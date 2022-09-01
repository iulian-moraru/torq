package lnd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lncapital/torq/internal/channels"
	"go.uber.org/ratelimit"
	"log"
	"time"
)

func storeLinkFailEvent(db *sqlx.DB, h *routerrpc.HtlcEvent, fwe *routerrpc.LinkFailEvent) error {

	jb, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("storeLinkFailEvent -> json.Marshal(%v): %v", h, err)
	}

	stm := `
	INSERT INTO
	htlc_event (
		time,
		event_origin,
		lnd_outgoing_short_channel_id,
		lnd_incoming_short_channel_id,
		outgoing_short_channel_id,
		incoming_short_channel_id,
		timestamp_ns,
		data,
		event_type,
		incoming_amt_msat,
		outgoing_amt_msat,
		incoming_timelock,
		Outgoing_timelock,
		outgoing_htlc_id,
		incoming_htlc_id,
		bolt_failure_code,
		bolt_failure_string,
		lnd_failure_detail
	)
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	timestampMs := time.Unix(0, int64(h.TimestampNs)).Round(time.Microsecond).UTC()

	incomingShortChannelId := channels.ConvertLNDShortChannelID(h.IncomingChannelId)
	outgoingShortChannelId := channels.ConvertLNDShortChannelID(h.OutgoingChannelId)

	_, err = db.Exec(stm,
		timestampMs,
		h.EventType,
		h.OutgoingChannelId,
		h.IncomingChannelId,
		outgoingShortChannelId,
		incomingShortChannelId,
		h.TimestampNs,
		jb,
		"LinkFailEvent",
		fwe.Info.IncomingAmtMsat,
		fwe.Info.OutgoingAmtMsat,
		fwe.Info.IncomingTimelock,
		fwe.Info.OutgoingTimelock,
		h.OutgoingHtlcId,
		h.IncomingHtlcId,
		fwe.WireFailure.String(),
		fwe.FailureString,
		fwe.FailureDetail.String(),
	)

	if err != nil {
		return fmt.Errorf(`storeLinkFailEvent -> db.Exec(%s, %v, %v, %v, %v, %v): %v`,
			stm, timestampMs, h.EventType, h.OutgoingChannelId, h.IncomingChannelId, jb, err)
	}

	return nil
}

func storeSettleEvent(db *sqlx.DB, h *routerrpc.HtlcEvent, fwe *routerrpc.SettleEvent) error {

	jb, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("storeForwardEvent -> json.Marshal(%v): %v", h, err)
	}

	stm := `
	INSERT INTO
	htlc_event (
		time,
		event_origin,
		lnd_outgoing_short_channel_id,
		lnd_incoming_short_channel_id,
		outgoing_short_channel_id,
		incoming_short_channel_id,
		timestamp_ns,
		data,
		event_type,
		outgoing_htlc_id,
		incoming_htlc_id
	)
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	timestampMs := time.Unix(0, int64(h.TimestampNs)).Round(time.Microsecond).UTC()
	incomingShortChannelId := channels.ConvertLNDShortChannelID(h.IncomingChannelId)
	outgoingShortChannelId := channels.ConvertLNDShortChannelID(h.OutgoingChannelId)

	_, err = db.Exec(stm,
		timestampMs,
		h.EventType,
		h.OutgoingChannelId,
		h.IncomingChannelId,
		outgoingShortChannelId,
		incomingShortChannelId,
		h.TimestampNs,
		jb,
		"SettleEvent",
		h.OutgoingHtlcId,
		h.IncomingHtlcId,
	)

	if err != nil {
		return fmt.Errorf(`storeSettleEvent -> db.Exec(%s, %v, %v, %v, %v, %v): %v`,
			stm, timestampMs, h.EventType, h.OutgoingChannelId, h.IncomingChannelId, jb, err)
	}

	return nil
}

