package channel_history

import (
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"time"
)

type ChannelEvent struct {
	// The date used by the chart library to place roughly in the timeline. Usually bucketed pr day
	Date time.Time `json:"date"`
	// Spesific time the event happened.
	Datetime time.Time `json:"datetime"`
	// The channel point
	LNDChannelPoint *string `json:"lndChannelPoint"`
	// The channel ID
	ShortChannelId *string `json:"shortChannelId"`
	// The type of event. E.g. disable/enable, change in fee rate, base fee, min/max htlc amount
	Type *string `json:"type"`
	// Was this changed by our node (outbound) or their node (inbound)
	Outbound *bool `json:"outbound"`
	// The node that announced the change
	AnnouncingPubKey *string `json:"announcing_pub_key"`
	// The value, in cases where there is a value change,
	//like with fee rate etc. Not used by disable/enable and channel open/close
	Value *uint64 `json:"value"`
	// The previous value
	PreviousValue *uint64 `json:"previous_value"`
}

func getChannelEventHistory(db *sqlx.DB, chanIds []string, from time.Time, to time.Time) (r []*ChannelEvent, err error) {

	sql := `WITH
    fromDate AS (VALUES (?)),
    toDate AS (VALUES (?)),
	pub_keys as (select array_agg(distinct pub_key) as pub_keys from channel_event where lnd_short_channel_id in (?)),
    tz AS (select preferred_timezone as tz from settings)
-- disabled changes
select date(ts)::timestamp AT TIME ZONE (table tz) as date,
        ts::timestamp AT TIME ZONE (table tz) as datetime,
        lnd_channel_point,
	   lnd_short_channel_id,
	   ((table pub_keys) && pub_key_array) = false as outbound,
       announcing_pub_key,
       case when disabled = true then 'disabled' else 'enabled' end as type,
       null as value,
 	   null as prev
from (SELECT ts as ts,
			 lnd_channel_point,
             lnd_short_channel_id,
             announcing_pub_key,
             ARRAY[announcing_pub_key] as pub_key_array,
             disabled,
             lag(disabled, 1, false) OVER (PARTITION BY lnd_short_channel_id, announcing_pub_key ORDER BY ts) AS prev
      FROM routing_policy
      where lnd_short_channel_id in (?)
        and ts::timestamp AT TIME ZONE (table tz) >= (table fromDate)::timestamp
        and ts::timestamp AT TIME ZONE (table tz) <= (table toDate)::timestamp
) as o
where prev != disabled

UNION
-- fee rate changes
select date(ts)::timestamp AT TIME ZONE (table tz) as date,
       ts::timestamp AT TIME ZONE (table tz) as datetime,
       lnd_channel_point,
       lnd_short_channel_id,
       ((table pub_keys) && pub_key_array) = false as outbound,
       announcing_pub_key,
       'fee_rate' as type,
       fee_rate as value,
       prev
from (SELECT ts as ts,
             lnd_channel_point,
             lnd_short_channel_id,
             announcing_pub_key,
             ARRAY[announcing_pub_key] as pub_key_array,
             fee_rate_mill_msat as fee_rate,
             lag(fee_rate_mill_msat, 1, 0) OVER (PARTITION BY lnd_short_channel_id,
announcing_pub_key ORDER BY ts) AS prev
      FROM routing_policy
      where lnd_short_channel_id in (?)
        and ts::timestamp AT TIME ZONE (table tz) >= (table fromDate)::timestamp
        and ts::timestamp AT TIME ZONE (table tz) <= (table toDate)::timestamp
) as o
where prev != fee_rate

UNION
-- base fee changes
select date(ts)::timestamp AT TIME ZONE (table tz) as date,
       ts::timestamp AT TIME ZONE (table tz) as datetime,
       lnd_channel_point,
       lnd_short_channel_id,
       ((table pub_keys) && pub_key_array) = false as outbound,
       announcing_pub_key,
       'base_fee' as type,
       round(fee_base / 1000) as value,
       round(prev / 1000) as prev
from (SELECT ts as ts,
             lnd_channel_point,
             lnd_short_channel_id,
             announcing_pub_key,
             ARRAY[announcing_pub_key] as pub_key_array,
             fee_base_msat as fee_base,
             lag(fee_base_msat, 1, 0) OVER (PARTITION BY lnd_short_channel_id, announcing_pub_key ORDER BY ts) AS prev
      FROM routing_policy
      where lnd_short_channel_id in (?)
        and ts::timestamp AT TIME ZONE (table tz) >= (table fromDate)::timestamp
        and ts::timestamp AT TIME ZONE (table tz) <= (table toDate)::timestamp
) as o
where prev != fee_base

UNION
-- max_htlc changes
select date(ts)::timestamp AT TIME ZONE (table tz) as date,
       ts::timestamp AT TIME ZONE (table tz) as datetime,
       lnd_channel_point,
       lnd_short_channel_id,
       ((table pub_keys) && pub_key_array) = false as outbound,
       announcing_pub_key,
       'max_htlc' as type,
       round(max_htlc_msat / 1000) as value,
       round(prev / 1000) as prev
from (SELECT ts as ts,
             lnd_channel_point,
             lnd_short_channel_id,
             announcing_pub_key,
             ARRAY[announcing_pub_key] as pub_key_array,
             max_htlc_msat,
             lag(max_htlc_msat, 1, 0) OVER (PARTITION BY lnd_short_channel_id, announcing_pub_key ORDER BY ts) AS prev
      FROM routing_policy
      where lnd_short_channel_id in (?)
        and ts::timestamp AT TIME ZONE (table tz) >= (table fromDate)::timestamp
        and ts::timestamp AT TIME ZONE (table tz) <= (table toDate)::timestamp
) as o
where prev != max_htlc_msat

UNION
-- min_htlc changes
select date(ts)::timestamp AT TIME ZONE (table tz) as date,
       ts::timestamp AT TIME ZONE (table tz) as datetime,
       lnd_channel_point,
       lnd_short_channel_id,
       ((table pub_keys) && pub_key_array) = false as outbound,
       announcing_pub_key,
       'min_htlc' as type,
       round(min_htlc / 1000) as value,
       round(prev / 1000) as prev
from (SELECT ts as ts,
             lnd_channel_point,
             lnd_short_channel_id,
             announcing_pub_key,
             ARRAY[announcing_pub_key] as pub_key_array,
             min_htlc,
             lag(min_htlc, 1, 0) OVER (PARTITION BY lnd_short_channel_id, announcing_pub_key ORDER BY ts) AS prev
      FROM routing_policy
      where lnd_short_channel_id in (?)
        and ts::timestamp AT TIME ZONE (table tz) >= (table fromDate)::timestamp
        and ts::timestamp AT TIME ZONE (table tz) <= (table toDate)::timestamp
) as o
where prev  != min_htlc
order by datetime desc;
`

	qs, args, err := sqlx.In(sql, from, to, chanIds, chanIds, chanIds, chanIds, chanIds, chanIds)
	if err != nil {
		return r, errors.Wrapf(err, "sqlx.In(%s, %v, %v, %v, %v, %v, %v, %v, %v)",
			sql, from, to, chanIds, chanIds, chanIds, chanIds, chanIds, chanIds)
	}

	qsr := db.Rebind(qs)
	rows, err := db.Query(qsr, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "Error running getChannelEventHistory query")
	}

	for rows.Next() {
		c := &ChannelEvent{}
		err = rows.Scan(
			&c.Date,
			&c.Datetime,
			&c.LNDChannelPoint,
			&c.ShortChannelId,
			&c.Outbound,
			&c.AnnouncingPubKey,
			&c.Type,
			&c.Value,
			&c.PreviousValue,
		)
		if err != nil {
			return r, err
		}

		// Append to the result
		r = append(r, c)

	}
	return r, nil
}
