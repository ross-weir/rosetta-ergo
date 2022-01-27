package errutil

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
		ErrUnableToDerive,
		ErrUnclearIntent,
		ErrUnableToParseIntermediateResult,
		ErrScriptPubKeysMissing,
		ErrInvalidCoin,
		ErrUnableToDecodeAddress,
		ErrUnableToDecodeScriptPubKey,
		ErrUnableToCalculateSignatureHash,
		ErrUnsupportedScriptType,
		ErrUnableToComputePkScript,
		ErrUnableToGetCoins,
		ErrTransactionNotFound,
		ErrCouldNotGetFeeRate,
		ErrUnableToGetBalance,
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

	// ErrNotReady is returned when ergo node is not
	// yet ready to serve queries.
	ErrNotReady = &types.Error{
		Code:      2, //nolint
		Message:   "ergo node is not ready",
		Retriable: true,
	}

	// ErrErgoNode is returned when ergo node
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

	// ErrUnableToDerive is returned when an address
	// cannot be derived from a provided public key.
	ErrUnableToDerive = &types.Error{
		Code:    5, //nolint
		Message: "Unable to derive address",
	}

	// ErrUnclearIntent is returned when operations
	// provided in /construction/preprocess or /construction/payloads
	// are not valid.
	ErrUnclearIntent = &types.Error{
		Code:    6, //nolint
		Message: "Unable to parse intent",
	}

	// ErrUnableToParseIntermediateResult is returned
	// when a data structure passed between Construction
	// API calls is not valid.
	ErrUnableToParseIntermediateResult = &types.Error{
		Code:    7, //nolint
		Message: "Unable to parse intermediate result",
	}

	// ErrScriptPubKeysMissing is returned when
	// the indexer cannot populate the required
	ErrScriptPubKeysMissing = &types.Error{
		Code:    8, //nolint
		Message: "Missing ScriptPubKeys",
	}

	// ErrInvalidCoin is returned when a *types.Coin
	// cannot be parsed during construction.
	ErrInvalidCoin = &types.Error{
		Code:    9, //nolint
		Message: "Coin is invalid",
	}

	// ErrUnableToDecodeAddress is returned when an address
	// cannot be parsed during construction.
	ErrUnableToDecodeAddress = &types.Error{
		Code:    10, //nolint
		Message: "Unable to decode address",
	}

	// ErrUnableToDecodeScriptPubKey is returned when a
	// ScriptPubKey cannot be parsed during construction.
	ErrUnableToDecodeScriptPubKey = &types.Error{
		Code:    11, //nolint
		Message: "Unable to decode ScriptPubKey",
	}

	// ErrUnableToCalculateSignatureHash is returned
	// when some payload to sign cannot be generated.
	ErrUnableToCalculateSignatureHash = &types.Error{
		Code:    12, //nolint
		Message: "Unable to calculate signature hash",
	}

	// ErrUnsupportedScriptType is returned when
	// trying to sign an input with an unsupported
	// script type.
	ErrUnsupportedScriptType = &types.Error{
		Code:    13, //nolint
		Message: "Script type is not supported",
	}

	// ErrUnableToComputePkScript is returned
	// when trying to compute the PkScript in
	// ConsructionParse.
	ErrUnableToComputePkScript = &types.Error{
		Code:    14, //nolint
		Message: "Unable to compute PK script",
	}

	// ErrUnableToGetCoins is returned by the indexer
	// when it is not possible to get the coins
	// owned by a *types.AccountIdentifier.
	ErrUnableToGetCoins = &types.Error{
		Code:    15, //nolint
		Message: "Unable to get coins",
	}

	// ErrTransactionNotFound is returned by the indexer
	// when it is not possible to find a transaction.
	ErrTransactionNotFound = &types.Error{
		Code:    16, // nolint
		Message: "Transaction not found",
	}

	// ErrCouldNotGetFeeRate is returned when the fetch
	// to get the suggested fee rate fails.
	ErrCouldNotGetFeeRate = &types.Error{
		Code:    17, // nolint
		Message: "Could not get suggested fee rate",
	}

	// ErrUnableToGetBalance is returned by the indexer
	// when it is not possible to get the balance
	// of a *types.AccountIdentifier.
	ErrUnableToGetBalance = &types.Error{
		Code:    18, //nolint
		Message: "Unable to get balance",
	}
)

// errutil.WrapErr adds details to the types.Error provided. We use a function
// to do this so that we don't accidentially overrwrite the standard errors.
func WrapErr(rErr *types.Error, err error) *types.Error {
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
