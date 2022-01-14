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

// NodeInfo struct for NodeInfo
type NodeInfo struct {
	Name       string `json:"name"`
	AppVersion string `json:"appVersion"`
	// Can be 'null' if state is empty (no full block is applied since node launch)
	FullHeight int32 `json:"fullHeight"`
	// Can be 'null' if state is empty (no header applied since node launch)
	HeadersHeight int32 `json:"headersHeight"`
	// Can be 'null' if no full block is applied since node launch
	BestFullHeaderId string `json:"bestFullHeaderId"`
	// Can be 'null' if no full block is applied since node launch
	PreviousFullHeaderId string `json:"previousFullHeaderId"`
	// Can be 'null' if no header applied since node launch
	BestHeaderId string `json:"bestHeaderId"`
	// Can be 'null' if state is empty (no full block is applied since node launch)
	StateRoot string `json:"stateRoot"`
	StateType string `json:"stateType"`
	// Can be 'null' if no full block is applied since node launch
	StateVersion string `json:"stateVersion"`
	IsMining     bool   `json:"isMining"`
	// Number of connected peers
	PeersCount int32 `json:"peersCount"`
	// Current unconfirmed transactions count
	UnconfirmedCount int32 `json:"unconfirmedCount"`
	// Difficulty on current bestFullHeaderId. Can be 'null' if no full block is applied since node launch
	Difficulty int64 `json:"difficulty"`
	// Current internal node time
	CurrentTime int64 `json:"currentTime"`
	// Time when the node was started
	LaunchTime int64 `json:"launchTime"`
	// Can be 'null' if no headers is applied since node launch
	HeadersScore int64 `json:"headersScore"`
	// Can be 'null' if no full block is applied since node launch
	FullBlocksScore int64 `json:"fullBlocksScore"`
	// Can be 'null' if genesis blocks is not produced yet
	GenesisBlockId string      `json:"genesisBlockId"`
	Parameters     *Parameters `json:"parameters"`
}
