package virtual_network

import (
	"bytes"
	"context"
	"github.com/cockroachdb/errors"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"log"
	"strings"
)

const defautDelayMS = 2000          // 2s
const defaultMaxDurationMS = 120000 // 60s

func GetChannelBalance(ctx context.Context, cli *client.Client,
	container dockercontainer.ContainerCreateCreatedBody) (balance string, err error) {
	err = Retry(func() error {
		var channelBalance struct {
			Balance string `json:"balance"`
		}
		cmd := []string{"lncli", "--network=simnet", "channelbalance"}
		err := ExecJSONReturningCommand(ctx, cli, container, cmd, &channelBalance)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on %s", container.ID)
		}
		if channelBalance.Balance == "" {
			return errors.New("Payment not received")
		}
		if channelBalance.Balance == "0" {
			return errors.New("Payment not received")
		}
		balance = channelBalance.Balance
		return nil
	}, defautDelayMS, defaultMaxDurationMS)
	if err != nil {
		return "", errors.Wrap(err, "Getting channel balance")
	}
	return balance, nil
}

func PayInvoice(ctx context.Context, cli *client.Client,
	container dockercontainer.ContainerCreateCreatedBody, invoice string) error {
	err := Retry(func() error {
		cmd := []string{"lncli", "--network=simnet", "sendpayment", "--force", "--pay_req=" + invoice}
		var stderr bytes.Buffer
		stdout, stderr, err := ExecCommand(ctx, cli, container, cmd)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on %s", container.ID)
		}
		if len(stderr.Bytes()) > 0 {
			log.Println("Standard error not empty, retrying")
			return errors.New("Payment not sent")
		}
		if len(stdout.Bytes()) == 0 {
			log.Println("Standard out is empty, retrying")
			return errors.New("Payment not sent")
		}
		if strings.Contains(strings.ToLower(stdout.String()), "error") {
			log.Println("Word error was found in stdout, retrying")
			return errors.New("Payment not sent")
		}
		log.Println("Pay invoice command complete")
		return nil
	}, defautDelayMS, defaultMaxDurationMS)
	if err != nil {
		return errors.Wrap(err, "Sending payment")
	}
	return nil
}

func GenerateInvoice(ctx context.Context, cli *client.Client,
	container dockercontainer.ContainerCreateCreatedBody, amount string) (encodedInvoice string, err error) {
	err = Retry(func() error {
		var addInvoice struct {
			EncodedPayReq string `json:"payment_request"`
		}
		cmd := []string{"lncli", "--network=simnet", "addinvoice", "--amt=" + amount}
		err := ExecJSONReturningCommand(ctx, cli, container, cmd, &addInvoice)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on %s", container.ID)
		}
		if addInvoice.EncodedPayReq == "" {
			return errors.New("Invoice not generated")
		}
		encodedInvoice = addInvoice.EncodedPayReq
		return nil
	}, defautDelayMS, defaultMaxDurationMS)
	if err != nil {
		return "", errors.Wrap(err, "Creating invoice")
	}
	return encodedInvoice, nil
}

func CheckPeerExists(ctx context.Context, cli *client.Client, container dockercontainer.ContainerCreateCreatedBody,
	remotePubkey string) (bool, error) {
	var listPeers struct {
		Peers []struct {
			Pubkey   string `json:"pub_key"`
			SyncType string `json:"sync_type"`
		} `json:"peers"`
	}
	err := Retry(func() error {
		cmd := []string{"lncli", "--network=simnet", "listpeers"}
		err := ExecJSONReturningCommand(ctx, cli, container, cmd, &listPeers)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on Alice %s", container.ID)
		}
		for _, peer := range listPeers.Peers {
			if peer.Pubkey == remotePubkey {
				// peer found, return from retry
				if peer.SyncType != "ACTIVE_SYNC" {
					return noRetryError{}
				}
				return nil
			}
		}
		return errors.New("peer not found")
	}, defautDelayMS, defaultMaxDurationMS)
	if err != nil {
		return false, errors.Wrap(err, "Checking nodes are peers")
	}
	return true, nil
}

func ConnectPeer(ctx context.Context, cli *client.Client, container dockercontainer.ContainerCreateCreatedBody,
	remotePubkey string, remoteIPAddress string) error {
	err := Retry(func() error {
		cmd := []string{"lncli", "--network=simnet", "connect", remotePubkey + "@" + remoteIPAddress}
		var stderr bytes.Buffer
		_, stderr, err := ExecCommand(ctx, cli, container, cmd)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on %s", container.ID)
		}
		if len(stderr.Bytes()) > 0 {
			if !strings.Contains(string(stderr.Bytes()), "already connected to peer") {
				return errors.New("Peer didn't connect")
			}
		}

		// immediately check if the peer is connected as sometimes it seems to succeed and didn't
		peerConnected, err := CheckPeerExists(ctx, cli, container, remotePubkey)
		if err != nil || !peerConnected {
			return errors.New("Peer didn't connect")
		}

		return nil
	}, defautDelayMS, defaultMaxDurationMS)
	if err != nil {
		return errors.Wrap(err, "Connecting peers")
	}
	return nil
}

