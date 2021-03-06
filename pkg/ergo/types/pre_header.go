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

// PreHeader struct for PreHeader
type PreHeader struct {
	// Basic timestamp definition
	Timestamp int64 `json:"timestamp"`
	// Ergo blockchain protocol version
	Version int32 `json:"version"`
	NBits   int64 `json:"nBits"`
	Height  int32 `json:"height"`
	// Base16-encoded 32 byte modifier id
	ParentID string `json:"parentId"`
	// Base16-encoded votes for a soft-fork and parameters
	Votes   string  `json:"votes"`
	MinerPk *string `json:"minerPk,omitempty"`
}
