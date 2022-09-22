package settings

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/pkg/lnd_connect"
	"github.com/lncapital/torq/pkg/server_errors"
	"github.com/rs/zerolog/log"
)

type settings struct {
	DefaultDateRange  string `json:"defaultDateRange" db:"default_date_range"`
	PreferredTimezone string `json:"preferredTimezone" db:"preferred_timezone"`
	WeekStartsOn      string `json:"weekStartsOn" db:"week_starts_on"`
}

type timeZone struct {
	Name string `json:"name" db:"name"`
}

func RegisterSettingRoutes(r *gin.RouterGroup, db *sqlx.DB, restartLNDSub func() error) {
	r.GET("", func(c *gin.Context) { getSettingsHandler(c, db) })
	r.PUT("", func(c *gin.Context) { updateSettingsHandler(c, db) })
	r.GET("local-nodes", func(c *gin.Context) { getLocalNodesHandler(c, db) })
	r.POST("local-nodes", func(c *gin.Context) { addLocalNodeHandler(c, db, restartLNDSub) })
	r.GET("local-nodes/:nodeId", func(c *gin.Context) { getLocalNodeHandler(c, db) })
	r.PUT("local-nodes/:nodeId", func(c *gin.Context) { updateLocalNodeHandler(c, db, restartLNDSub) })
	r.DELETE("local-nodes/:nodeId", func(c *gin.Context) { updateLocalNodeDeletedHandler(c, db, restartLNDSub) })
	r.PUT("local-nodes/:nodeId/set-disabled", func(c *gin.Context) { updateLocalNodeDisabledHandler(c, db, restartLNDSub) })
}
func RegisterUnauthenticatedRoutes(r *gin.RouterGroup, db *sqlx.DB) {
	r.GET("timezones", func(c *gin.Context) { getTimeZonesHandler(c, db) })
}

func getTimeZonesHandler(c *gin.Context, db *sqlx.DB) {
	timeZones, err := getTimeZones(db)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, timeZones)
}
func getSettingsHandler(c *gin.Context, db *sqlx.DB) {
	settings, err := getSettings(db)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, settings)
}

func updateSettingsHandler(c *gin.Context, db *sqlx.DB) {
	var settings settings
	if err := c.BindJSON(&settings); err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}
	err := updateSettings(db, settings)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, settings)
}

type localNode struct {
	LocalNodeId       int                   `json:"localNodeId" form:"localNodeId" db:"local_node_id"`
	Implementation    string                `json:"implementation" form:"implementation" db:"implementation"`
	GRPCAddress       *string               `json:"grpcAddress" form:"grpcAddress" db:"grpc_address"`
	TLSFileName       *string               `json:"tlsFileName" db:"tls_file_name"`
	TLSFile           *multipart.FileHeader `form:"tlsFile"`
	MacaroonFileName  *string               `json:"macaroonFileName" db:"macaroon_file_name"`
	MacaroonFile      *multipart.FileHeader `form:"macaroonFile"`
	CreateOn          time.Time             `json:"createdOn" db:"created_on"`
	UpdatedOn         *time.Time            `json:"updatedOn"  db:"updated_on"`
	TLSDataBytes      []byte                `db:"tls_data"`
	MacaroonDataBytes []byte                `db:"macaroon_data"`
	PubKey            *string               `json:"pubKey" db:"pub_key"`
	Disabled          bool                  `json:"disabled" db:"disabled"`
	Deleted           bool                  `json:"deleted" db:"deleted"`
}

func getLocalNodeHandler(c *gin.Context, db *sqlx.DB) {
	nodeId, err := strconv.Atoi(c.Param("nodeId"))
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}
	localNode, err := getLocalNode(db, nodeId)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, localNode)
}

func getLocalNodesHandler(c *gin.Context, db *sqlx.DB) {
	localNode, err := getLocalNodes(db)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, localNode)
}

