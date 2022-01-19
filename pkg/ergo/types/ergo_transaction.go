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

// ErgoTransaction Ergo transaction
type ErgoTransaction struct {
	// Base16-encoded transaction id bytes
	ID *string `json:"id,omitempty"`
	// Inputs of the transaction
	Inputs *[]ErgoTransactionInput `json:"inputs"`
	// Data inputs of the transaction
	DataInputs *[]ErgoTransactionDataInput `json:"dataInputs"`
	// Outputs of the transaction
	Outputs *[]ErgoTransactionOutput `json:"outputs"`
	// Size in bytes
	Size *int32 `json:"size,omitempty"`
}