func CloseChannel(ctx context.Context, cli *client.Client, container dockercontainer.ContainerCreateCreatedBody,
	channelPoint string) (closeTxId string, err error) {
	err = Retry(func() error {
		var closeChannel struct {
			ClosingTxId string `json:"closing_txid"`
		}
		fundingTxId := channelPoint[:strings.IndexByte(channelPoint, ':')]
		outputIndex := channelPoint[strings.IndexByte(channelPoint, ':')+1:]
		cmd := []string{"lncli", "--network=simnet", "closechannel", "--funding_txid=" + fundingTxId, "--output_index=" + outputIndex}
		err := ExecJSONReturningCommand(ctx, cli, container, cmd, &closeChannel)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on Alice %s", container.ID)
		}
		if closeChannel.ClosingTxId == "" {
			return errors.New("Channel not closed")
		}
		closeTxId = closeChannel.ClosingTxId
		return nil
	}, defautDelayMS, defaultMaxDurationMS)
	if err != nil {
		return "", errors.Wrap(err, "Closing channel")
	}
	return closeTxId, nil
}

func CreateChannel(ctx context.Context, cli *client.Client, container dockercontainer.ContainerCreateCreatedBody,
	remotePubkey string, amount string, btcd dockercontainer.ContainerCreateCreatedBody) (channelPoint string, err error) {

	var fundingTxId string
	err = Retry(func() error {
		var openChannel struct {
			FundingTxId string `json:"funding_txid"`
		}
		cmd := []string{"lncli", "--network=simnet", "openchannel", "--node_key=" + remotePubkey, "--local_amt=" + amount}
		err := ExecJSONReturningCommand(ctx, cli, container, cmd, &openChannel)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on %s", container.ID)
		}
		if openChannel.FundingTxId == "" {
			return errors.New("Channel not created")
		}
		fundingTxId = openChannel.FundingTxId
		return nil
	}, defautDelayMS, defaultMaxDurationMS)
	if err != nil {
		return "", errors.Wrap(err, "Creating channel")
	}
	log.Printf("Funding transaction ID: %s\n", fundingTxId)

	log.Println("Include funding transaction in block thereby opening the channel")

	err = MineBlocks(ctx, cli, btcd, 30)
	if err != nil {
		return "", errors.Wrap(err, "Mining blocks")
	}

	log.Println("Blocks mined")
	log.Println("Checking channel is open")

	err = Retry(func() error {
		var listChannels struct {
			Channels []struct {
				ChannelPoint string `json:"channel_point"`
			} `json:"channels"`
		}
		cmd := []string{"lncli", "--network=simnet", "listchannels"}
		err := ExecJSONReturningCommand(ctx, cli, container, cmd, &listChannels)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on %s", container.ID)
		}
		if len(listChannels.Channels) == 0 {
			return errors.New("Channel not open")
		}
		if listChannels.Channels[0].ChannelPoint == "" {
			return errors.New("Channel not open")
		}
		channelPoint = listChannels.Channels[0].ChannelPoint
		return nil
	}, defautDelayMS, defaultMaxDurationMS)
	if err != nil {
		return "", errors.Wrap(err, "Creating channel")
	}
	log.Printf("Channel point: %s\n", channelPoint)

	return channelPoint, nil

}

func GetNewAddress(ctx context.Context, cli *client.Client, instance dockercontainer.ContainerCreateCreatedBody) (addr string, err error) {

	err = Retry(func() error {
		var address struct {
			Address string `json:"address"`
		}
		cmd := []string{"lncli", "--network=simnet", "newaddress", "np2wkh"}
		err := ExecJSONReturningCommand(ctx, cli, instance, cmd, &address)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on Alice %s", instance.ID)
		}
		if address.Address == "" {
			return errors.New("Not a valid address")
		}
		addr = address.Address
		return nil
	}, defautDelayMS, defaultMaxDurationMS)
	if err != nil {
		return addr, errors.Wrap(err, "Getting new address")
	}

	return addr, nil
}

func GetPubKey(ctx context.Context, cli *client.Client, container dockercontainer.ContainerCreateCreatedBody) (pubkey string, err error) {
	var getInfo struct {
		IdentityPubkey string `json:"identity_pubkey"`
	}
	err = Retry(func() error {
		cmd := []string{"lncli", "--network=simnet", "getinfo"}
		err = ExecJSONReturningCommand(ctx, cli, container, cmd, &getInfo)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on %s", container.ID)
		}
		if getInfo.IdentityPubkey == "" {
			return errors.New("Invalid Pubkey")
		}
		pubkey = getInfo.IdentityPubkey
		return nil
	}, defautDelayMS, defaultMaxDurationMS)
	if err != nil {
		return "", errors.Wrap(err, "Calling getinfo")
	}
	return pubkey, nil
}

func GetOnchainBalance(ctx context.Context, cli *client.Client, container dockercontainer.ContainerCreateCreatedBody) (balance string, err error) {
	err = Retry(func() error {
		var walletBalance struct {
			ConfirmedBalance string `json:"confirmed_balance"`
		}
		cmd := []string{"lncli", "--network=simnet", "walletbalance"}
		err := ExecJSONReturningCommand(ctx, cli, container, cmd, &walletBalance)
		if err != nil {
			return errors.Wrapf(err, "Running exec command on %s", container.ID)
		}
		if walletBalance.ConfirmedBalance == "" {
			return errors.New("Balance not confirmed")
		}
		if walletBalance.ConfirmedBalance == "0" {
			return errors.New("Balance not confirmed")
		}
		balance = walletBalance.ConfirmedBalance
		return nil
	}, defautDelayMS, defaultMaxDurationMS)
	if err != nil {
		return "", errors.Wrap(err, "Getting balance")
	}
	return balance, nil

}
