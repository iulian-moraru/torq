package peers

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/internal/settings"
	"github.com/lncapital/torq/pkg/lnd_connect"
	"github.com/lncapital/torq/pkg/server_errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"net/http"
	"strconv"
)

type LndAddress struct {
	PubKey string `json:"pubKey"`
	Host   string `json:"host"`
}

type ConnectPeerRequest struct {
	NodeId     int        `json:"nodeId"`
	LndAddress LndAddress `json:"lndAddress"`
	Perm       *bool      `json:"perm"`
	TimeOut    *uint64    `json:"timeOut"`
}

func connectPeerHandler(c *gin.Context, db *sqlx.DB) {
	var requestBody ConnectPeerRequest

	if err := c.BindJSON(&requestBody); err != nil {
		log.Error().Msgf("JSON binding the request body")
		server_errors.WrapLogAndSendServerError(c, err, "JSON binding the request body")
		return
	}

	conn, err := connectLND(db, requestBody.NodeId)
	if err != nil {
		server_errors.WrapLogAndSendServerError(c, err, "can't connect to LND")
	}
	defer conn.Close()

	client := lnrpc.NewLightningClient(conn)
	ctx := context.Background()

	resp, err := ConnectPeer(client, ctx, requestBody)
	if err != nil {
		server_errors.WrapLogAndSendServerError(c, err, "Connect peer")
	}

	c.JSON(http.StatusOK, resp)
}

func listPeersHandler(c *gin.Context, db *sqlx.DB) {

	nodeId, err := strconv.Atoi(c.Param("nodeId"))
	latestErr := c.Param("le")

	if err != nil {
		server_errors.WrapLogAndSendServerError(c, err, "Err getting node id")
	}

	conn, err := connectLND(db, nodeId)
	if err != nil {
		server_errors.WrapLogAndSendServerError(c, err, "can't connect to LND")
	}
	defer conn.Close()

	client := lnrpc.NewLightningClient(conn)
	ctx := context.Background()

	resp, err := ListPeers(client, ctx, latestErr)
	if err != nil {
		server_errors.WrapLogAndSendServerError(c, err, "List peers")
	}

	c.JSON(http.StatusOK, resp)
}

func connectLND(db *sqlx.DB, nodeId int) (conn *grpc.ClientConn, err error) {
	connectionDetails, err := settings.GetConnectionDetails(db, false, nodeId)

	if err != nil {
		log.Error().Err(err).Msgf("Error getting node connection details from the db: %s", err.Error())
		//server_errors.WrapLogAndSendServerError(c, err, "Error getting node connecting details from the db")
		return nil, err
	}

	if len(connectionDetails) == 0 {
		//log.Debug().Msgf("Node is deleted or disabled")
		return nil, errors.New("Local node disabled or deleted")
	}

	conn, err = lnd_connect.Connect(
		connectionDetails[0].GRPCAddress,
		connectionDetails[0].TLSFileBytes,
		connectionDetails[0].MacaroonFileBytes)
	if err != nil {
		log.Error().Err(err).Msgf("can't connect to LND: %s", err.Error())
		return nil, err
	}

	return conn, nil

}

func RegisterPeersRoutes(r *gin.RouterGroup, db *sqlx.DB) {
	r.POST("add", func(c *gin.Context) { connectPeerHandler(c, db) })
	r.GET("list/:nodeId/*le", func(c *gin.Context) { listPeersHandler(c, db) })
}
