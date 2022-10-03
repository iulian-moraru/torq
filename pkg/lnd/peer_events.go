package lnd

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/rs/zerolog/log"
	"go.uber.org/ratelimit"
	"google.golang.org/grpc"
)

type peerEventUpdate struct {
	Type      string `json:"type"`
	EventType string `json:"eventType"`
}

type peerEventsClient interface {
	SubscribePeerEvents(ctx context.Context, in *lnrpc.PeerEventSubscription, opts ...grpc.CallOption) (lnrpc.Lightning_SubscribePeerEventsClient, error)
}

func SubscribePeerEvents(ctx context.Context, client peerEventsClient, wsChan chan interface{}) error {

	peerEventStream, err := client.SubscribePeerEvents(ctx, &lnrpc.PeerEventSubscription{})

	if err != nil {
		log.Error().Msgf("subscribe and store invoices - lnrpc subscribe:  %v", err)
		return errors.Wrap(err, "lnrpc subscribe invoices")
	}

	rl := ratelimit.New(1) // 1 per second maximum rate limit

	for {

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		peerEvent, err := peerEventStream.Recv()

		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				log.Error().Msgf("Peer events subscription - Context canceled")
				break
			}
			log.Error().Msgf("Peer events subscription err: %v", err)
			// rate limited resubscribe
			for {
				rl.Take()
				peerEventStream, err = client.SubscribePeerEvents(ctx, &lnrpc.PeerEventSubscription{})
				if err == nil {
					//log.Debug().Msgf("Reconnected to invoice subscription")
					break
				}
			}
			continue
		}

		updatePeer := peerEventUpdate{
			"peerEvent",
			peerEvent.GetType().String(),
		}

		wsChan <- updatePeer

	}

	return nil
}
