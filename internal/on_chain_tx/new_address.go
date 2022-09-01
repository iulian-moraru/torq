package on_chain_tx

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/internal/settings"
	"github.com/lncapital/torq/pkg/lnd_connect"
	"github.com/rs/zerolog/log"
)

func newAddress(db *sqlx.DB, req newAddressRequest) (r string, err error) {
	addressType := req.Type
	account := req.Account

	connectionDetails, err := settings.GetConnectionDetails(db)
	conn, err := lnd_connect.Connect(
		connectionDetails.GRPCAddress,
		connectionDetails.TLSFileBytes,
		connectionDetails.MacaroonFileBytes)
	if err != nil {
		log.Error().Err(err).Msgf("can't connect to LND: %s", err.Error())
		return r, errors.Newf("can't connect to LND")
	}

	defer conn.Close()

	lnAddressType := lnrpc.AddressType(addressType)
	newAddressReq := lnrpc.NewAddressRequest{Type: lnAddressType, Account: account}

	client := lnrpc.NewLightningClient(conn)
	ctx := context.Background()

	resp, err := client.NewAddress(ctx, &newAddressReq)
	if err != nil {
		log.Error().Msgf("Err creating new address: %v", err)
		return r, err
	}

	//log.Debug().Msgf("New address : %v", resp.Address)
	return resp.Address, nil
}
