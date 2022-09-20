package peers

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/internal/settings"
	"github.com/lncapital/torq/pkg/lnd_connect"
	"github.com/rs/zerolog/log"
)

func connectPeer(db *sqlx.DB, req connectPeerRequest) (r string, err error) {
	connPeerReq, err := processRequest(req)
	if err != nil {
		return r, err
	}
	connectionDetails, err := settings.GetConnectionDetails(db, false, req.NodeId)

	if err != nil {
		log.Error().Err(err).Msgf("Error getting node connection details from the db: %s", err.Error())
		return r, errors.New("Error getting node connecting details from the db")
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

	_, err = client.ConnectPeer(ctx, &connPeerReq)
	if err != nil {
		log.Error().Msgf("Error connect peer: %v", err)
		return r, err
	}

	return "Peer connected", nil
}

func processRequest(req connectPeerRequest) (r lnrpc.ConnectPeerRequest, err error) {
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
