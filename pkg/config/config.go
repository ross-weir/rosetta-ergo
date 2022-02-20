package config

import (
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/pkg/ergo"
)

// Version is populated at build time using ldflags
var Version = "UNKNOWN"

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

	// Appended to the data directory supplied by RosettaDataDirEnv
	indexerPath = "indexer"
	// Appended to the data directory, contains utxos that existed before genesis block
	genesisUtxoPath = "genesis_utxos.json"
	// Path to balances to bootstrap
	bootstrapBalancePath = "bootstrap_balances.json"

	// allFilePermissions specifies anyone can do anything
	// to the file.
	allFilePermissions = 0777

	nodeConfigPath = "node.conf"

	// The following env vars will be prefixed with this value
	EnvVarPrefix = "ERGO_"

	// The mode we're operating in for the rosetta API
	// Online vs Offline mode, see more:
	// https://www.rosetta-api.org/docs/node_deployment.html#multiple-modes
	RosettaModeEnv = "ROSETTA_MODE"

	// The directory used for rosetta data, this should almost always be /data, unless testing
	// locally
	// See more: https://docs.cloud.coinbase.com/rosetta/docs/standard-storage-location
	RosettaDataDirEnv = "ROSETTA_DATA_DIR"

	// The network we're operating in for the ergo blockchain
	// mainnet vs testnet
	NetworkEnv = "NETWORK"

	// The port the rosetta API is listening on
	RosettaPortEnv = "ROSETTA_PORT"

	// The directory containing configuration files
	ConfigEnv = "CONFIG_DIR"
)

type Configuration struct {
	Mode                   Mode
	Version                string
	Network                *types.NetworkIdentifier
	Currency               *types.Currency
	GenesisBlockIdentifier *types.BlockIdentifier
	RosettaPort            int
	NodePort               int
	IndexerPath            string
	GenesisUtxoPath        string
	BootstrapBalancePath   string
	NodeConfigPath         string
}

// Create a new configuration based on settings above and env variables
func LoadConfiguration() (*Configuration, error) {
	cfg := &Configuration{}
	baseDirectory := getEnv(RosettaDataDirEnv)
	configDirectory := getEnv(ConfigEnv)
	networkValue := getEnv(NetworkEnv)

	cfg.NodeConfigPath = path.Join(configDirectory, nodeConfigPath)

	modeValue := Mode(getEnv(RosettaModeEnv))
	switch modeValue {
	case Online:
		cfg.Mode = Online
		cfg.IndexerPath = path.Join(baseDirectory, indexerPath)
		if err := ensurePathExists(cfg.IndexerPath); err != nil {
			return nil, fmt.Errorf("%w: unable to create indexer path", err)
		}

		cfg.GenesisUtxoPath = path.Join(configDirectory, networkValue, genesisUtxoPath)
		cfg.BootstrapBalancePath = path.Join(configDirectory, networkValue, bootstrapBalancePath)
	case Offline:
		cfg.Mode = Offline
	case "":
		return nil, fmt.Errorf("%s%s must be populated", EnvVarPrefix, RosettaModeEnv)
	default:
		return nil, fmt.Errorf("%s is not a valid mode", modeValue)
	}

	cfg.Currency = ergo.Currency
	cfg.Version = Version
	switch networkValue {
	case Mainnet:
		cfg.Network = &types.NetworkIdentifier{
			Blockchain: ergo.Blockchain,
			Network:    ergo.MainnetNetwork,
		}
		cfg.GenesisBlockIdentifier = ergo.MainnetGenesisBlockIdentifier
		cfg.NodePort = mainnetNodePort
	case Testnet:
		cfg.Network = &types.NetworkIdentifier{
			Blockchain: ergo.Blockchain,
			Network:    ergo.TestnetNetwork,
		}
		cfg.GenesisBlockIdentifier = ergo.TestnetGenesisBlockIdentifier
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

// ensurePathsExist directories along
// a path if they do not exist.
func ensurePathExists(path string) error {
	if err := os.MkdirAll(path, os.FileMode(allFilePermissions)); err != nil {
		return fmt.Errorf("%w: unable to create %s directory", err, path)
	}

	return nil
}
