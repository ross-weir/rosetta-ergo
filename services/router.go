package services

import (
	"net/http"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/ross-weir/rosetta-ergo/configuration"
	"github.com/ross-weir/rosetta-ergo/ergo"
)

func NewBlockchainRouter(
	cfg *configuration.Configuration,
	client *ergo.Client,
	asserter *asserter.Asserter,
	i Indexer,
) http.Handler {
	networkAPIService := NewNetworkAPIService(cfg, client)
	networkAPIController := server.NewNetworkAPIController(
		networkAPIService,
		asserter,
	)

	blockAPIService := NewBlockAPIService(cfg, i)
	blockAPIController := server.NewBlockAPIController(
		blockAPIService,
		asserter,
	)

	return server.NewRouter(networkAPIController, blockAPIController)
}
