package configuration

import (
	"fmt"
	"os"
	"strconv"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/ergo"
)

// Version is populated at build time using ldflags
var Version string = "UNKNOWN"

// Mode is the setting that determines if
// the implementation is "online" or "offline".
type Mode string

const (
	Online  Mode = "ONLINE"
	Offline Mode = "OFFLINE"

	Mainnet string = "mainnet"
	Testnet string = "testnet"

	mainnetNodePort = 9053
	testnetNodePort = 9052

	DataDir = "/data"

	// TODO: node configuration
	// TODO: node config file path, etc

	// The following env vars will be prefixed with this value
	EnvVarPrefix = "ERGO_"

	// The mode we're operating in for the rosetta API
	// Online vs Offline mode, see more:
	// https://www.rosetta-api.org/docs/node_deployment.html#multiple-modes
	RosettaModeEnv = "ROSETTA_MODE"

	// The network we're operating in for the ergo blockchain
	// mainnet vs testnet
	NetworkEnv = "NETWORK"

	// The port the rosetta API is listening on
	RosettaPortEnv = "ROSETTA_PORT"
)

type Configuration struct {
	Mode                   Mode
	Version                string
	Network                *types.NetworkIdentifier
	Currency               *types.Currency
	GenesisBlockIdentifier *types.BlockIdentifier
	RosettaPort            int
	NodePort               int
}

// Create a new configuration based on settings above and env variables
func LoadConfiguration(baseDirectory string) (*Configuration, error) {
	cfg := &Configuration{}

	modeValue := Mode(getEnv(RosettaModeEnv))
	switch modeValue {
	case Online:
		cfg.Mode = Online
	case Offline:
		cfg.Mode = Offline
	case "":
		return nil, fmt.Errorf("%s%s must be populated", EnvVarPrefix, RosettaModeEnv)
	default:
		return nil, fmt.Errorf("%s is not a valid mode", modeValue)
	}

	cfg.Currency = ergo.Currency
	cfg.Version = Version

	// network specific configuration
	networkValue := getEnv(NetworkEnv)
	switch networkValue {
	case Mainnet:
		cfg.Network = &types.NetworkIdentifier{
			Blockchain: ergo.Blockchain,
			Network:    ergo.MainnetNetwork,
		}
		cfg.GenesisBlockIdentifier = ergo.MainnetGenesisBlockIdentifier
		// config.Params = bitcoin.MainnetParams
		// config.ConfigPath = mainnetConfigPath
		cfg.NodePort = mainnetNodePort
	case Testnet:
		cfg.Network = &types.NetworkIdentifier{
			Blockchain: ergo.Blockchain,
			Network:    ergo.TestnetNetwork,
		}
		cfg.GenesisBlockIdentifier = ergo.TestnetGenesisBlockIdentifier
		// config.Params = bitcoin.TestnetParams
		// config.ConfigPath = testnetConfigPath
		cfg.NodePort = testnetNodePort
	case "":
		return nil, fmt.Errorf("%s%s must be populated", EnvVarPrefix, NetworkEnv)
	default:
		return nil, fmt.Errorf("%s is not a valid network", networkValue)
	}

	rosettaPortStr := getEnv(RosettaPortEnv)
	if len(rosettaPortStr) == 0 {
		return nil, fmt.Errorf("%s%s must be populated", EnvVarPrefix, RosettaPortEnv)
	}

	rosettaPort, err := strconv.Atoi(rosettaPortStr)
	if err != nil || len(rosettaPortStr) == 0 || rosettaPort <= 0 {
		return nil, fmt.Errorf("%w: unable to parse port %s", err, rosettaPortStr)
	}

	cfg.RosettaPort = rosettaPort

	return cfg, nil
}

func getEnv(key string) string {
	return os.Getenv(EnvVarPrefix + key)
}
