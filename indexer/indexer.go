package indexer

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/storage/database"
	"github.com/coinbase/rosetta-sdk-go/storage/modules"
	"github.com/coinbase/rosetta-sdk-go/syncer"
	"github.com/coinbase/rosetta-sdk-go/types"
	sdkUtils "github.com/coinbase/rosetta-sdk-go/utils"
	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	"github.com/ross-weir/rosetta-ergo/configuration"
	"github.com/ross-weir/rosetta-ergo/ergo"
	ergotype "github.com/ross-weir/rosetta-ergo/ergo/types"
	"github.com/ross-weir/rosetta-ergo/services"
	"go.uber.org/zap"
)

const (
	// initialStartIndex is provided to the syncer
	// to indicate where we should start syncing from.
	// This can be updated based on the previous
	// sync'd block in storage
	initialStartIndex = -1

	// Index to stop syncing at
	// -1 means to sync forever until app is stopped
	endIndex = -1

	retryDelay = 10 * time.Second
	retryLimit = 5

	// Sleep time between checking node status
	nodeWaitSleep = 3 * time.Second

	// zeroValue is 0 as a string
	zeroValue = "0"
)

var _ syncer.Handler = (*Indexer)(nil)
var _ syncer.Helper = (*Indexer)(nil)

type Indexer struct {
	cancel context.CancelFunc

	network *types.NetworkIdentifier

	client         *ergo.Client
	asserter       *asserter.Asserter
	db             database.Database
	blockStorage   *modules.BlockStorage
	balanceStorage *modules.BalanceStorage
	coinStorage    *modules.CoinStorage
	workers        []modules.BlockWorker

	logger *zap.SugaredLogger
}

// defaultBadgerOptions returns a set of badger.Options optimized
// for running a Rosetta implementation.
func defaultBadgerOptions(
	dir string,
) badger.Options {
	opts := badger.DefaultOptions(dir)

	// By default, we do not compress the table at all. Doing so can
	// significantly increase memory usage.
	opts.Compression = options.None

	return opts
}

func InitIndexer(
	ctx context.Context,
	cancel context.CancelFunc,
	cfg *configuration.Configuration,
	client *ergo.Client,
	logger *zap.Logger,
) (*Indexer, error) {
	localDB, err := database.NewBadgerDatabase(
		ctx,
		cfg.IndexerPath,
		database.WithCustomSettings(defaultBadgerOptions((cfg.IndexerPath))),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to initialize storage", err)
	}

	blockStorage := modules.NewBlockStorage(localDB, runtime.NumCPU())

	asserter, err := asserter.NewClientWithOptions(
		cfg.Network,
		cfg.GenesisBlockIdentifier,
		ergo.OperationTypes,
		ergo.OperationStatuses,
		services.Errors,
		nil,
		new(asserter.Validations),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to initialize asserter", err)
	}

	i := &Indexer{
		cancel:       cancel,
		network:      cfg.Network,
		client:       client,
		asserter:     asserter,
		db:           localDB,
		blockStorage: blockStorage,
		logger:       logger.Sugar().Named("indexer"),
	}

	coinStorage := modules.NewCoinStorage(
		localDB,
		&CoinStorageHelper{blockStorage},
		asserter,
	)
	i.coinStorage = coinStorage

	balanceStorage := modules.NewBalanceStorage(localDB)
	balanceStorage.Initialize(
		&BalanceStorageHelper{asserter},
		&BalanceStorageHandler{},
	)
	i.balanceStorage = balanceStorage

	i.workers = []modules.BlockWorker{coinStorage, balanceStorage}

	i.logger.Info("indexer initialized")

	return i, nil
}

// CloseDatabase closes a storage.Database. This should be called
// before exiting.
func (i *Indexer) CloseDatabase(ctx context.Context) {
	err := i.db.Close(ctx)
	if err != nil {
		i.logger.Fatalw("unable to close indexer database", "error", err)
	}

	i.logger.Infow("database closed successfully")
}

// waitForNode returns once ergo node is ready to serve
// block queries.
func (i *Indexer) waitForNode(ctx context.Context) error {
	for {
		_, err := i.client.NetworkStatus(ctx)
		if err == nil {
			return nil
		}

		i.logger.Infow("waiting for ergo node...")
		if err := sdkUtils.ContextSleep(ctx, nodeWaitSleep); err != nil {
			return err
		}
	}
}

