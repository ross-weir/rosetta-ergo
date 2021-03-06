package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/coinbase/rosetta-sdk-go/storage/modules"

	_ "embed"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/pkg/ergo"
)

//go:embed mainnet/bootstrap_balances.json
var mainnetBootstrapBalanceBytes []byte

//go:embed mainnet/genesis_utxos.json
var mainnetGenesisUtxosBytes []byte

//go:embed testnet/bootstrap_balances.json
var testnetBootstrapBalanceBytes []byte

//go:embed testnet/genesis_utxos.json
var testnetGenesisUtxosBytes []byte

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

	// Appended to the data directory supplied by RosettaDataDirEnv
	indexerPath = "indexer"

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

	// The port that the blockchain node is listening on
	// Not currently prefixed with EnvVarPrefix because I want to change that from ERGO_ -> ROSETTA_.
	NodePortEnv = "ROSETTA_NODE_PORT"
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
	GenesisUtxos           []types.AccountCoin
	BootstrapBalances      []*modules.BootstrapBalance
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
	case Offline:
		cfg.Mode = Offline
	case "":
		return nil, fmt.Errorf("%s%s must be populated", EnvVarPrefix, RosettaModeEnv)
	default:
		return nil, fmt.Errorf("%s is not a valid mode", modeValue)
	}

	nodePortStr := os.Getenv(NodePortEnv)
	nodePort, err := parsePort(nodePortStr)
	if err != nil {
		return nil, err
	}

	var genesisUtxos []types.AccountCoin
	var bootstrapBalances []*modules.BootstrapBalance

	cfg.NodePort = *nodePort
	cfg.Currency = ergo.Currency
	cfg.Version = Version
	switch networkValue {
	case Mainnet:
		cfg.Network = &types.NetworkIdentifier{
			Blockchain: ergo.Blockchain,
			Network:    ergo.MainnetNetwork,
		}
		err := parseInterface(mainnetGenesisUtxosBytes, &genesisUtxos)

		if err != nil {
			return nil, err
		}

		err = parseInterface(mainnetBootstrapBalanceBytes, &bootstrapBalances)
		if err != nil {
			return nil, err
		}

		cfg.GenesisUtxos = genesisUtxos
		cfg.BootstrapBalances = bootstrapBalances
		cfg.GenesisBlockIdentifier = ergo.MainnetGenesisBlockIdentifier
	case Testnet:
		cfg.Network = &types.NetworkIdentifier{
			Blockchain: ergo.Blockchain,
			Network:    ergo.TestnetNetwork,
		}
		err := parseInterface(testnetGenesisUtxosBytes, &genesisUtxos)

		if err != nil {
			return nil, err
		}

		err = parseInterface(testnetBootstrapBalanceBytes, &bootstrapBalances)
		if err != nil {
			return nil, err
		}

		cfg.GenesisUtxos = genesisUtxos
		cfg.BootstrapBalances = bootstrapBalances
		cfg.GenesisBlockIdentifier = ergo.TestnetGenesisBlockIdentifier
	case "":
		return nil, fmt.Errorf("%s%s must be populated", EnvVarPrefix, NetworkEnv)
	default:
		return nil, fmt.Errorf("%s is not a valid network", networkValue)
	}

	rosettaPortStr := getEnv(RosettaPortEnv)
	if len(rosettaPortStr) == 0 {
		return nil, fmt.Errorf("%s%s must be populated", EnvVarPrefix, RosettaPortEnv)
	}

	rosettaPort, err := parsePort(rosettaPortStr)
	if err != nil {
		return nil, err
	}

	cfg.RosettaPort = *rosettaPort

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

func parsePort(port string) (*int, error) {
	p, err := strconv.Atoi(port)

	if err != nil || len(port) == 0 || p <= 0 {
		return nil, fmt.Errorf("unable to parse port %s: %w", port, err)
	}

	return &p, nil
}

func parseInterface(b []byte, output interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&output); err != nil {
		return fmt.Errorf("%w: unable to unmarshal", err)
	}

	return nil
}
