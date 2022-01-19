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

// NipopowProof struct for NipopowProof
type NipopowProof struct {
	// security parameter (min μ-level superchain length)
	M float32 `json:"m"`
	// security parameter (min suffix length, >= 1)
	K float32 `json:"k"`
	// proof prefix headers
	Prefix     *[]PopowHeader `json:"prefix"`
	SuffixHead *PopowHeader   `json:"suffixHead"`
	// tail of the proof suffix headers
	SuffixTail *[]BlockHeader `json:"suffixTail"`
}