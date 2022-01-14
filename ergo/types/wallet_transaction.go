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

// WalletTransaction Transaction augmented with some useful information
type WalletTransaction struct {
	// Base16-encoded transaction id bytes
	ID *string `json:"id,omitempty"`
	// Transaction inputs
	Inputs *[]ErgoTransactionInput `json:"inputs"`
	// Transaction data inputs
	DataInputs *[]ErgoTransactionDataInput `json:"dataInputs"`
	// Transaction outputs
	Outputs *[]ErgoTransactionOutput `json:"outputs"`
	// Height of a block the transaction was included in
	InclusionHeight int32 `json:"inclusionHeight"`
	// Number of transaction confirmations
	NumConfirmations int32 `json:"numConfirmations"`
	// Scan identifiers the transaction relates to
	Scans []int32 `json:"scans"`
	// Size in bytes
	Size *int32 `json:"size,omitempty"`
}
