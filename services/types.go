package services

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	// HistoricalBalanceLookup indicates
	// that historical balance lookup is supported.
	HistoricalBalanceLookup = true

	// MempoolCoins indicates that
	// including mempool coins in the /account/coins
	// response is not supported.
	MempoolCoins = false

	// inlineFetchLimit is the maximum number
	// of transactions to fetch inline.
	inlineFetchLimit = 100
)

// Indexer is used by the servicers to get block and account data.
// Defined here to avoid circular imports
type Indexer interface {
	GetBlockLazy(
		context.Context,
		*types.PartialBlockIdentifier,
	) (*types.BlockResponse, error)
	GetBlockTransaction(
		context.Context,
		*types.BlockIdentifier,
		*types.TransactionIdentifier,
	) (*types.Transaction, error)
}
