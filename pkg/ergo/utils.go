package ergo

import (
	"github.com/coinbase/rosetta-sdk-go/utils"
	ergotype "github.com/ross-weir/rosetta-ergo/pkg/ergo/types"
)

// GetInputsForTxs gathers all input boxes required for a set of txs
// Useful for the indexer to be able to wait for txs containing a desired input
func GetInputsForTxs(txs *[]ergotype.ErgoTransaction) []*InputCtx {
	inputCoins := []*InputCtx{}
	outputs := []string{}
	for _, tx := range *txs {
		// keep track of outputs so we know if they're later used as an input
		for _, output := range *tx.Outputs {
			outputs = append(outputs, *output.BoxID)
		}

		for _, input := range *tx.Inputs {
			// If an output is used as an input in the same block
			// there's no need to fetch it from coin storage as it won't exist
			if !utils.ContainsString(outputs, input.BoxID) {
				i := InputCtx{
					TxID:    *tx.ID,
					InputID: input.BoxID,
				}
				inputCoins = append(inputCoins, &i)
			}
		}
	}

	return inputCoins
}
