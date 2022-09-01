package virtual_network

import (
	"context"
	"github.com/cockroachdb/errors"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"strconv"
)

func MineBlocks(ctx context.Context, cli *client.Client, container dockercontainer.ContainerCreateCreatedBody,
	numberOfBlocks int) error {
	err := Retry(func() error {
		var output []string
		cmd := []string{"/start-btcctl.sh", "generate", strconv.Itoa(numberOfBlocks)}
		err := ExecJSONReturningCommand(ctx, cli, container, cmd, &output)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on btcd %s", container.ID)
		}
		if len(output) == 0 {
			return errors.New("Blocks not mined")
		}
		return nil
	}, defautDelayMS, defaultMaxDurationMS)
	if err != nil {
		errors.Wrap(err, "btcd mining blocks")
	}
	return nil
}

func SegWitActive(ctx context.Context, cli *client.Client, container dockercontainer.ContainerCreateCreatedBody) error {
	err := Retry(func() error {
		var blockchainInfo struct {
			Bip9Softforks struct {
				Segwit struct {
					Status string `json:"status"`
				} `json:"segwit"`
			} `json:"bip9_softforks"`
		}
		cmd := []string{"/start-btcctl.sh", "getblockchaininfo"}
		err := ExecJSONReturningCommand(ctx, cli, container, cmd, &blockchainInfo)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on btcd %s", container.ID)
		}
		if blockchainInfo.Bip9Softforks.Segwit.Status != "active" {
			return errors.New("Segwit not active")
		}
		return nil
	}, defautDelayMS, defaultMaxDurationMS)
	// Return the error
	return err
}
