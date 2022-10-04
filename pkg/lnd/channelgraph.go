package lnd

import (
	"context"
	"encoding/json"
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/internal/channels"
	"github.com/rs/zerolog/log"
	"go.uber.org/ratelimit"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"time"
)

type subscribeChannelGrpahClient interface {
	SubscribeChannelGraph(ctx context.Context, in *lnrpc.GraphTopologySubscription,
		opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeChannelGraphClient, error)
}

// SubscribeAndStoreChannelGraph Subscribes to channel updates
func SubscribeAndStoreChannelGraph(ctx context.Context, client subscribeChannelGrpahClient, db *sqlx.DB, ourNodePubKeys []string) error {

	req := lnrpc.GraphTopologySubscription{}
	stream, err := client.SubscribeChannelGraph(ctx, &req)

	if err != nil {
		return errors.Wrapf(err, "SubscribeAndStoreChannelGraph -> client.SubscribeChannelGraph(%v, %v)", ctx, req)
	}

	rl := ratelimit.New(1) // 1 per second maximum rate limit

	for {

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		gpu, err := stream.Recv()

		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				break
			}
			log.Error().Msgf("Subscribe channel graph stream receive: %v\n", err)
			// rate limited resubscribe
			log.Info().Msg("Attempting reconnect to channel graph")
			for {
				rl.Take()
				stream, err = client.SubscribeChannelGraph(ctx, &req)
				if err == nil {
					log.Info().Msg("Reconnected to channel graph")
					break
				}
				log.Printf("Reconnecting to channel graph: %v\n", err)
			}
			continue
		}

		err = processNodeUpdates(gpu.NodeUpdates, db, ourNodePubKeys)
		if err != nil {
			return errors.Wrap(err, "Process node updates")
		}

		err = processChannelUpdates(gpu.ChannelUpdates, db, ourNodePubKeys)
		if err != nil {
			return errors.Wrap(err, "Process channel updates")
		}

	}

	return nil
}

func processNodeUpdates(nus []*lnrpc.NodeUpdate, db *sqlx.DB, ourNodePubKeys []string) error {

	for _, nu := range nus {
		// Check if this node update is relevant to a node we have or have had a channel with
		relevant, _ := isRelevantOrOurNode(nu.IdentityKey, ourNodePubKeys)

		if relevant {
			ts := time.Now().UTC()
			err := insertNodeEvent(db, ts, nu.IdentityKey, nu.Alias, nu.Color,
				nu.NodeAddresses, nu.Features)
			if err != nil {
				return errors.Wrapf(err, "processNodeUpdates ->insertNodeEvent(%v, %s, %s, %s, %s, %v, %v)",
					db, ts, nu.IdentityKey, nu.Alias, nu.Color, nu.NodeAddresses, nu.Features)
			}
		}

	}

	return nil
}

func processChannelUpdates(cus []*lnrpc.ChannelEdgeUpdate, db *sqlx.DB, ourNodePubKeys []string) error {
	for _, cu := range cus {
		// Check if this channel update is relevant to one of our channels
		// And if one of our nodes is advertising the channel update (meaning
		// we have changed our the channel policy).
		ourNode := slices.Contains(ourNodePubKeys, cu.AdvertisingNode)

		chanPoint, err := chanPointFromByte(cu.ChanPoint.GetFundingTxidBytes(), cu.ChanPoint.GetOutputIndex())
		if err != nil {
			return errors.Wrapf(err, "SubscribeChannelEvents ->getChanPoint(%b, %d)",
				cu.ChanPoint.GetFundingTxidBytes(), cu.ChanPoint.GetOutputIndex())
		}
		relevantChannel := isRelevantChannel(chanPoint)

		if relevantChannel {
			ts := time.Now().UTC()
			err := insertRoutingPolicy(db, ts, ourNode, cu)
			if err != nil {
				return errors.Wrapf(err, "SubscribeChannelEvents ->insertRoutingPolicy(%v, %s, %t, %v)",
					db, ts, ourNode, cu)
			}
		}

	}

	return nil
}

const rpQuery = `
INSERT INTO routing_policy (ts,
	lnd_short_channel_id,
	short_channel_id,
	announcing_pub_key,
	lnd_channel_point,
	outbound,
	disabled,
	time_lock_delta,
	min_htlc,
	max_htlc_msat,
	fee_base_msat,
	fee_rate_mill_msat)
select $1, $2, $3,$4, $5, $6, $7, $8, $9, $10, $11, $12
WHERE NOT EXISTS (
	select true
	from (select
            last(lnd_short_channel_id,ts) lnd_short_channel_id,
            last(short_channel_id,ts) short_channel_id,
			last(announcing_pub_key, ts) as announcing_pub_key,
			last(disabled,ts) disabled,
			last(time_lock_delta,ts) time_lock_delta,
			last(min_htlc,ts) min_htlc,
			last(max_htlc_msat,ts) max_htlc_msat,
			last(fee_base_msat,ts) fee_base_msat,
			last(fee_rate_mill_msat, ts) fee_rate_mill_msat
		from routing_policy
		group by lnd_short_channel_id, announcing_pub_key) as a
	where a.lnd_short_channel_id = $13 and
		  a.announcing_pub_key = $14 and
		  a.disabled = $15 and
		  a.time_lock_delta = $16 and
		  a.min_htlc = $17 and
		  a.max_htlc_msat = $18 and
		  a.fee_base_msat = $19 and
		  a.fee_rate_mill_msat = $20
);`

