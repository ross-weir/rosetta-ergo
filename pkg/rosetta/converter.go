package rosetta

import (
	"context"
	"fmt"
	"strconv"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/pkg/ergo"
	ergotype "github.com/ross-weir/rosetta-ergo/pkg/ergo/types"
)

// BlockConverter converts a full ergo block into a full rosetta block
type BlockConverter struct {
	e *ergo.Client

	// blockTxInputs is a running cache of outputs that are
	// produced from txs in this block
	// We keep track of these in case they're used as
	// inputs in txs later in the block
	blockTxInputs map[string]*types.AccountCoin

	// Boxes to be used as inputs for operations
	// in this converted block
	preFetchedInputs map[string]*types.AccountCoin
}

func NewBlockConverter(
	e *ergo.Client,
	preFetchedInputs map[string]*types.AccountCoin,
) *BlockConverter {
	return &BlockConverter{
		e:                e,
		blockTxInputs:    map[string]*types.AccountCoin{},
		preFetchedInputs: preFetchedInputs,
	}
}

func (c *BlockConverter) BlockToRosettaBlock(
	ctx context.Context,
	block *ergotype.FullBlock,
) (*types.Block, error) {
	b := HeaderToRosettaBlock(block.Header)
	ergoTxs := *block.BlockTransactions.Transactions
	parsedTxs := make([]*types.Transaction, len(ergoTxs))

	for idx, ergoTx := range ergoTxs {
		parsedTx, err := c.TxToRosettaTx(ctx, &ergoTxs[idx])
		if err != nil {
			return nil, fmt.Errorf(
				"%w: error parsing transaction operations, txId: %s",
				err,
				*ergoTx.ID,
			)
		}

		parsedTxs[idx] = parsedTx

		// store outputs for this tx in `c.inputCoins`
		// incase they're used in later txs
		c.storeTxCoins(parsedTx)
	}

	b.Transactions = parsedTxs

	return b, nil
}

func (c *BlockConverter) TxToRosettaTx(
	ctx context.Context,
	ergoTx *ergotype.ErgoTransaction,
) (*types.Transaction, error) {
	txOps := []*types.Operation{}
	inputs := *ergoTx.Inputs

	for inputIdx, input := range inputs {
		acctCoin, err := c.findCoinForInput(input.BoxID)
		if err != nil {
			return nil, fmt.Errorf("%w: unable to find input (tx: %s)", err, *ergoTx.ID)
		}

		txOp, err := c.InputToRosettaOp(
			&inputs[inputIdx],
			int64(len(txOps)),
			int64(inputIdx),
			acctCoin,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: error parsing tx input, boxId: %s", err, input.BoxID)
		}

		txOps = append(txOps, txOp)
	}

	outputs := *ergoTx.Outputs
	for outputIdx, output := range outputs {
		txOp, err := c.OutputToRosettaOp(ctx, &outputs[outputIdx], int64(len(txOps)))
		if err != nil {
			return nil, fmt.Errorf("%w: error parsing tx output, boxId: %s", err, *output.BoxID)
		}

		txOps = append(txOps, txOp)
	}

	// TODO: metadata?
	return &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: *ergoTx.ID,
		},
		Operations: txOps,
	}, nil
}

func (c *BlockConverter) InputToRosettaOp(
	i *ergotype.ErgoTransactionInput,
	operationIndex int64,
	inputIndex int64,
	inputCoin *types.AccountCoin,
) (*types.Operation, error) {
	negatedVal, err := types.NegateValue(inputCoin.Coin.Amount.Value)
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
		Account: inputCoin.Account,
		Amount: &types.Amount{
			Value:    negatedVal,
			Currency: Currency,
		},
		CoinChange: &types.CoinChange{
			CoinIdentifier: &types.CoinIdentifier{
				Identifier: i.BoxID,
			},
			CoinAction: types.CoinSpent,
		},
		// metadata
	}, nil
}

func (c *BlockConverter) OutputToRosettaOp(
	ctx context.Context,
	o *ergotype.ErgoTransactionOutput,
	operationIndex int64,
) (*types.Operation, error) {
	coinChange := &types.CoinChange{
		CoinIdentifier: &types.CoinIdentifier{
			Identifier: *o.BoxID,
		},
		CoinAction: types.CoinCreated,
	}

	addr, err := c.e.TreeToAddress(ctx, o.ErgoTree)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get address from ergo tree", err)
	}

	account := &types.AccountIdentifier{Address: *addr}
	intBase := 10

	return &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index:        operationIndex,
			NetworkIndex: types.Int64(int64(*o.Index)),
		},
		Type:    OutputOpType,
		Status:  types.String(SuccessStatus),
		Account: account,
		Amount: &types.Amount{
			Value:    strconv.FormatInt(o.Value, intBase),
			Currency: Currency,
		},
		CoinChange: coinChange,
		// Metadata
	}, nil
}

// Preserve input coins used in transactions so they can
// be used for later transactions in the block if required
func (c *BlockConverter) storeTxCoins(tx *types.Transaction) {
	for _, op := range tx.Operations {
		if op.CoinChange == nil {
			continue
		}

		if op.CoinChange.CoinAction != types.CoinCreated {
			continue
		}

		c.blockTxInputs[op.CoinChange.CoinIdentifier.Identifier] = &types.AccountCoin{
			Coin: &types.Coin{
				CoinIdentifier: op.CoinChange.CoinIdentifier,
				Amount:         op.Amount,
			},
			Account: op.Account,
		}
	}
}

func (c *BlockConverter) findCoinForInput(inputID string) (*types.AccountCoin, error) {
	acctCoin, ok := c.blockTxInputs[inputID]
	if ok {
		return acctCoin, nil
	}

	acctCoin, ok = c.preFetchedInputs[inputID]
	if ok {
		return acctCoin, nil
	}

	return nil, fmt.Errorf("unable to find input with ID: %s", inputID)
}

func PeerToRosettaPeer(p *ergotype.Peer) (*types.Peer, error) {
	metadata, err := types.MarshalMap(p)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to marshal peer metadata", err)
	}

	return &types.Peer{
		PeerID:   p.Address,
		Metadata: metadata,
	}, nil
}

func HeaderToRosettaBlock(bh *ergotype.BlockHeader) *types.Block {
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
	}
}
