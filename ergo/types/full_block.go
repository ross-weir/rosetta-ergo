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

// FullBlock Block with header and transactions
type FullBlock struct {
	Header            *BlockHeader       `json:"header"`
	BlockTransactions *BlockTransactions `json:"blockTransactions"`
	AdProofs          *BlockAdProofs     `json:"adProofs"`
	Extension         *Extension         `json:"extension"`
	// Size in bytes
	Size int32 `json:"size"`
}
