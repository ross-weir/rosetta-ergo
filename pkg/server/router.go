package server

import (
	"net/http"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/ross-weir/rosetta-ergo/pkg/config"
	"github.com/ross-weir/rosetta-ergo/pkg/ergo"
	"github.com/ross-weir/rosetta-ergo/pkg/services"
	"github.com/ross-weir/rosetta-ergo/pkg/storage"
)

func NewBlockchainRouter(
	cfg *config.Configuration,
	client *ergo.Client,
	asserter *asserter.Asserter,
	storage *storage.Storage,
) http.Handler {
	networkAPIService := services.NewNetworkAPIService(cfg, client, storage)
	networkAPIController := server.NewNetworkAPIController(
		networkAPIService,
		asserter,
	)

	blockAPIService := services.NewBlockAPIService(cfg, storage)
	blockAPIController := server.NewBlockAPIController(
		blockAPIService,
		asserter,
	)

	accountAPIService := services.NewAccountAPIService(cfg, storage)
	accountAPIController := server.NewAccountAPIController(
		accountAPIService,
		asserter,
	)

	mempoolAPIService := services.NewMempoolAPIService(cfg, client, storage)
	mempoolAPIController := server.NewMempoolAPIController(
		mempoolAPIService,
		asserter,
	)

	return server.NewRouter(
		networkAPIController,
		blockAPIController,
		accountAPIController,
		mempoolAPIController,
	)
}
