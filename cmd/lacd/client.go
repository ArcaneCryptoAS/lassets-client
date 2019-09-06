package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/lnrpc"

	"gitlab.com/arcanecrypto/lnassets/larpc"
)

var _ larpc.AssetClientServer = &AssetClient{}

type AssetClient struct {
	lncli      lnrpc.LightningClient
	db         *bolt.DB
	contracts  *bolt.Bucket
	port       int
	netAddress string
	server     *grpcServerConnection

	// channels
	paymentsCh chan larpc.Payment
	contractCh chan larpc.ClientContract
}

func (a AssetClient) CreateContract(ctx context.Context, req *larpc.ClientCreateContractRequest) (*larpc.ClientCreateContractResponse, error) {
	log.Infoln("received create contract request")

	if req.Amount == 0 {
		return nil, fmt.Errorf("amount can not be 0")
	}

	res, err := a.server.server.NewContract(ctx, &larpc.ServerNewContractRequest{
		Asset:        req.Asset,
		Amount:       req.Amount,
		Host:         a.netAddress,
		ContractType: req.ContractType,
	})
	if err != nil {
		return nil, err
	}

	marginInv, err := a.lncli.DecodePayReq(ctx, &lnrpc.PayReqString{
		PayReq: res.MarginPayReq,
	})
	if err != nil {
		return nil, err
	}

	contract := larpc.ClientContract{
		Uuid:            res.Uuid,
		Asset:           req.Asset,
		Amount:          req.Amount,
		AmountSatMargin: marginInv.NumSatoshis,
		MarginInvoice:   res.MarginPayReq,
		ContractType:    req.ContractType,
	}

	expectedInitAmount := convertPercentOfAssetToSats(req.Amount, latest[req.Asset], 100)
	expectedMarginAmount := convertPercentOfAssetToSats(req.Amount, latest[req.Asset], res.PercentMargin)

	switch req.ContractType {
	case larpc.ContractType_FUNDED:
		initInv, err := a.lncli.DecodePayReq(ctx, &lnrpc.PayReqString{
			PayReq: res.InitiatingPayReq,
		})
		if err != nil {
			return nil, err
		}

		// update the necessary fields
		contract.AmountSatInit = initInv.NumSatoshis
		contract.InitInvoice = res.InitiatingPayReq

	case larpc.ContractType_UNFUNDED:
		// do some special logic if necesssary
		expectedInitAmount = 0
	default:
		return nil, fmt.Errorf("contract type %v not supported", req.ContractType)
	}

	err = saveContract(a.db, a.contractCh, contract)
	if err != nil {
		return nil, err
	}

	return &larpc.ClientCreateContractResponse{
		Contract: &contract,

		ExpectedMarginAmount: expectedMarginAmount,
		ExpectedInitAmount:   expectedInitAmount,
		OurPrice:             price[req.Asset],

		ServerPrice:   res.AssetPrice,
		PercentMargin: res.PercentMargin,
	}, nil
}

func (a AssetClient) OpenContract(ctx context.Context, req *larpc.ClientOpenContractRequest) (*larpc.ClientOpenContractResponse, error) {
	log.Infoln("received open contract request")

	var contract larpc.ClientContract

	err := a.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(contractsBucket)

		asBytes := b.Get([]byte(req.Uuid))

		err := json.Unmarshal(asBytes, &contract)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not get contract from database: %w", err)
	}

	err = a.PayInvoice(contract.MarginInvoice)
	if err != nil {
		return nil, err
	}

	if contract.ContractType == larpc.ContractType_FUNDED {
		err = a.PayInvoice(contract.InitInvoice)
		if err != nil {
			return nil, err
		}
	}

	contract.InvoicesPaid = true
	err = saveContract(a.db, a.contractCh, contract)
	if err != nil {
		return nil, fmt.Errorf("could not save contract in DB: %w", err)
	}

	log.Infof("opened contract %s", contract.Uuid)

	return &larpc.ClientOpenContractResponse{
		Contract: &contract,
	}, nil
}

// convertPercentOfAssetToSats converts a percentage of an amount of a given asset to satoshis
func convertPercentOfAssetToSats(amount float64, price float64, percent float64) int64 {
	amountSat := (amount / price) * btcutil.SatoshiPerBitcoin

	return int64(math.Round(amountSat / 100 * percent))
}

