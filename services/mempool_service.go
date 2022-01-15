package services

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/configuration"
	"github.com/ross-weir/rosetta-ergo/ergo"
)

// MempoolAPIService implements the server.MempoolAPIServicer interface.
type MempoolAPIService struct {
	config *configuration.Configuration
	client *ergo.Client
}

// NewMempoolAPIService creates a new instance of a MempoolAPIService.
func NewMempoolAPIService(
	config *configuration.Configuration,
	client *ergo.Client,
) server.MempoolAPIServicer {
	return &MempoolAPIService{
		config: config,
		client: client,
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
func (s *MempoolAPIService) MempoolTransaction(
	ctx context.Context,
	request *types.MempoolTransactionRequest,
) (*types.MempoolTransactionResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, wrapErr(ErrUnavailableOffline, nil)
	}

	return nil, wrapErr(ErrUnimplemented, nil)
}
