package ergo

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/coinbase/rosetta-sdk-go/types"

	ergotype "github.com/ross-weir/rosetta-ergo/ergo/types"
)

const (
	genesisBlockIndex = 1
)

// Convert a ergo `FullBlock` to a rosetta `Block`
func ErgoBlockToRosetta( //revive:disable-line:exported
	ctx context.Context,
	b *ergotype.FullBlock,
	coins map[string]*types.AccountCoin,
) (*types.Block, error) {
	if b == nil {
		return nil, errors.New("error parsing nil block")
	}

	block, err := ergoBlockHeaderToRosettaBlock(ctx, b.Header)
	if err != nil {
		return nil, err
	}

	txs, err := ergoBlockToRosettaTxs(ctx, b, coins)
	if err != nil {
		return nil, err
	}

	block.Transactions = txs

	return block, nil
}

func ergoBlockHeaderToRosettaBlock(
	ctx context.Context,
	bh *ergotype.BlockHeader,
) (*types.Block, error) {
	if bh == nil {
		return nil, errors.New("ergoBlockHeaderToRosettaBlock: error parsing nil block")
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

// ergoBlockToRosettaTxs extracts rosetta transactions from ergo full block
func ergoBlockToRosettaTxs(
	ctx context.Context,
	fb *ergotype.FullBlock,
	coins map[string]*types.AccountCoin,
) ([]*types.Transaction, error) {
	if fb == nil {
		return nil, errors.New("ergoBlockToRosettaTxs: error parsing nil block")
	}

	// TODO: might be able to just pass BlockTransactions
	txs := make([]*types.Transaction, len(*fb.BlockTransactions.Transactions))

	for index, transaction := range *fb.BlockTransactions.Transactions {
		txOps, err := ergoTransactionToRosettaOps(&transaction)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: error parsing transaction operations, txId: %s",
				err,
				*transaction.ID,
			)
		}

		// get metadata
		tx := &types.Transaction{
			TransactionIdentifier: &types.TransactionIdentifier{
				Hash: *transaction.ID,
			},
			Operations: txOps,
		}

		txs[index] = tx

		// In some cases, a transaction will spent an output
		// from the same block.
		// It won't have been previously fetched from coin storage so populate it now.
		for _, op := range tx.Operations {
			if op.CoinChange == nil {
				continue
			}

			if op.CoinChange.CoinAction != types.CoinCreated {
				continue
			}

			coins[op.CoinChange.CoinIdentifier.Identifier] = &types.AccountCoin{
				Coin: &types.Coin{
					CoinIdentifier: op.CoinChange.CoinIdentifier,
					Amount:         op.Amount,
				},
				Account: op.Account,
			}
		}
	}

	return txs, nil
}

// ergoTransactionToRosettaOps extracts rosetta operations from an ergo transactions inputs/outputs
func ergoTransactionToRosettaOps(tx *ergotype.ErgoTransaction) ([]*types.Operation, error) {
	txOps := []*types.Operation{}

	for inputIndex, input := range *tx.Inputs {
		// get the account coin related to this input (i.e the previous unspent utxo)
		// this is needed for value, etc

		txOp, err := ergoInputToRosettaTxOp(&input, int64(len(txOps)), int64(inputIndex))
		if err != nil {
			return nil, fmt.Errorf("%w: error parsing tx input", err)
		}

		txOps = append(txOps, txOp)
	}

	for _, output := range *tx.Outputs {
		txOp, err := ergoOutputToRosettaTxOp(&output, int64(len(txOps)))
		if err != nil {
			return nil, fmt.Errorf("%w: error parsing tx output, boxId: %s", err, *output.BoxID)
		}

		txOps = append(txOps, txOp)
	}

	return txOps, nil
}

func ergoInputToRosettaTxOp(
	input *ergotype.ErgoTransactionInput,
	operationIndex int64,
	inputIndex int64,
) (*types.Operation, error) {

	return nil, nil
}

func ergoOutputToRosettaTxOp(
	output *ergotype.ErgoTransactionOutput,
	operationIndex int64,
) (*types.Operation, error) {
	coinChange := &types.CoinChange{
		CoinIdentifier: &types.CoinIdentifier{
			Identifier: *output.BoxID,
		},
		CoinAction: types.CoinCreated,
	}

	// get the account (pk)

	return &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index:        operationIndex,
			NetworkIndex: types.Int64(int64(*output.Index)),
		},
		Type:   OutputOpType,
		Status: types.String(SuccessStatus),
		//Account
		Amount: &types.Amount{
			Value:    strconv.FormatInt(output.Value, 10),
			Currency: Currency,
		},
		CoinChange: coinChange,
		// Metadata
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
