package subscribe

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lncapital/torq/pkg/lnd"
	// "github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// Start runs the background server. It subscribes to events, gossip and
// fetches data as needed and stores it in the database.
// It is meant to run as a background task / daemon and is the bases for all
// of Torqs data collection
func Start(ctx context.Context, conn *grpc.ClientConn, db *sqlx.DB, localNodeId int) error {

	router := routerrpc.NewRouterClient(conn)
	client := lnrpc.NewLightningClient(conn)

	// Create an error group to catch errors from go routines.
	// TODO: Improve this by using the context to propogate the error,
	//   shutting down the if one of the subscribe go routines fail.
	//   https://www.fullstory.com/blog/why-errgroup-withcontext-in-golang-server-handlers/
	// TODO: Also consider using the same context used by the gRPC connection from Golang and the
	//   gRPC server of Torq
	errs, ctx := errgroup.WithContext(ctx)

	// Store a list of public keys belonging to our nodes
	err := lnd.InitOurNodesList(ctx, client, db)
	if err != nil {
		return err
	}

	//Import Open channels
	err = lnd.ImportChannelList(lnrpc.ChannelEventUpdate_OPEN_CHANNEL, db, client, localNodeId)
	if err != nil {
		return errors.Wrapf(err, "Start -> importChannelList(%s, %v, %v)",
			lnrpc.ChannelEventUpdate_OPEN_CHANNEL, db, client)
	}

	// Import Closed channels
	err = lnd.ImportChannelList(lnrpc.ChannelEventUpdate_CLOSED_CHANNEL, db, client, localNodeId)
	if err != nil {
		return errors.Wrapf(err, "Start -> importChannelList(%s, %v, %v)",
			lnrpc.ChannelEventUpdate_CLOSED_CHANNEL, db, client)
	}

	// Import Node info (based on channels)
	err = lnd.ImportMissingNodeEvents(client, db)
	if err != nil {
		return errors.Wrapf(err, "Start -> ImportMissingNodeEvents(%v, %v)", client, db)
	}

	// Import routing policies from open channels
	err = lnd.ImportRoutingPolicies(client, db)
	if err != nil {
		return errors.Wrapf(err, "Start -> ImportRoutingPolicies(%v, %v)", client, db)
	}

	// Initialize the peer list
	err = lnd.InitPeerList(db)
	if err != nil {
		return errors.Wrapf(err, "start -> InitPeerList(%v)", db)
	}

	// Initialize the channel id list
	err = lnd.InitChanIdList(db)
	if err != nil {
		return errors.Wrapf(err, "start -> InitChanIdList(%v)", db)
	}
	// Create a channel to update the list of public key for nodes we have
	// or have had channels with
	pubKeyChan := make(chan string)

	// Start listening for updates to the public key list
	go lnd.UpdatePeerList(pubKeyChan)

	// Create a channel to update the list of channel points for our currently active with
	chanPointChan := make(chan string)

	// Start listening for updates to the channel point list
	go lnd.UpdateChanIdList(chanPointChan)

	// Transactions
	errs.Go(func() error {
		err := lnd.SubscribeAndStoreTransactions(ctx, client, db)
		if err != nil {
			return errors.Wrapf(err, "Start->SubscribeAndStoreTransactions(%v, %v, %v)", ctx, client, db)
		}
		return nil
	})

	// Graph (Node updates, fee updates etc.)
	errs.Go(func() error {
		err := lnd.SubscribeAndStoreChannelGraph(ctx, client, db)
		if err != nil {
			return errors.Wrapf(err, "Start->SubscribeAndStoreChannelGraph(%v, %v, %v)", ctx, client, db)
		}
		return nil
	})

	// // HTLC events
	errs.Go(func() error {
		err := lnd.SubscribeAndStoreHtlcEvents(ctx, router, db)
		if err != nil {
			return errors.Wrapf(err, "Start->SubscribeAndStoreHtlcEvents(%v, %v, %v)", ctx, router, db)
		}
		return nil
	})

	// // Channel Events
	errs.Go(func() error {
		err := lnd.SubscribeAndStoreChannelEvents(ctx, client, db, pubKeyChan, chanPointChan, localNodeId)
		if err != nil {
			return errors.Wrapf(err, "Start->SubscribeAndStoreChannelEvents(%v, %v, %v)", ctx, router, db)
		}
		return nil
	})

	// Forwarding history
	errs.Go(func() error {
		err := lnd.SubscribeForwardingEvents(ctx, client, db, nil)
		if err != nil {
			return errors.Wrapf(err, "Start->SubscribeForwardingEvents(%v, %v, %v, %v)", ctx,
				client, db, nil)
		}
		return nil
	})

	// Invoices
	errs.Go(func() error {
		err := lnd.SubscribeAndStoreInvoices(ctx, client, db)
		if err != nil {
			return errors.Wrapf(err, "Start->SubscribeAndStoreInvoices(%v, %v, %v)", ctx,
				client, db)
		}
		return nil
	})

	// Payments
	errs.Go(func() error {
		err := lnd.SubscribeAndStorePayments(ctx, client, db, nil)
		if err != nil {
			return errors.Wrapf(err, "Start->SubscribeAndStorePayments(%v, %v, %v)", ctx,
				client, db)
		}
		return nil
	})

	// Update in flight payments
	errs.Go(func() error {
		err := lnd.SubscribeAndUpdatePayments(ctx, client, db, nil)
		if err != nil {
			return errors.Wrapf(err, "Start->SubscribeAndUpdatePayments(%v, %v, %v)", ctx,
				client, db)
		}
		return nil
	})

	err = errs.Wait()

	return err
}
