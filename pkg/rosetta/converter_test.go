package rosetta_test

import (
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/google/go-cmp/cmp"
	ergotype "github.com/ross-weir/rosetta-ergo/pkg/ergo/types"
	"github.com/ross-weir/rosetta-ergo/pkg/rosetta"
	"github.com/stretchr/testify/assert"
)

func TestPeerToRosettaPeer(t *testing.T) {
	ergoPeer := ergotype.Peer{
		Address:        "/127.0.0.1:9020",
		Name:           types.String("ergo-testnet-4.0.0"),
		ConnectionType: types.String("Outgoing"),
	}
	actual, err := rosetta.PeerToRosettaPeer(&ergoPeer)
	expected := &types.Peer{
		PeerID: "/127.0.0.1:9020",
		Metadata: map[string]interface{}{
			"address":        "/127.0.0.1:9020",
			"connectionType": types.String("Outgoing"),
			"name":           types.String("ergo-testnet-4.0.0"),
		},
	}

	assert.Nil(t, err)
	assert.True(t, cmp.Equal(actual, expected))
}

func TestHeaderToRosettaBlock(t *testing.T) {
	ergoHeader := ergotype.BlockHeader{
		ID:        "3aca13c532e2085b2158f924b1ca92539d4612b1ac2e62a48ceb2d0043b33db7",
		ParentID:  "daf460f25be46b6b7d3683d61ff98562c4c5cf2c4fc76021b983c6c750b11548",
		Height:    2310,
		Timestamp: 1622012800818,
	}
	actual := rosetta.HeaderToRosettaBlock(&ergoHeader)
	expected := types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Hash:  "3aca13c532e2085b2158f924b1ca92539d4612b1ac2e62a48ceb2d0043b33db7",
			Index: 2310,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "daf460f25be46b6b7d3683d61ff98562c4c5cf2c4fc76021b983c6c750b11548",
			Index: 2309,
		},
		Timestamp: 1622012800818,
	}

	assert.True(t, cmp.Equal(actual, &expected))
}

// The parent block identifier for the genesis block should be the genesis block
func TestHeaderToRosettaBlockGenesis(t *testing.T) {
	ergoHeader := ergotype.BlockHeader{
		ID:        "78bc468ff36d5d6fa9707d382fe5b8947d6dd0ed2bc5c44707426ee12557c7fb",
		Height:    1,
		Timestamp: 1621820178111,
	}
	actual := rosetta.HeaderToRosettaBlock(&ergoHeader)
	expected := types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Hash:  "78bc468ff36d5d6fa9707d382fe5b8947d6dd0ed2bc5c44707426ee12557c7fb",
			Index: 1,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Hash:  "78bc468ff36d5d6fa9707d382fe5b8947d6dd0ed2bc5c44707426ee12557c7fb",
			Index: 1,
		},
		Timestamp: 1621820178111,
	}

	assert.True(t, cmp.Equal(actual, &expected))
}

// func TestBlockToRosettaBlock(t *testing.T) {
// 	ergoBlockJSON, _ := ioutil.ReadFile("testdata/ergo_block.json")
// 	ergoBlock := ergotype.FullBlock{}
// 	_ = json.Unmarshal([]byte(ergoBlockJSON), &ergoBlock)

// 	expected := &types.Block{
// 		BlockIdentifier: &types.BlockIdentifier{
// 			Index: 2310,
// 			Hash:  "3aca13c532e2085b2158f924b1ca92539d4612b1ac2e62a48ceb2d0043b33db7",
// 		},
// 		ParentBlockIdentifier: &types.BlockIdentifier{
// 			Index: 2309,
// 			Hash:  "daf460f25be46b6b7d3683d61ff98562c4c5cf2c4fc76021b983c6c750b11548",
// 		},
// 		Timestamp: 1622012800818,
// 		Transactions: []*types.Transaction{
// 			&types.Transaction{
// 				TransactionIdentifier: &types.TransactionIdentifier{
// 					Hash: "5f1a470c1c9e3a18b2d9a13ad319603b55e4df31d37cc2fbcae6dabe43257b6f",
// 				},
// 				Operations: []*types.Operation{
// 					&types.Operation{
// 						OperationIdentifier: &types.OperationIdentifier{
// 							Index:        0,
// 							NetworkIndex: types.Int64(0),
// 						},
// 						Type:   "INPUT",
// 						Status: types.String("SUCCESS"),
// 						Account: &types.AccountIdentifier{
// 							Address: "AfYgQf5PappexKq8Vpig4vwEuZLjrq7gV97BWBVcKymTYqRzCoJLE9cDBpGHvtAAkAgQf8Yyv7NQUjSphKSjYxk3dB3W8VXzHzz5MuCcNbqqKHnMDZAa6dbHH1uyMScq5rXPLFD5P8MWkD5FGE6RbHKrKjANcr6QZHcBpppdjh9r5nra4c7dsCgULFZfWYTaYqHpx646BUHhhp8jDCHzzF33G8XfgKYo93ABqmdqagbYRzrqCgPHv5kxRmFt7Y99z26VQTgXoEmXJ2aRu6LoB59rKN47JxWGos27D79kKzJRiyYNEVzXU8MYCxtAwV",
// 						},
// 						Amount: &types.Amount{
// 							Value: "93253275000000000",
// 							Currency: &types.Currency{
// 								Symbol:   "ERG",
// 								Decimals: 9,
// 							},
// 						},
// 						CoinChange: &types.CoinChange{
// 							CoinIdentifier: &types.CoinIdentifier{
// 								Identifier: "0d7b164a0341b726130cb5a6c4f8f4f4b227ca54099323f32a83de004567b76e",
// 							},
// 							CoinAction: "coin_spent",
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	assert.Equal(t, ergoBlock, nil)
// }
