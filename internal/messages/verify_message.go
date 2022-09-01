package messages

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/internal/settings"
	"github.com/lncapital/torq/pkg/lnd_connect"
	"github.com/rs/zerolog/log"
)

func verifyMessage(db *sqlx.DB, req VerifyMessageRequest) (r VerifyMessageResponse, err error) {
	connectionDetails, err := settings.GetConnectionDetails(db)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting node connection details from the db: %s", err.Error())
		return r, errors.New("Error getting node connection details from the db")
	}

	conn, err := lnd_connect.Connect(
		connectionDetails.GRPCAddress,
		connectionDetails.TLSFileBytes,
		connectionDetails.MacaroonFileBytes)
	if err != nil {
		log.Error().Err(err).Msgf("can't connect to LND: %s", err.Error())
		return r, errors.Newf("can't connect to LND")
	}
	defer conn.Close()

	client := lnrpc.NewLightningClient(conn)

	verifyMsgReq := lnrpc.VerifyMessageRequest{
		Msg:       []byte(req.Message),
		Signature: req.Signature,
	}

	ctx := context.Background()

	verifyMsgResp, err := client.VerifyMessage(ctx, &verifyMsgReq)
	if err != nil {
		log.Error().Err(err).Msgf("Error verifying message: %v", err)
		return r, errors.Newf("Error verifying message")
	}

	r.Valid = verifyMsgResp.GetValid()
	r.PubKey = verifyMsgResp.GetPubkey()

	return r, nil
}
