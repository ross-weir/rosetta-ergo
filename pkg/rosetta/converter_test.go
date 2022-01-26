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
