package services

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ross-weir/rosetta-ergo/configuration"
	"github.com/ross-weir/rosetta-ergo/ergo"
)

// NetworkAPIService implements the server.NetworkAPIServicer interface.
type NetworkAPIService struct {
	cfg    *configuration.Configuration
	client *ergo.Client
	i      Indexer
}

// NewNetworkAPIService creates a new instance of a NetworkAPIService.
func NewNetworkAPIService(
	config *configuration.Configuration,
	client *ergo.Client,
	indexer Indexer,
) server.NetworkAPIServicer {
	return &NetworkAPIService{
		cfg:    config,
		client: client,
		i:      indexer,
	}
}

// NetworkList implements the /network/list endpoint
func (s *NetworkAPIService) NetworkList(
	ctx context.Context,
	request *types.MetadataRequest,
) (*types.NetworkListResponse, *types.Error) {
	return &types.NetworkListResponse{
		NetworkIdentifiers: []*types.NetworkIdentifier{
			s.cfg.Network,
		},
	}, nil
}

// NetworkStatus implements the /network/status endpoint.
func (s *NetworkAPIService) NetworkStatus(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkStatusResponse, *types.Error) {
	if s.cfg.Mode != configuration.Online {
		return nil, wrapErr(ErrUnavailableOffline, nil)
	}

	peers, err := s.client.GetPeers(ctx)
	if err != nil {
		return nil, wrapErr(ErrErgoNode, err)
	}

	cachedBlockResponse, err := s.i.GetBlockLazy(ctx, nil)
	if err != nil {
		return nil, wrapErr(ErrNotReady, nil)
	}

	return &types.NetworkStatusResponse{
		CurrentBlockIdentifier: cachedBlockResponse.Block.BlockIdentifier,
		CurrentBlockTimestamp:  cachedBlockResponse.Block.Timestamp,
		GenesisBlockIdentifier: s.cfg.GenesisBlockIdentifier,
		Peers:                  peers,
	}, nil
}

// NetworkOptions implements the /network/options endpoint.
func (s *NetworkAPIService) NetworkOptions(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkOptionsResponse, *types.Error) {
	// TODO: this can be cached since it won't change after startup
	nodeInfo, err := s.client.GetNodeInfo(ctx)
	if err != nil {
		return nil, wrapErr(ErrErgoNode, err)
	}

	return &types.NetworkOptionsResponse{
		Version: &types.Version{
			RosettaVersion:    types.RosettaAPIVersion,
			NodeVersion:       nodeInfo.AppVersion,
			MiddlewareVersion: types.String(s.cfg.Version),
		},
		Allow: &types.Allow{
			OperationStatuses:       ergo.OperationStatuses,
			OperationTypes:          ergo.OperationTypes,
			Errors:                  Errors,
			HistoricalBalanceLookup: HistoricalBalanceLookup,
			MempoolCoins:            MempoolCoins,
		},
	}, nil
}
