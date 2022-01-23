package ergo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/coinbase/rosetta-sdk-go/types"
	ergotype "github.com/ross-weir/rosetta-ergo/pkg/ergo/types"
	"go.uber.org/zap"
)

type nodeEndpoint string

const (
	nodeEndpointNodeInfo       nodeEndpoint = "info"
	nodeEndpointConnectedPeers nodeEndpoint = "peers/connected"

	nodeEndpointBlock            nodeEndpoint = "blocks"
	nodeEndpointLastBlockHeaders nodeEndpoint = "blocks/lastHeaders"
	nodeEndpointBlockAtHeight    nodeEndpoint = "blocks/at"

	nodeEndpointTxUnconfirmed nodeEndpoint = "transactions/unconfirmed"

	nodeEndpointTreeToAddress nodeEndpoint = "utils/ergoTreeToAddress"
)

const (
	defaultTimeout = 100 * time.Second
	dialTimeout    = 5 * time.Second

	// Using fixed credentials as the nodes REST API should never be exposed on the internet
	nodeAPIToken = "rosetta"
)

type Client struct {
	baseURL string

	httpClient *http.Client
	logger     *zap.SugaredLogger
}

// Return the URL to use for a client where the node is running locally
func LocalNodeURL(nodePort int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", nodePort)
}

// Create a new ergo node client
func NewClient(
	baseURL string,
	logger *zap.Logger,
) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: newHTTPClient(defaultTimeout),
		logger:     logger.Sugar().Named("ergoclient"),
	}
}

func newHTTPClient(timeout time.Duration) *http.Client {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: dialTimeout,
		}).Dial,
	}

	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: netTransport,
	}

	return httpClient
}

// GetPeers gets a list of connected `Peer`s for ergo
func (e *Client) GetPeers(ctx context.Context) ([]*types.Peer, error) {
	connectedPeers, err := e.GetConnectedPeers(ctx)
	if err != nil {
		return nil, err
	}

	peers := make([]*types.Peer, len(connectedPeers))
	for i := range connectedPeers {
		peer, err := ergoPeerToRosettaPeer(&connectedPeers[i])
		if err != nil {
			return nil, fmt.Errorf("%w: unable to convert ergo peer to rosetta peer", err)
		}

		peers[i] = peer
	}

	return peers, nil
}

// GetNodeInfo gets information about the connected Ergo node
func (e *Client) GetNodeInfo(ctx context.Context) (*ergotype.NodeInfo, error) {
	nodeInfo := &ergotype.NodeInfo{}

	err := e.makeRequest(ctx, nodeEndpointNodeInfo, http.MethodGet, nil, nodeInfo)
	if err != nil {
		return nil, fmt.Errorf("%w: error fetching node info", err)
	}

	return nodeInfo, nil
}

// GetUnconfirmedTxs gets all the unconfirmed transactions currently in mempool
func (e *Client) GetUnconfirmedTxs(ctx context.Context) ([]ergotype.ErgoTransaction, error) {
	txs := []ergotype.ErgoTransaction{}

	err := e.makeRequest(ctx, nodeEndpointTxUnconfirmed, http.MethodGet, nil, &txs)
	if err != nil {
		return nil, fmt.Errorf("%w: error fetching unconfirmed txs", err)
	}

	return txs, nil
}

func (e *Client) TreeToAddress(ctx context.Context, et string) (*string, error) {
	var addr = &ergotype.AddressHolder{}

	err := e.makeRequest(
		ctx,
		nodeEndpoint(fmt.Sprintf("%s/%s", nodeEndpointTreeToAddress, et)),
		http.MethodGet,
		nil,
		addr,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: error converting ergo tree to address", err)
	}

	return &addr.Address, nil
}

func (e *Client) GetConnectedPeers(ctx context.Context) ([]ergotype.Peer, error) {
	peers := []ergotype.Peer{}

	err := e.makeRequest(ctx, nodeEndpointConnectedPeers, http.MethodGet, nil, &peers)
	if err != nil {
		return nil, fmt.Errorf("%w: error fetching connected peers", err)
	}

	return peers, nil
}

func (e *Client) GetLatestBlockHeaders(
	ctx context.Context,
	count int32,
) ([]ergotype.BlockHeader, error) {
	blockHeaders := []ergotype.BlockHeader{}

	err := e.makeRequest(
		ctx,
		nodeEndpoint(fmt.Sprintf("%s/%d", nodeEndpointLastBlockHeaders, count)),
		http.MethodGet,
		nil,
		&blockHeaders,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: error fetching last %d block headers", err, count)
	}

	return blockHeaders, nil
}

// Get a ergo block by index (height)
func (e *Client) GetBlockByIndex(
	ctx context.Context,
	indexPtr *int64,
) (*ergotype.FullBlock, error) {
	var blockHeaders []string
	index := *indexPtr

	err := e.makeRequest(
		ctx,
		nodeEndpoint(fmt.Sprintf("%s/%d", nodeEndpointBlockAtHeight, index)),
		http.MethodGet,
		nil,
		&blockHeaders,
	)

	if err != nil {
		return nil, fmt.Errorf("%w: error fetching block at index %d", err, index)
	}
	if len(blockHeaders) == 0 {
		return nil, fmt.Errorf("%w: no blocks found at index %d", err, index)
	}

	return e.GetBlockByID(ctx, &blockHeaders[0])
}

func (e *Client) GetBlockByID(
	ctx context.Context,
	idPtr *string,
) (*ergotype.FullBlock, error) {
	block := &ergotype.FullBlock{}
	id := *idPtr

	err := e.makeRequest(
		ctx,
		nodeEndpoint(fmt.Sprintf("%s/%s", nodeEndpointBlock, id)),
		http.MethodGet,
		nil,
		block,
	)

	if err != nil {
		return nil, fmt.Errorf("%w: error fetching block with id %s", err, id)
	}

	return block, nil
}

// Make a request to the node API endpoint
// Attempts to marshal the http response into the `response` parameter
func (e *Client) makeRequest(
	ctx context.Context,
	endpoint nodeEndpoint,
	requestMethod string,
	requestBody []interface{},
	response interface{},
) error {
	var requestBodyJSON []byte
	var err error

	if requestBody != nil {
		requestBodyJSON, err = json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("%w: error marshalling api request body", err)
		}
	}

	req, err := http.NewRequest(
		requestMethod,
		e.baseURL+"/"+string(endpoint),
		bytes.NewReader(requestBodyJSON),
	)
	if err != nil {
		return fmt.Errorf("%w: error constructing request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", nodeAPIToken)

	res, err := e.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("%w: error posting to api", err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid response: %s %s", res.Status, string(body))
	}

	if err != nil {
		return fmt.Errorf("%w: failed to read response body", err)
	}

	if err = json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("%w: error unmarshalling response body", err)
	}

	return nil
}