func addLocalNodeHandler(c *gin.Context, db *sqlx.DB, restartLNDSub func() error) {
	var localNode localNode

	if err := c.Bind(&localNode); err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}

	if localNode.TLSFile == nil || localNode.MacaroonFile == nil ||
		localNode.GRPCAddress == nil || *localNode.GRPCAddress == "" || localNode.Implementation == "" {
		server_errors.SendBadRequest(c, "All node details are required to add a new node")
		return
	}

	existingNodes, err := getLocalNodes(db)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}

	if len(existingNodes) > 0 {

		tlsDataFile, err := localNode.TLSFile.Open()
		if err != nil {
			server_errors.LogAndSendServerError(c, err)
			return
		}
		tlsCert, err := io.ReadAll(tlsDataFile)
		if err != nil {
			server_errors.LogAndSendServerError(c, err)
			return
		}
		if len(tlsCert) == 0 {
			server_errors.SendBadRequest(c, "Can't check new GRPC details without TLS Cert")
			return
		}

		macaroonDataFile, err := localNode.MacaroonFile.Open()
		if err != nil {
			server_errors.LogAndSendServerError(c, err)
			return
		}
		macaroonFile, err := io.ReadAll(macaroonDataFile)
		if err != nil {
			server_errors.LogAndSendServerError(c, err)
			return
		}
		if len(macaroonFile) == 0 {
			server_errors.SendBadRequest(c, "Can't check new GRPC details without Macaroon File")
			return
		}

		publicKey, err := getPublicKeyFromNode(*localNode.GRPCAddress, tlsCert, macaroonFile)
		if err != nil {
			server_errors.LogAndSendServerError(c, err)
			return
		}
		for _, existingNode := range existingNodes {
			if *existingNode.PubKey == publicKey {

				server_errors.SendUnprocessableEntity(c, "This node already exists")
				return
			}
		}
	}

	localNodeId, err := insertLocalNodeDetails(db, localNode)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}
	localNode.LocalNodeId = localNodeId

	err = saveTLSAndMacaroon(localNode, c, db)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}

	go func() {
		if err := restartLNDSub(); err != nil {
			log.Warn().Msg("Already restarting subscriptions, discarding restart request")
		}
	}()

	c.JSON(http.StatusOK, localNode)
}

func updateLocalNodeHandler(c *gin.Context, db *sqlx.DB, restartLNDSub func() error) {
	var localNode localNode
	if err := c.Bind(&localNode); err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}

	nodeId, err := strconv.Atoi(c.Param("nodeId"))
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}
	localNode.LocalNodeId = nodeId

	existingNodeDetails, err := getLocalNodeConnectionDetailsById(db, nodeId)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}

	// if GRPC details have changed need to check that the public keys (if existing) match
	if existingNodeDetails.GRPCAddress != localNode.GRPCAddress &&
		(existingNodeDetails.PubKey != nil && len(*existingNodeDetails.PubKey) != 0) {

		var TLSCert []byte
		if localNode.TLSFile != nil {
			tlsDataFile, err := localNode.TLSFile.Open()
			if err != nil {
				server_errors.LogAndSendServerError(c, err)
				return
			}
			tlsData, err := io.ReadAll(tlsDataFile)
			if err != nil {
				server_errors.LogAndSendServerError(c, err)
				return
			}
			TLSCert = tlsData
		}
		if len(TLSCert) == 0 && len(existingNodeDetails.TLSDataBytes) != 0 {
			TLSCert = existingNodeDetails.TLSDataBytes
		}
		if len(TLSCert) == 0 {
			server_errors.SendBadRequest(c, "Can't check new GRPC details without TLS Cert")
			return
		}

		var macaroonFile []byte
		if localNode.MacaroonFile != nil {
			macaroonDataFile, err := localNode.MacaroonFile.Open()
			if err != nil {
				server_errors.LogAndSendServerError(c, err)
				return
			}
			macaroonData, err := io.ReadAll(macaroonDataFile)
			if err != nil {
				server_errors.LogAndSendServerError(c, err)
				return
			}
			macaroonFile = macaroonData
		}
		if len(macaroonFile) == 0 && len(existingNodeDetails.MacaroonDataBytes) != 0 {
			macaroonFile = existingNodeDetails.MacaroonDataBytes
		}
		if len(macaroonFile) == 0 {
			server_errors.SendBadRequest(c, "Can't check new GRPC details without Macaroon File")
			return
		}

		publicKey, err := getPublicKeyFromNode(*localNode.GRPCAddress, TLSCert, macaroonFile)
		if err != nil {
			server_errors.LogAndSendServerError(c, err)
			return
		}

		if publicKey != *existingNodeDetails.PubKey {
			server_errors.SendUnprocessableEntity(c, "Pubkey does not match, create a new node instead of updating this one")
			return
		}
	}

	err = updateLocalNodeDetails(db, localNode)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}

	err = saveTLSAndMacaroon(localNode, c, db)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}

	maxTries := 30
	attempts := 0
	for {
		attempts++
		if attempts > maxTries {
			server_errors.LogAndSendServerError(c, errors.New("Failed to restart node subscriptions"))
			return
		}
		if err := restartLNDSub(); err != nil {
			log.Warn().Msg("Already restarting subscriptions, retrying")
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	c.JSON(http.StatusOK, localNode)
}

func getPublicKeyFromNode(grpcAddress string, tlsCert []byte, macaroonFile []byte) (publicKey string, err error) {
	conn, err := lnd_connect.Connect(grpcAddress, tlsCert, macaroonFile)
	if err != nil {
		return "", errors.Wrap(err, "Can't connect to node to verify public key, check all details including TLS Cert and Macaroon")
	}
	defer conn.Close()

	client := lnrpc.NewLightningClient(conn)
	ctx := context.Background()
	info, err := client.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return "", err
	}
	return info.IdentityPubkey, nil
}

