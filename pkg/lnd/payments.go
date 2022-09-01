package lnd

import (
	"context"
	"encoding/json"
	"github.com/benbjohnson/clock"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/zpay32"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"time"
)

type lightningClient_ListPayments interface {
	ListPayments(ctx context.Context, in *lnrpc.ListPaymentsRequest,
		opts ...grpc.CallOption) (*lnrpc.ListPaymentsResponse,
		error)
}

// PayOptions allows the caller to adjust the number of payments can be requested at a time
// and set a custom time interval between requests.
type PayOptions struct {
	Tick <-chan time.Time
}

func SubscribeAndStorePayments(ctx context.Context, client lightningClient_ListPayments, db *sqlx.DB, opt *PayOptions) error {

	// Create the default ticker used to fetch forwards at a set interval
	c := clock.New()
	ticker := c.Tick(10 * time.Second)

	// If a custom ticker is set in the options, override the default ticker.
	if (opt != nil) && (opt.Tick != nil) {
		ticker = opt.Tick
	}

	// Request the Payments at the requested interval.
	// NB!: This timer is slowly being shifted because of the time required to
	// fetch and store the response.
	for {
		// Exit if canceled
		select {
		case <-ctx.Done():
			return nil
		case <-ticker:

			// Fetch the last payment index we have stored
			last, err := fetchLastPaymentIndex(db)
			if err != nil {
				return errors.Wrapf(err, "SubscribeAndStorePayments->fetchLastPaymentIndex(%v)", db)
			}

			// Keep fetching until LND returns less than the max number of records requested.
			for {

				p, err := fetchPayments(ctx, client, last)
				if errors.As(err, &context.Canceled) {
					return nil
				}
				if err != nil {
					log.Printf("Fetch payments: %v\n", err)
					break
				}

				last = p.LastIndexOffset

				// Store the payments
				err = storePayments(db, p.Payments)
				if err != nil {
					log.Printf("Store payments: %v\n", err)
					break
				}

				// Stop fetching if there are fewer forwards than max requested
				// (indicates that we have the last forwarding record)
				if len(p.Payments) == 0 {
					break
				}
			}
		}
	}
}

func fetchLastPaymentIndex(db *sqlx.DB) (uint64, error) {
	var last uint64

	row := db.QueryRow(`select coalesce(max(payment_index), 0) as latest from payment;`)
	err := row.Scan(&last)

	if err != nil {
		return 0, errors.Wrap(err, "fetching last payment index")
	}

	return last, nil
}

// fetchPayments fetches completed payments from LND.
func fetchPayments(ctx context.Context, client lightningClient_ListPayments, last uint64) (
	r *lnrpc.ListPaymentsResponse, err error) {

	//retry:
	req := &lnrpc.ListPaymentsRequest{
		IncludeIncomplete: true,
		IndexOffset:       last,
		MaxPayments:       1, // Only fetch one at a time due to the size of failed payments
		Reversed:          false,
	}
	r, err = client.ListPayments(ctx, req)

	if err != nil {
		return nil, errors.Wrap(err, "fetch payments: lnrpc list payments")
	}

	return r, nil
}

func storePayments(db *sqlx.DB, p []*lnrpc.Payment) error {

	const q = `INSERT INTO payment(
				  payment_hash,
				  creation_timestamp,
				  payment_preimage,
				  value_msat,
				  payment_request,
				  status,
				  fee_msat,
				  creation_time_ns,
				  htlcs,
				  payment_index,
				  failure_reason,
				  created_on)
			  VALUES ($1, $2, $3,$4, $5,$6, $7, $8, $9, $10, $11, $12)
			  ON CONFLICT (creation_timestamp, payment_index) DO NOTHING;`

	if len(p) > 0 {
		tx := db.MustBegin()

		for _, payment := range p {

			htlcJson, err := json.Marshal(payment.Htlcs)
			if err != nil {
				return err
			}

			if _, err := tx.Exec(q,
				payment.PaymentHash,
				time.Unix(0, payment.CreationTimeNs).Round(time.Microsecond).UTC(),
				payment.PaymentPreimage,
				payment.ValueMsat,
				payment.PaymentRequest,
				payment.Status.String(),
				payment.FeeMsat,
				payment.CreationTimeNs,
				htlcJson,
				payment.PaymentIndex,
				payment.FailureReason.String(),
				time.Now().UTC(),
			); err != nil {
				return errors.Wrapf(err, "store payments: db exec")
			}
		}
		err := tx.Commit()
		if err != nil {
			return err
		}
	}

	return nil
}

