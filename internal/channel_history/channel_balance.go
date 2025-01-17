package channel_history

import (
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"

	"github.com/lncapital/torq/internal/channels"
	"github.com/lncapital/torq/pkg/commons"
)

type Balance struct {
	Date             time.Time `db:"date" json:"date"`
	InboundCapacity  int64     `db:"inbound_capacity" json:"inboundCapacity"`
	OutboundCapacity int64     `db:"outbound_capacity" json:"outboundCapacity"`
	CapacityDiff     *int64    `db:"capacity_diff" json:"capacityDiff"`
}

type ChannelBalance struct {
	LNDShortChannelId string     `json:"lndShortChannelId"`
	Balances          []*Balance `json:"balances"`
}

func getChannelBalance(db *sqlx.DB, lndShortChannelIdString string, from time.Time, to time.Time) (ChannelBalance, error) {
	lndShortChannelId, err := strconv.ParseUint(lndShortChannelIdString, 10, 64)
	if err != nil {
		return ChannelBalance{}, errors.Wrapf(err, "Converting LND short channel id %v", lndShortChannelId)
	}
	shortChannelId := channels.ConvertLNDShortChannelID(lndShortChannelId)
	channelId := commons.GetChannelIdFromShortChannelId(shortChannelId)

	cb := ChannelBalance{LNDShortChannelId: lndShortChannelIdString}
	q := `WITH
		   initial_balance as (
				select coalesce((-t.amount)-t.total_fees, 0) as initial_balance
				from channel_event ce
				JOIN channel c ON c.channel_id=ce.channel_id
				left join tx t on c.funding_transaction_hash = t.tx_hash
				where ce.event_type in (0,1) and
					  c.channel_id = $1
				limit 1
			)
		select time as date,
		       outbound_capacity,
		       outbound_capacity - lag(outbound_capacity) over (order by time) as capacity_diff
		from (
			select time,
			       floor((table initial_balance) + sum(amt/1000) over(order by time)) as outbound_capacity
			from (
				(select time,
				   -outgoing_amount_msat as amt
				from forward
				where outgoing_channel_id = $1
				order by time)
				UNION
				(select time,
					   incoming_amount_msat as amt
				from forward
				where incoming_channel_id = $1
				order by time)
				UNION
				(select creation_timestamp as time,
					-(select sum(a) from UNNEST(ARRAY(SELECT jsonb_array_elements_text(jsonb_path_query_array(htlcs, ('$.route[*].hops[0]?(@.chan_id=='|| $5 ||').amt_to_forward_msat')::jsonpath)))::numeric[]) as a) amt
				from payment p
				where (status = 'SUCCEEDED')
					and jsonb_path_query_array(htlcs, ('$.route[*].hops[0].chan_id')::jsonpath) @> $5::jsonb
				order by time)
				UNION
				(select settle_date as time,
					   -- We need to fetch the amount paid to a channel using MPP.
				   (select sum(a) from UNNEST(ARRAY(SELECT jsonb_array_elements_text(jsonb_path_query_array(htlcs, ('$?(@.chan_id=='|| $5 ||' && @.state==1).amt_msat')::jsonpath)))::numeric[]) as a) amt
				from invoice
				where invoice_state = 'SETTLED'
					and jsonb_path_query_array(htlcs, '$[*].chan_id') @> $5::jsonb
				order by time)
			) a
		) b
	where time::timestamp AT TIME ZONE ($4) between $2 and $3
;`

	rows, err := db.Queryx(q, channelId, from, to, commons.GetSettings().PreferredTimeZone, lndShortChannelIdString)
	if err != nil {
		return cb, err
	}
	for rows.Next() {
		b := Balance{}

		err = rows.StructScan(&b)
		if err != nil {
			return cb, err
		}

		if len(cb.Balances) == 0 {
			cb.Balances = []*Balance{&b}
		} else {
			cb.Balances = append(cb.Balances, &b)
		}

	}

	return cb, nil

}
