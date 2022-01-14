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

// Parameters struct for Parameters
type Parameters struct {
	// Height when current parameters were considered(not actual height). Can be '0' if state is
	// empty
	Height int32 `json:"height"`
	// Storage fee coefficient (per byte per storage period ~4 years)
	StorageFeeFactor int32 `json:"storageFeeFactor"`
	// Minimum value per byte of an output
	MinValuePerByte int32 `json:"minValuePerByte"`
	// Maximum block size (in bytes)
	MaxBlockSize int32 `json:"maxBlockSize"`
	// Maximum cumulative computational cost of input scripts in block transactions
	MaxBlockCost int32 `json:"maxBlockCost"`
	// Ergo blockchain protocol version
	BlockVersion int32 `json:"blockVersion"`
	// Validation cost of a single token
	TokenAccessCost int32 `json:"tokenAccessCost"`
	// Validation cost per one transaction input
	InputCost int32 `json:"inputCost"`
	// Validation cost per one data input
	DataInputCost int32 `json:"dataInputCost"`
	// Validation cost per one transaction output
	OutputCost int32 `json:"outputCost"`
}
