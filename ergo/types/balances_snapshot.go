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

// BalancesSnapshot Amount of Ergo tokens and assets
type BalancesSnapshot struct {
	Height  int32    `json:"height"`
	Balance int64    `json:"balance"`
	Assets  *[]Asset `json:"assets,omitempty"`
}
