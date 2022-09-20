package peers

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/lncapital/torq/pkg/server_errors"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
)

type lndAddress struct {
	PubKey string `json:"pubKey"`
	Host   string `json:"host"`
}

type connectPeerRequest struct {
	NodeId     int        `json:"nodeId"`
	LndAddress lndAddress `json:"lndAddress"`
	Perm       *bool      `json:"perm"`
	TimeOut    *uint64    `json:"timeOut"`
}

func connectPeerHandler(c *gin.Context, db *sqlx.DB) {
	var requestBody connectPeerRequest

	if err := c.BindJSON(&requestBody); err != nil {
		log.Error().Msgf("JSON binding the request body")
		server_errors.WrapLogAndSendServerError(c, err, "JSON binding the request body")
		return
	}

	resp, err := connectPeer(db, requestBody)
	if err != nil {
		server_errors.WrapLogAndSendServerError(c, err, "Connect peer")
	}

	c.JSON(http.StatusOK, resp)
}

func listPeersHandler(c *gin.Context, db *sqlx.DB) {

	nodeId, err := strconv.Atoi(c.Param("nodeId"))
	latestErr := c.Param("le")

	if err != nil {
		server_errors.WrapLogAndSendServerError(c, err, "List peers")
	}

	resp, err := listPeers(db, nodeId, latestErr)
	if err != nil {
		server_errors.WrapLogAndSendServerError(c, err, "List peers")
	}

	c.JSON(http.StatusOK, resp)
}

func RegisterPeersRoutes(r *gin.RouterGroup, db *sqlx.DB) {
	r.POST("add", func(c *gin.Context) { connectPeerHandler(c, db) })
	r.GET("list/:nodeId/*le", func(c *gin.Context) { listPeersHandler(c, db) })
}
