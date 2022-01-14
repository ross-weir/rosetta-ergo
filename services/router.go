package services

import (
	"net/http"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/ross-weir/rosetta-ergo/configuration"
)

func NewBlockchainRouter(
	cfg *configuration.Configuration,
	asserter *asserter.Asserter,
) http.Handler {

	return server.NewRouter()
}
