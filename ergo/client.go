package ergo

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	defaultTimeout = 100 * time.Second
	dialTimeout    = 5 * time.Second

	// Using fixed credentials as the nodes REST API should never be exposed on the internet
	nodeToken = "rosetta"
)

type Client struct {
	baseURL string

	genesisBlockIdentifier *types.BlockIdentifier
	currency               *types.Currency

	httpClient *http.Client
}

// Return the URL to use for a client where the node is running locally
func LocalNodeURL(nodePort int) string {
	return fmt.Sprintf("127.0.0.1:%d", nodePort)
}

func NewClient() *Client {
	return &Client{}
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
