package invoices

import (
	"context"
	"encoding/hex"
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/internal/settings"
	"github.com/lncapital/torq/pkg/lnd_connect"
	"github.com/rs/zerolog/log"
)

func newInvoice(db *sqlx.DB, req newInvoiceRequest) (r newInvoiceResponse, err error) {
	newInvoiceReq, err := processInvoiceReq(req)
	if err != nil {
		return r, err
	}

	connectionDetails, err := settings.GetConnectionDetails(db, false, req.NodeId)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting node connection details from the db: %s", err.Error())
		return r, errors.New("Error getting node connection details from the db")
	}

	if len(connectionDetails) == 0 {
		//log.Debug().Msgf("Node is deleted or disabled")
		return r, errors.Newf("Local node disabled or deleted")
	}

	conn, err := lnd_connect.Connect(
		connectionDetails[0].GRPCAddress,
		connectionDetails[0].TLSFileBytes,
		connectionDetails[0].MacaroonFileBytes)
	if err != nil {
		log.Error().Err(err).Msgf("can't connect to LND: %s", err.Error())
		return r, errors.Newf("can't connect to LND")
	}

	defer conn.Close()

	client := lnrpc.NewLightningClient(conn)

	ctx := context.Background()

	resp, err := client.AddInvoice(ctx, &newInvoiceReq)
	if err != nil {
		log.Error().Msgf("Err creating new invoice: %v", err)
		return r, err
	}

	//log.Debug().Msgf("Invoice : %v", resp.PaymentRequest)

	r.PaymentRequest = resp.GetPaymentRequest()
	r.AddIndex = resp.GetAddIndex()
	r.PaymentAddress = hex.EncodeToString(resp.GetPaymentAddr())

	return r, nil
}

func processInvoiceReq(req newInvoiceRequest) (inv lnrpc.Invoice, err error) {

	if req.NodeId == 0 {
		return inv, errors.New("Node id is missing")
	}

	if req.Memo != nil {
		inv.Memo = *req.Memo
	}

	if req.RPreImage != nil {
		rPreImage, err := hex.DecodeString(*req.RPreImage)
		if err != nil {
			return inv, errors.New("error decoding preimage")
		}
		inv.RPreimage = rPreImage
	}

	if req.ValueMsat != nil {
		inv.ValueMsat = *req.ValueMsat
	}

	//Default value is 3600 seconds
	if req.Expiry != nil {
		inv.Expiry = *req.Expiry
	}

	if req.FallBackAddress != nil {
		inv.FallbackAddr = *req.FallBackAddress
	}

	if req.Private != nil {
		inv.Private = *req.Private
	}

	if req.IsAmp != nil {
		inv.IsAmp = *req.IsAmp
	}

	return inv, nil
}
