package lnd

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/rs/zerolog/log"
	"go.uber.org/ratelimit"
	"google.golang.org/grpc"

	"github.com/lncapital/torq/internal/channels"
	"github.com/lncapital/torq/internal/graph_events"
	"github.com/lncapital/torq/internal/nodes"
	"github.com/lncapital/torq/pkg/broadcast"
	"github.com/lncapital/torq/pkg/commons"
)

type subscribeChannelGrpahClient interface {
	SubscribeChannelGraph(ctx context.Context, in *lnrpc.GraphTopologySubscription,
		opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeChannelGraphClient, error)
}

// SubscribeAndStoreChannelGraph Subscribes to channel updates
func SubscribeAndStoreChannelGraph(ctx context.Context, client subscribeChannelGrpahClient, db *sqlx.DB,
	nodeSettings commons.ManagedNodeSettings, eventChannel chan interface{}) error {

	req := lnrpc.GraphTopologySubscription{}
	stream, err := client.SubscribeChannelGraph(ctx, &req)
	if err != nil {
		return errors.Wrap(err, "LND Subscribe Channel Graph")
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
			log.Error().Err(err).Msgf("Subscribe channel graph stream receive")
			// rate limited resubscribe
			log.Info().Msg("Attempting reconnect to channel graph")
			for {
				rl.Take()
				stream, err = client.SubscribeChannelGraph(ctx, &req)
				if err == nil {
					log.Info().Msg("Reconnected to channel graph")
					break
				}
				log.Debug().Err(err).Msg("Reconnecting to channel graph")
			}
			continue
		}

		err = processNodeUpdates(gpu.NodeUpdates, db, nodeSettings, eventChannel)
		if err != nil {
			return errors.Wrap(err, "Process node updates")
		}

		err = processChannelUpdates(gpu.ChannelUpdates, db, nodeSettings, eventChannel)
		if err != nil {
			return errors.Wrap(err, "Process channel updates")
		}

	}

	return nil
}

func processNodeUpdates(nus []*lnrpc.NodeUpdate, db *sqlx.DB, nodeSettings commons.ManagedNodeSettings,
	eventChannel chan interface{}) error {
	for _, nu := range nus {
		eventNodeId := commons.GetActiveNodeIdFromPublicKey(nu.IdentityKey, nodeSettings.Chain, nodeSettings.Network)
		if eventNodeId != 0 {
			err := insertNodeEvent(db, time.Now().UTC(), eventNodeId, nu.Alias, nu.Color,
				nu.NodeAddresses, nu.Features, nodeSettings.NodeId, eventChannel)
			if err != nil {
				return errors.Wrapf(err, "Insert node event")
			}
		}
	}
	return nil
}

func processChannelUpdates(cus []*lnrpc.ChannelEdgeUpdate, db *sqlx.DB,
	nodeSettings commons.ManagedNodeSettings, eventChannel chan interface{}) error {
	for _, cu := range cus {
		channelPoint, err := chanPointFromByte(cu.ChanPoint.GetFundingTxidBytes(), cu.ChanPoint.GetOutputIndex())
		if err != nil {
			return errors.Wrap(err, "Creating channel point from byte")
		}
		fundingTransactionHash, fundingOutputIndex := channels.ParseChannelPoint(channelPoint)

		channelId := commons.GetActiveChannelIdFromFundingTransaction(fundingTransactionHash, fundingOutputIndex)
		if channelId != 0 {
			err := insertRoutingPolicy(db, time.Now().UTC(), channelId, nodeSettings, cu, eventChannel)
			if err != nil {
				return errors.Wrap(err, "Insert routing policy")
			}
		}
	}
	return nil
}

