package services

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/configuration"
)

// BlockAPIService implements the server.BlockAPIServicer interface.
type BlockAPIService struct {
	config *configuration.Configuration
	i      Indexer
}

// NewBlockAPIService creates a new instance of a BlockAPIService.
func NewBlockAPIService(
	config *configuration.Configuration,
	i Indexer,
) server.BlockAPIServicer {
	return &BlockAPIService{
		config: config,
		i:      i,
	}
}

// Block implements the /block endpoint.
func (s *BlockAPIService) Block(
	ctx context.Context,
	request *types.BlockRequest,
) (*types.BlockResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, wrapErr(ErrUnavailableOffline, nil)
	}

	blockResponse, err := s.i.GetBlockLazy(ctx, request.BlockIdentifier)
	if err != nil {
		return nil, wrapErr(ErrBlockNotFound, err)
	}

	// Direct client to fetch transactions individually if
	// more than inlineFetchLimit.
	if len(blockResponse.OtherTransactions) > inlineFetchLimit {
		return blockResponse, nil
	}

	txs := make([]*types.Transaction, len(blockResponse.OtherTransactions))
	for i, otherTx := range blockResponse.OtherTransactions {
		transaction, err := s.i.GetBlockTransaction(
			ctx,
			blockResponse.Block.BlockIdentifier,
			otherTx,
		)
		if err != nil {
			return nil, wrapErr(ErrTransactionNotFound, err)
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
	if s.config.Mode != configuration.Online {
		return nil, wrapErr(ErrUnavailableOffline, nil)
	}

	transaction, err := s.i.GetBlockTransaction(
		ctx,
		request.BlockIdentifier,
		request.TransactionIdentifier,
	)
	if err != nil {
		return nil, wrapErr(ErrTransactionNotFound, err)
	}

	return &types.BlockTransactionResponse{
		Transaction: transaction,
	}, nil
}
