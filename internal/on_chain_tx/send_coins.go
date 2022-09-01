package on_chain_tx

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/internal/settings"
	"github.com/lncapital/torq/pkg/lnd_connect"
	"github.com/rs/zerolog/log"
)

func sendCoins(db *sqlx.DB, req sendCoinsRequest) (r string, err error) {

	sendCoinsReq, err := processSendRequest(req)
	if err != nil {
		return r, err
	}
	connectionDetails, err := settings.GetConnectionDetails(db)
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
	ctx := context.Background()

	resp, err := client.SendCoins(ctx, &sendCoinsReq)
	if err != nil {
		log.Error().Msgf("Err sending coins: %v", err)
		return "Err sending coins", err
	}

	return resp.Txid, nil

}

func processSendRequest(req sendCoinsRequest) (r lnrpc.SendCoinsRequest, err error) {
	if req.Addr == "" {
		log.Error().Msgf("Address must be provided")
		return r, errors.New("Address must be provided")
	}

	if req.AmountSat <= 0 {
		log.Error().Msgf("Invalid amount")
		return r, errors.New("Invalid amount")
	}

	if req.TargetConf != nil && req.SatPerVbyte != nil {
		log.Error().Msgf("Either targetConf or satPerVbyte accepted")
		return r, errors.New("Either targetConf or satPerVbyte accepted")
	}

	r.Addr = req.Addr
	r.Amount = req.AmountSat

	if req.TargetConf != nil {
		r.TargetConf = *req.TargetConf
	}

	if req.SatPerVbyte != nil {
		r.SatPerVbyte = *req.SatPerVbyte
	}

	if req.SendAll != nil {
		r.SendAll = *req.SendAll
	}

	if req.Label != nil {
		r.Label = *req.Label
	}

	if req.MinConfs != nil {
		r.MinConfs = *req.MinConfs
	}

	if req.SpendUnconfirmed != nil {
		r.SpendUnconfirmed = *req.SpendUnconfirmed
	}

	return r, nil
}
