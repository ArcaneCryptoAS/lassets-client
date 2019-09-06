package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"gitlab.com/arcanecrypto/lnassets/larpc"
)

var log = logrus.New()

var openContractCommand = cli.Command{
	Name:     "opencontract",
	Category: "Contracts",
	Usage:    "Open a new contract with another lightning asset server",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "asset",
			Usage: "which asset to denominate the contract in, USD or NOK",
			Value: "USD",
		},
		cli.IntFlag{
			Name:  "amount",
			Usage: "the amount denominated in `asset`",
		},
		cli.StringFlag{
			Name:  "type",
			Usage: "the contract type as a string, either FUNDED or UNFUNDED",
			Value: "UNFUNDED",
		},
	},
	Action: openContract,
}

func openContract(ctx *cli.Context) error {
	// connect to our local lad daemon
	client, cleanup := connectToDaemon(ctx.GlobalInt(flag_rpcport))
	defer cleanup()

	asset := ctx.String("asset")
	amount := ctx.Float64("amount")
	cType, ok := larpc.ContractType_value[ctx.String("type")]
	if !ok {
		return fmt.Errorf("contract type %q not supported", ctx.String("type"))
	}

	// create a contract at the server. This is not open before we have paid
	// the invoices related to the contract, and won't start rebalancing until they are paid
	createRes, err := client.CreateContract(context.Background(),
		&larpc.ClientCreateContractRequest{
			Asset:        asset,
			Amount:       amount,
			ContractType: larpc.ContractType(cType),
		})
	if err != nil {
		log.WithFields(logrus.Fields{
			"Asset":        asset,
			"Amount":       amount,
			"ContractType": ctx.String("type"),
		}).WithError(err).Error("could not open new contract")
		return err
	}

	if err = displayQuote(
		createRes.Contract.AmountSatMargin,
		createRes.ServerPrice,
		createRes.PercentMargin,
		createRes.OurPrice); err != nil {
		return fmt.Errorf("user did not accept terms: %w", err)
	}

	log.WithField("uuid", createRes.Contract.Uuid).Info("user accepted terms")

	// Open the contract by paying the invoices, the client daemon
	// makes sure the amounts are correct
	openResponse, err := client.OpenContract(context.Background(), &larpc.ClientOpenContractRequest{
		Uuid: createRes.Contract.Uuid,
	})
	if err != nil {
		log.WithError(err).Error("could not open contract")
		return fmt.Errorf("could not open contract")
	}

	log.Infof("opened contract: %+v", openResponse.Contract)

	return nil
}

var closeContractCommand = cli.Command{
	Name:     "closecontract",
	Category: "Contracts",
	Usage:    "Close a contract",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "uuid",
			Usage: "the uuid of the contract",
		},
	},
	Action: closeContract,
}

func closeContract(ctx *cli.Context) error {
	conn, cleanup := connectToDaemon(ctx.GlobalInt(flag_rpcport))
	defer cleanup()

	uuid := ctx.String("uuid")

	_, err := conn.CloseContract(context.Background(), &larpc.ClientCloseContractRequest{
		Uuid: uuid,
	})
	if err != nil {
		log.WithFields(logrus.Fields{
			"UUID": uuid,
		}).WithError(err).Error("could not close contract")
		return err
	}

	log.WithField("uuid", uuid).Infof("closed contract")

	return nil
}

var listContractsCommand = cli.Command{
	Name:     "listcontracts",
	Category: "Contracts",
	Usage:    "list all open contracts",
	Action:   listContracts,
}

func listContracts(ctx *cli.Context) error {
	conn, cleanup := connectToDaemon(ctx.GlobalInt(flag_rpcport))
	defer cleanup()

	contract, err := conn.ListContracts(context.Background(), &larpc.ClientListContractsRequest{})
	if err != nil {
		log.WithError(err).Error("could not list contract")
		return err
	}

	log.Infof("contracts: %+v\n", contract.Contracts)

	return nil
}

func displayQuote(amountSats int64, assetPrice, percentMargin, ourPrice float64) error {
	fmt.Printf("Initiating contract for requires %.2f percent margin, which equals %d sats\n"+
		"Server used a price of %.2f, we have a price of %.2f\n",
		percentMargin, amountSats, assetPrice, ourPrice)

	fmt.Printf("CONTINUE OPENING CONTRACT? (y/n)")

	var answer string
	fmt.Scanln(&answer)

	switch answer {
	case "y":
		return nil
	}

	return errors.New("opening contract canceled")
}
