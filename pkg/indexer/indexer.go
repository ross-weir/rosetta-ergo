package indexer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	storageErrs "github.com/coinbase/rosetta-sdk-go/storage/errors"
	"github.com/coinbase/rosetta-sdk-go/syncer"
	"github.com/coinbase/rosetta-sdk-go/types"
	sdkUtils "github.com/coinbase/rosetta-sdk-go/utils"
	"github.com/ross-weir/rosetta-ergo/pkg/config"
	"github.com/ross-weir/rosetta-ergo/pkg/ergo"
	ergotype "github.com/ross-weir/rosetta-ergo/pkg/ergo/types"
	"github.com/ross-weir/rosetta-ergo/pkg/rosetta"
	"github.com/ross-weir/rosetta-ergo/pkg/storage"
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
	nodeWaitSleep           = 6 * time.Second
	missingTransactionDelay = 200 * time.Millisecond

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

	client   *ergo.Client
	cfg      *config.Configuration
	asserter *asserter.Asserter
	storage  *storage.Storage
	logger   *zap.SugaredLogger

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

func InitIndexer(
	cancel context.CancelFunc,
	cfg *config.Configuration,
	client *ergo.Client,
	logger *zap.Logger,
	asserter *asserter.Asserter,
	storage *storage.Storage,
) (*Indexer, error) {
	i := &Indexer{
		cancel:         cancel,
		client:         client,
		cfg:            cfg,
		asserter:       asserter,
		storage:        storage,
		waiter:         newWaitTable(),
		logger:         logger.Sugar().Named("indexer"),
		coinCache:      map[string]*types.AccountCoin{},
		coinCacheMutex: new(sdkUtils.PriorityMutex),
		seenSemaphore:  semaphore.NewWeighted(int64(runtime.NumCPU())),
	}

	return i, nil
}

