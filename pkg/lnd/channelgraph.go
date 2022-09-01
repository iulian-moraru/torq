package lnd

import (
	"context"
	"encoding/json"
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/internal/channels"
	"go.uber.org/ratelimit"
	"google.golang.org/grpc"
	"log"
	"time"
)

type subscribeChannelGrpahClient interface {
	SubscribeChannelGraph(ctx context.Context, in *lnrpc.GraphTopologySubscription,
		opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeChannelGraphClient, error)
}

// SubscribeAndStoreChannelGraph Subscribes to channel updates
func SubscribeAndStoreChannelGraph(ctx context.Context, client subscribeChannelGrpahClient, db *sqlx.DB) error {

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
			if errors.As(err, &context.Canceled) {
				break
			}
			log.Printf("Subscribe channel graph stream receive: %v\n", err)
			// rate limited resubscribe
			log.Println("Attempting reconnect to channel graph")
			for {
				rl.Take()
				stream, err = client.SubscribeChannelGraph(ctx, &req)
				if err == nil {
					log.Println("Reconnected to channel graph")
					break
				}
				log.Printf("Reconnecting to channel graph: %v\n", err)
			}
			continue
		}

		err = processNodeUpdates(gpu.NodeUpdates, db)
		if err != nil {
			return errors.Wrap(err, "Process node updates")
		}

		err = processChannelUpdates(gpu.ChannelUpdates, db)
		if err != nil {
			return errors.Wrap(err, "Process channel updates")
		}

	}

	return nil
}

func processNodeUpdates(nus []*lnrpc.NodeUpdate, db *sqlx.DB) error {

	for _, nu := range nus {
		// Check if this node update is relevant to a node we have or have had a channel with
		relevant, _ := isRelevantOrOurNode(nu.IdentityKey)

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

func processChannelUpdates(cus []*lnrpc.ChannelEdgeUpdate, db *sqlx.DB) error {
	for _, cu := range cus {
		// Check if this channel update is relevant to one of our channels
		// And if one of our nodes is advertising the channel update (meaning
		// we have changed our the channel policy).
		ourNode := isOurNode(cu.AdvertisingNode)

		chanPoint, err := getChanPoint(cu.ChanPoint.GetFundingTxidBytes(), cu.ChanPoint.GetOutputIndex())
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

	cp, err := getChanPoint(cu.ChanPoint.GetFundingTxidBytes(), cu.ChanPoint.GetOutputIndex())
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

var chanPointList []string

func InitChanIdList(db *sqlx.DB) error {
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

	err := db.QueryRowx(q).Scan(pq.Array(&chanPointList))
	if err != nil {
		return errors.Wrapf(err, "InitChanIdList -> db.QueryRow(%s).Scan(pq.Array(%v))", q, chanPointList)
	}

	return nil
}

func UpdateChanIdList(c chan string) {

	var chanPoint string

waitForUpdate:
	for {
		// Wait for new peers to enter
		chanPoint = <-c

		// Remove chanPoint to the list, if it's already present.
		// continue the outer loop and wait for new update
		for i, cp := range chanPointList {
			if cp == chanPoint {
				chanPointList = append(chanPointList[:i], chanPointList[i+1:]...)
				continue waitForUpdate
			}
		}

		// If not present add it to the chanID list
		chanPointList = append(chanPointList, chanPoint)

	}
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

var ourNodePubKeys []string

func InitOurNodesList(ctx context.Context, client lnrpc.LightningClient, db *sqlx.DB) error {

	var pubKey *string
	var grpcAddress string

	q := `select grpc_address, pub_key from local_node;`
	r, err := db.Query(q)

	for r.Next() {
		err := r.Scan(&grpcAddress, &pubKey)
		if err != nil {
			return errors.Wrapf(err, "r.Scan(&grpcAddress, &pubKey)")
		}

		// If the pub key is missing from the local_node table, add it.
		if pubKey == nil || len(*pubKey) == 0 {
			pubKey, err = addMissingLocalPubkey(ctx, client, grpcAddress, db)
			if err != nil {
				return errors.Wrapf(err, "addMissingLocalPubkey(ctx, client, grpcAddress, db)")
			}
		}
		ourNodePubKeys = append(ourNodePubKeys, *pubKey)
	}
	if err != nil {
		return errors.Wrapf(err, "db.Query(%s)", q)
	}

	return nil
}

// pubKeyList is used to store which node and channel updates to store. We only want to store
// updates that are relevant to our channels and their nodes.
var pubKeyList []string

// InitPeerList fetches all public keys from the list of all channels. This is used to
// filter out noise from the graph updates.
func InitPeerList(db *sqlx.DB) error {
	q := `select array_agg(distinct pub_key) as all_nodes from channel_event where event_type in (0, 1);`
	err := db.QueryRow(q).Scan(pq.Array(&pubKeyList))
	if err != nil {
		return errors.Wrapf(err, "InitPeerList -> db.QueryRow(%s).Scan(pq.Array(%v))", q, pubKeyList)
	}

	return nil
}

// UpdatePeerList is meant to run as a gorouting. It adds new public keys to the pubKeyList
// and removes existing ones.
func UpdatePeerList(c chan string) {

	var pubKey string
	var present bool

	for {
		// Wait for new peers to enter
		pubKey = <-c
		// Add it to the peer list, if not already present.
		for _, p := range pubKeyList {
			if p == pubKey {
				present = true
				break
			}
		}

		// If not present add it to the public key  list
		if present == false {
			pubKeyList = append(pubKeyList, pubKey)
		}

		// Reset to false in order to allow the next public key to be added.
		present = false
	}
}

// isRelevantOrOurNode is used to check if any public key is in the pubKeyList.
// The first boolean returned indicate if the key is relevant, the second boolean
// indicates that it is one of our own nodes.
func isRelevantOrOurNode(pubKey string) (bool, bool) {

	if isOurNode(pubKey) {
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
	for _, cid := range chanPointList {
		if cid == chanPoint {
			return true
		}
	}
	return false
}

// isRelevant is used to check if any public key is in the pubKeyList.
func isRelevant(pubKey string) bool {

	for _, p := range pubKeyList {

		// Check if any of the provided public keys equals the current public key.
		if p == pubKey {
			// If found, no reason to search further, immediately return true.
			return true
		}

	}

	return false
}

// isOurNode is used to check if the public key is from one of our own nodes.
func isOurNode(pubKey string) bool {

	for _, p := range ourNodePubKeys {

		// Check if the public key belongs to one of our nodes.
		if p == pubKey {
			// If found, no reason to search further, immediately return true, and true
			return true
		}

	}
	return false
}
