package channels

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/internal/peers"
	"github.com/lncapital/torq/internal/settings"
	"github.com/lncapital/torq/pkg/lnd_connect"
	"github.com/rs/zerolog/log"
	"io"
	"strings"
)

type OpenChannelRequest struct {
	NodeId             int     `json:"nodeId"`
	SatPerVbyte        *uint64 `json:"satPerVbyte"`
	NodePubKey         string  `json:"nodePubKey"`
	Host               *string `json:"host"`
	LocalFundingAmount int64   `json:"localFundingAmount"`
	PushSat            *int64  `json:"pushSat"`
	TargetConf         *int32  `json:"targetConf"`
	Private            *bool   `json:"private"`
	MinHtlcMsat        *int64  `json:"minHtlcMsat"`
	RemoteCsvDelay     *uint32 `json:"remoteCsvDelay"`
	MinConfs           *int32  `json:"minConfs"`
	SpendUnconfirmed   *bool   `json:"spendUnconfirmed"`
	CloseAddress       *string `json:"closeAddress"`
}

type OpenChannelResponse struct {
	ReqId               string `json:"reqId"`
	Status              string `json:"status"`
	ChannelPoint        string `json:"channelPoint,omitempty"`
	PendingChannelPoint string `json:"pendingChannelPoint,omitempty"`
}

type PsbtDetails struct {
	FundingAddress string `json:"funding_address,omitempty"`
	FundingAmount  int64  `json:"funding_amount,omitempty"`
	Psbt           []byte `json:"psbt,omitempty"`
}

func OpenChannel(db *sqlx.DB, wChan chan interface{}, req OpenChannelRequest, reqId string) (err error) {
	// TODO: Add support for batch opening channels

	openChanReq, err := prepareOpenRequest(req)
	if err != nil {
		return err
	}

	connectionDetails, err := settings.GetConnectionDetails(db, false, req.NodeId)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting node connection details from the db: %s", err.Error())
		return errors.New("Error getting node connection details from the db")
	}

	if len(connectionDetails) == 0 {
		//log.Debug().Msgf("Node is deleted or disabled")
		return errors.Newf("Local node disabled or deleted")
	}

	conn, err := lnd_connect.Connect(
		connectionDetails[0].GRPCAddress,
		connectionDetails[0].TLSFileBytes,
		connectionDetails[0].MacaroonFileBytes)
	if err != nil {
		log.Error().Err(err).Msgf("can't connect to LND: %s", err.Error())
		return errors.Newf("can't connect to LND")
	}
	defer conn.Close()

	client := lnrpc.NewLightningClient(conn)
	ctx := context.Background()

	//If host provided - check if peer and if needed connect peer
	if req.NodePubKey != "" && req.Host != nil {
		log.Debug().Msgf("Host provided. connect peer")
		if err := checkConnectPeer(client, ctx, req.NodeId, req.NodePubKey, *req.Host); err != nil {
			return err
		}
	}

	//Send open channel request
	openChanRes, err := client.OpenChannel(ctx, &openChanReq)
	// TODO: Add automatic peer connection: https://api.lightning.community/#connectpeer
	//   If the node is not connected and the user did not specify any connection details get the connection options and
	//   ask the user to choose.
	//   https://api.lightning.community/#getnodeinfo
	if err != nil { // Use switch and check error type for peer not connected.
		log.Error().Msgf("Err opening channel: %v", err)
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		resp, err := openChanRes.Recv()

		if err == io.EOF {
			//log.Info().Msgf("Open channel EOF")
			return nil
		}

		if err != nil {
			if strings.Contains(err.Error(), "is not online") {
				log.Error().Msgf("Peer is not connected")
				wChan <- errors.Newf("Peer is not connected. Provide full IP")
				return errors.Newf("Peer is not connected. Provide full IP")
			}
			log.Error().Msgf("could not open channel: %v", err)
			wChan <- errors.Newf("could not open channel: %v", err)
			return err
		}

		r, err := processOpenResponse(resp)
		if err != nil {
			return err
		}
		wChan <- r

	}
}

