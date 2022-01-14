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

// CandidateBlock Can be null if node is not mining or candidate block is not ready
type CandidateBlock struct {
	Version *int32 `json:"version,omitempty"`
	// Base16-encoded 32 byte digest
	ExtensionHash string `json:"extensionHash"`
	// Basic timestamp definition
	Timestamp *int64 `json:"timestamp,omitempty"`
	// Base16-encoded 33 byte digest - digest with extra byte with tree height
	StateRoot *string `json:"stateRoot,omitempty"`
	NBits     *int64  `json:"nBits,omitempty"`
	// Base16-encoded ad proofs
	AdProofBytes *string `json:"adProofBytes,omitempty"`
	// Base16-encoded 32 byte modifier id
	ParentID           string `json:"parentId"`
	TransactionsNumber *int32 `json:"transactionsNumber,omitempty"`
	// Ergo transaction objects
	Transactions *[]ErgoTransaction `json:"transactions,omitempty"`
	// Base16-encoded votes for a soft-fork and parameters
	Votes *string `json:"votes,omitempty"`
}
