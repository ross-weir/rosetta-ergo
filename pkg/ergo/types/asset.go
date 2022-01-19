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

// Asset Token detail in the transaction
type Asset struct {
	// Base16-encoded 32 byte digest
	TokenID string `json:"tokenId"`
	// Amount of the token
	Amount int64 `json:"amount"`
}