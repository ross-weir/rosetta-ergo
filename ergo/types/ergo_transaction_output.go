/*
 * Ergo Node API
 *
 * API docs for Ergo Node. Models are shared between all Ergo products
 *
 * API version: 4.0.20.2
 * Contact: ergoplatform@protonmail.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package ergo

// ErgoTransactionOutput struct for ErgoTransactionOutput
type ErgoTransactionOutput struct {
	// Base16-encoded transaction box id bytes. Should be 32 bytes long
	BoxId *string `json:"boxId,omitempty"`
	// Amount of Ergo token
	Value int64 `json:"value"`
	// Base16-encoded ergo tree bytes
	ErgoTree string `json:"ergoTree"`
	// Height the output was created at
	CreationHeight int32 `json:"creationHeight"`
	// Assets list in the transaction
	Assets *[]Asset `json:"assets,omitempty"`
	// Ergo box registers
	AdditionalRegisters map[string]string `json:"additionalRegisters"`
	// Base16-encoded transaction id bytes
	TransactionId *string `json:"transactionId,omitempty"`
	// Index in the transaction outputs
	Index *int32 `json:"index,omitempty"`
}
