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

type timeStampedError struct {
	Timestamp uint64 `json:"timestamp"`
	Error     string `json:"error"`
}

type feature struct {
	Name       string `json:"name"`
	IsRequired bool   `json:"isRequired"`
	IsKnown    bool   `json:"isKnown"`
}

type featuresEntry struct {
	Key   uint32  `json:"key"`
	Value feature `json:"value"`
}

type peer struct {
	PubKey          string             `json:"pubKey"`
	Address         string             `json:"address"`
	BytesSent       uint64             `json:"bytesSent"`
	BytesRecv       uint64             `json:"bytesRecv"`
	SatSent         int64              `json:"satSent"`
	SatRecv         int64              `json:"satRecv"`
	Inbound         bool               `json:"inbound"`
	PingTime        int64              `json:"pingTime"`
	SyncType        string             `json:"syncType"`
	Features        []featuresEntry    `json:"features"`
	Errors          []timeStampedError `json:"errors"`
	FlapCount       int32              `json:"flapCount"`
	LastFlapNs      int64              `json:"lastFlapNs"`
	LastPingPayload []byte             `json:"lastPingPayload"`
}

func listPeers(db *sqlx.DB, nodeId int, latestErr string) (r []peer, err error) {

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
	if latestErr != "/" {
		listPeerReq.LatestError = true
	}

	client := lnrpc.NewLightningClient(conn)
	ctx := context.Background()

	resp, err := client.ListPeers(ctx, &listPeerReq)
	if err != nil {
		log.Error().Msgf("Error connect peer: %v", err)
		return r, err
	}

	var peerList []peer
	for _, connPeer := range resp.Peers {
		var p peer
		var features []featuresEntry
		var tsErrs []timeStampedError
		p.PubKey = connPeer.PubKey
		p.Address = connPeer.Address
		p.BytesSent = connPeer.BytesSent
		p.BytesRecv = connPeer.BytesRecv
		p.SatSent = connPeer.SatSent
		p.SatRecv = connPeer.SatRecv
		p.Inbound = connPeer.Inbound
		p.PingTime = connPeer.PingTime
		p.SyncType = connPeer.SyncType.String()

		for k, feat := range connPeer.Features {
			var ftr featuresEntry
			ftr.Key = k

			ftr.Value.Name = feat.Name
			ftr.Value.IsRequired = feat.IsRequired
			ftr.Value.IsKnown = feat.IsKnown
			features = append(features, ftr)
		}
		p.Features = features
		for _, tsErr := range connPeer.Errors {
			var tmpsErr timeStampedError
			tmpsErr.Timestamp = tsErr.Timestamp
			tmpsErr.Error = tsErr.Error
			tsErrs = append(tsErrs, tmpsErr)
		}
		p.Errors = tsErrs
		p.FlapCount = connPeer.FlapCount
		p.LastFlapNs = connPeer.LastFlapNs
		p.LastPingPayload = connPeer.LastPingPayload
		peerList = append(peerList, p)
	}

	return peerList, nil
}
