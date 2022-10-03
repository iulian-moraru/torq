package virtual_network

import (
	"context"
	"fmt"
	"github.com/cockroachdb/errors"
	"github.com/docker/docker/client"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func createDockerEnvironment(name string, createDatabase bool) (de DockerDevEnvironment, err error) {

	btcdName := name + "-btcd"
	aliceName := name + "-alice"
	bobName := name + "-bob"
	carolName := name + "-carol"

	cli, err := client.NewClientWithOpts()
	if err != nil {
		return de, errors.Newf("Getting new docker client: %v\n", err)
	}

	de = DockerDevEnvironment{
		Client:           cli,
		NetworkName:      name,
		SharedVolumeName: name + "-shared",
	}

	if createDatabase {
		// Add config for Torq database
		de.AddContainer(name+"-torq-db",
			"timescale/timescaledb:latest-pg14",
			nil,
			[]string{"POSTGRES_PASSWORD=password", "PGPORT=5444"},
			nil,
			"5444")
	}

	// Add config for btcd
	de.AddContainer(
		btcdName,
		name+"/btcd",
		[]string{
			de.SharedVolumeName + ":/rpc",
			btcdName + ":/data",
		},
		[]string{"NETWORK=simnet"},
		nil,
		"",
	)

	// Add config for alice
	de.AddContainer(
		aliceName,
		name+"/lnd",
		[]string{
			de.SharedVolumeName + ":/rpc",
			aliceName + ":/root/.lnd",
		},
		[]string{"NETWORK=simnet"},
		nil,
		"",
	)

	// Add config for bob
	de.AddContainer(
		bobName,
		name+"/lnd",
		[]string{
			de.SharedVolumeName + ":/rpc",
			bobName + ":/root/.lnd",
		},
		[]string{"NETWORK=simnet"},
		nil,
		"10009",
	)

	// Add config for carol
	de.AddContainer(
		carolName,
		name+"/lnd",
		[]string{
			de.SharedVolumeName + ":/rpc",
			carolName + ":/root/.lnd",
		},
		[]string{"NETWORK=simnet"},
		nil,
		"",
	)

	return de, nil
}

func StartVirtualNetwork(name string, withDatabase bool) error {

	torqDbName := name + "-torq-db"
	btcdName := name + "-btcd"
	aliceName := name + "-alice"
	bobName := name + "-bob"
	carolName := name + "-carol"

	de, err := createDockerEnvironment(name, withDatabase)
	if err != nil {
		return err
	}

	torqDbConf := de.Containers[torqDbName]
	btcdConf := de.Containers[btcdName]
	aliceConf := de.Containers[aliceName]
	bobConf := de.Containers[bobName]
	carolConf := de.Containers[carolName]

	ctx := context.Background()

	if withDatabase {
		if withDatabase {
			err = de.StartContainer(ctx, torqDbConf)
			if err != nil {
				return err
			}
		}
	}

	log.Println("Starting btcd")
	err = de.StartContainer(ctx, btcdConf)
	if err != nil {
		log.Fatal(err)
	}

	/*
		// Start alice
	*/
	log.Println("Starting Alice")
	err = de.StartContainer(ctx, aliceConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Get Alice's IP")
	log.Println(aliceConf.Instance)
	aliceInspection, err := de.Client.ContainerInspect(ctx, aliceConf.Instance.ID)
	if err != nil {
		return errors.Newf("Getting Alice's IP Address: %v", err)
	}
	log.Println(aliceInspection.NetworkSettings.Networks)
	aliceIPAddress := aliceInspection.NetworkSettings.Networks[name].IPAddress
	log.Println("Alice's IP address is:")
	log.Println(aliceIPAddress)

	log.Println("Getting Alice's pubkey")
	alicePubkey, err := GetPubKey(ctx, de.Client, aliceConf.Instance)
	if err != nil {
		return errors.Newf("Getting Alice's pubkey: %v", err)
	}
	log.Printf("Alice's pubkey: %s\n", alicePubkey)

	/*
		// Start bob
	*/
	log.Println("Starting Bob")
	err = de.StartContainer(ctx, bobConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Get Bob's IP")
	bobInspection, err := de.Client.ContainerInspect(ctx, bobConf.Instance.ID)
	if err != nil {
		return errors.Newf("Getting Bob's IP Address: %v", err)
	}
	bobIPAddress := bobInspection.NetworkSettings.Networks[name].IPAddress
	log.Println("Bob's IP address is:")
	log.Println(bobIPAddress)

	log.Println("Getting Bob's pubkey")
	bobPubkey, err := GetPubKey(ctx, de.Client, bobConf.Instance)
	if err != nil {
		return errors.Newf("Getting Bob's pubkey: %v", err)
	}
	log.Printf("Bob's pubkey: %s\n", bobPubkey)

	/*
	  Connecting Bob and Alice
	*/

	log.Println("Connecting Alice to Bob")

	err = ConnectPeer(ctx, de.Client, bobConf.Instance, alicePubkey, aliceIPAddress)
	if err != nil {
		return errors.Newf("Connecting Alice to Bob: %v", err)
	}

	log.Println("Verifing Alice is a peer of Bob")

	alicePeerExists, err := CheckPeerExists(ctx, de.Client, bobConf.Instance, alicePubkey)
	if err != nil || !alicePeerExists {
		return errors.Newf("Checking that Alice is a peer of Bob: %v", err)
	}

	/*
		// Start Carol
	*/
	log.Println("Starting Carol")
	err = de.StartContainer(ctx, carolConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Getting Carol's pubkey")
	carolPubkey, err := GetPubKey(ctx, de.Client, carolConf.Instance)
	if err != nil {
		return errors.Newf("Getting Carol's pubkey: %v", err)
	}
	log.Printf("Carol's pubkey: %s\n", carolPubkey)

	carolInspection, err := de.Client.ContainerInspect(ctx, carolConf.Instance.ID)
	if err != nil {
		return errors.Newf("Getting Carol's IP Address: %v", err)
	}
	carolIPAddress := carolInspection.NetworkSettings.Networks[name].IPAddress
	log.Println("Carol's IP address is:")
	log.Println(carolIPAddress)

	log.Println("Connecting Carol to Bob")

	err = ConnectPeer(ctx, de.Client, bobConf.Instance, carolPubkey, carolIPAddress)
	if err != nil {
		return errors.Newf("Connecting Carol to Bob: %v", err)
	}

	log.Println("Verifing Carol is a peer of Bob")

	carolPeerExists, err := CheckPeerExists(ctx, de.Client, bobConf.Instance, carolPubkey)
	if err != nil || !carolPeerExists {
		return errors.Newf("Checking that Carol is a peer of Bob: %v", err)
	}

	WriteConnectionDetails(ctx, de.Client, bobName, bobIPAddress)
	PrintInstructions()

	return nil
}

func StopVirtualNetwork(name string, withDatabase bool) error {

	de, err := createDockerEnvironment(name, withDatabase)
	if err != nil {
		return err
	}

	ctx := context.Background()
	for _, container := range de.Containers {
		// Start btcd first
		err := de.StopContainer(ctx, container.Name)
		if err != nil {
			return errors.Newf("Starting container: %v\n", err)
		}
	}

	return nil
}

func PurgeVirtualNetwork(name string, withDatabase bool) error {

	de, err := createDockerEnvironment(name, withDatabase)
	if err != nil {
		return err
	}

	ctx := context.Background()

	err = de.CleanupContainers(ctx)
	if err != nil {
		return errors.Newf("cleaning up containers: %v\n", err)
	}
	err = de.CleanupDefaultVolumes(ctx)
	if err != nil {
		return errors.Newf("cleaning up volumes: %v\n", err)
	}

	err = de.FindAndRemoveNetwork(ctx, de.NetworkName)
	if err != nil {
		return errors.Newf("removing network: %v\n", err)
	}

	return nil
}

// CreateNewVirtualNetwork creates a new virtual network with the given name,
// this is used for creating the development Lightning network and database.
func CreateNewVirtualNetwork(name string, createDatabase bool, purge bool) error {

	torqDbName := name + "-torq-db"
	btcdName := name + "-btcd"
	aliceName := name + "-alice"
	bobName := name + "-bob"
	carolName := name + "-carol"

	de, err := createDockerEnvironment(name, createDatabase)

	var torqDbCont *ContainerConfig
	if createDatabase {
		torqDbCont = de.Containers[torqDbName]
	}
	btcdConf := de.Containers[btcdName]
	aliceConf := de.Containers[aliceName]
	bobConf := de.Containers[bobName]
	carolConf := de.Containers[carolName]

	ctx := context.Background()
	if purge {
		// Clean up old containers and network before initiating new ones.
		err := de.CleanupContainers(ctx)
		if err != nil {
			return errors.Newf("cleaning up containers: %v\n", err)
		}
		err = de.CleanupDefaultVolumes(ctx)
		if err != nil {
			return errors.Newf("cleaning up volumes: %v\n", err)
		}

		err = de.FindAndRemoveNetwork(ctx, de.NetworkName)
		if err != nil {
			return errors.Newf("removing network: %v\n", err)
		}
	}

	// Create the shared network
	networkingConfig, err := de.CreateNetwork(ctx)
	if err != nil {
		log.Fatal(err)
	}
	de.NetworkingConfig = networkingConfig

	log.Println("Creating network for " + name)

	// Start the database
	if createDatabase {
		err = de.InitContainer(ctx, torqDbCont)
		if err != nil {
			return err
		}
	}

	log.Println("Building btcd image from dockerfile")
	err = de.BuildImage(ctx, "virtual_network/docker/btcd/", name+"/btcd")
	if err != nil {
		return errors.Newf("building image: %v\n", err)
	}

	log.Println("Building lnd image from dockerfile")
	de.BuildImage(ctx, "virtual_network/docker/lnd/", name+"/lnd")
	if err != nil {
		return errors.Newf("building image: %v\n", err)
	}

	log.Println("Starting btcd")
	err = de.InitContainer(ctx, btcdConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting Alice")
	err = de.InitContainer(ctx, aliceConf)
	if err != nil {
		log.Fatal(err)
	}

	// Example looking at container logs
	//out, err := de.Client.ContainerLogs(ctx, btcdConf.Instance.ID, types.ContainerLogsOptions{ShowStdout: true})
	//if err != nil {
	//	panic(err)
	//}
	//stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	log.Println("Creating new mining address on Alice")
	aliceAddress, err := GetNewAddress(ctx, de.Client, aliceConf.Instance)
	if err != nil {
		return errors.Newf("Getting alice mining address: %v", err)
	}
	log.Println("Alice receive address created")
	log.Println(aliceAddress)

	log.Println("Shutting Alice down before btcd restart")
	de.FindAndRemoveContainer(ctx, aliceConf.Name)

	log.Println("Recreating btcd container with Alice's mining address")
	de.FindAndRemoveContainer(ctx, btcdConf.Name)

	log.Println("Starting new btcd container")
	// Update the container config with the minind addres instead of adding a new one
	btcdConf.Env = []string{"NETWORK=simnet", "MINING_ADDRESS=" + aliceAddress}
	err = de.InitContainer(ctx, btcdConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Generate 400 blocks (we need at least \"100 >=\" blocks because of coinbase block maturity and \"300 ~=\" in order to activate segwit)")

	err = MineBlocks(ctx, de.Client, btcdConf.Instance, 400)
	if err != nil {
		return errors.Newf("Mining blocks: %v", err)
	}

	log.Println("Blocks mined")

	log.Println("Recreating Alice now that btcd is back online")

	// Initiation new Alice (based on the same configuration
	err = de.InitContainer(ctx, aliceConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Checking that segwit is active")
	err = SegWitActive(ctx, de.Client, btcdConf.Instance)
	if err != nil {
		return errors.Newf("btcd checking segwit is active: %v", err)
	}
	log.Println("Segwit is active")

	aliceInitBalance, err := GetOnchainBalance(ctx, de.Client, aliceConf.Instance)
	if err != nil {
		return errors.Newf("Getting Bob's balance: %v", err)
	}
	log.Printf("Alice's initial onchain balance: %s\n", aliceInitBalance)

	// Starting carol
	log.Println("Starting Carol")
	err = de.InitContainer(ctx, carolConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Creating new mining address on Carol")
	carolAddress, err := GetNewAddress(ctx, de.Client, carolConf.Instance)
	if err != nil {
		return errors.Newf("Getting Carol mining address: %v", err)
	}
	log.Println("Carol receive address created")
	log.Println(carolAddress)

	log.Println("Shutting Carol down before btcd restart")
	de.FindAndRemoveContainer(ctx, carolConf.Name)

	log.Println("Recreating btcd container with Carol's mining address")
	de.FindAndRemoveContainer(ctx, btcdConf.Name)

	log.Println("Starting new btcd container")
	// Update the container config with the minind addres instead of adding a new one
	btcdConf.Env = []string{"NETWORK=simnet", "MINING_ADDRESS=" + carolAddress}
	err = de.InitContainer(ctx, btcdConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Generate 400 blocks (we need at least \"100 >=\" blocks because of coinbase block maturity and \"300 ~=\" in order to activate segwit)")
	err = MineBlocks(ctx, de.Client, btcdConf.Instance, 400)
	if err != nil {
		return errors.Newf("Mining blocks: %v", err)
	}
	log.Println("Blocks mined")

	// Starting carol
	log.Println("Recreating Carol now that btcd is back online")
	err = de.InitContainer(ctx, carolConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Checking that segwit is active")
	err = SegWitActive(ctx, de.Client, btcdConf.Instance)
	if err != nil {
		return errors.Newf("btcd checking segwit is active: %v", err)
	}
	log.Println("Segwit is active")

	carolInitBalance, err := GetOnchainBalance(ctx, de.Client, carolConf.Instance)
	if err != nil {
		return errors.Newf("Getting Bob's balance: %v", err)
	}
	log.Printf("Carol's initial onchain balance: %s\n", carolInitBalance)

	// Start Bob
	log.Println("Starting Bob")
	err = de.InitContainer(ctx, bobConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Creating new mining address on Carol")
	bobAddress, err := GetNewAddress(ctx, de.Client, bobConf.Instance)
	if err != nil {
		return errors.Newf("Getting Bob mining address: %v", err)
	}
	log.Println("Bob receive address created")
	log.Println(bobAddress)

	log.Println("Shutting Bob down before btcd restart")
	de.FindAndRemoveContainer(ctx, bobConf.Name)

	log.Println("Recreating btcd container with Bob's mining address")
	de.FindAndRemoveContainer(ctx, btcdConf.Name)

	log.Println("Starting new btcd container")
	// Update the container config with the minind addres instead of adding a new one
	btcdConf.Env = []string{"NETWORK=simnet", "MINING_ADDRESS=" + bobAddress}
	err = de.InitContainer(ctx, btcdConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Generate 400 blocks (we need at least \"100 >=\" blocks because of coinbase block maturity and \"300 ~=\" in order to activate segwit)")
	err = MineBlocks(ctx, de.Client, btcdConf.Instance, 400)
	if err != nil {
		return errors.Newf("Mining blocks: %v", err)
	}
	log.Println("Blocks mined")

	log.Println("Recreating Bob now that btcd is back online")
	err = de.InitContainer(ctx, bobConf)
	if err != nil {
		log.Fatal(err)
	}

	bobInitBalance, err := GetOnchainBalance(ctx, de.Client, bobConf.Instance)
	if err != nil {
		return errors.Newf("Getting Bob's balance: %v", err)
	}
	log.Printf("Bob's initial onchain balance: %s\n", bobInitBalance)

	log.Println("Checking that segwit is active")
	err = SegWitActive(ctx, de.Client, btcdConf.Instance)
	if err != nil {
		return errors.Newf("btcd checking segwit is active: %v", err)
	}
	log.Println("Segwit is active")

	log.Println("Get Bob's pubkey")
	bobInspection, err := de.Client.ContainerInspect(ctx, bobConf.Instance.ID)
	if err != nil {
		return errors.Newf("Getting Bob's IP Address: %v", err)
	}
	bobIPAddress := bobInspection.NetworkSettings.Networks[name].IPAddress
	log.Println("Bob's IP address is:")
	log.Println(bobIPAddress)

	bobPubkey, err := GetPubKey(ctx, de.Client, bobConf.Instance)
	if err != nil {
		return errors.Newf("Getting Bob's pubkey: %v", err)
	}
	log.Printf("Bob's pubkey is: %s\n", bobPubkey)

	log.Println("Connecting Bob to Alice")

	err = ConnectPeer(ctx, de.Client, aliceConf.Instance, bobPubkey, bobIPAddress)
	if err != nil {
		return errors.Newf("Connecting Bob to Alice: %v", err)
	}

	log.Println("Verifing Bob is a peer of Alice")

	bobPeerExists, err := CheckPeerExists(ctx, de.Client, aliceConf.Instance, bobPubkey)
	if err != nil || !bobPeerExists {
		return errors.Newf("Checking that Bob is a peer of Alice: %v", err)
	}

	log.Println("Bob confirmed as peer of Alice")

	log.Println("Getting Alice's pubkey")
	alicePubkey, err := GetPubKey(ctx, de.Client, aliceConf.Instance)
	if err != nil {
		return errors.Newf("Getting Alice's pubkey: %v", err)
	}

	log.Printf("Alice's pubkey is: %s\n", alicePubkey)

	log.Println("Verifing Alice is a peer of Bob")

	alicePeerExists, err := CheckPeerExists(ctx, de.Client, bobConf.Instance, alicePubkey)
	if err != nil || !alicePeerExists {
		return errors.Newf("Checking that Alice is a peer of Bob: %v", err)
	}
	log.Println("Alice confirmed as peer of Bob")

	log.Println("Create the Alice<->Bob channel")

	aliceBobChannelPoint, err := CreateChannel(ctx, de.Client, aliceConf.Instance, bobPubkey, "12000000",
		btcdConf.Instance)
	if err != nil {
		return errors.Newf("Creating Alice<->Bob channel: %v", err)
	}

	log.Println("Generating invoice for payment to Bob")

	bobEncodedInvoice, err := GenerateInvoice(ctx, de.Client, bobConf.Instance, "4100000")
	if err != nil {
		return errors.Newf("Creating Bob invoice: %v", err)
	}

	log.Printf("Encoded payment request: %s\n", bobEncodedInvoice)

	log.Println("Alice paying invoice sending payment to Bob")

	err = PayInvoice(ctx, de.Client, aliceConf.Instance, bobEncodedInvoice)
	if err != nil {
		return errors.Newf("Sending Alice->Bob payment: %v", err)
	}

	log.Println("Checking payment received by Bob")
	bobChannelBalance, err := GetChannelBalance(ctx, de.Client, bobConf.Instance)
	if err != nil {
		return errors.Newf("Checking Bob's balance: %v", err)
	}

	log.Println("Payment received by Bob")
	log.Printf("Bob's channel balance: %s\n", bobChannelBalance)

	log.Println("Close Alice<->Bob channel to gain on chain funds for Bob")

	aliceBobClosingTxId, err := CloseChannel(ctx, de.Client, aliceConf.Instance, aliceBobChannelPoint)
	if err != nil {
		return errors.Newf("Closing Alice<->Bob channel: %v", err)
	}

	log.Printf("Alice<->Bob channel closing transaction id: %s\n", aliceBobClosingTxId)

	log.Println("Mining some blocks to confirm closing transaction")

	err = MineBlocks(ctx, de.Client, btcdConf.Instance, 3)
	if err != nil {
		return errors.Newf("Mining blocks: %v", err)
	}

	bobOnChainBalance, err := GetOnchainBalance(ctx, de.Client, bobConf.Instance)
	if err != nil {
		return errors.Newf("Getting Bob's balance: %v", err)
	}
	log.Printf("Bob's onchain balance: %s\n", bobOnChainBalance)

	log.Println("Getting Carol's pubkey")
	carolPubkey, err := GetPubKey(ctx, de.Client, carolConf.Instance)
	if err != nil {
		return errors.Newf("Getting Carol's pubkey: %v", err)
	}
	log.Printf("Carol's pubkey: %s\n", carolPubkey)

	carolInspection, err := de.Client.ContainerInspect(ctx, carolConf.Instance.ID)
	if err != nil {
		return errors.Newf("Getting Carol's IP Address: %v", err)
	}
	carolIPAddress := carolInspection.NetworkSettings.Networks[name].IPAddress
	log.Println("Carol's IP address is:")
	log.Println(carolIPAddress)

	log.Println("Connecting Carol to Bob")

	err = ConnectPeer(ctx, de.Client, bobConf.Instance, carolPubkey, carolIPAddress)
	if err != nil {
		return errors.Newf("Connecting Carol to Bob: %v", err)
	}

	log.Println("Verifing Carol is a peer of Bob")

	carolPeerExists, err := CheckPeerExists(ctx, de.Client, bobConf.Instance, carolPubkey)
	if err != nil || !carolPeerExists {
		return errors.Newf("Checking that Carol is a peer of Bob: %v", err)
	}

	log.Println("Carol confirmed as peer of Bob")

	log.Println("Verifing Bob is a peer of Carol")
	carolBobPeerExists, err := CheckPeerExists(ctx, de.Client, carolConf.Instance, bobPubkey)
	if err != nil || !carolBobPeerExists {
		return errors.Newf("Checking that Bob is a peer of Carol: %v", err)
	}
	log.Println("Bob confirmed as peer of Carol")

	err = MineBlocks(ctx, de.Client, btcdConf.Instance, 30)
	if err != nil {
		return errors.Newf("Mining blocks: %v\n", err)
	}
	log.Println("Created the Bob<->Carol channel")

	_, err = CreateChannel(ctx, de.Client, bobConf.Instance, carolPubkey, "100000", btcdConf.Instance)
	if err != nil {
		return errors.Newf("Creating Bob<->Carol channel: %v", err)
	}

	log.Println("Recreate the Alice<->Bob channel")

	aliceBobChannelPoint, err = CreateChannel(ctx, de.Client, aliceConf.Instance, bobPubkey, "1000000",
		btcdConf.Instance)
	if err != nil {
		return errors.Newf("Creating Alice<->Bob channel: %v", err)
	}

	log.Println("Generating invoice for payment to Carol")

	carolEncodedInvoice, err := GenerateInvoice(ctx, de.Client, carolConf.Instance, "10")
	if err != nil {
		return errors.Newf("Creating Carol invoice: %v", err)
	}

	log.Printf("Encoded payment request: %s\n", carolEncodedInvoice)

	log.Println("Alice paying invoice sending payment via Bob to Carol")

	err = PayInvoice(ctx, de.Client, aliceConf.Instance, carolEncodedInvoice)
	if err != nil {
		return errors.Newf("Sending Alice->Bob->Carol payment: %v", err)
	}

	log.Println("Checking payment received by Carol")
	carolChannelBalance, err := GetChannelBalance(ctx, de.Client, carolConf.Instance)
	if err != nil {
		return errors.Newf("Checking Carol's balance: %v", err)
	}

	log.Println("Payment received by Carol")
	log.Printf("Carol's channel balance: %s\n", carolChannelBalance)

	log.Println("Cluster setup complete")

	WriteConnectionDetails(ctx, de.Client, bobName, bobIPAddress)
	PrintInstructions()
	//err = StopVirtualNetwork(name, createDatabase)
	//if err != nil {
	//	return errors.Newf("Stopping virtual network: %v", err)
	//}

	return nil
}

func PrintInstructions() {
	fmt.Println("\nVirtual network is ready. Start Torq by running:")
	fmt.Println("\n\tgo build ./cmd/torq && ./torq --torq.password password --db.user postgres --db.port 5444 " +
		"--db.password password start")

	fmt.Println("\nThe frontend password is 'password'.")

	fmt.Println("\nRemember to upload the tls.cert and admin.macaroon files in the settings page. " +
		"Set localhost:10009 as the lnd address")

	fmt.Println("You can find the macaroon and tls files in /virtual_network/generated_files")

	fmt.Println("\nYou might need to stop and start Torq after uploading the tls and macaroon files.")

	fmt.Println("\nIf you want to interact with the lnd nodes alice, bob and carol with a shortcut. Run this command:")
	fmt.Println("\n" +
		"alice() { docker exec -it  dev-alice /bin/bash -c \"lncli --macaroonpath=\"/root/." +
		"lnd/data/chain/bitcoin/simnet/admin.macaroon\" --network=simnet $@\"}; \n" +
		"bob() { docker exec -it  dev-bob /bin/bash -c \"lncli --macaroonpath=\"/root/." +
		"lnd/data/chain/bitcoin/simnet/admin.macaroon\" --network=simnet $@\"}; \n" +
		"carol() { docker exec -it  dev-carol /bin/bash -c \"lncli --macaroonpath=\"/root/." +
		"lnd/data/chain/bitcoin/simnet/admin.macaroon\" --network=simnet $@\"}; \n" +
		"vbtcd() { docker exec -it  dev-btcd /bin/bash -c \"btcctl --simnet --rpcuser=devuser --rpcpass=devpass" +
		" --rpccert=/rpc/rpc.cert --rpcserver=localhost $@\"};")

}

// NodeFLowLoop
// Run a loop that
// - creates and pays invoices to/from random nodes at a user defined interval
// - creates addresses and sends coins from/to random nodes at a user defined interval
// - opens a channel between random nodes at user defined interval -no more than 2 duplicate channels
// - closes channels randomly
// invfrq flag = invoice creation frequency - default to 1 time per second
// sendcoins flag = create address and sencoins frequency - default to 1 time per 30 seconds
// openChan flag = open channel frequency - default to 1 time per 10 minutes
func NodeFLowLoop(name string, invfrq int, scofrq int, ochfrq int) error {
	log.Printf("inv freq: %v\n", invfrq)
	log.Printf("send coins freq: %v\n", scofrq)
	log.Printf("ochfreq freq: %v\n", ochfrq)

	torqDbName := name + "-torq-db"
	btcdName := name + "-btcd"
	aliceName := name + "-alice"
	bobName := name + "-bob"
	carolName := name + "-carol"

	de, err := createDockerEnvironment(name, true)
	if err != nil {
		return err
	}

	torqDbConf := de.Containers[torqDbName]
	btcdConf := de.Containers[btcdName]
	aliceConf := de.Containers[aliceName]
	bobConf := de.Containers[bobName]
	carolConf := de.Containers[carolName]

	ctx := context.Background()

	err = de.StartContainer(ctx, torqDbConf)
	if err != nil {
		return err
	}

	log.Println("Starting btcd")
	err = de.StartContainer(ctx, btcdConf)
	if err != nil {
		log.Fatal(err)
	}

	/*
		// Start alice
	*/
	log.Println("Starting Alice")
	err = de.StartContainer(ctx, aliceConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Get Alice's IP")
	log.Println(aliceConf.Instance)
	aliceInspection, err := de.Client.ContainerInspect(ctx, aliceConf.Instance.ID)
	if err != nil {
		return errors.Newf("Getting Alice's IP Address: %v", err)
	}
	log.Println(aliceInspection.NetworkSettings.Networks)

	aliceIPAddress := aliceInspection.NetworkSettings.Networks[name].IPAddress
	log.Println("Alice's IP address is:")
	log.Println(aliceIPAddress)

	log.Println("Getting Alice's pubkey")
	alicePubkey, err := GetPubKey(ctx, de.Client, aliceConf.Instance)
	if err != nil {
		return errors.Newf("Getting Alice's pubkey: %v", err)
	}
	log.Printf("Alice's pubkey: %s\n", alicePubkey)

	aliceWalletBalance, err := GetOnchainBalance(ctx, de.Client, aliceConf.Instance)
	if err != nil {

		return errors.Newf("Checking Carol's balance: %v", err)
		//continue
	}
	log.Printf("Alice wallet balance: %v", aliceWalletBalance)

	/*
		// Start bob
	*/
	log.Println("Starting Bob")
	err = de.StartContainer(ctx, bobConf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Get Bob's IP")
	bobInspection, err := de.Client.ContainerInspect(ctx, bobConf.Instance.ID)
	if err != nil {
		return errors.Newf("Getting Bob's IP Address: %v", err)
	}
	bobIPAddress := bobInspection.NetworkSettings.Networks[name].IPAddress
	log.Println("Bob's IP address is:")
	log.Println(bobIPAddress)

	log.Println("Getting Bob's pubkey")
	bobPubkey, err := GetPubKey(ctx, de.Client, bobConf.Instance)
	if err != nil {
		return errors.Newf("Getting Bob's pubkey: %v", err)
	}
	log.Printf("Bob's pubkey: %s\n", bobPubkey)

	bobWalletBalance, err := GetOnchainBalance(ctx, de.Client, bobConf.Instance)
	if err != nil {

		return errors.Newf("Checking Carol's balance: %v", err)
		//continue
	}
	log.Printf("Bob wallet balance: %v", bobWalletBalance)

	/*
		// Start Carol
	*/
	log.Println("Starting Carol")
	err = de.StartContainer(ctx, carolConf)
	if err != nil {
		log.Fatal(err)
	}

	carolInspection, err := de.Client.ContainerInspect(ctx, carolConf.Instance.ID)
	if err != nil {
		return errors.Newf("Getting Carol's IP Address: %v", err)
	}
	carolIPAddress := carolInspection.NetworkSettings.Networks[name].IPAddress
	log.Println("Carol's IP address is:")
	log.Println(carolIPAddress)

	log.Println("Getting Carol's pubkey")
	carolPubkey, err := GetPubKey(ctx, de.Client, carolConf.Instance)
	if err != nil {
		return errors.Newf("Getting Carol's pubkey: %v", err)
	}
	log.Printf("Carol's pubkey: %s\n", carolPubkey)

	carolWalletBalance, err := GetOnchainBalance(ctx, de.Client, carolConf.Instance)
	if err != nil {

		return errors.Newf("Checking Carol's balance: %v", err)
		//continue
	}
	log.Printf("Carol wallet balance: %v", carolWalletBalance)

	/*
	 Connecting Bob and Alice
	*/

	log.Println("Connecting Alice to Bob")

	err = ConnectPeer(ctx, de.Client, bobConf.Instance, alicePubkey, aliceIPAddress)
	if err != nil {
		return errors.Newf("Connecting Alice to Bob: %v", err)
	}

	log.Println("Verifing Alice is a peer of Bob")

	alicePeerExists, err := CheckPeerExists(ctx, de.Client, bobConf.Instance, alicePubkey)
	if err != nil || !alicePeerExists {
		return errors.Newf("Checking that Alice is a peer of Bob: %v", err)
	}

	err = MineBlocks(ctx, de.Client, btcdConf.Instance, 30)
	if err != nil {
		return errors.Newf("Mining blocks: %v\n", err)
	}

	/*
	 Connecting Bob and Carol
	*/

	log.Println("Connecting Carol to Bob")

	err = ConnectPeer(ctx, de.Client, bobConf.Instance, carolPubkey, carolIPAddress)
	if err != nil {
		return errors.Newf("Connecting Carol to Bob: %v", err)
	}

	log.Println("Verifing Carol is a peer of Bob")

	carolPeerExists, err := CheckPeerExists(ctx, de.Client, bobConf.Instance, carolPubkey)
	if err != nil || !carolPeerExists {
		return errors.Newf("Checking that Carol is a peer of Bob: %v", err)
	}

	err = MineBlocks(ctx, de.Client, btcdConf.Instance, 30)
	if err != nil {
		return errors.Newf("Mining blocks: %v\n", err)
	}

	/*
	 Connecting Alice and Carol
	*/

	log.Println("Connecting Carol to Bob")

	err = ConnectPeer(ctx, de.Client, aliceConf.Instance, carolPubkey, carolIPAddress)
	if err != nil {
		return errors.Newf("Connecting Carol to Bob: %v", err)
	}

	log.Println("Verifing Carol is a peer of Bob")

	carolPeerAliceExists, err := CheckPeerExists(ctx, de.Client, aliceConf.Instance, carolPubkey)
	if err != nil || !carolPeerAliceExists {
		return errors.Newf("Checking that Carol is a peer of Bob: %v", err)
	}

	err = MineBlocks(ctx, de.Client, btcdConf.Instance, 30)
	if err != nil {
		return errors.Newf("Mining blocks: %v\n", err)
	}

	WriteConnectionDetails(ctx, de.Client, bobName, bobIPAddress)

	//go openRandomChann(name, ochfrq, ctx, de, alicePubkey, bobPubkey, carolPubkey)
	//go closeRandomChann(name, ochfrq, ctx, de, alicePubkey, bobPubkey, carolPubkey)
	go createPayInvoice(name, invfrq, ctx, de, alicePubkey, bobPubkey, carolPubkey)
	//go addressSendCoins(name, scofrq, ctx, de, alicePubkey, bobPubkey, carolPubkey)
	select {}
	return nil
}

func openRandomChann(name string, ochfrq int, ctx context.Context, de DockerDevEnvironment, alicePK string, bobPK string, carolPK string) {
	log.Println("Opening random channel")
	freq := time.Duration(ochfrq)
	ticker := time.NewTicker(freq * time.Minute)

	btcdName := name + "-btcd"
	btcdConf := de.Containers[btcdName]

	var cnt int
	for range ticker.C {
		cnt++
		peer1, peer2PK, peer2 := pickRandomNodes(de, name, alicePK, bobPK, carolPK)

		log.Printf("%v opening channel to %v\n", peer1.Name, peer2.Name)

		channels, err := ListNodeChannels(ctx, de.Client, peer1.Instance, peer2PK)
		if err != nil {
			continue
		}

		if len(channels) >= 2 {
			log.Println("There are already 2 active channels.Skipping!")
			continue
		}

		var min int
		var max int
		var size string

		rand.Seed(time.Now().UnixNano())
		if cnt == 4 {
			min = 40000000
			max = 100000000
			size = strconv.Itoa(rand.Intn(max-min) + min)
			cnt = 0
		} else {
			min = 500000
			max = 15000000
			size = strconv.Itoa(rand.Intn(max-min) + min)
		}

		_, err = CreateChannel(ctx, de.Client, peer1.Instance, peer2PK, size, btcdConf.Instance)
		if err != nil {
			continue
		}
		log.Println("Channel size: ", size)
	}
}

func closeRandomChann(name string, ochfrq int, ctx context.Context, de DockerDevEnvironment, alicePK string, bobPK string, carolPK string) {
	freq := time.Duration(ochfrq)
	ticker := time.NewTicker(freq * time.Minute)

	btcdName := name + "-btcd"
	btcdConf := de.Containers[btcdName]

	for range ticker.C {
		log.Println("Closing random channel")
		peer1, peer2PK, peer2 := pickRandomNodes(de, name, alicePK, bobPK, carolPK)

		log.Printf("%v closing channel to %v\n", peer1.Name, peer2.Name)

		channels, err := ListNodeChannels(ctx, de.Client, peer1.Instance, peer2PK)
		if err != nil {
			continue
		}

		if len(channels) == 0 {
			log.Println("0 active channels. Skipping!")
			continue
		}

		rand.Seed(time.Now().UnixNano())
		randomIndex := rand.Intn(len(channels))
		channelPoint := channels[randomIndex]

		closeTxid, err := CloseChannel(ctx, de.Client, peer1.Instance, channelPoint)
		//err = CreateChannel(ctx, de.Client, peer1.Instance, peer2, amt, btcdConf.Instance)
		if err != nil {
			continue
		}
		log.Println("Channel closed. Txid: ", closeTxid)
		err = MineBlocks(ctx, de.Client, btcdConf.Instance, 1)
		if err != nil {
			continue
		}

		log.Println("Blocks mined")
	}
}

func createPayInvoice(name string, invfrq int, ctx context.Context, de DockerDevEnvironment, alicePK string, bobPK string, carolPK string) {
	freq := time.Duration(invfrq)
	ticker := time.NewTicker(freq * time.Second)

	btcdName := name + "-btcd"
	btcdConf := de.Containers[btcdName]

	for range ticker.C {
		log.Println("Create new invoice and pay this invoice")
		peer1, _, peer2 := pickRandomNodes(de, name, alicePK, bobPK, carolPK)

		rand.Seed(time.Now().UnixNano())
		min := 1
		max := 1000000
		amt := strconv.Itoa(rand.Intn(max-min) + min)

		newInvoice, err := GenerateInvoice(ctx, de.Client, peer1.Instance, amt)
		if err != nil {
			//return errors.Newf("Creating Carol invoice: %v", err)
			continue
		}

		log.Printf("%v generated new invoice: %s\n", peer1.Name, newInvoice)
		//log.Printf("Encoded payment request: %s\n", newInvoice)

		err = MineBlocks(ctx, de.Client, btcdConf.Instance, 6)
		if err != nil {
			continue
		}
		log.Println("Blocks mined")

		log.Printf("%v paying invoice to %v", peer2.Name, peer1.Name)
		err = PayInvoice(ctx, de.Client, peer2.Instance, newInvoice)
		if err != nil {
			log.Println("Sending payment: %v", err)
			continue
		}
		err = MineBlocks(ctx, de.Client, btcdConf.Instance, 6)
		if err != nil {
			continue
		}
		log.Println("Blocks mined")

		log.Printf("Checking payment received by %v", peer1.Name)
		peer1ChannelBalance, err := GetChannelBalance(ctx, de.Client, peer1.Instance)
		if err != nil {
			//return errors.Newf("Checking Carol's balance: %v", err)
			continue
		}

		log.Printf("Payment received by %v", peer1.Name)
		log.Printf("%v's channel balance: %s\n", peer1.Name, peer1ChannelBalance)
	}
}

func addressSendCoins(name string, scofrq int, ctx context.Context, de DockerDevEnvironment, alicePK string, bobPK string, carolPK string) {
	freq := time.Duration(scofrq)
	ticker := time.NewTicker(freq * time.Second)

	btcdName := name + "-btcd"
	btcdConf := de.Containers[btcdName]

	for range ticker.C {
		log.Println("Creating new address and sending coins to this address")
		peer1, _, peer2 := pickRandomNodes(de, name, alicePK, bobPK, carolPK)
		log.Println("Creating new mining address on", peer1.Name)

		log.Println("Getting on-chain balance for:", peer1.Name)
		peer1OnChainBal, err := GetOnchainBalance(ctx, de.Client, peer1.Instance)
		if err != nil {
			log.Println("Err getting before on-chain balance")
		}
		log.Printf("Before on-chain balance of %v: %s", peer1.Name, peer1OnChainBal)

		peer1NewAddr, err := GetNewAddress(ctx, de.Client, peer1.Instance)
		if err != nil {
			log.Println("Error creating new addres for ", peer1.Name, ". Skipping")
			continue
		}

		rand.Seed(time.Now().UnixNano())
		min := 200000
		max := 100000000
		amt := strconv.Itoa(rand.Intn(max-min) + min)

		log.Printf("%v sending %s on-chain to %v\n", peer2.Name, amt, peer1.Name)
		txId, err := AddressSendCoins(ctx, de.Client, peer2.Instance, peer1NewAddr, amt)
		if err != nil {
			log.Println("Payment failed")
			continue
		}

		err = MineBlocks(ctx, de.Client, btcdConf.Instance, 6)
		if err != nil {
			continue
		}
		log.Println("Blocks mined")

		peer1OnChainBal, err = GetOnchainBalance(ctx, de.Client, peer1.Instance)
		if err != nil {
			log.Println("Err getting after on-chain balance")
		}

		log.Println("Coins sent. TxId: ", txId)
		log.Printf("After on-chain balance of %v: %s", peer1.Name, peer1OnChainBal)
	}

}

func pickRandomNodes(de DockerDevEnvironment, name string, alicePK string, bobPK string, carolPK string) (peer1 *ContainerConfig, peer2PK string, peer2 *ContainerConfig) {
	//1 = alice
	//2 = bob
	//3 = carol
	nodeCombo := []string{"1-2", "1-3", "2-3", "2-1", "3-1", "3-2"}
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(nodeCombo))
	pick := nodeCombo[randomIndex]

	combo := strings.Split(pick, "-")
	val1 := combo[0]
	val2 := combo[1]

	aliceName := name + "-alice"
	bobName := name + "-bob"
	carolName := name + "-carol"

	aliceConf := de.Containers[aliceName]
	bobConf := de.Containers[bobName]
	carolConf := de.Containers[carolName]

	switch {
	case val1 == "1" && val2 == "2":
		peer1 = aliceConf
		peer2PK = bobPK
		peer2 = bobConf
	case val1 == "1" && val2 == "3":
		peer1 = aliceConf
		peer2PK = carolPK
		peer2 = carolConf
	case val1 == "2" && val2 == "3":
		peer1 = bobConf
		peer2PK = carolPK
		peer2 = carolConf
	case val1 == "2" && val2 == "1":
		peer1 = bobConf
		peer2PK = alicePK
		peer2 = aliceConf
	case val1 == "3" && val2 == "1":
		peer1 = carolConf
		peer2PK = alicePK
		peer2 = aliceConf
	case val1 == "3" && val2 == "2":
		peer1 = carolConf
		peer2PK = bobPK
		peer2 = bobConf
	}

	return peer1, peer2PK, peer2
}
