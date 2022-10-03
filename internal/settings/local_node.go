package settings

import (
	"github.com/cockroachdb/errors"
	"github.com/jmoiron/sqlx"
	"golang.org/x/exp/slices"
)

func GetNodeIdByGRPC(db *sqlx.DB, nodeConnectionDetails ConnectionDetails) (int, error) {
	localNodes, err := getLocalNodeConnectionDetails(db)
	if err != nil {
		return 0, errors.Wrap(err, "Getting local nodes from db")
	}

	index := slices.IndexFunc(localNodes, func(node localNode) bool {
		return node.GRPCAddress != nil && *node.GRPCAddress == nodeConnectionDetails.GRPCAddress
	})

	return index, nil
}

func AddNodeToDB(db *sqlx.DB, nodeConnectionDetails ConnectionDetails) error {

	publicKey, err := getPublicKeyFromNode(nodeConnectionDetails.GRPCAddress, nodeConnectionDetails.TLSFileBytes,
		nodeConnectionDetails.MacaroonFileBytes)
	if err != nil {
		return errors.Wrap(err, "Getting public key from node")
	}
	existingNodes, err := getLocalNodeConnectionDetails(db)
	if err != nil {
		return errors.Wrap(err, "Getting local nodes from db")
	}
	localNodeFromConfig := localNode{
		Implementation: "LND",
		GRPCAddress:    &nodeConnectionDetails.GRPCAddress,
	}

	for _, existingNode := range existingNodes {
		// if the public key already exists, update the item in the database to match new details
		if existingNode.PubKey != nil && *existingNode.PubKey == publicKey {
			err = updateLocalNodeDetails(db, localNodeFromConfig)
			if err != nil {
				return errors.Wrap(err, "Inserting local node to database")
			}
			nodeConnectionDetails.LocalNodeId = existingNode.LocalNodeId
			err = UpdateNodeFiles(db, nodeConnectionDetails)
			if err != nil {
				return err
			}
			return nil
		}
	}

	id, err := insertLocalNodeDetails(db, localNodeFromConfig)
	if err != nil {
		return errors.Wrap(err, "Inserting local node to database")
	}
	nodeConnectionDetails.LocalNodeId = id
	err = UpdateNodeFiles(db, nodeConnectionDetails)
	if err != nil {
		return err
	}

	return nil
}

func UpdateNodeFiles(db *sqlx.DB, nodeConnectionDetails ConnectionDetails) error {
	tlsFileName := "tls.cert"
	macaroonFileName := "lnd.macaroon"
	localNodeFromConfig := localNode{
		Implementation:    "LND",
		LocalNodeId:       nodeConnectionDetails.LocalNodeId,
		GRPCAddress:       &nodeConnectionDetails.GRPCAddress,
		TLSFileName:       &tlsFileName,
		TLSDataBytes:      nodeConnectionDetails.TLSFileBytes,
		MacaroonFileName:  &macaroonFileName,
		MacaroonDataBytes: nodeConnectionDetails.MacaroonFileBytes,
	}
	err := updateLocalNodeTLS(db, localNodeFromConfig)
	if err != nil {
		return errors.Wrap(err, "Updating local node TLS file")
	}
	err = updateLocalNodeMacaroon(db, localNodeFromConfig)
	if err != nil {
		return errors.Wrap(err, "Updating local node Macaroon file")
	}
	return nil
}
