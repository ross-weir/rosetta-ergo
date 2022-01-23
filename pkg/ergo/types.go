package ergo

import (
	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	// The rosetta blockchain is Ergo.
	Blockchain string = "Ergo"

	// The identifier for the mainnet network identifier in rosetta
	MainnetNetwork string = "mainnet"

	// The identifier for the testnet network identifier in rosetta
	TestnetNetwork string = "testnet"

	ErgDecimals = 9

	// The amount of nano ergs = 1 erg
	NanoErgInErg = 1000000000

	// Input op type represents a UTXO input in a transaction
	InputOpType = "INPUT"

	// Output op type represents a UTXO output in a transaction
	OutputOpType = "OUTPUT"
)

var (
	MainnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Index: 1,
		Hash:  "b0244dfc267baca974a4caee06120321562784303a8a688976ae56170e4d175b",
	}

	TestnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Index: 1,
		Hash:  "78bc468ff36d5d6fa9707d382fe5b8947d6dd0ed2bc5c44707426ee12557c7fb",
	}

	Currency = &types.Currency{
		Symbol:   "ERG",
		Decimals: ErgDecimals,
	}

	OperationTypes = []string{
		InputOpType,
		OutputOpType,
	}

	// SuccessStatus is the status of all
	// Ergo operations because anything
	// on-chain is considered successful.
	SuccessStatus = "SUCCESS"

	// OperationStatuses are all supported operation.Status.
	OperationStatuses = []*types.OperationStatus{
		{
			Status:     SuccessStatus,
			Successful: true,
		},
	}
)
