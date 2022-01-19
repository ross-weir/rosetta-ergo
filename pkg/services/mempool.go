package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/server"
	storageErrs "github.com/coinbase/rosetta-sdk-go/storage/errors"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/pkg/config"
	"github.com/ross-weir/rosetta-ergo/pkg/ergo"
	ergotype "github.com/ross-weir/rosetta-ergo/pkg/ergo/types"
	"github.com/ross-weir/rosetta-ergo/pkg/errutil"
	"github.com/ross-weir/rosetta-ergo/pkg/storage"
)

// MempoolAPIService implements the server.MempoolAPIServicer interface.
type MempoolAPIService struct {
	config  *config.Configuration
	client  *ergo.Client
	storage *storage.Storage
}

// NewMempoolAPIService creates a new instance of a MempoolAPIService.
func NewMempoolAPIService(
	config *config.Configuration,
	client *ergo.Client,
	storage *storage.Storage,
) server.MempoolAPIServicer {
	return &MempoolAPIService{
		config:  config,
		client:  client,
		storage: storage,
	}
}

// Mempool implements the /mempool endpoint.
func (s *MempoolAPIService) Mempool(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.MempoolResponse, *types.Error) {
	if s.config.Mode != config.Online {
		return nil, errutil.WrapErr(errutil.ErrUnavailableOffline, nil)
	}

	mempoolTransactions, err := s.client.GetUnconfirmedTxs(ctx)
	if err != nil {
		return nil, errutil.WrapErr(errutil.ErrErgoNode, err)
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
	if s.config.Mode != config.Online {
		return nil, errutil.WrapErr(errutil.ErrUnavailableOffline, nil)
	}

	mempoolTransactions, err := s.client.GetUnconfirmedTxs(ctx)
	if err != nil {
		return nil, errutil.WrapErr(errutil.ErrErgoNode, err)
	}

	var requestedTx *ergotype.ErgoTransaction
	for i := range mempoolTransactions {
		if *mempoolTransactions[i].ID == request.TransactionIdentifier.Hash {
			requestedTx = &mempoolTransactions[i]
		}
	}
	if requestedTx == nil {
		return nil, errutil.WrapErr(errutil.ErrTransactionNotFound, err)
	}

	inputCoins := ergo.GetInputsForTxs(&mempoolTransactions)
	accountCoins, err := s.findCoinsForMempoolTx(ctx, inputCoins)
	if err != nil {
		return nil, errutil.WrapErr(errutil.ErrUnableToGetCoins, err)
	}

	ops, err := ergo.ErgoTransactionToRosettaOps(ctx, s.client, requestedTx, accountCoins)
	if err != nil {
		return nil, errutil.WrapErr(errutil.ErrErgoNode, err)
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

func (s *MempoolAPIService) findCoinsForMempoolTx(ctx context.Context, inputs []*ergo.InputCtx,
) (map[string]*types.AccountCoin, error) {
	coinMap := map[string]*types.AccountCoin{}

	for _, inputCtx := range inputs {
		databaseTransaction := s.storage.Db().ReadTransaction(ctx)
		defer databaseTransaction.Discard(ctx)

		// Attempt to find coin
		coin, owner, err := s.storage.Coin().GetCoinTransactional(
			ctx,
			databaseTransaction,
			&types.CoinIdentifier{
				Identifier: inputCtx.InputID,
			},
		)
		if err == nil {
			coinMap[inputCtx.InputID] = &types.AccountCoin{
				Account: owner,
				Coin:    coin,
			}

			continue
		}

		if !errors.Is(err, storageErrs.ErrCoinNotFound) {
			return nil, fmt.Errorf("%w: unable to lookup coin %s", err, inputCtx.InputID)
		}

		// Not sure how often this can happen, no coins
		// We can also check the coinCache in the indexer, not sure how much
		// help that would be though
	}

	return coinMap, nil
}