func SubscribeAndUpdatePayments(ctx context.Context, client lightningClient_ListPayments, db *sqlx.DB, opt *PayOptions) error {

	// Create the default ticker used to fetch forwards at a set interval
	c := clock.New()
	ticker := c.Tick(10 * time.Second)

	// If a custom ticker is set in the options, override the default ticker.
	if (opt != nil) && (opt.Tick != nil) {
		ticker = opt.Tick
	}

	// Request the in flight payments at the requested interval.
	// NB!: This timer is slowly being shifted because of the time required to
	// fetch and store the response.
	for {
		// Exit if canceled
		select {
		case <-ctx.Done():
			return nil
		case <-ticker:

			// Fetch the last payment index we have stored
			inFlightindexes, err := fetchInFlightPaymentIndexes(db)

			if err != nil {
				log.Printf("Subscribe and update payments: %v\n", err)
				continue
			}

			// Keep fetching until LND returns less than the max number of records requested.
			for _, i := range inFlightindexes {
				ifPayIndex := i - 1 // Subtract one to get that index, otherwise we would get the one after.
				p, err := fetchPayments(ctx, client, ifPayIndex)
				if errors.As(err, &context.Canceled) {
					return nil
				}
				if err != nil {
					log.Printf("Subscribe and update payments: %v\n", err)
					continue
				}

				// Store the payments
				err = updatePayments(db, p.Payments)
				if err != nil {
					log.Printf("Subscribe and update payments: %v\n", err)
					continue
				}
			}
		}
	}
}

type Payment struct {
	PaymentIndex      uint64 `json:"payment_index" db:"payment_index"`
	PaymentHash       string `json:"payment_hash" db:"payment_hash"`
	PaymentPreimage   string `json:"payment_preimage" db:"payment_preimage"`
	PaymentRequest    string `json:"payment_request" db:"payment_request"`
	Status            string `json:"status" db:"status"`
	ValueMsat         int64  `json:"value_msat" db:"value_msat"`
	FeeMsat           int64  `json:"fee_msat" db:"fee_msat"`
	Htlcs             []byte `json:"htlcs" db:"htlcs"`
	FailureReason     string `json:"failure_reason" db:"failure_reason"`
	CreationTimeNs    int64  `json:"creation_time_ns" db:"creation_time_ns"`
	CreationTimestamp int64  `json:"creation_timestamp" db:"creation_timestamp"`
}

func fetchInFlightPaymentIndexes(db *sqlx.DB) (r []uint64, err error) {

	rows, err := db.Query(`
		select payment_index
		from payment
		where status = 'IN_FLIGHT'
		order by payment_index asc;
	`)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var res *uint64
		err = rows.Scan(&res)

		if err != nil {
			return r, err
		}

		// Append to the result
		r = append(r, *res)

	}

	return r, nil
}

func updatePayments(db *sqlx.DB, p []*lnrpc.Payment) error {

	const q = `update payment set(
				  payment_hash,
				  payment_preimage,
				  value_msat,
				  payment_request,
				  status,
				  fee_msat,
				  htlcs,
				  failure_reason,
				  updated_on)
			  = ($1, $2, $3,$4, $5,$6, $7, $8, $9)
				where payment_index = $10;`

	if len(p) > 0 {
		tx := db.MustBegin()

		for _, payment := range p {

			htlcJson, err := json.Marshal(payment.Htlcs)
			if err != nil {
				return err
			}

			status := payment.Status.String()
			fr := payment.FailureReason.String()

			// Payments can be stuck as IN_FLIGHT if they are interrupted.
			if status == "IN_FLIGHT" {
				// Check expiry time for IN_FLIGHT payments

				// Default expiry (1 week, just to be sure)
				expiry := time.Duration(24 * 7 * time.Hour)

				// Update the expiry time if the PaymentRequest is available
				if payment.PaymentRequest != "" {
					inva, err := zpay32.Decode(payment.PaymentRequest, &chaincfg.MainNetParams)
					if err != nil {
						return err
					}
					expiry = inva.Expiry()
				}

				currentTime := time.Now().UTC()
				created := time.Unix(0, payment.CreationTimeNs).UTC()

				// Add 10 minutes to the invoice expiry time to be safe.
				expiredAt := created.Add(expiry).Add(10 * time.Minute)

				// If the invoice has expired, mark the payment as failed.
				if currentTime.Unix() > expiredAt.Unix() {
					status = "FAILED"

					// Set the failure reason to timeout, if none is already set.
					if fr == "FAILURE_REASON_NONE" {
						fr = "FAILURE_REASON_TIMEOUT"
					}
				}
			}

			_, err = db.Exec(q,
				payment.PaymentHash,
				payment.PaymentPreimage,
				payment.ValueMsat,
				payment.PaymentRequest,
				status,
				payment.FeeMsat,
				htlcJson,
				fr,
				time.Now().UTC(),
				payment.PaymentIndex,
			)

			if err != nil {
				return errors.Wrapf(err, "updatePayments->tx.Exec(%v)", q)
			}
		}
		err := tx.Commit()
		if err != nil {
			return err
		}
	}

	return nil
}
