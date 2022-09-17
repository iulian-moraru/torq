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

func listPeers(db *sqlx.DB, nodeId int) (r []*lnrpc.Peer, err error) {

	connectionDetails, err := settings.GetConnectionDetails(db, false, nodeId)

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

	listPeerReq := lnrpc.ListPeersRequest{}

	//if req.LatestError != nil {
	//	listPeerReq.LatestError = *req.LatestError
	//}

	client := lnrpc.NewLightningClient(conn)
	ctx := context.Background()

	resp, err := client.ListPeers(ctx, &listPeerReq)
	if err != nil {
		log.Error().Msgf("Error connect peer: %v", err)
		return r, err
	}

	//log.Debug().Msgf("REsponse: %v", resp.Peers)

	return resp.Peers, nil
}
