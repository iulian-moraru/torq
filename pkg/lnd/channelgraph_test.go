package lnd

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/testutil"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"testing"
	"time"
)

type stubLNDSubscribeChannelGraphRPC struct {
	grpc.ClientStream
	GraphTopologyUpdate []*lnrpc.GraphTopologyUpdate
}

func (s *stubLNDSubscribeChannelGraphRPC) Recv() (*lnrpc.GraphTopologyUpdate, error) {
	if len(s.GraphTopologyUpdate) == 0 {
		return nil, context.Canceled
	}
	var gtu interface{}
	gtu, s.GraphTopologyUpdate = s.GraphTopologyUpdate[0], s.GraphTopologyUpdate[1:]
	if u, ok := gtu.(*lnrpc.GraphTopologyUpdate); ok {
		return u, nil
	}
	return nil, context.Canceled
}

func (s *stubLNDSubscribeChannelGraphRPC) SubscribeChannelGraph(ctx context.Context, in *lnrpc.GraphTopologySubscription,
	opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeChannelGraphClient, error) {

	return &stubLNDSubscribeChannelGraphRPC{GraphTopologyUpdate: s.GraphTopologyUpdate}, nil
}

func TestSubscribeChannelGraphUpdates(t *testing.T) {
	srv, err := testutil.InitTestDBConn()
	if err != nil {
		panic(err)
	}

	db, err := srv.NewTestDatabase(true)
	if err != nil {
		t.Fatal(err)
	}

	chanPoint := &lnrpc.ChannelPoint{FundingTxid: &lnrpc.
		ChannelPoint_FundingTxidBytes{
		FundingTxidBytes: []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
		OutputIndex: 1}

	chanPointStr, err := getChanPoint([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, chanPoint.OutputIndex)
	if err != nil {
		t.Fatal(err)
	}

	// Add our public key to the list
	ourNodePubKeys = append(ourNodePubKeys, "ourNodePubkey")

	// Add our public key to the list
	pubKeyList = append(pubKeyList, "firstNodePubkey", "secondNodePubkey")

	// Add the channel to the list of relevant channels
	chanPointList = append(chanPointList, chanPointStr)

	t.Run("Irrelevant routing policy updates are ignored", func(t *testing.T) {

		irrelevatChannelPoint := &lnrpc.ChannelPoint{FundingTxid: &lnrpc.
			ChannelPoint_FundingTxidBytes{
			FundingTxidBytes: []byte{2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1}},
			OutputIndex: 2}

		irrelecantUpdateEvent := lnrpc.GraphTopologyUpdate{
			NodeUpdates: nil,
			ChannelUpdates: []*lnrpc.ChannelEdgeUpdate{{
				ChanId:    12345678,
				ChanPoint: irrelevatChannelPoint,
				Capacity:  2000000,
				RoutingPolicy: &lnrpc.RoutingPolicy{
					TimeLockDelta:    0,
					MinHtlc:          0,
					FeeBaseMsat:      2,
					FeeRateMilliMsat: 200,
					Disabled:         true,
					MaxHtlcMsat:      1000,
					LastUpdate:       0,
				},
				AdvertisingNode: "secondNodePubkey",
				ConnectingNode:  "secondIrrelevantPubkey",
			}},
			ClosedChans: nil,
		}

		result := simulateChannelGraphUpdate(t, db, &stubLNDSubscribeChannelGraphRPC{GraphTopologyUpdate: []*lnrpc.
			GraphTopologyUpdate{&irrelecantUpdateEvent}})

		if len(result) != 0 {
			testutil.Fatalf(t, "Expected to find no routing policy record stored in the database. Found %d",
				len(result))
		}
	})

	t.Run("Relevant routing policies are correctly stored", func(t *testing.T) {

		updateEvent := lnrpc.GraphTopologyUpdate{
			NodeUpdates: nil,
			ChannelUpdates: []*lnrpc.ChannelEdgeUpdate{{
				ChanId:    1234567,
				ChanPoint: chanPoint,
				Capacity:  1000000,
				RoutingPolicy: &lnrpc.RoutingPolicy{
					TimeLockDelta:    0,
					MinHtlc:          0,
					FeeBaseMsat:      2,
					FeeRateMilliMsat: 100,
					Disabled:         true,
					MaxHtlcMsat:      1000,
					LastUpdate:       0,
				},
				AdvertisingNode: "theirNodePubkey",
				ConnectingNode:  "ourNodePubkey",
			}},
			ClosedChans: nil,
		}

		expected := routingPolicyData{
			Ts:                time.Now(),
			LNDChannelPoint:   chanPointStr,
			LNDShortChannelId: fmt.Sprint(updateEvent.ChannelUpdates[0].ChanId),
			Outbound:          false,
			AnnouncingPubKey:  updateEvent.ChannelUpdates[0].AdvertisingNode,
			FeeRateMillMsat:   updateEvent.ChannelUpdates[0].RoutingPolicy.FeeRateMilliMsat,
			FeeBaseMsat:       updateEvent.ChannelUpdates[0].RoutingPolicy.FeeBaseMsat,
			MaxHtlcMsat:       updateEvent.ChannelUpdates[0].RoutingPolicy.MaxHtlcMsat,
			MinHtlc:           updateEvent.ChannelUpdates[0].RoutingPolicy.MinHtlc,
			TimeLockDelta:     updateEvent.ChannelUpdates[0].RoutingPolicy.TimeLockDelta,
			Disabled:          updateEvent.ChannelUpdates[0].RoutingPolicy.Disabled,
		}

		result := simulateChannelGraphUpdate(t, db,
			&stubLNDSubscribeChannelGraphRPC{GraphTopologyUpdate: []*lnrpc.GraphTopologyUpdate{&updateEvent}},
		)

		if len(result) != 1 {
			testutil.Fatalf(t, "Expected to find a single routing policy record stored in the database. Found %d",
				len(result))
		}

		if result[0].AnnouncingPubKey != expected.AnnouncingPubKey {
			testutil.Errorf(t, "Incorrect announcing pub key. Expected: %v, got %v", expected.AnnouncingPubKey,
				result[0].AnnouncingPubKey)
		}

		if result[0].LNDChannelPoint != expected.LNDChannelPoint {
			testutil.Errorf(t, "Incorrect channel point. Expected: %v, got %v", expected.LNDChannelPoint, result[0].LNDChannelPoint)
		}

		if result[0].LNDShortChannelId != expected.LNDShortChannelId {
			testutil.Errorf(t, "Incorrect channel id. Expected: %v, got %v", expected.LNDShortChannelId, result[0].LNDShortChannelId)
		}

		if result[0].Disabled != expected.Disabled {
			testutil.Errorf(t, "Incorrect channel disabled state. Expected: %v, got %v", expected.Disabled,
				result[0].Disabled)
		}

		if result[0].FeeRateMillMsat != expected.FeeRateMillMsat {
			testutil.Errorf(t, "Incorrect fee rate. Expected: %v, got %v", expected.FeeRateMillMsat,
				result[0].FeeRateMillMsat)
		}

		if result[0].FeeBaseMsat != expected.FeeBaseMsat {
			testutil.Errorf(t, "Incorrect base fee state. Expected: %v, got %v", expected.FeeBaseMsat,
				result[0].FeeBaseMsat)
		}

		if result[0].MinHtlc != expected.MinHtlc {
			testutil.Errorf(t, "Incorrect min htlc. Expected: %v, got %v", expected.MinHtlc, result[0].MinHtlc)
		}

		if result[0].MaxHtlcMsat != expected.MaxHtlcMsat {
			testutil.Errorf(t, "Incorrect max htlc. Expected: %v, got %v", expected.MaxHtlcMsat, result[0].MaxHtlcMsat)
		}

		if result[0].Outbound != expected.Outbound {
			testutil.Errorf(t, "Incorrect outbound state. Expected: %v, got %v", expected.Outbound, result[0].Outbound)
		}

		if result[0].TimeLockDelta != expected.TimeLockDelta {
			testutil.Errorf(t, "Incorrect timelock delta. Expected: %v, got %v", expected.TimeLockDelta,
				result[0].TimeLockDelta)
		}

		r2 := simulateChannelGraphUpdate(t, db, &stubLNDSubscribeChannelGraphRPC{GraphTopologyUpdate: []*lnrpc.
			GraphTopologyUpdate{&updateEvent}})

		if len(r2) != 1 {
			testutil.Fatalf(t, "Expected to find a single routing policy record stored in the database. Found %d",
				len(r2))
		}

		if t.Failed() {
			t.FailNow()
		}

		secondUpdateEvent := lnrpc.GraphTopologyUpdate{
			NodeUpdates: nil,
			ChannelUpdates: []*lnrpc.ChannelEdgeUpdate{{
				ChanId:    12345678,
				ChanPoint: chanPoint,
				Capacity:  2000000,
				RoutingPolicy: &lnrpc.RoutingPolicy{
					TimeLockDelta:    0,
					MinHtlc:          0,
					FeeBaseMsat:      2,
					FeeRateMilliMsat: 200,
					Disabled:         true,
					MaxHtlcMsat:      1000,
					LastUpdate:       0,
				},
				AdvertisingNode: "ourNodePubkey",
				ConnectingNode:  "secondIrrelevantPubkey",
			}},
			ClosedChans: nil,
		}

		e3 := routingPolicyData{
			Ts:                time.Now(),
			LNDChannelPoint:   chanPointStr,
			LNDShortChannelId: fmt.Sprint(secondUpdateEvent.ChannelUpdates[0].ChanId),
			Outbound:          true,
			AnnouncingPubKey:  "ourNodePubkey",
			FeeRateMillMsat:   secondUpdateEvent.ChannelUpdates[0].RoutingPolicy.FeeRateMilliMsat,
			FeeBaseMsat:       secondUpdateEvent.ChannelUpdates[0].RoutingPolicy.FeeBaseMsat,
			MaxHtlcMsat:       secondUpdateEvent.ChannelUpdates[0].RoutingPolicy.MaxHtlcMsat,
			MinHtlc:           secondUpdateEvent.ChannelUpdates[0].RoutingPolicy.MinHtlc,
			TimeLockDelta:     secondUpdateEvent.ChannelUpdates[0].RoutingPolicy.TimeLockDelta,
			Disabled:          secondUpdateEvent.ChannelUpdates[0].RoutingPolicy.Disabled,
		}

		r3 := simulateChannelGraphUpdate(t, db, &stubLNDSubscribeChannelGraphRPC{GraphTopologyUpdate: []*lnrpc.
			GraphTopologyUpdate{&secondUpdateEvent}})

		if r3[1].AnnouncingPubKey != e3.AnnouncingPubKey {
			testutil.Errorf(t, "Incorrect announcing pub key. Expected: %v, got %v", e3.AnnouncingPubKey,
				r3[1].AnnouncingPubKey)
		}

		if r3[1].Outbound != e3.Outbound {
			testutil.Errorf(t, "Incorrect outbound state. Expected: %v, got %v", e3.Outbound, r3[1].Outbound)
		}

	})

}

type routingPolicyData struct {
	Ts                time.Time
	LNDChannelPoint   string `db:"lnd_channel_point"`
	LNDShortChannelId string `db:"lnd_short_channel_id"`
	Outbound          bool   `db:"outbound"`
	AnnouncingPubKey  string `db:"announcing_pub_key"`
	FeeRateMillMsat   int64  `db:"fee_rate_mill_msat"`
	FeeBaseMsat       int64  `db:"fee_base_msat"`
	MaxHtlcMsat       uint64 `db:"max_htlc_msat"`
	MinHtlc           int64  `db:"min_htlc"`
	TimeLockDelta     uint32 `db:"time_lock_delta"`
	Disabled          bool   `db:"disabled"`
}

func simulateChannelGraphUpdate(t *testing.T, db *sqlx.DB, client *stubLNDSubscribeChannelGraphRPC) (r []routingPolicyData) {
	ctx := context.Background()
	errs, ctx := errgroup.WithContext(ctx)

	errs.Go(func() error {
		err := SubscribeAndStoreChannelGraph(ctx, client, db)
		if err != nil {
			t.Fatalf("Problem subscribing to channel graph: %v", err)
		}
		return err
	})

	// Wait for subscriptions to complete
	err := errs.Wait()
	if err != nil {
		t.Fatal(err)
	}

	var result []routingPolicyData
	err = db.Select(&result, `
			select ts,
				   lnd_channel_point,
				   lnd_short_channel_id,
				   outbound,
				   announcing_pub_key,
				   fee_rate_mill_msat,
				   fee_base_msat,
				   max_htlc_msat,
				   min_htlc,
				   time_lock_delta,
				   disabled
			from routing_policy;`)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			t.Fatal("There were no routing policies but I did expect there to be some")
		}
		t.Fatalf("Problem executing sql: %v", err)
	}

	return result
}