func insertRoutingPolicy(db *sqlx.DB, ts time.Time, outbound bool, cu *lnrpc.ChannelEdgeUpdate) error {

	if cu == nil || cu.RoutingPolicy == nil {
		log.Warn().Msg("Routing policy nil, skipping")
		return nil
	}

	cp, err := chanPointFromByte(cu.ChanPoint.GetFundingTxidBytes(), cu.ChanPoint.GetOutputIndex())
	if err != nil {
		return errors.Wrapf(err, "insertRoutingPolicy -> getChanPoint(%v, %d)",
			cu.ChanPoint.GetFundingTxidBytes(), cu.ChanPoint.GetOutputIndex())
	}

	shortChannelId := channels.ConvertLNDShortChannelID(cu.ChanId)
	// Check if the routing policy is unchanged

	_, err = db.Exec(rpQuery, ts, cu.ChanId, shortChannelId, cu.AdvertisingNode, cp, outbound,
		cu.RoutingPolicy.Disabled, cu.RoutingPolicy.TimeLockDelta, cu.RoutingPolicy.MinHtlc,
		cu.RoutingPolicy.MaxHtlcMsat, cu.RoutingPolicy.FeeBaseMsat, cu.RoutingPolicy.FeeRateMilliMsat,
		// Variables to check if it exists
		cu.ChanId, cu.AdvertisingNode, cu.RoutingPolicy.Disabled, cu.RoutingPolicy.TimeLockDelta, cu.RoutingPolicy.MinHtlc,
		cu.RoutingPolicy.MaxHtlcMsat, cu.RoutingPolicy.FeeBaseMsat, cu.RoutingPolicy.FeeRateMilliMsat)

	if err != nil {
		return errors.Wrapf(err, "insertRoutingPolicy -> db.Exec(%s)", rpQuery)
	}

	return nil
}

const neQuery = `INSERT INTO node_event (timestamp, pub_key, alias, color, node_addresses, features)
SELECT $1,$2,$3,$4,$5,$6
WHERE NOT EXISTS (
select true
from (select pub_key,
        last(alias, timestamp) as  alias,
        last(color,timestamp) as color,
        last(node_addresses,timestamp) as node_addresses,
        last(features,timestamp) as features
    from node_event
    group by pub_key) as a
where a.pub_key = $2 and
      a.alias = $3 and
      a.color = $4 and
      a.node_addresses = $5 and
      a.features = $6
);`

func insertNodeEvent(db *sqlx.DB, ts time.Time, pubKey string, alias string, color string,
	na []*lnrpc.NodeAddress, f map[uint32]*lnrpc.Feature) error {

	// Create json byte object from node address map
	najb, err := json.Marshal(na)
	if err != nil {
		return errors.Wrapf(err, "insertNodeEvent -> json.Marshal(%v)", na)
	}

	// Create json byte object from features list
	fjb, err := json.Marshal(f)
	if err != nil {
		return errors.Wrapf(err, "insertNodeEvent -> json.Marshal(%v)", f)
	}

	db.Exec(neQuery, ts, pubKey, alias, color, najb, fjb)

	return nil
}

func AddOpenChanPoint(chanPoint string)      { addOpenChanPointChan <- chanPoint }
func RemoveClosedChanPoint(chanPoint string) { removeClosedChanPointChan <- chanPoint }
func GetOpenChanPoints() []string            { return <-getOpenChanPointsChan }

var addOpenChanPointChan = make(chan string)
var removeClosedChanPointChan = make(chan string)
var getOpenChanPointsChan = make(chan []string)

func OpenChanPointListMonitor(ctx context.Context) {
	var openChanPointList []string
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("Chan point list monitor is closing")
			return
		case chanPoint := <-addOpenChanPointChan:
			if !slices.Contains(openChanPointList, chanPoint) {
				openChanPointList = append(openChanPointList, chanPoint)
			}
		case chanPoint := <-removeClosedChanPointChan:
			index := slices.Index(openChanPointList, chanPoint)
			if index != -1 {
				openChanPointList = append(openChanPointList[:index], openChanPointList[index+1:]...)
			}
		case getOpenChanPointsChan <- openChanPointList:
		}
	}
}

