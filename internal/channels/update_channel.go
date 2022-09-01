package channels

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/internal/settings"
	"github.com/lncapital/torq/pkg/lnd_connect"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"
)

//UpdateChannel
//Returns status, failed updates array
func updateChannels(db *sqlx.DB, req updateChanRequestBody) (r updateResponse, err error) {

	policyReq, err := createPolicyRequest(req)
	if err != nil {
		log.Error().Msgf("Err building policy request: %v", err)
		r.Status = "Err building policy request"
		return r, err
	}

	connectionDetails, err := settings.GetConnectionDetails(db)
	conn, err := lnd_connect.Connect(
		connectionDetails.GRPCAddress,
		connectionDetails.TLSFileBytes,
		connectionDetails.MacaroonFileBytes)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting node connection details from the db: %s", err.Error())
		return r, errors.New("Error getting node connection details from the db")
	}

	defer conn.Close()

	ctx := context.Background()
	client := lnrpc.NewLightningClient(conn)

	resp, err := client.UpdateChannelPolicy(ctx, &policyReq)

	if err != nil {
		log.Error().Msgf("Err updating channel/s: %v", err)
		return r, err
	}

	r = processUpdateResponse(resp)

	return r, nil
}

func createPolicyRequest(req updateChanRequestBody) (r lnrpc.PolicyUpdateRequest, err error) {

	updChanReq := lnrpc.PolicyUpdateRequest{}

	//Minimum supported value for TimeLockDelta is 18
	if req.TimeLockDelta < 18 {
		updChanReq.TimeLockDelta = 18
	} else {
		updChanReq.TimeLockDelta = req.TimeLockDelta
	}

	if req.ChannelPoint != nil {
		updChanReq.Scope, err = processChannelPoint(*req.ChannelPoint)
		if err != nil {
			return r, err
		}
	} else {
		updChanReq.Scope = &lnrpc.PolicyUpdateRequest_Global{Global: true}
	}

	if req.FeeRatePpm != nil {
		updChanReq.FeeRatePpm = *req.FeeRatePpm
	}

	if req.MaxHtlcMsat != nil {
		updChanReq.MaxHtlcMsat = *req.MaxHtlcMsat
	}

	if req.MinHtlcMsat != nil {
		updChanReq.MinHtlcMsat = *req.MinHtlcMsat
		updChanReq.MinHtlcMsatSpecified = true
	}

	if req.BaseFeeMsat != nil {
		updChanReq.BaseFeeMsat = *req.BaseFeeMsat
	}
	return updChanReq, nil
}

//processChannelPoint
//Split received channel point string into fundingtxid and outputindex
//Build PolicyUpdateRequest_ChanPoint: ChannelPoint_FundingTxidStr, ChannelPoint,
//Return PolicyUpdateRequest_ChanPoint
func processChannelPoint(chanPoint string) (cp *lnrpc.PolicyUpdateRequest_ChanPoint, err error) {

	//Split string into funding txid and output index
	splitChanPoint := strings.Split(chanPoint, ":")
	if len(splitChanPoint) != 2 {
		log.Error().Msgf("invalid channel point format: %v", err)
		return cp, errors.New("invalid channel point format")
	}

	txid := splitChanPoint[0]

	oIndxUint, err := strconv.ParseUint(splitChanPoint[1], 10, 1)
	if err != nil {
		log.Error().Msgf("parsing channel point output index: %v", err)
		return cp, errors.Newf("parsing channel point output index: %v", err)
	}

	outputIndex := uint32(oIndxUint)

	fundingTxid := lnrpc.ChannelPoint_FundingTxidStr{FundingTxidStr: txid}

	lnrpcCP := &lnrpc.ChannelPoint{FundingTxid: &fundingTxid, OutputIndex: outputIndex}
	cp = &lnrpc.PolicyUpdateRequest_ChanPoint{ChanPoint: lnrpcCP}

	return cp, nil
}

func processUpdateResponse(resp *lnrpc.PolicyUpdateResponse) (r updateResponse) {
	var failedUpdSlice []failedUpdate
	//log.Debug().Msgf("There are failed updates")
	if len(resp.GetFailedUpdates()) > 0 {
		for _, failUpdate := range resp.GetFailedUpdates() {
			//log.Debug().Msgf("txid byte: %v", failUpdate.Outpoint.TxidBytes)
			failedUpd := failedUpdate{}
			failedUpd.Reason = failUpdate.UpdateError
			failedUpd.UpdateError = failUpdate.UpdateError
			failedUpd.OutPoint.OutIndx = failUpdate.Outpoint.OutputIndex
			failedUpd.OutPoint.Txid = failUpdate.Outpoint.TxidStr
			failedUpdSlice = append(failedUpdSlice, failedUpd)
		}
		r.Status = "Channel/s update failed"
		r.FailedUpdates = failedUpdSlice
	} else {
		r.Status = "Channel/s updated"
	}
	return r
}