func saveContract(db *bolt.DB, contractCh chan larpc.ClientContract, contract larpc.ClientContract) error {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(contractsBucket)

		reqBytes, err := json.Marshal(contract)
		if err != nil {
			return err
		}

		return b.Put([]byte(contract.Uuid), reqBytes)
	})
	if err != nil {
		return err
	}

	// pass the saved contract on to the contractCh, in case someone is subscribed
	select {
	case contractCh <- contract:
	default:
	}

	return nil
}

func (a AssetClient) CloseContract(ctx context.Context, req *larpc.ClientCloseContractRequest) (*larpc.ClientCloseContractResponse, error) {
	log.Infoln("received close contract request")

	if req == nil {
		return nil, fmt.Errorf("request can not be nil")
	}

	var contract larpc.ClientContract
	err := a.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(contractsBucket)

		rawContract := b.Get([]byte(req.Uuid))

		err := json.Unmarshal(rawContract, &contract)
		if err != nil {
			return fmt.Errorf("could not unmarshal contract: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	_, err = a.server.server.CloseContract(ctx,
		&larpc.ServerCloseContractRequest{
		Uuid: req.Uuid,
	})
	if err != nil {
		return nil, fmt.Errorf("could not close contract with server")
	}

	// delete the contract from db
	err = a.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(contractsBucket)

		return b.Delete([]byte(req.Uuid))
	})
	if err != nil {
		return nil, err
	}

	return &larpc.ClientCloseContractResponse{}, nil
}

func (a AssetClient) RequestPaymentRequest(ctx context.Context, req *larpc.ClientRequestPaymentRequestRequest) (*larpc.ClientRequestPaymentRequestResponse, error) {
	log.Infoln("received request payment request request")

	res, err := a.lncli.AddInvoice(ctx, &lnrpc.Invoice{
		Value: req.AmountSat,
	})
	if err != nil {
		return nil, err
	}

	return &larpc.ClientRequestPaymentRequestResponse{
		PayReq: res.PaymentRequest,
	}, nil

}

func (a AssetClient) RequestPayment(ctx context.Context, req *larpc.ClientRequestPaymentRequest) (*larpc.ClientRequestPaymentResponse, error) {
	log.Infoln("received request payment request")

	// TODO: Check amount is correct
	err := a.PayInvoice(req.PayReq)
	if err != nil {
		return nil, err
	}

	return &larpc.ClientRequestPaymentResponse{}, nil
}

func (a AssetClient) ListContracts(ctx context.Context, req *larpc.ClientListContractsRequest) (*larpc.ClientListContractsResponse, error) {
	log.Infoln("received list contracts request")

	if req == nil {
		return nil, fmt.Errorf("request can not be nil")
	}

	var contracts []*larpc.ClientContract
	err := a.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(contractsBucket)

		return b.ForEach(func(k, v []byte) error {
			var contract larpc.ClientContract
			err := json.Unmarshal(v, &contract)
			if err != nil {
				return err
			}

			contracts = append(contracts, &contract)

			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return &larpc.ClientListContractsResponse{
		Contracts: contracts,
	}, nil
}

func (a AssetClient) SubscribeClientContracts(req *larpc.
ClientSubscribeContractsRequest, updateStream larpc.
AssetClient_SubscribeClientContractsServer) error {
	log.Infoln("received subscribe client contracts request")

	contractCh := a.contractCh

	for {
		newContract := <-contractCh

		if err := updateStream.Send(&newContract); err != nil {
			return err
		}
	}
}

// PayInvoice does not exist in grpc, but is a util method defined on an AssetClient
func (a AssetClient) PayInvoice(paymentRequest string) error {

	res, err := a.lncli.SendPaymentSync(context.Background(), &lnrpc.SendRequest{
		PaymentRequest: paymentRequest,
	})
	if err != nil {
		return err
	}

	if res.PaymentError != "" {
		return fmt.Errorf("could not send payment: %s", res.PaymentError)
	}

	log.WithField("paymentRequest", paymentRequest).Info("paid")

	return nil
}