// waitForNode returns once ergo node is ready to serve
// block queries.
func (i *Indexer) waitForNode(ctx context.Context) error {
	for {
		// Wait until the node can start returning blocks by id.
		// This seems to happen after the node has fully sync'd.
		_, err := i.client.GetBlockByID(ctx, &i.cfg.GenesisBlockIdentifier.Hash)
		if err == nil {
			return nil
		}

		i.logger.Infow("waiting for ergo node...", "err", err)
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

	i.storage.Initialize()
	blockStorage := i.storage.Block()

	startIndex := int64(initialStartIndex)
	head, err := blockStorage.GetHeadBlockIdentifier(ctx)
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
	pastBlocks := blockStorage.CreateBlockCache(ctx, syncer.DefaultPastBlockLimit)

	syncer := syncer.New(
		i.cfg.Network,
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
	var requiredInputs []*ergo.InputCtx
	var err error

	retries := 0
	for ctx.Err() == nil {
		// replace with fetcher `Block()`?
		// We would then need to get requiredInputs from `rosetta.Transaction` inputs instead
		// I don't think we can replace with fetcher because fetcher will use the
		// rosetta API to request a block, that block wouldn't exist yet if
		// we're processing it here in the indexer?
		// ergoBlock, requiredInputs, err = i.client.GetRawBlock(ctx, blockIdentifier)
		ergoBlock, err = i.getBlockByIdentifier(ctx, blockIdentifier)
		if err == nil {
			requiredInputs = ergo.GetInputsForTxs(ergoBlock.BlockTransactions.Transactions)

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
	coinMap, err := i.findCoins(ctx, ergoBlock, requiredInputs)
	if err != nil {
		return nil, fmt.Errorf("findCoins failed: %w", err)
	}

	converter := rosetta.NewBlockConverter(i.client, coinMap)
	block, err := converter.BlockToRosettaBlock(ctx, ergoBlock)

	if err != nil {
		return nil, err
	}

	if err := i.asserter.Block(block); err != nil {
		return nil, fmt.Errorf("%w: block is not valid %+v", err, blockIdentifier)
	}

	return block, nil
}

func (i *Indexer) getBlockByIdentifier(
	ctx context.Context,
	blockID *types.PartialBlockIdentifier,
) (*ergotype.FullBlock, error) {
	var block *ergotype.FullBlock
	var err error

	if blockID.Hash != nil {
		block, err = i.client.GetBlockByID(ctx, blockID.Hash)
		if err != nil {
			return nil, err
		}
	}

	if blockID.Index != nil {
		block, err = i.client.GetBlockByIndex(ctx, blockID.Index)
		if err != nil {
			return nil, err
		}
	}

	return block, err
}

// BlockAdded is called by the syncer when a block is added.
// Stores the block Hash/Index only. `BlockSeen` adds the actual block to storage.
func (i *Indexer) BlockAdded(ctx context.Context, block *types.Block) error {
	blockStorage := i.storage.Block()
	err := i.ensureNegativeValueInputs(block)
	if err != nil {
		return fmt.Errorf("failed to ensure negative value inputs: %w", err)
	}

	err = blockStorage.AddBlock(ctx, block)
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
	err := i.storage.Block().RemoveBlock(ctx, blockIdentifier)
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

	err := i.storage.Block().SeeBlock(ctx, block)
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

// Check if the block is orphaned
func (i *Indexer) checkHeaderMatch(
	ctx context.Context,
	block *ergotype.FullBlock,
) error {
	headBlock, err := i.storage.Block().GetHeadBlockIdentifier(ctx)
	if err != nil && !errors.Is(err, storageErrs.ErrHeadBlockNotFound) {
		return fmt.Errorf("%w: unable to lookup head block", err)
	}

	// If block we are trying to process is next but it is not connected, we
	// should return syncer.ErrOrphanHead to manually trigger a reorg.
	if headBlock != nil && block != nil &&
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
	isGenesis := i.isGenesis(block)
	blockStorage := i.storage.Block()

	for ctx.Err() == nil {
		startSeen := i.seen
		databaseTransaction := i.storage.DB().ReadTransaction(ctx)
		defer databaseTransaction.Discard(ctx)

		coinHeadBlock, err := blockStorage.GetHeadBlockIdentifierTransactional(
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
		coin, owner, err := i.storage.Coin().GetCoinTransactional(
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
		currHeadBlock, err := blockStorage.GetHeadBlockIdentifier(ctx)
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

func (i *Indexer) ensureNegativeValueInputs(block *types.Block) error {
	for _, tx := range block.Transactions {
		for _, op := range tx.Operations {
			if op.CoinChange.CoinAction != types.CoinSpent {
				continue
			}

			bigVal, ok := new(big.Int).SetString(op.Amount.Value, 10)
			if !ok {
				return fmt.Errorf("%s is not an integer", bigVal)
			}

			if bigVal.Sign() == 1 {
				negatedVal, err := types.NegateValue(op.Amount.Value)
				if err != nil {
					return fmt.Errorf("%w: unable to negate input", err)
				}
				op.Amount.Value = negatedVal
			}
		}
	}

	return nil
}

// bootstrapGenesisState loads pre-genesis block utxos into the db
func (i *Indexer) bootstrapGenesisState(ctx context.Context) error {
	err := i.storage.Balance().BootstrapBalances(
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

	err = i.storage.Coin().SetCoinsImported(ctx, importedCoins)
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
	connectedPeers, err := i.client.GetConnectedPeers(ctx)
	if err != nil {
		return nil, err
	}

	peers := make([]*types.Peer, len(connectedPeers))
	for i := range connectedPeers {
		peer, err := rosetta.PeerToRosettaPeer(&connectedPeers[i])
		if err != nil {
			return nil, err
		}

		peers[i] = peer
	}

	currentHeaders, err := i.client.GetLatestBlockHeaders(ctx, 1)
	if err != nil {
		return nil, err
	}
	if len(currentHeaders) == 0 {
		return nil, errors.New("failed to get latest block header from node")
	}

	block := rosetta.HeaderToRosettaBlock(&currentHeaders[0])

	return &types.NetworkStatusResponse{
		CurrentBlockIdentifier: block.BlockIdentifier,
		CurrentBlockTimestamp:  block.Timestamp,
		GenesisBlockIdentifier: i.cfg.GenesisBlockIdentifier,
		Peers:                  peers,
	}, nil
}

// isGenesis checks if the provided block is the genesis block
func (i *Indexer) isGenesis(b *ergotype.FullBlock) bool {
	return i.cfg.GenesisBlockIdentifier.Hash == b.Header.ID
}