func saveTLSAndMacaroon(localNode localNode, c *gin.Context, db *sqlx.DB) error {
	if localNode.TLSFile != nil {
		localNode.TLSFileName = &localNode.TLSFile.Filename
		tlsDataFile, err := localNode.TLSFile.Open()
		if err != nil {
			return err
		}
		tlsData, err := io.ReadAll(tlsDataFile)
		if err != nil {
			return err
		}
		localNode.TLSDataBytes = tlsData

		err = updateLocalNodeTLS(db, localNode)
		if err != nil {
			return err
		}
	}

	if localNode.MacaroonFile != nil {
		localNode.MacaroonFileName = &localNode.MacaroonFile.Filename
		macaroonDataFile, err := localNode.MacaroonFile.Open()
		if err != nil {
			return err
		}
		macaroonData, err := io.ReadAll(macaroonDataFile)
		if err != nil {
			return err
		}
		localNode.MacaroonDataBytes = macaroonData
		err = updateLocalNodeMacaroon(db, localNode)
		if err != nil {
			return err
		}
	}
	return nil
}

type disabledJSON struct {
	Disabled bool `json:"disabled"`
}

func updateLocalNodeDisabledHandler(c *gin.Context, db *sqlx.DB, restartLNDSub func() error) {
	var disabledJSON disabledJSON

	nodeId, err := strconv.Atoi(c.Param("nodeId"))
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}

	if err := c.Bind(&disabledJSON); err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}
	err = updateLocalNodeDisabledFlag(db, nodeId, disabledJSON.Disabled)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}

	maxTries := 30
	attempts := 0
	for {
		attempts++
		if attempts > maxTries {
			server_errors.LogAndSendServerError(c, errors.New("Failed to restart node subscriptions"))
			return
		}
		if err := restartLNDSub(); err != nil {
			log.Warn().Msg("Already restarting subscriptions, retrying")
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	c.Status(http.StatusOK)
}

func updateLocalNodeDeletedHandler(c *gin.Context, db *sqlx.DB, restartLNDSub func() error) {
	nodeId, err := strconv.Atoi(c.Param("nodeId"))
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}

	err = updateLocalNodeSetDeleted(db, nodeId)
	if err != nil {
		server_errors.LogAndSendServerError(c, err)
		return
	}

	go func() {
		if err := restartLNDSub(); err != nil {
			log.Warn().Msg("Already restarting subscriptions, discarding restart request")
		}
	}()

	c.Status(http.StatusOK)
}
