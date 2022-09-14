package settings

import (
	"github.com/jmoiron/sqlx"
)

type ConnectionDetails struct {
	LocalNodeId       int
	GRPCAddress       string
	TLSFileBytes      []byte
	MacaroonFileBytes []byte
	Disabled          bool
	Deleted           bool
}

func GetConnectionDetails(db *sqlx.DB, allNodes bool, nodeId int) ([]ConnectionDetails, error) {
	var localNodes []localNode
	var activeNode localNode
	var err error
	connectionDetailsList := []ConnectionDetails{}
	if allNodes {
		//Get all nodes not disabled and not deleted
		localNodes, err = getLocalNodeConnectionDetails(db)
		if err != nil {
			return []ConnectionDetails{}, err
		}

		for _, localNodeDetails := range localNodes {
			if (localNodeDetails.GRPCAddress == nil) || (localNodeDetails.TLSDataBytes == nil) || (localNodeDetails.
				MacaroonDataBytes == nil) {
				continue
			}
			connectionDetailsList = append(connectionDetailsList, ConnectionDetails{
				LocalNodeId:       localNodeDetails.LocalNodeId,
				GRPCAddress:       *localNodeDetails.GRPCAddress,
				TLSFileBytes:      localNodeDetails.TLSDataBytes,
				MacaroonFileBytes: localNodeDetails.MacaroonDataBytes})
		}
	} else {
		//Get node details based on node id
		activeNode, err = getLocalNodeConnectionDetailsById(db, nodeId)
		if err != nil {
			return []ConnectionDetails{}, err
		}

		if activeNode.Deleted || activeNode.Disabled {
			return []ConnectionDetails{}, nil
		}
		connectionDetailsList = append(connectionDetailsList, ConnectionDetails{
			LocalNodeId:       activeNode.LocalNodeId,
			GRPCAddress:       *activeNode.GRPCAddress,
			TLSFileBytes:      activeNode.TLSDataBytes,
			MacaroonFileBytes: activeNode.MacaroonDataBytes,
			Disabled:          activeNode.Disabled,
			Deleted:           activeNode.Deleted,
		})
	}

	return connectionDetailsList, nil
}
