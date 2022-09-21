package peers

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/rs/zerolog/log"
)

func ConnectPeer(client lnrpc.LightningClient, ctx context.Context, req ConnectPeerRequest) (r string, err error) {
	connPeerReq, err := processRequest(req)

	_, err = client.ConnectPeer(ctx, &connPeerReq)
	if err != nil {
		log.Error().Msgf("Error connect peer: %v", err)
		return r, err
	}

	return "Peer connected", nil
}

func processRequest(req ConnectPeerRequest) (r lnrpc.ConnectPeerRequest, err error) {
	if req.NodeId == 0 {
		return r, errors.New("Node id missing")
	}

	if req.LndAddress.PubKey == "" || req.LndAddress.Host == "" {
		return r, errors.New("LND address not provided")
	}

	addr := lnrpc.LightningAddress{
		Pubkey: req.LndAddress.PubKey,
		Host:   req.LndAddress.Host,
	}

	r.Addr = &addr

	if req.Perm != nil {
		r.Perm = *req.Perm
	}

	if req.TimeOut != nil {
		r.Timeout = *req.TimeOut
	} else {
		r.Timeout = 30
	}

	return r, err
}