func insertRoutingPolicy(
	db *sqlx.DB,
	eventTime time.Time,
	channelId int,
	nodeSettings commons.ManagedNodeSettings,
	cu *lnrpc.ChannelEdgeUpdate,
	eventChannel chan interface{}) error {

	var err error
	if cu == nil || cu.RoutingPolicy == nil {
		log.Warn().Msg("Routing policy nil, skipping")
		return nil
	}

	announcingNodeId := commons.GetNodeIdFromPublicKey(cu.AdvertisingNode, nodeSettings.Chain, nodeSettings.Network)
	if announcingNodeId == 0 {
		newNode := nodes.Node{
			PublicKey: cu.AdvertisingNode,
			Chain:     nodeSettings.Chain,
			Network:   nodeSettings.Network,
		}
		announcingNodeId, err = nodes.AddNodeWhenNew(db, newNode)
		if err != nil {
			return errors.Wrapf(err, "Adding node (publicKey: %v)", cu.AdvertisingNode)
		}
	}
	connectingNodeId := commons.GetNodeIdFromPublicKey(cu.ConnectingNode, nodeSettings.Chain, nodeSettings.Network)
	if connectingNodeId == 0 {
		newNode := nodes.Node{
			PublicKey: cu.ConnectingNode,
			Chain:     nodeSettings.Chain,
			Network:   nodeSettings.Network,
		}
		connectingNodeId, err = nodes.AddNodeWhenNew(db, newNode)
		if err != nil {
			return errors.Wrapf(err, "Adding node (publicKey: %v shortChannelId: %v)", cu.ConnectingNode)
		}
	}

	channelEvent := graph_events.ChannelEventFromGraph{}
	err = db.Get(&channelEvent, `
				SELECT *
				FROM routing_policy
				WHERE channel_id=$1 AND announcing_node_id=$2 AND connecting_node_id=$3
				ORDER BY ts DESC
				LIMIT 1;`, channelId, announcingNodeId, connectingNodeId)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return errors.Wrapf(err, "insertNodeEvent -> getPreviousChannelEvent.")
		}
	}

	// If one of our active torq nodes is announcing_node_id then the channel update was by our node
	// TODO FIXME ignore if previous update was from the same node so if announcing_node_id=node_id on previous record
	// and the current parameters are announcing_node_id!=node_id
	if cu.RoutingPolicy.Disabled != channelEvent.Disabled ||
		cu.RoutingPolicy.FeeBaseMsat != channelEvent.FeeBaseMsat ||
		cu.RoutingPolicy.FeeRateMilliMsat != channelEvent.FeeRateMilliMsat ||
		cu.RoutingPolicy.MaxHtlcMsat != channelEvent.MaxHtlcMsat ||
		cu.RoutingPolicy.MinHtlc != channelEvent.MinHtlc ||
		cu.RoutingPolicy.TimeLockDelta != channelEvent.TimeLockDelta {

		_, err := db.Exec(`
		INSERT INTO routing_policy
			(ts,disabled,time_lock_delta,min_htlc,max_htlc_msat,fee_base_msat,fee_rate_mill_msat,
			 channel_id,announcing_node_id,connecting_node_id,node_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);`, eventTime,
			cu.RoutingPolicy.Disabled, cu.RoutingPolicy.TimeLockDelta, cu.RoutingPolicy.MinHtlc,
			cu.RoutingPolicy.MaxHtlcMsat, cu.RoutingPolicy.FeeBaseMsat, cu.RoutingPolicy.FeeRateMilliMsat,
			channelId, announcingNodeId, connectingNodeId, nodeSettings.NodeId)
		if err != nil {
			return errors.Wrapf(err, "insertRoutingPolicy")
		}

		if eventChannel != nil {
			channelGraphEvent := broadcast.ChannelGraphEvent{
				GraphEventData: broadcast.GraphEventData{
					EventData: broadcast.EventData{
						EventTime: eventTime,
						NodeId:    nodeSettings.NodeId,
					},
					AnnouncingNodeId: &announcingNodeId,
					ConnectingNodeId: &connectingNodeId,
					ChannelId:        &channelId,
				},
				ChannelGraphEventData: broadcast.ChannelGraphEventData{
					TimeLockDelta:    cu.RoutingPolicy.TimeLockDelta,
					FeeRateMilliMsat: cu.RoutingPolicy.FeeRateMilliMsat,
					FeeBaseMsat:      cu.RoutingPolicy.FeeBaseMsat,
					MaxHtlcMsat:      cu.RoutingPolicy.MaxHtlcMsat,
					Disabled:         cu.RoutingPolicy.Disabled,
					MinHtlc:          cu.RoutingPolicy.MinHtlc,
				},
			}
			if channelEvent.ChannelId != 0 {
				channelGraphEvent.PreviousEventTime = channelEvent.EventTime
				channelGraphEvent.PreviousEventData = broadcast.ChannelGraphEventData{
					TimeLockDelta:    channelEvent.TimeLockDelta,
					FeeRateMilliMsat: channelEvent.FeeRateMilliMsat,
					FeeBaseMsat:      channelEvent.FeeBaseMsat,
					MaxHtlcMsat:      channelEvent.MaxHtlcMsat,
					Disabled:         channelEvent.Disabled,
					MinHtlc:          channelEvent.MinHtlc,
				}
			}
			eventChannel <- channelGraphEvent
		}
	}
	return nil
}

