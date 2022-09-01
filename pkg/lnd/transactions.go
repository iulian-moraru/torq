package lnd

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lightningnetwork/lnd/lnrpc"
	"go.uber.org/ratelimit"
	"log"
	"time"
)

func fetchLastTxHeight(db *sqlx.DB) (txHeight int32, err error) {

	sqlLatest := `select max(block_height) from tx;`

	row := db.QueryRow(sqlLatest)
	err = row.Scan(&txHeight)

	if err == sql.ErrNoRows {
		return 1, nil
	}

	if err != nil {
		return 1, err
	}

	return txHeight, nil
}

func ImportTransactions(ctx context.Context, client lnrpc.LightningClient, db *sqlx.DB) error {

	txheight, err := fetchLastTxHeight(db)
	if err != nil {
		return errors.Wrap(err, "Fetch Last Tx Height")
	}

	req := lnrpc.GetTransactionsRequest{
		StartHeight: txheight,
	}
	res, err := client.GetTransactions(ctx, &req)
	if err != nil {
		return errors.Wrap(err, "Get Transactions")
	}

	for _, tx := range res.Transactions {
		err = storeTransaction(db, tx)
		if err != nil {
			return errors.Wrap(err, "Store Transaction")
		}
	}

	return nil
}

// SubscribeAndStoreTransactions Subscribes to on-chain transaction events from LND and stores them in the
// database as a time series. It will also import unregistered transactions on startup.
func SubscribeAndStoreTransactions(ctx context.Context, client lnrpc.LightningClient, db *sqlx.DB) error {

	// Imports transactions not captured on the stream
	err := ImportTransactions(ctx, client, db)
	if err != nil {
		return errors.Wrapf(err, "ImportTransactions(%v, %v, %v)", ctx, client, db)
	}

	req := lnrpc.GetTransactionsRequest{}
	stream, err := client.SubscribeTransactions(ctx, &req)
	if err != nil {
		return err
	}
	rl := ratelimit.New(1) // 1 per second maximum rate limit

	for {

		select {
		case <-ctx.Done():
			return nil
		default:

			tx, err := stream.Recv()

			if err != nil {
				if errors.As(err, &context.Canceled) {
					break
				}
				log.Printf("Subscribe transactions stream receive: %v\n", err)
				// rate limited resubscribe
				log.Println("Attempting reconnect to transactions")
				for {
					rl.Take()
					stream, err = client.SubscribeTransactions(ctx, &req)
					if err == nil {
						log.Println("Reconnected to transactions")
						break
					}
					log.Printf("Reconnecting to transactions: %v\n", err)
				}
				continue
			}

			err = storeTransaction(db, tx)
			if err != nil {
				fmt.Printf("Subscribe transaction events store transaction error: %v", err)
				// rate limit for caution but hopefully not needed
				rl.Take()
				continue
			}
		}
	}

}

var insertTx = `INSERT INTO tx (timestamp, tx_hash, amount, num_confirmations, block_hash, block_height,
                total_fees, dest_addresses, raw_tx_hex, label) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                ON CONFLICT (timestamp, tx_hash) DO NOTHING;`

func storeTransaction(db *sqlx.DB, tx *lnrpc.Transaction) error {

	_, err := db.Exec(insertTx,
		time.Unix(tx.TimeStamp, 0).UTC(),
		tx.TxHash,
		tx.Amount,
		tx.NumConfirmations,
		tx.BlockHash,
		tx.BlockHeight,
		tx.TotalFees,
		pq.Array(tx.DestAddresses),
		tx.RawTxHex,
		tx.Label,
	)

	if err != nil {
		return errors.Wrapf(err, `storeTransaction -> db.Exec(%s, %s, %s, %d, %d, %s, %d, %d, %v, %s, %s)`, insertTx,
			time.Unix(tx.TimeStamp, 0).UTC(),
			tx.TxHash,
			tx.Amount,
			tx.NumConfirmations,
			tx.BlockHash,
			tx.BlockHeight,
			tx.TotalFees,
			pq.Array(tx.DestAddresses),
			tx.RawTxHex,
			tx.Label)
	}
	return nil

}
