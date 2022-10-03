package invoices

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/internal/channels"
	"github.com/lncapital/torq/internal/settings"
	"github.com/lncapital/torq/pkg/lnd_connect"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
)

type feature struct {
	Name       string `json:"name"`
	IsKnown    bool   `json:"isKnown"`
	IsRequired bool   `json:"isRequired"`
}

type featureMap map[uint32]feature

type hopHint struct {
	LNDShortChannelId uint64 `json:"lndShortChannelId"`
	ShortChannelId    string `json:"shortChannelId"`
	NodeId            string `json:"nodeId"`
	FeeBase           uint32 `json:"feeBase"`
	CltvExpiryDelta   uint32 `json:"cltvExpiryDelta"`
	FeeProportional   uint32 `json:"feeProportionalMillionths"`
}

type routeHint struct {
	HopHints []hopHint `json:"hopHints"`
}

type DecodedInvoice struct {
	NodeAlias         string      `json:"nodeAlias"`
	PaymentRequest    string      `json:"paymentRequest"`
	DestinationPubKey string      `json:"destinationPubKey"`
	RHash             string      `json:"rHash"`
	Memo              string      `json:"memo"`
	ValueMsat         int64       `json:"valueMsat"`
	PaymentAddr       string      `json:"paymentAddr"`
	FallbackAddr      string      `json:"fallbackAddr"`
	Expiry            int64       `json:"expiry"`
	CreatedAt         int64       `json:"createdAt"`
	CltvExpiry        int64       `json:"cltvExpiry"`
	Private           bool        `json:"private"`
	Features          featureMap  `json:"features"`
	RouteHints        []routeHint `json:"routeHints"`
}

func constructRouteHints(routeHints []*lnrpc.RouteHint) []routeHint {

	var r []routeHint

	for _, rh := range routeHints {
		var hopHints []hopHint
		for _, hh := range rh.HopHints {
			hopHints = append(hopHints, hopHint{
				LNDShortChannelId: hh.ChanId,
				ShortChannelId:    channels.ConvertLNDShortChannelID(hh.ChanId),
				NodeId:            hh.NodeId,
				FeeBase:           hh.FeeBaseMsat,
				CltvExpiryDelta:   hh.CltvExpiryDelta,
				FeeProportional:   hh.FeeProportionalMillionths,
			})
		}
		r = append(r, routeHint{
			HopHints: hopHints,
		})
	}

	return r
}

func constructFeatureMap(features map[uint32]*lnrpc.Feature) featureMap {

	f := featureMap{}
	for n, v := range features {
		f[n] = feature{
			Name:       v.Name,
			IsKnown:    v.IsKnown,
			IsRequired: v.IsRequired,
		}
	}

	return f
}

func constructDecodedInvoice(decodedInvoice *lnrpc.PayReq) *DecodedInvoice {
	return &DecodedInvoice{
		DestinationPubKey: decodedInvoice.Destination,
		RHash:             decodedInvoice.PaymentHash,
		Memo:              decodedInvoice.Description,
		ValueMsat:         decodedInvoice.NumMsat,
		FallbackAddr:      decodedInvoice.FallbackAddr,
		CreatedAt:         decodedInvoice.Timestamp,
		Expiry:            decodedInvoice.Expiry,
		CltvExpiry:        decodedInvoice.CltvExpiry,
		RouteHints:        constructRouteHints(decodedInvoice.RouteHints),
		Features:          constructFeatureMap(decodedInvoice.Features),
		PaymentAddr:       hex.EncodeToString(decodedInvoice.PaymentAddr),
	}
}

// decodeInvoice Decode a lightning invoice
func decodeInvoice(db *sqlx.DB, invoice string) (*DecodedInvoice, error) {
	log.Info().Msgf("Decoding invoice: %s", invoice)
	// Get lnd client
	connectionDetails, err := settings.GetConnectionDetails(db)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting node connection details from the db: %s", err.Error())
		return nil, errors.New("Error getting node connection details from the db")
	}

	// TODO: change to select which local node
	conn, err := lnd_connect.Connect(
		connectionDetails[0].GRPCAddress,
		connectionDetails[0].TLSFileBytes,
		connectionDetails[0].MacaroonFileBytes)
	if err != nil {
		log.Error().Err(err).Msgf("can't connect to LND: %s", err.Error())
		return nil, errors.Newf("can't connect to LND")
	}
	defer conn.Close()

	client := lnrpc.NewLightningClient(conn)
	// Decode invoice
	ctx := context.Background()
	// TODO: Handle different error types like incorrect checksum etc to explain why the decode failed.
	decodedInvoice, err := client.DecodePayReq(ctx, &lnrpc.PayReqString{
		PayReq: invoice,
	})
	if err != nil {
		return nil, err
	}

	nodeInfo, err := client.GetNodeInfo(ctx, &lnrpc.NodeInfoRequest{
		PubKey:          decodedInvoice.Destination,
		IncludeChannels: false,
	})
	if err != nil {
		return nil, err
	}

	torqDecodedInvoice := constructDecodedInvoice(decodedInvoice)
	torqDecodedInvoice.NodeAlias = nodeInfo.Node.Alias

	return torqDecodedInvoice, nil
}

func decodeInvoiceHandler(c *gin.Context, db *sqlx.DB) {
	invoice := c.Query("invoice")

	di, err := decodeInvoice(db, invoice)

	if err != nil {
		fmt.Println(err)

		if strings.Contains(err.Error(), "checksum failed") {
			//errResponse := server_errors.SingleFieldError("invoice", "CHECKSUM_FAILED")
			c.JSON(http.StatusBadRequest, gin.H{"error": "CHECKSUM_FAILED"})
			return
		}
		//server_errors.WrapLogAndSendServerError(c, err, "could not decode invoice")
		c.JSON(http.StatusBadRequest, gin.H{"error": "COULD_NOT_DECODE_INVOICE"})
		return
	}

	c.JSON(http.StatusOK, di)
}