func storeForwardFailEvent(db *sqlx.DB, h *routerrpc.HtlcEvent) error {

	jb, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("storeForwardFailEvent -> json.Marshal(%v): %v", h, err)
	}

	stm := `
	INSERT INTO
	htlc_event (
		time,
		event_origin,
		lnd_outgoing_short_channel_id,
		lnd_incoming_short_channel_id,
		outgoing_short_channel_id,
		incoming_short_channel_id,
		timestamp_ns,
		data,
		event_type,
		outgoing_htlc_id,
		incoming_htlc_id
	)
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	timestampMs := time.Unix(0, int64(h.TimestampNs)).Round(time.Microsecond).UTC()
	incomingShortChannelId := channels.ConvertLNDShortChannelID(h.IncomingChannelId)
	outgoingShortChannelId := channels.ConvertLNDShortChannelID(h.OutgoingChannelId)

	_, err = db.Exec(stm,
		timestampMs,
		h.EventType,
		h.OutgoingChannelId,
		h.IncomingChannelId,
		outgoingShortChannelId,
		incomingShortChannelId,
		h.TimestampNs,
		jb,
		"ForwardFailEvent",
		h.OutgoingHtlcId,
		h.IncomingHtlcId,
	)

	if err != nil {
		return fmt.Errorf(`storeForwardFailEvent -> db.Exec(%s, %v, %v, %v, %v, %v): %v`,
			stm, timestampMs, h.EventType, h.OutgoingChannelId, h.IncomingChannelId, jb, err)
	}

	return nil
}

func storeForwardEvent(db *sqlx.DB, h *routerrpc.HtlcEvent, fwe *routerrpc.ForwardEvent) error {

	jb, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("storeForwardEvent -> json.Marshal(%v): %v", h, err)
	}

	stm := `
	INSERT INTO
	htlc_event (
		time,
		event_origin,
		lnd_outgoing_short_channel_id,
		lnd_incoming_short_channel_id,
		outgoing_short_channel_id,
		incoming_short_channel_id,
		timestamp_ns,
		data,
		event_type,
		incoming_amt_msat,
		outgoing_amt_msat,
		incoming_timelock,
		Outgoing_timelock,
		outgoing_htlc_id,
		incoming_htlc_id
	)
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	timestampMs := time.Unix(0, int64(h.TimestampNs)).Round(time.Microsecond).UTC()
	incomingShortChannelId := channels.ConvertLNDShortChannelID(h.IncomingChannelId)
	outgoingShortChannelId := channels.ConvertLNDShortChannelID(h.OutgoingChannelId)

	_, err = db.Exec(stm,
		timestampMs,
		h.EventType,
		h.OutgoingChannelId,
		h.IncomingChannelId,
		outgoingShortChannelId,
		incomingShortChannelId,
		h.TimestampNs,
		jb,
		"ForwardEvent",
		fwe.Info.IncomingAmtMsat,
		fwe.Info.OutgoingAmtMsat,
		fwe.Info.IncomingTimelock,
		fwe.Info.OutgoingTimelock,
		h.OutgoingHtlcId,
		h.IncomingHtlcId,
	)

	if err != nil {
		return fmt.Errorf(`storeForwardEvent -> db.Exec(%s, %v, %v, %v, %v, %v): %v`,
			stm, timestampMs, h.EventType, h.OutgoingChannelId, h.IncomingChannelId, jb, err)
	}

	return nil
}

// SubscribeAndStoreHtlcEvents subscribes to HTLC events from LND and stores them in the database as time series.
// NB: LND has marked HTLC event streaming as experimental. Delivery is not guaranteed, so dataset might not be complete
// HTLC events is primarily used to diagnose how good a channel / node is. And if the channel allocation should change.
func SubscribeAndStoreHtlcEvents(ctx context.Context, router routerrpc.RouterClient, db *sqlx.DB) error {

	htlcStream, err := router.SubscribeHtlcEvents(ctx, &routerrpc.SubscribeHtlcEventsRequest{})
	if err != nil {
		return fmt.Errorf("SubscribeAndStoreHtlcEvents -> SubscribeHtlcEvents(): %v", err)
	}

	rl := ratelimit.New(1) // 1 per second maximum rate limit

	for {

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		htlcEvent, err := htlcStream.Recv()

		if err != nil {
			if errors.As(err, &context.Canceled) {
				break
			}
			log.Printf("Subscribe htlc events stream receive: %v\n", err)
			// rate limited resubscribe
			log.Println("Attempting reconnect to HTLC events")
			for {
				rl.Take()
				htlcStream, err = router.SubscribeHtlcEvents(ctx, &routerrpc.SubscribeHtlcEventsRequest{})
				if err == nil {
					log.Println("Reconnected to HTLC events")
					break
				}
				log.Printf("Reconnecting to HTLC events: %v\n", err)
			}
			continue
		}

		switch htlcEvent.Event.(type) {
		case *routerrpc.HtlcEvent_ForwardEvent:
			err = storeForwardEvent(db, htlcEvent, htlcEvent.GetForwardEvent())
			if err != nil {
				log.Printf("Subscribe htlc events stream: %v", err)
				// rate limit for caution but hopefully not needed
				rl.Take()
			}
		case *routerrpc.HtlcEvent_ForwardFailEvent:
			err = storeForwardFailEvent(db, htlcEvent)
			if err != nil {
				log.Printf("Subscribe htlc events stream: %v", err)
				// rate limit for caution but hopefully not needed
				rl.Take()
			}
		case *routerrpc.HtlcEvent_LinkFailEvent:
			err = storeLinkFailEvent(db, htlcEvent, htlcEvent.GetLinkFailEvent())
			if err != nil {
				log.Printf("Subscribe htlc events stream: %v", err)
				// rate limit for caution but hopefully not needed
				rl.Take()
			}
		case *routerrpc.HtlcEvent_SettleEvent:
			err = storeSettleEvent(db, htlcEvent, htlcEvent.GetSettleEvent())
			if err != nil {
				log.Printf("Subscribe htlc events stream: %v", err)
				// rate limit for caution but hopefully not needed
				rl.Take()
			}
		}

	}

	return nil
}