func prepareOpenRequest(ocReq OpenChannelRequest) (r lnrpc.OpenChannelRequest, err error) {
	if ocReq.NodeId == 0 {
		return r, errors.New("Node id is missing")
	}

	if ocReq.SatPerVbyte != nil && ocReq.TargetConf != nil {
		return r, errors.New("Cannot set both SatPerVbyte and TargetConf")
	}

	pubKeyHex, err := hex.DecodeString(ocReq.NodePubKey)
	if err != nil {
		return r, errors.New("error decoding public key hex")
	}

	//open channel request
	openChanReq := lnrpc.OpenChannelRequest{
		NodePubkey: pubKeyHex,

		// This is the amount we are putting into the channel (channel size)
		LocalFundingAmount: ocReq.LocalFundingAmount,
	}

	// The amount to give the other node in the opening process.
	// NB: This means you will give the other node this amount of sats
	if ocReq.PushSat != nil {
		openChanReq.PushSat = *ocReq.PushSat
	}

	if ocReq.SatPerVbyte != nil {
		openChanReq.SatPerVbyte = *ocReq.SatPerVbyte
	}

	if ocReq.TargetConf != nil {
		openChanReq.TargetConf = *ocReq.TargetConf
	}

	if ocReq.Private != nil {
		openChanReq.Private = *ocReq.Private
	}

	if ocReq.MinHtlcMsat != nil {
		openChanReq.MinHtlcMsat = *ocReq.MinHtlcMsat
	}

	if ocReq.RemoteCsvDelay != nil {
		openChanReq.RemoteCsvDelay = *ocReq.RemoteCsvDelay
	}

	if ocReq.MinConfs != nil {
		openChanReq.MinConfs = *ocReq.MinConfs
	}

	if ocReq.SpendUnconfirmed != nil {
		openChanReq.SpendUnconfirmed = *ocReq.SpendUnconfirmed
	}

	if ocReq.CloseAddress != nil {
		openChanReq.CloseAddress = *ocReq.CloseAddress
	}
	return openChanReq, nil
}

func processOpenResponse(resp *lnrpc.OpenStatusUpdate) (*OpenChannelResponse, error) {

	switch resp.GetUpdate().(type) {
	case *lnrpc.OpenStatusUpdate_ChanPending:
		log.Info().Msgf("Channel pending")

		pc := resp.GetChanPending()
		pcp, err := translateChanPoint(pc.Txid, pc.OutputIndex)
		if err != nil {
			log.Error().Msgf("Error translating pending channel point")
			return nil, err
		}

		return &OpenChannelResponse{
			Status:              "PENDING",
			PendingChannelPoint: pcp,
		}, nil

	case *lnrpc.OpenStatusUpdate_ChanOpen:
		log.Info().Msgf("Channel open")

		oc := resp.GetChanOpen()
		ocp, err := translateChanPoint(oc.ChannelPoint.GetFundingTxidBytes(), oc.ChannelPoint.OutputIndex)
		if err != nil {
			log.Error().Msgf("Error translating channel point")
			return nil, err
		}

		return &OpenChannelResponse{
			Status:       "OPEN",
			ChannelPoint: ocp,
		}, nil

	case *lnrpc.OpenStatusUpdate_PsbtFund:
		log.Error().Msg("Channel psbt fund response received. Can't process this response")
		return nil, errors.New("Channel psbt fund response received. Can't process this response")
	default:
	}

	return nil, nil
}

func translateChanPoint(cb []byte, oi uint32) (string, error) {
	ch, err := chainhash.NewHash(cb)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%d", ch.String(), oi), nil
}

func checkConnectPeer(client lnrpc.LightningClient, ctx context.Context, nodeId int, remotePubkey string, host string) (err error) {

	peerList, err := peers.ListPeers(client, ctx, "/le")
	if err != nil {
		return err
	}

	for _, peer := range peerList {
		if peer.PubKey == remotePubkey {
			log.Debug().Msgf("Peer is connected")
			// peer found
			return nil
		}
	}

	req := peers.ConnectPeerRequest{
		NodeId: nodeId,
		LndAddress: peers.LndAddress{
			PubKey: remotePubkey,
			Host:   host,
		},
	}

	_, err = peers.ConnectPeer(client, ctx, req)
	if err != nil {
		log.Error().Msgf("Err connecting peer")
		return err
	}
	//connect peer
	log.Debug().Msgf("Peer connected. Open channel next")

	return nil
}
