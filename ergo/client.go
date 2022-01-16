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
	"github.com/coinbase/rosetta-sdk-go/utils"
	ergotype "github.com/ross-weir/rosetta-ergo/ergo/types"
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
)

const (
	defaultTimeout = 100 * time.Second
	dialTimeout    = 5 * time.Second

	// Using fixed credentials as the nodes REST API should never be exposed on the internet
	nodeAPIToken = "rosetta"
)

type Client struct {
	baseURL string

	genesisBlockIdentifier *types.BlockIdentifier
	currency               *types.Currency

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
	genesisBlockIdentifier *types.BlockIdentifier,
	currency *types.Currency,
	logger *zap.Logger,
) *Client {
	return &Client{
		baseURL:                baseURL,
		genesisBlockIdentifier: genesisBlockIdentifier,
		currency:               currency,
		httpClient:             newHTTPClient(defaultTimeout),
		logger:                 logger.Sugar().Named("ergoclient"),
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

// Get `NetworkStatusResponse` for ergo
func (e *Client) NetworkStatus(ctx context.Context) (*types.NetworkStatusResponse, error) {
	// get current block id / timestamp
	currentBlockHeader, err := e.getLatestBlockHeaders(ctx, 1)
	if err != nil {
		return nil, err
	}

	currentBlock, err := ergoBlockHeaderToRosettaBlock(ctx, &currentBlockHeader[0])
	if err != nil {
		return nil, fmt.Errorf("%w: error converting ergo block header to rosetta", err)
	}

	peers, err := e.GetPeers(ctx)
	if err != nil {
		return nil, err
	}

	return &types.NetworkStatusResponse{
		CurrentBlockIdentifier: currentBlock.BlockIdentifier,
		CurrentBlockTimestamp:  currentBlock.Timestamp,
		GenesisBlockIdentifier: e.genesisBlockIdentifier,
		Peers:                  peers,
	}, nil
}

// Get list of connected `Peer`s for ergo
func (e *Client) GetPeers(ctx context.Context) ([]*types.Peer, error) {
	connectedPeers, err := e.getConnectedPeers(ctx)
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

// Get information about the connected Ergo node
func (e *Client) GetNodeInfo(ctx context.Context) (*ergotype.NodeInfo, error) {
	nodeInfo := &ergotype.NodeInfo{}

	err := e.makeRequest(ctx, nodeEndpointNodeInfo, http.MethodGet, nil, nodeInfo)
	if err != nil {
		return nil, fmt.Errorf("%w: error fetching node info", err)
	}

	return nodeInfo, nil
}

// GetRawBlock fetches a full Ergo block and returns all the newly created utxos (coins) created
func (e *Client) GetRawBlock(
	ctx context.Context,
	identifier *types.PartialBlockIdentifier,
) (*ergotype.FullBlock, []*InputCtx, error) {
	var block *ergotype.FullBlock
	var err error

	if identifier.Hash != nil {
		block, err = e.getBlockByID(ctx, identifier.Hash)
		if err != nil {
			return nil, nil, err
		}
	}

	if identifier.Index != nil {
		block, err = e.getBlockByIndex(ctx, identifier.Index)
		if err != nil {
			return nil, nil, err
		}
	}

	// TODO: as per types.PartialBlockIdentifier if neither is specified get the current block

	// We want to return all the input box ids here so they can be fetched
	// from coin storage for usage when assembling the rosetta block
	inputCoins := []*InputCtx{}
	outputs := []string{}
	for _, tx := range *block.BlockTransactions.Transactions {
		// keep track of outputs so we know if they're later used as an input
		for _, output := range *tx.Outputs {
			outputs = append(outputs, *output.BoxID)
		}

		for _, input := range *tx.Inputs {
			// If an output is used as an input in the same block
			// there's no need to fetch it from coin storage as it won't exist
			if !utils.ContainsString(outputs, input.BoxID) {
				i := InputCtx{
					TxID:    *tx.ID,
					InputID: input.BoxID,
				}
				inputCoins = append(inputCoins, &i)
			}
		}
	}

	return block, inputCoins, nil
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

func (e *Client) getConnectedPeers(ctx context.Context) ([]ergotype.Peer, error) {
	peers := []ergotype.Peer{}

	err := e.makeRequest(ctx, nodeEndpointConnectedPeers, http.MethodGet, nil, &peers)
	if err != nil {
		return nil, fmt.Errorf("%w: error fetching connected peers", err)
	}

	return peers, nil
}

func (e *Client) getLatestBlockHeaders(
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
func (e *Client) getBlockByIndex(
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

	return e.getBlockByID(ctx, &blockHeaders[0])
}

func (e *Client) getBlockByID(
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