func (i *Indexer) Sync(ctx context.Context) error {
	if err := i.waitForNode(ctx); err != nil {
		return fmt.Errorf("%w: failed to wait for node", err)
	}

	i.logger.Info("ergo node running, initializing storage")

	i.blockStorage.Initialize(i.workers)

	startIndex := int64(initialStartIndex)
	head, err := i.blockStorage.GetHeadBlockIdentifier(ctx)
	if err == nil {
		startIndex = head.Index + 1
	}

	i.logger.Infof("start syncing from index %d", startIndex)

	// Load in previous blocks into syncer cache to handle reorgs.
	pastBlocks := i.blockStorage.CreateBlockCache(ctx, syncer.DefaultPastBlockLimit)

	syncer := syncer.New(
		i.network,
		i,
		i,
		i.cancel,
		syncer.WithPastBlocks(pastBlocks),
	)

	return syncer.Sync(ctx, startIndex, endIndex)
}

// Block is called by the syncer to fetch a block.
func (i *Indexer) Block(
	ctx context.Context,
	network *types.NetworkIdentifier,
	blockIdentifier *types.PartialBlockIdentifier,
) (*types.Block, error) {
	var ergoBlock *ergotype.FullBlock
	var err error

	retries := 0
	for ctx.Err() == nil {
		ergoBlock, err = i.client.GetRawBlock(ctx, blockIdentifier)
		if err == nil {
			break
		}

		retries++
		if retries > retryLimit {
			return nil, fmt.Errorf("%w: unable to get raw block %+v", err, blockIdentifier)
		}

		if err := sdkUtils.ContextSleep(ctx, retryDelay); err != nil {
			return nil, err
		}
	}

	block, err := ergo.ErgoBlockToRosetta(ergoBlock)
	if err != nil {
		return nil, err
	}

	if err := i.asserter.Block(block); err != nil {
		return nil, fmt.Errorf("%w: block is not valid %+v", err, blockIdentifier)
	}

	return block, nil
}

// BlockAdded is called by the syncer when a block is added.
// Stores the block Hash/Index only. `BlockSeen` adds the actual block to storage.
func (i *Indexer) BlockAdded(ctx context.Context, block *types.Block) error {
	err := i.blockStorage.AddBlock(ctx, block)
	if err != nil {
		return fmt.Errorf(
			"%w: unable to add block to storage %s:%d",
			err,
			block.BlockIdentifier.Hash,
			block.BlockIdentifier.Index,
		)
	}

	return nil
}

// BlockRemoved is called by the syncer when a block is removed.
func (i *Indexer) BlockRemoved(
	ctx context.Context,
	blockIdentifier *types.BlockIdentifier,
) error {
	return nil
}

// BlockSeen is called by the syncer when a block is encountered.
// Stores the rosetta `Block` for /block/ API calls.
func (i *Indexer) BlockSeen(ctx context.Context, block *types.Block) error {
	err := i.blockStorage.SeeBlock(ctx, block)
	if err != nil {
		return fmt.Errorf(
			"%w: unable to encounter block to storage %s:%d",
			err,
			block.BlockIdentifier.Hash,
			block.BlockIdentifier.Index,
		)
	}

	return nil
}

// NetworkStatus is called by the syncer to get the current
// network status.
func (i *Indexer) NetworkStatus(
	ctx context.Context,
	network *types.NetworkIdentifier,
) (*types.NetworkStatusResponse, error) {
	return i.client.NetworkStatus(ctx)
}

// GetBlockLazy returns a *types.BlockResponse from the indexer's block storage.
// All transactions in a block must be fetched individually.
func (i *Indexer) GetBlockLazy(
	ctx context.Context,
	blockIdentifier *types.PartialBlockIdentifier,
) (*types.BlockResponse, error) {
	return i.blockStorage.GetBlockLazy(ctx, blockIdentifier)
}

// GetBlockTransaction returns a *types.Transaction if it is in the provided
// *types.BlockIdentifier.
func (i *Indexer) GetBlockTransaction(
	ctx context.Context,
	blockIdentifier *types.BlockIdentifier,
	transactionIdentifier *types.TransactionIdentifier,
) (*types.Transaction, error) {
	return i.blockStorage.GetBlockTransaction(
		ctx,
		blockIdentifier,
		transactionIdentifier,
	)
}

func (i *Indexer) GetHeadBlockIdentifier(ctx context.Context) (*types.BlockIdentifier, error) {
	return i.blockStorage.GetHeadBlockIdentifier(ctx)
}

func (i *Indexer) GetBlock(
	ctx context.Context,
	id *types.PartialBlockIdentifier,
) (*types.Block, error) {
	return i.blockStorage.GetBlock(ctx, id)
}
