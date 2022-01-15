package services

import "github.com/coinbase/rosetta-sdk-go/types"

var (
	// Errors contains all errors that could be returned
	// by this Rosetta implementation.
	Errors = []*types.Error{
		ErrUnimplemented,
		ErrUnavailableOffline,
		ErrNotReady,
		ErrErgoNode,
		ErrBlockNotFound,
		// ErrUnableToDerive,
		// ErrUnclearIntent,
		// ErrUnableToParseIntermediateResult,
		// ErrScriptPubKeysMissing,
		// ErrInvalidCoin,
		// ErrUnableToDecodeAddress,
		// ErrUnableToDecodeScriptPubKey,
		// ErrUnableToCalculateSignatureHash,
		// ErrUnsupportedScriptType,
		// ErrUnableToComputePkScript,
		// ErrUnableToGetCoins,
		ErrTransactionNotFound,
		// ErrCouldNotGetFeeRate,
		// ErrUnableToGetBalance,
	}

	// ErrUnimplemented is returned when an endpoint
	// is called that is not implemented.
	ErrUnimplemented = &types.Error{
		Code:    0, //nolint
		Message: "Endpoint not implemented",
	}

	// ErrUnavailableOffline is returned when an endpoint
	// is called that is not available offline.
	ErrUnavailableOffline = &types.Error{
		Code:    1, //nolint
		Message: "Endpoint unavailable offline",
	}

	// ErrNotReady is returned when node is not
	// yet ready to serve queries.
	ErrNotReady = &types.Error{
		Code:      2, //nolint
		Message:   "Ergo node is not ready",
		Retriable: true,
	}

	// ErrErgoNode is returned when ergo
	// errors on a request.
	ErrErgoNode = &types.Error{
		Code:    3, //nolint
		Message: "Ergo node error",
	}

	// ErrBlockNotFound is returned when a block
	// is not available in the indexer.
	ErrBlockNotFound = &types.Error{
		Code:    4, //nolint
		Message: "Block not found",
	}

	// ErrTransactionNotFound is returned by the indexer
	// when it is not possible to find a transaction.
	ErrTransactionNotFound = &types.Error{
		Code:    16, // nolint
		Message: "Transaction not found",
	}
)

// wrapErr adds details to the types.Error provided. We use a function
// to do this so that we don't accidentially overrwrite the standard errors.
func wrapErr(rErr *types.Error, err error) *types.Error {
	newErr := &types.Error{
		Code:      rErr.Code,
		Message:   rErr.Message,
		Retriable: rErr.Retriable,
	}
	if err != nil {
		newErr.Details = map[string]interface{}{
			"context": err.Error(),
		}
	}

	return newErr
}
