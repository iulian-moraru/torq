package lnd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lightningnetwork/lnd/lnrpc"
	"go.uber.org/ratelimit"

	"github.com/lncapital/torq/pkg/broadcast"
	"github.com/lncapital/torq/pkg/commons"
)

func fetchLastTxHeight(db *sqlx.DB) (txHeight int32, err error) {

	sqlLatest := `select coalesce(max(block_height),1) from tx;`

	row := db.QueryRow(sqlLatest)
	err = row.Scan(&txHeight)

	if err != nil {
		return 1, err
	}

	return txHeight, nil
}

func ImportTransactions(ctx context.Context, client lnrpc.LightningClient, db *sqlx.DB, nodeId int) error {

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
		err = storeTransaction(db, tx, nodeId)
		if err != nil {
			return errors.Wrap(err, "Store Transaction")
		}
	}

	return nil
}

// SubscribeAndStoreTransactions Subscribes to on-chain transaction events from LND and stores them in the
// database as a time series. It will also import unregistered transactions on startup.
func SubscribeAndStoreTransactions(ctx context.Context, client lnrpc.LightningClient, db *sqlx.DB,
	nodeSettings commons.ManagedNodeSettings, eventChannel chan interface{}) error {

	// Imports transactions not captured on the stream
	err := ImportTransactions(ctx, client, db, nodeSettings.NodeId)
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
				if errors.Is(ctx.Err(), context.Canceled) {
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

			err = storeTransaction(db, tx, nodeSettings.NodeId)
			if err != nil {
				fmt.Printf("Subscribe transaction events store transaction error: %v", err)
				// rate limit for caution but hopefully not needed
				rl.Take()
				continue
			}

			if eventChannel != nil {
				eventChannel <- broadcast.TransactionEvent{
					EventData: broadcast.EventData{
						EventTime: time.Now().UTC(),
						NodeId:    nodeSettings.NodeId,
					},
					Amount:    tx.Amount,
					Timestamp: time.Unix(tx.TimeStamp, 0),
					TotalFees: tx.TotalFees,
				}
			}
		}
	}
}

func storeTransaction(db *sqlx.DB, tx *lnrpc.Transaction, nodeId int) error {
	if tx == nil {
		return nil
	}

	// Here we're only storing the output addresses, not the output index, amount or if these
	// transactions are ours or not. We might want to add this.
	var destinationAddresses []string
	for _, output := range tx.OutputDetails {
		destinationAddresses = append(destinationAddresses, output.Address)
	}

	var insertTx = `INSERT INTO tx (timestamp, tx_hash, amount, num_confirmations, block_hash, block_height,
                total_fees, dest_addresses, raw_tx_hex, label, node_id)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
                ON CONFLICT (timestamp, tx_hash) DO NOTHING;`

	_, err := db.Exec(insertTx,
		time.Unix(tx.TimeStamp, 0).UTC(),
		tx.TxHash,
		tx.Amount,
		tx.NumConfirmations,
		tx.BlockHash,
		tx.BlockHeight,
		tx.TotalFees,
		pq.Array(destinationAddresses),
		tx.RawTxHex,
		tx.Label,
		nodeId,
	)

	if err != nil {
		return errors.Wrapf(err, `inserting transaction`)
	}

	return nil
}
