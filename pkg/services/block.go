package services

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/pkg/config"
	"github.com/ross-weir/rosetta-ergo/pkg/errutil"
	"github.com/ross-weir/rosetta-ergo/pkg/storage"
)

// txFetchLimit is the maximum number
// of transactions to fetch inline.
const txFetchLimit = 100

// BlockAPIService implements the server.BlockAPIServicer interface.
type BlockAPIService struct {
	config  *config.Configuration
	storage *storage.Storage
}

// NewBlockAPIService creates a new instance of a BlockAPIService.
func NewBlockAPIService(
	config *config.Configuration,
	storage *storage.Storage,
) server.BlockAPIServicer {
	return &BlockAPIService{
		config:  config,
		storage: storage,
	}
}

// Block implements the /block endpoint.
func (s *BlockAPIService) Block(
	ctx context.Context,
	request *types.BlockRequest,
) (*types.BlockResponse, *types.Error) {
	if s.config.Mode != config.Online {
		return nil, errutil.WrapErr(errutil.ErrUnavailableOffline, nil)
	}

	blockStorage := s.storage.Block()
	blockResponse, err := blockStorage.GetBlockLazy(ctx, request.BlockIdentifier)
	if err != nil {
		return nil, errutil.WrapErr(errutil.ErrBlockNotFound, err)
	}

	// Direct client to fetch transactions individually if
	// more than inlineFetchLimit.
	if len(blockResponse.OtherTransactions) > txFetchLimit {
		return blockResponse, nil
	}

	txs := make([]*types.Transaction, len(blockResponse.OtherTransactions))
	for i, otherTx := range blockResponse.OtherTransactions {
		transaction, err := blockStorage.GetBlockTransaction(
			ctx,
			blockResponse.Block.BlockIdentifier,
			otherTx,
		)
		if err != nil {
			return nil, errutil.WrapErr(errutil.ErrTransactionNotFound, err)
		}

		txs[i] = transaction
	}
	blockResponse.Block.Transactions = txs

	blockResponse.OtherTransactions = nil
	return blockResponse, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (s *BlockAPIService) BlockTransaction(
	ctx context.Context,
	request *types.BlockTransactionRequest,
) (*types.BlockTransactionResponse, *types.Error) {
	if s.config.Mode != config.Online {
		return nil, errutil.WrapErr(errutil.ErrUnavailableOffline, nil)
	}

	transaction, err := s.storage.Block().GetBlockTransaction(
		ctx,
		request.BlockIdentifier,
		request.TransactionIdentifier,
	)
	if err != nil {
		return nil, errutil.WrapErr(errutil.ErrTransactionNotFound, err)
	}

	return &types.BlockTransactionResponse{
		Transaction: transaction,
	}, nil
}
