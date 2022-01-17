package services

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/configuration"
	"github.com/ross-weir/rosetta-ergo/ergo"
	ergotype "github.com/ross-weir/rosetta-ergo/ergo/types"
)

// MempoolAPIService implements the server.MempoolAPIServicer interface.
type MempoolAPIService struct {
	config *configuration.Configuration
	client *ergo.Client
	i      Indexer
}

// NewMempoolAPIService creates a new instance of a MempoolAPIService.
func NewMempoolAPIService(
	config *configuration.Configuration,
	client *ergo.Client,
	indexer Indexer,
) server.MempoolAPIServicer {
	return &MempoolAPIService{
		config: config,
		client: client,
		i:      indexer,
	}
}

// Mempool implements the /mempool endpoint.
func (s *MempoolAPIService) Mempool(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.MempoolResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, wrapErr(ErrUnavailableOffline, nil)
	}

	mempoolTransactions, err := s.client.GetUnconfirmedTxs(ctx)
	if err != nil {
		return nil, wrapErr(ErrErgoNode, err)
	}

	transactionIdentifiers := make([]*types.TransactionIdentifier, len(mempoolTransactions))
	for i, mempoolTransaction := range mempoolTransactions {
		transactionIdentifiers[i] = &types.TransactionIdentifier{Hash: *mempoolTransaction.ID}
	}

	return &types.MempoolResponse{
		TransactionIdentifiers: transactionIdentifiers,
	}, nil
}

// MempoolTransaction implements the /mempool/transaction endpoint.
// This endpoint has some leeway, the rosetta specification states it can be an estimate of the
// final tx.
func (s *MempoolAPIService) MempoolTransaction(
	ctx context.Context,
	request *types.MempoolTransactionRequest,
) (*types.MempoolTransactionResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, wrapErr(ErrUnavailableOffline, nil)
	}

	mempoolTransactions, err := s.client.GetUnconfirmedTxs(ctx)
	if err != nil {
		return nil, wrapErr(ErrErgoNode, err)
	}

	var requestedTx *ergotype.ErgoTransaction
	for i := range mempoolTransactions {
		if *mempoolTransactions[i].ID == request.TransactionIdentifier.Hash {
			requestedTx = &mempoolTransactions[i]
		}
	}
	if requestedTx == nil {
		return nil, wrapErr(ErrTransactionNotFound, err)
	}

	inputCoins := ergo.GetInputsForTxs(&mempoolTransactions)
	accountCoins, err := s.i.FindCoinsForMempoolTx(ctx, inputCoins)
	if err != nil {
		return nil, wrapErr(ErrUnableToGetCoins, err)
	}

	ops, err := ergo.ErgoTransactionToRosettaOps(ctx, s.client, requestedTx, accountCoins)
	if err != nil {
		return nil, wrapErr(ErrErgoNode, err)
	}

	tx := &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: *requestedTx.ID,
		},
		Operations: ops,
	}

	return &types.MempoolTransactionResponse{
		Transaction: tx,
	}, nil
}
