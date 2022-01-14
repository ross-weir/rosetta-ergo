package ergo

import "github.com/coinbase/rosetta-sdk-go/types"

const (
	// The rosetta blockchain is Ergo.
	Blockchain string = "Ergo"

	// The identifier for the mainnet network identifier in rosetta
	MainnetNetwork string = "Mainnet"

	// The identifier for the testnet network identifier in rosetta
	TestnetNetwork string = "Testnet"

	// The amount of nano ergs = 1 erg
	NanoErgInErg = 1000000000

	// Input op type represents a UTXO input in a transaction
	InputOpType = "INPUT"

	// Output op type represents a UTXO output in a transaction
	OutputOpType = "OUTPUT"
)

var (
	MainnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Hash: "b0244dfc267baca974a4caee06120321562784303a8a688976ae56170e4d175b",
	}

	TestnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Hash: "78bc468ff36d5d6fa9707d382fe5b8947d6dd0ed2bc5c44707426ee12557c7fb",
	}

	Currency = &types.Currency{
		Symbol:   "ERG",
		Decimals: 9,
	}

	OperationTypes = []string{
		InputOpType,
		OutputOpType,
	}
)
