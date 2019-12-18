package main

import (
	"fmt"
	"github.com/ArcaneCryptoAS/lassets-client/build"
	"github.com/ArcaneCryptoAS/lassets-client/larpc"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"os"
)

// default value for flags
const (
	defaultRPCPort = 10456
)

// all flags for laccli command
const (
	flag_rpcport = "rpcport"
)

func main() {
	app := cli.NewApp()
	app.Name = "laccli"
	app.Version = build.Version()
	app.Usage = "control plane for your Lightning Assets Client Daemon (lac)"
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  flag_rpcport,
			Value: defaultRPCPort,
			Usage: "port to listen for grpc connections on",
		},
	}
	app.Commands = []cli.Command{
		openContractCommand,
		closeContractCommand,
		listContractsCommand,
	}

	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

// connectToDaemon opens a connection to the lightning assets client daemon
func connectToDaemon(rpcPort int) (larpc.AssetClientClient, func()) {
	// Load the specified TLS certificate and build transport credentials
	// with it.
	// TODO: Add tls here..
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
	}

	rpcServer := fmt.Sprintf("localhost:%d", rpcPort)

	conn, err := grpc.Dial(rpcServer, opts...)
	if err != nil {
		log.Fatalf("unable to connect to RPC server: %w", err)
	}

	cleanUp := func() {
		conn.Close()
	}

	return larpc.NewAssetClientClient(conn), cleanUp
}
