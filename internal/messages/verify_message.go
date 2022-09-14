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
	if req.NodeId == 0 {
		return r, errors.Newf("Node Id missing")
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
