package channels

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

func batchOpenChannels(db *sqlx.DB, req BatchOpenRequest) (r BatchOpenResponse, err error) {
	bOpenChanReq, err := checkPrepareReq(req)
	if err != nil {
		log.Error().Msgf("Error checkPrepareReq: %v", err)
		return r, err
	}

	connectionDetails, err := settings.GetConnectionDetails(db)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting node connection details from the db: %s", err.Error())
		return r, errors.New("Error getting node connection details from the db")
	}

	// TODO: change to select which local node
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

	bocResponse, err := client.BatchOpenChannel(ctx, &bOpenChanReq)
	if err != nil {
		log.Error().Msgf("Error BatchOpenChannel: %v", err)
		return r, err
	}

	resp, err := processBocResponse(bocResponse)
	if err != nil {
		log.Error().Msgf("Error processBocResponse: %v", err)
		return r, err
	}
	//log.Info().Msgf("pending channels: %v", bocResponse.String())
	return resp, nil

}

func checkPrepareReq(bocReq BatchOpenRequest) (req lnrpc.BatchOpenChannelRequest, err error) {
	if len(bocReq.Channels) == 0 {
		log.Debug().Msgf("channel array empty")
		return req, errors.New("Channels array is empty")
	}

	if bocReq.TargetConf != nil && bocReq.SatPerVbyte != nil {
		log.Error().Msgf("Only one fee model accepted")
		return req, errors.New("Either targetConf or satPerVbyte accepted")
	}

	var boChannels []*lnrpc.BatchOpenChannel

	for i, channel := range bocReq.Channels {
		var boChannel lnrpc.BatchOpenChannel
		pubKeyHex, err := hex.DecodeString(channel.NodePubkey)
		if err != nil {
			log.Error().Msgf("Err decoding string: %v, %v", i, err)
			return req, err
		}
		boChannel.NodePubkey = pubKeyHex

		if channel.LocalFundingAmount == 0 {
			log.Debug().Msgf("local funding amt 0")
			return req, errors.New("Local funding amount 0")
		}
		boChannel.LocalFundingAmount = channel.LocalFundingAmount

		if channel.Private != nil {
			boChannel.Private = *channel.Private
		}

		if channel.MinHtlcMsat != nil {
			boChannel.MinHtlcMsat = *channel.MinHtlcMsat
		}

		if channel.PushSat != nil {
			boChannel.PushSat = *channel.PushSat
		}
		boChannels = append(boChannels, &boChannel)
	}

	batchOpnReq := lnrpc.BatchOpenChannelRequest{
		Channels: boChannels,
	}

	if bocReq.SatPerVbyte != nil {
		batchOpnReq.SatPerVbyte = *bocReq.SatPerVbyte
	}

	if bocReq.TargetConf != nil {
		batchOpnReq.TargetConf = *bocReq.TargetConf
	}

	return batchOpnReq, nil
}

func processBocResponse(resp *lnrpc.BatchOpenChannelResponse) (r BatchOpenResponse, err error) {
	for _, pc := range resp.GetPendingChannels() {
		var bocPC pendingChannel
		chanPoint, err := translateChanPoint(pc.Txid, pc.OutputIndex)
		if err != nil {
			r = BatchOpenResponse{}
			log.Error().Msgf("Translate channel point err: %v", err)
			return r, err
		}

		bocPC.PendingChannelPoint = chanPoint
		r.PendingChannels = append(r.PendingChannels, bocPC)
	}
	return r, nil
}
