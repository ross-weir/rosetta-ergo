package ergo

import (
	"errors"
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/types"

	ergotype "github.com/ross-weir/rosetta-ergo/ergo/types"
)

const (
	genesisBlockIndex = 1
)

// Convert a ergo `FullBlock` to a rosetta `Block`
func ErgoBlockToRosetta( //revive:disable-line:exported
	b *ergotype.FullBlock,
) (*types.Block, error) {
	if b == nil {
		return nil, errors.New("error parsing nil block")
	}

	blockWithoutTxs, err := ergoBlockHeaderToRosettaBlock(b.Header)
	if err != nil {
		return nil, errors.New("error parsing block using header")
	}

	// TODO: add txs

	return blockWithoutTxs, nil
}

func ergoBlockHeaderToRosettaBlock(bh *ergotype.BlockHeader) (*types.Block, error) {
	if bh == nil {
		return nil, errors.New("error parsing nil block")
	}

	blockIndex := int64(bh.Height)
	previousBlockIndex := blockIndex - 1
	previousBlockHash := bh.ParentID

	// the genesis blocks parent is itself according to the rosetta spec
	if blockIndex == genesisBlockIndex {
		previousBlockIndex = genesisBlockIndex
		previousBlockHash = bh.ID
	}

	return &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Hash:  bh.ID,
			Index: blockIndex,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Hash:  previousBlockHash,
			Index: previousBlockIndex,
		},
		Timestamp: bh.Timestamp,
	}, nil
}

func ergoPeerToRosettaPeer(p *ergotype.Peer) (*types.Peer, error) {
	if p == nil {
		return nil, errors.New("error parsing nil peer")
	}

	metadata, err := types.MarshalMap(p)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to marshal peer metadata", err)
	}

	return &types.Peer{
		PeerID:   p.Address,
		Metadata: metadata,
	}, nil
}