func InitChanPointList(db *sqlx.DB) error {
	q := `
		select array_agg(lnd_channel_point) as lnd_channel_point from (
			select
				last(event_type, time) as event_type,
				last(lnd_channel_point,time) as lnd_channel_point
			from channel_event
			where event_type in(0,1)
			group by lnd_channel_point
		) as t
		where t.event_type = 0;`

	var chanPointList []string
	err := db.QueryRowx(q).Scan(pq.Array(&chanPointList))
	if err != nil {
		return errors.Wrap(err, "Query row into chanPointList")
	}
	for _, chanPoint := range chanPointList {
		AddOpenChanPoint(chanPoint)
	}

	return nil
}

func addMissingLocalPubkey(ctx context.Context, client lnrpc.LightningClient, grpcAddress string,
	db *sqlx.DB) (r *string, err error) {

	// Get the public key of our node
	// TODO: Update this when adding support for multiple nodes
	ni, err := client.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, errors.Wrapf(err, "client.GetNodeInfo(ctx, &lnrpc.NodeInfoRequest{})")
	}

	const q = `update local_node set(pub_key, updated_on) = ($1, $2) where grpc_address = $3`

	_, err = db.Exec(q,
		ni.IdentityPubkey,
		time.Now().UTC(),
		grpcAddress,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "tx.Exec(%v, %v, %v, %v)",
			q,
			ni.IdentityPubkey,
			time.Now().UTC(),
			grpcAddress,
		)
	}

	return &ni.IdentityPubkey, nil
}

func InitOurNodesList(ctx context.Context, client lnrpc.LightningClient, db *sqlx.DB) (ourNodePubKeys []string, err error) {

	var pubKey *string
	var grpcAddress *string

	q := `select grpc_address, pub_key from local_node;`
	r, err := db.Query(q)

	ourNodePubKeys = []string{}
	for r.Next() {
		err = r.Scan(&grpcAddress, &pubKey)
		if err != nil {
			return []string{}, errors.Wrapf(err, "Reading grpc_address and pub_key from db")
		}

		if grpcAddress == nil || *grpcAddress == "" {
			continue
		}
		// If the pub key is missing from the local_node table, add it.
		if pubKey == nil || len(*pubKey) == 0 {
			pubKey, err = addMissingLocalPubkey(ctx, client, *grpcAddress, db)
			if err != nil {
				return []string{}, errors.Wrapf(err, "addMissingLocalPubkey(ctx, client, grpcAddress, db)")
			}
		}
		ourNodePubKeys = append(ourNodePubKeys, *pubKey)
	}
	if err != nil {
		return []string{}, errors.Wrapf(err, "db.Query(%s)", q)
	}

	return ourNodePubKeys, nil
}

// InitPeerList fetches all public keys from the list of all channels. This is used to
// filter out noise from the graph updates.
func InitPeerList(db *sqlx.DB) error {
	q := `select array_agg(distinct pub_key) as all_nodes from channel_event where event_type in (0, 1);`
	var peerPubKeyList []string
	err := db.QueryRow(q).Scan(pq.Array(&peerPubKeyList))
	if err != nil {
		return errors.Wrap(err, "Selecting distinct pub keys from channel_event table")
	}
	for _, peerPubKey := range peerPubKeyList {
		AddPeerPubKey(peerPubKey)
	}

	return nil
}

func AddPeerPubKey(pubKey string) { addPeerPubKeyChan <- pubKey }
func GetPeerPubKeys() []string    { return <-getPeerPubKeysChan }

var addPeerPubKeyChan = make(chan string)
var getPeerPubKeysChan = make(chan []string)

func PeerPubKeyListMonitor(ctx context.Context) {
	// pubKeyList is used to store which node and channel updates to store. We only want to store
	// updates that are relevant to our channels and their nodes.
	var pubKeyList []string
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("Peer pub key list monitor is closing")
			return
		case pubKey := <-addPeerPubKeyChan:
			if !slices.Contains(pubKeyList, pubKey) {
				pubKeyList = append(pubKeyList, pubKey)
			}
		case getPeerPubKeysChan <- pubKeyList:
		}
	}
}

// isRelevantOrOurNode is used to check if any public key is in the pubKeyList.
// The first boolean returned indicate if the key is relevant, the second boolean
// indicates that it is one of our own nodes.
func isRelevantOrOurNode(pubKey string, ourNodePubKeys []string) (bool, bool) {

	if slices.Contains(ourNodePubKeys, pubKey) {
		// Is relevant (first boolean), _and_ our node (second boolean).
		return true, true
	}

	if isRelevant(pubKey) {
		// Is relevant (first boolean), _but not_ our node (second boolean).
		return true, false
	}

	return false, false
}

func isRelevantChannel(chanPoint string) bool {
	for _, c := range GetOpenChanPoints() {
		if c == chanPoint {
			return true
		}
	}
	return false
}

// isRelevant is used to check if any public key is in the pubKeyList.
func isRelevant(pubKey string) bool {

	for _, p := range GetPeerPubKeys() {

		// Check if any of the provided public keys equals the current public key.
		if p == pubKey {
			// If found, no reason to search further, immediately return true.
			return true
		}

	}

	return false
}