func insertNodeEvent(db *sqlx.DB, eventTime time.Time, eventNodeId int, alias string, color string,
	nodeAddress []*lnrpc.NodeAddress, features map[uint32]*lnrpc.Feature, nodeId int, eventChannel chan interface{}) error {

	// Create json byte object from node address map
	najb, err := json.Marshal(nodeAddress)
	if err != nil {
		return errors.Wrap(err, "JSON Marshall node address map")
	}

	// Create json byte object from features list
	fjb, err := json.Marshal(features)
	if err != nil {
		return errors.Wrap(err, "JSON Marshal feature list")
	}

	nodeEvent := graph_events.NodeEventFromGraph{}
	err = db.Get(&nodeEvent, `
				SELECT *
				FROM node_event
				WHERE event_node_id=$1
				ORDER BY timestamp DESC
				LIMIT 1;`, eventNodeId)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return errors.Wrapf(err, "insertNodeEvent -> getPreviousNodeEvent.")
		}
	}
	// TODO FIXME ignore if previous update was from the same node so if event_node_id=node_id on previous record
	// and the current parameters are event_node_id!=node_id
	if alias != nodeEvent.Alias ||
		color != nodeEvent.Color ||
		string(najb) != nodeEvent.NodeAddresses ||
		string(fjb) != nodeEvent.Features {

		_, err = db.Exec(`INSERT INTO node_event
    		(timestamp, event_node_id, alias, color, node_addresses, features, node_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7);`,
			eventTime, eventNodeId, alias, color, najb, fjb, nodeId)
		if err != nil {
			return errors.Wrap(err, "Executing SQL")
		}

		if eventChannel != nil {
			nodeGraphEvent := broadcast.NodeGraphEvent{
				GraphEventData: broadcast.GraphEventData{
					EventData: broadcast.EventData{
						EventTime: eventTime,
						NodeId:    nodeId,
					},
					EventNodeId: &eventNodeId,
				},
				NodeGraphEventData: broadcast.NodeGraphEventData{
					Alias:     alias,
					Color:     color,
					Addresses: string(najb),
					Features:  string(fjb),
				},
			}
			if nodeEvent.NodeId != 0 {
				nodeGraphEvent.PreviousEventTime = nodeEvent.EventTime
				nodeGraphEvent.PreviousEventData = broadcast.NodeGraphEventData{
					Alias:     nodeEvent.Alias,
					Color:     nodeEvent.Color,
					Addresses: nodeEvent.NodeAddresses,
					Features:  nodeEvent.Features,
				}
			}
			eventChannel <- nodeGraphEvent
		}
	}
	return nil
}
