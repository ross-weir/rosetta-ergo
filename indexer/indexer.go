package indexer

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/storage/database"
	storageErrs "github.com/coinbase/rosetta-sdk-go/storage/errors"
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
	"golang.org/x/sync/semaphore"
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
	nodeWaitSleep           = 3 * time.Second
	missingTransactionDelay = 200 * time.Millisecond

	// zeroValue is 0 as a string
	zeroValue = "0"

	// semaphoreWeight is the weight of each semaphore request.
	semaphoreWeight = int64(1)
)

var (
	errMissingTransaction = errors.New("missing transaction")
)

var _ syncer.Handler = (*Indexer)(nil)
var _ syncer.Helper = (*Indexer)(nil)

type Indexer struct {
	cancel context.CancelFunc

	network *types.NetworkIdentifier

	client         *ergo.Client
	cfg            *configuration.Configuration
	asserter       *asserter.Asserter
	db             database.Database
	blockStorage   *modules.BlockStorage
	balanceStorage *modules.BalanceStorage
	coinStorage    *modules.CoinStorage
	workers        []modules.BlockWorker

	logger *zap.SugaredLogger

	waiter *waitTable

	// Store coins created in pre-store before persisted
	// in add block so we can optimistically populate
	// blocks before committed.
	coinCache      map[string]*types.AccountCoin
	coinCacheMutex *sdkUtils.PriorityMutex

	// When populating blocks using pre-stored blocks,
	// we should retry if a new block was seen (similar
	// to trying again if head block changes).
	seen      int64
	seenMutex sync.Mutex

	seenSemaphore *semaphore.Weighted
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
		cancel:         cancel,
		network:        cfg.Network,
		client:         client,
		cfg:            cfg,
		asserter:       asserter,
		waiter:         newWaitTable(),
		db:             localDB,
		blockStorage:   blockStorage,
		logger:         logger.Sugar().Named("indexer"),
		coinCache:      map[string]*types.AccountCoin{},
		coinCacheMutex: new(sdkUtils.PriorityMutex),
		seenSemaphore:  semaphore.NewWeighted(int64(runtime.NumCPU())),
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
	if err != nil {
		i.logger.Info("first run, bootstrapping pre-genesis block utxo state")
		err := i.bootstrapGenesisState(ctx)

		if err != nil {
			i.logger.Fatalf("%w: failed to bootstrap utxos, cannot continue", err)

			return err
		}
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
	var inputCoins []*ergo.InputCtx
	var err error

	retries := 0
	for ctx.Err() == nil {
		ergoBlock, inputCoins, err = i.client.GetRawBlock(ctx, blockIdentifier)
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

	// determine which coins must be fetched and get from coin storage
	// coins are needed during block conversion to populate transaction inputs/rosetta operations
	coinMap, err := i.findCoins(ctx, ergoBlock, inputCoins)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to find inputs", err)
	}

	block, err := ergo.ErgoBlockToRosetta(ctx, i.client, ergoBlock, coinMap)
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

	// clean cache intermediate
	// these coins would be added to storage at this point
	i.coinCacheMutex.Lock(true)
	for _, tx := range block.Transactions {
		for _, op := range tx.Operations {
			if op.CoinChange == nil {
				continue
			}

			if op.CoinChange.CoinAction != types.CoinCreated {
				continue
			}

			delete(i.coinCache, op.CoinChange.CoinIdentifier.Identifier)
		}
	}
	i.coinCacheMutex.Unlock()

	// Look for all remaining waiting transactions associated
	// with the next block that have not yet been closed. We should
	// abort these waits as they will never be closed by a new transaction.
	i.waiter.Lock()
	for txHash, val := range i.waiter.table {
		if val.earliestBlock == block.BlockIdentifier.Index+1 && !val.channelClosed {
			i.logger.Debugw(
				"aborting channel",
				"hash", block.BlockIdentifier.Hash,
				"index", block.BlockIdentifier.Index,
				"channel", txHash,
			)
			val.channelClosed = true
			val.aborted = true
			close(val.channel)
		}
	}
	i.waiter.Unlock()

	ops := 0
	for _, transaction := range block.Transactions {
		ops += len(transaction.Operations)
	}

	i.logger.Debugw(
		"block added",
		"hash", block.BlockIdentifier.Hash,
		"index", block.BlockIdentifier.Index,
		"transactions", len(block.Transactions),
		"ops", ops,
	)

	return nil
}

// BlockRemoved is called by the syncer when a block is removed.
func (i *Indexer) BlockRemoved(
	ctx context.Context,
	blockIdentifier *types.BlockIdentifier,
) error {
	i.logger.Debugw(
		"block removed",
		"hash", blockIdentifier.Hash,
		"index", blockIdentifier.Index,
	)
	err := i.blockStorage.RemoveBlock(ctx, blockIdentifier)
	if err != nil {
		return fmt.Errorf(
			"%w: unable to remove block from storage %s:%d",
			err,
			blockIdentifier.Hash,
			blockIdentifier.Index,
		)
	}

	return nil
}

// BlockSeen is called by the syncer when a block is encountered.
// Stores the rosetta `Block` for /block/ API calls.
func (i *Indexer) BlockSeen(ctx context.Context, block *types.Block) error {
	if err := i.seenSemaphore.Acquire(ctx, semaphoreWeight); err != nil {
		return err
	}
	defer i.seenSemaphore.Release(semaphoreWeight)

	// load intermediate
	// utxo cache, eagerly add new output coins to cache
	i.coinCacheMutex.Lock(false)
	for _, tx := range block.Transactions {
		for _, op := range tx.Operations {
			if op.CoinChange == nil {
				continue
			}

			// We only care about newly accessible coins.
			if op.CoinChange.CoinAction != types.CoinCreated {
				continue
			}

			i.coinCache[op.CoinChange.CoinIdentifier.Identifier] = &types.AccountCoin{
				Account: op.Account,
				Coin: &types.Coin{
					CoinIdentifier: op.CoinChange.CoinIdentifier,
					Amount:         op.Amount,
				},
			}
		}
	}
	i.coinCacheMutex.Unlock()

	// Update so that lookers know it exists
	i.seenMutex.Lock()
	i.seen++
	i.seenMutex.Unlock()

	err := i.blockStorage.SeeBlock(ctx, block)
	if err != nil {
		return fmt.Errorf(
			"%w: unable to encounter block to storage %s:%d",
			err,
			block.BlockIdentifier.Hash,
			block.BlockIdentifier.Index,
		)
	}

	// Close channels of all blocks waiting for this particular transaction to be created.
	// For example a future block might be waiting for a output in this transaction
	// to use as an input in its blocks transaction.
	i.waiter.Lock()
	for _, transaction := range block.Transactions {
		txHash := transaction.TransactionIdentifier.Hash
		val, ok := i.waiter.Get(txHash, false)
		if !ok {
			continue
		}

		if val.channelClosed {
			i.logger.Debugw(
				"channel already closed",
				"hash", block.BlockIdentifier.Hash,
				"index", block.BlockIdentifier.Index,
				"channel", txHash,
			)
			continue
		}

		// Closing channel will cause all listeners to continue
		val.channelClosed = true
		close(val.channel)
	}
	i.waiter.Unlock()

	i.logger.Debugw(
		"block seen",
		"hash", block.BlockIdentifier.Hash,
		"index", block.BlockIdentifier.Index,
	)

	return nil
}

func (i *Indexer) checkHeaderMatch(
	ctx context.Context,
	block *ergotype.FullBlock,
) error {
	headBlock, err := i.blockStorage.GetHeadBlockIdentifier(ctx)
	if err != nil && !errors.Is(err, storageErrs.ErrHeadBlockNotFound) {
		return fmt.Errorf("%w: unable to lookup head block", err)
	}

	// If block we are trying to process is next but it is not connected, we
	// should return syncer.ErrOrphanHead to manually trigger a reorg.
	if headBlock != nil &&
		int64(block.Header.Height) == headBlock.Index+1 &&
		block.Header.ParentID != headBlock.Hash {
		return syncer.ErrOrphanHead
	}

	return nil
}

func (i *Indexer) findCoins(
	ctx context.Context,
	block *ergotype.FullBlock,
	coins []*ergo.InputCtx,
) (map[string]*types.AccountCoin, error) {
	if err := i.checkHeaderMatch(ctx, block); err != nil {
		return nil, fmt.Errorf("%w: check header match failed", err)
	}

	coinMap := map[string]*types.AccountCoin{}
	var remainingCoins []*ergo.InputCtx
	for _, inputCtx := range coins {
		coin, owner, err := i.findCoin(
			ctx,
			block,
			inputCtx,
		)
		if err == nil {
			coinMap[inputCtx.InputID] = &types.AccountCoin{
				Account: owner,
				Coin:    coin,
			}
			continue
		}

		if errors.Is(err, errMissingTransaction) {
			remainingCoins = append(remainingCoins, inputCtx)
			continue
		}

		return nil, fmt.Errorf(
			"%w: unable to find coin, input id: %s, tx id: %s",
			err,
			inputCtx.InputID,
			inputCtx.TxID,
		)
	}

	if len(remainingCoins) == 0 {
		return coinMap, nil
	}

	// Wait for remaining transactions
	shouldAbort := false
	for _, coin := range remainingCoins {
		// Wait on Channel
		txHash := coin.TxID
		entry, ok := i.waiter.Get(txHash, true)
		if !ok {
			return nil, fmt.Errorf("transaction %s not in waiter", txHash)
		}

		select {
		case <-entry.channel:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		// Delete Transaction from WaitTable if last listener
		i.waiter.Lock()
		val, ok := i.waiter.Get(txHash, false)
		if !ok {
			return nil, fmt.Errorf("transaction %s not in waiter", txHash)
		}

		// Don't exit right away to make sure
		// we remove all closed entries from the
		// waiter.
		if val.aborted {
			shouldAbort = true
		}

		val.listeners--
		if val.listeners == 0 {
			i.waiter.Delete(txHash, false)
		} else {
			i.waiter.Set(txHash, val, false)
		}
		i.waiter.Unlock()
	}

	// Wait to exit until we have decremented our listeners
	if shouldAbort {
		return nil, syncer.ErrOrphanHead
	}

	// In the case of a reorg, we may still not be able to find
	// the transactions. So, we need to repeat this same process
	// recursively until we find the transactions we are looking for.
	foundCoins, err := i.findCoins(ctx, block, remainingCoins)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to get remaining transactions", err)
	}

	for k, v := range foundCoins {
		coinMap[k] = v
	}

	return coinMap, nil
}

func (i *Indexer) findCoin(
	ctx context.Context,
	block *ergotype.FullBlock,
	inputCtx *ergo.InputCtx,
) (*types.Coin, *types.AccountIdentifier, error) {
	isGenesis := i.client.IsGenesis(block)

	for ctx.Err() == nil {
		startSeen := i.seen
		databaseTransaction := i.db.ReadTransaction(ctx)
		defer databaseTransaction.Discard(ctx)

		coinHeadBlock, err := i.blockStorage.GetHeadBlockIdentifierTransactional(
			ctx,
			databaseTransaction,
		)
		if errors.Is(err, storageErrs.ErrHeadBlockNotFound) && !isGenesis {
			if err := sdkUtils.ContextSleep(ctx, missingTransactionDelay); err != nil {
				return nil, nil, err
			}

			continue
		}
		if err != nil && !isGenesis {
			return nil, nil, fmt.Errorf(
				"%w: unable to get transactional head block identifier",
				err,
			)
		}

		// Attempt to find coin
		coin, owner, err := i.coinStorage.GetCoinTransactional(
			ctx,
			databaseTransaction,
			&types.CoinIdentifier{
				Identifier: inputCtx.InputID,
			},
		)
		if err == nil {
			return coin, owner, nil
		}

		if !errors.Is(err, storageErrs.ErrCoinNotFound) {
			return nil, nil, fmt.Errorf("%w: unable to lookup coin %s", err, inputCtx.InputID)
		}

		// Check seen CoinCache
		i.coinCacheMutex.Lock(false)
		accCoin, ok := i.coinCache[inputCtx.InputID]
		i.coinCacheMutex.Unlock()
		if ok {
			return accCoin.Coin, accCoin.Account, nil
		}

		// Locking here prevents us from adding sending any done
		// signals while we are determining whether or not to add
		// to the WaitTable.
		i.waiter.Lock()

		// Check to see if head block has increased since
		// we created our databaseTransaction.
		currHeadBlock, err := i.blockStorage.GetHeadBlockIdentifier(ctx)
		if err != nil {
			i.waiter.Unlock()
			return nil, nil, fmt.Errorf("%w: unable to get head block identifier", err)
		}

		// If the block has changed, we try to look up the transaction
		// again.
		if types.Hash(currHeadBlock) != types.Hash(coinHeadBlock) || i.seen != startSeen {
			i.waiter.Unlock()
			continue
		}

		// Put Transaction in WaitTable if doesn't already exist (could be
		// multiple listeners)
		transactionHash := inputCtx.TxID
		val, ok := i.waiter.Get(transactionHash, false)
		if !ok {
			val = &waitTableEntry{
				channel:       make(chan struct{}),
				earliestBlock: int64(block.Header.Height),
			}
		}
		if val.earliestBlock > int64(block.Header.Height) {
			val.earliestBlock = int64(block.Header.Height)
		}
		val.listeners++
		i.waiter.Set(transactionHash, val, false)
		i.waiter.Unlock()

		return nil, nil, errMissingTransaction
	}

	return nil, nil, ctx.Err()
}

// bootstrapGenesisState loads pre-genesis block utxos into the db
func (i *Indexer) bootstrapGenesisState(ctx context.Context) error {
	err := i.balanceStorage.BootstrapBalances(
		ctx,
		i.cfg.BootstrapBalancePath,
		i.cfg.GenesisBlockIdentifier,
	)
	if err != nil {
		return err
	}

	var accountCoins []types.AccountCoin

	err = sdkUtils.LoadAndParse(i.cfg.GenesisUtxoPath, &accountCoins)
	if err != nil {
		return err
	}

	importedCoins := make([]*sdkUtils.AccountBalance, len(accountCoins))

	for i, coin := range accountCoins {
		importedCoins[i] = &sdkUtils.AccountBalance{
			Account: coin.Account,
			Coins:   []*types.Coin{coin.Coin},
		}
	}

	err = i.coinStorage.SetCoinsImported(ctx, importedCoins)
	if err != nil {
		return err
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

// GetCoins returns all unspent coins for a particular *types.AccountIdentifier.
func (i *Indexer) GetCoins(
	ctx context.Context,
	accountIdentifier *types.AccountIdentifier,
) ([]*types.Coin, *types.BlockIdentifier, error) {
	return i.coinStorage.GetCoins(ctx, accountIdentifier)
}

// GetBalance returns the balance of an account
// at a particular *types.PartialBlockIdentifier.
func (i *Indexer) GetBalance(
	ctx context.Context,
	accountIdentifier *types.AccountIdentifier,
	currency *types.Currency,
	blockIdentifier *types.PartialBlockIdentifier,
) (*types.Amount, *types.BlockIdentifier, error) {
	dbTx := i.db.ReadTransaction(ctx)
	defer dbTx.Discard(ctx)

	blockResponse, err := i.blockStorage.GetBlockLazyTransactional(
		ctx,
		blockIdentifier,
		dbTx,
	)
	if err != nil {
		return nil, nil, err
	}

	amount, err := i.balanceStorage.GetBalanceTransactional(
		ctx,
		dbTx,
		accountIdentifier,
		currency,
		blockResponse.Block.BlockIdentifier.Index,
	)
	if errors.Is(err, storageErrs.ErrAccountMissing) {
		return &types.Amount{
			Value:    zeroValue,
			Currency: currency,
		}, blockResponse.Block.BlockIdentifier, nil
	}
	if err != nil {
		return nil, nil, err
	}

	return amount, blockResponse.Block.BlockIdentifier, nil
}
