package ergo

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/coinbase/rosetta-sdk-go/types"

	ergotype "github.com/ross-weir/rosetta-ergo/pkg/ergo/types"
)

const (
	genesisBlockIndex = 1
)

// ErgoBlockToRosetta converts a ergo `FullBlock` to a rosetta `Block`
func ErgoBlockToRosetta( //revive:disable-line:exported
	ctx context.Context,
	e *Client,
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

	txs, err := ErgoBlockTxsToRosettaTxs(ctx, e, b.BlockTransactions.Transactions, coins)
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

// ErgoBlockTxsToRosettaTxs extracts rosetta transactions from ergo full block
func ErgoBlockTxsToRosettaTxs( //revive:disable-line:exported
	ctx context.Context,
	e *Client,
	txs *[]ergotype.ErgoTransaction,
	coins map[string]*types.AccountCoin,
) ([]*types.Transaction, error) {
	if txs == nil {
		return nil, errors.New("ergoBlockToRosettaTxs: error parsing nil txs")
	}

	blockTxs := *txs
	parsedTxs := make([]*types.Transaction, len(blockTxs))

	for index, transaction := range blockTxs {
		txOps, err := ErgoTransactionToRosettaOps(ctx, e, &blockTxs[index], coins)
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

		parsedTxs[index] = tx

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

	return parsedTxs, nil
}

// ErgoTransactionToRosettaOps extracts rosetta operations from an ergo transactions inputs/outputs
func ErgoTransactionToRosettaOps( //revive:disable-line:exported
	ctx context.Context,
	e *Client,
	tx *ergotype.ErgoTransaction,
	coins map[string]*types.AccountCoin,
) ([]*types.Operation, error) {
	txOps := []*types.Operation{}
	inputs := *tx.Inputs

	for inputIndex, input := range inputs {
		// get the account coin related to this input (i.e the previous unspent utxo)
		// this is needed for value, etc
		accountCoin, ok := coins[input.BoxID]
		if !ok {
			return nil, fmt.Errorf("error finding input %s, for tx: %s", input.BoxID, *tx.ID)
		}

		txOp, err := ergoInputToRosettaTxOp(
			&inputs[inputIndex],
			int64(len(txOps)),
			int64(inputIndex),
			accountCoin,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: error parsing tx input, boxId: %s", err, input.BoxID)
		}

		txOps = append(txOps, txOp)
	}

	outputs := *tx.Outputs
	for i, output := range outputs {
		txOp, err := ergoOutputToRosettaTxOp(ctx, e, &outputs[i], int64(len(txOps)))
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
	coin *types.AccountCoin,
) (*types.Operation, error) {
	negatedVal, err := types.NegateValue(coin.Coin.Amount.Value)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to negate previous output", err)
	}

	return &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index:        operationIndex,
			NetworkIndex: &inputIndex,
		},
		Type:    InputOpType,
		Status:  types.String(SuccessStatus),
		Account: coin.Account,
		Amount: &types.Amount{
			Value:    negatedVal,
			Currency: Currency,
		},
		CoinChange: &types.CoinChange{
			CoinIdentifier: &types.CoinIdentifier{
				Identifier: input.BoxID,
			},
			CoinAction: types.CoinSpent,
		},
		// metadata
	}, nil
}

func ergoOutputToRosettaTxOp(
	ctx context.Context,
	e *Client,
	output *ergotype.ErgoTransactionOutput,
	operationIndex int64,
) (*types.Operation, error) {
	coinChange := &types.CoinChange{
		CoinIdentifier: &types.CoinIdentifier{
			Identifier: *output.BoxID,
		},
		CoinAction: types.CoinCreated,
	}

	addr, err := e.TreeToAddress(ctx, output.ErgoTree)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get address from ergo tree", err)
	}

	account := &types.AccountIdentifier{Address: *addr}
	intBase := 10

	return &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index:        operationIndex,
			NetworkIndex: types.Int64(int64(*output.Index)),
		},
		Type:    OutputOpType,
		Status:  types.String(SuccessStatus),
		Account: account,
		Amount: &types.Amount{
			Value:    strconv.FormatInt(output.Value, intBase),
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
