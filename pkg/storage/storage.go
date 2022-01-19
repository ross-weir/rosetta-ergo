package storage

import (
	"context"
	"runtime"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/storage/database"
	"github.com/coinbase/rosetta-sdk-go/storage/modules"
	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	"go.uber.org/zap"
)

type Storage struct {
	logger  *zap.SugaredLogger
	db      database.Database
	block   *modules.BlockStorage
	balance *modules.BalanceStorage
	coin    *modules.CoinStorage
	workers []modules.BlockWorker
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

func NewStorage(
	ctx context.Context,
	dbPath string,
	log *zap.Logger,
	asserter *asserter.Asserter,
) (*Storage, error) {
	logger := log.Sugar().Named("storage")

	logger.Debug("initializing database")

	db, err := database.NewBadgerDatabase(
		ctx,
		dbPath,
		database.WithCustomSettings(defaultBadgerOptions(dbPath)),
	)
	if err != nil {
		return nil, err
	}

	logger.Debug("initializing storage modules")

	blockStorage := modules.NewBlockStorage(db, runtime.NumCPU())
	coinStorage := modules.NewCoinStorage(
		db,
		&CoinStorageHelper{blockStorage},
		asserter,
	)

	balanceStorage := modules.NewBalanceStorage(db)
	balanceStorage.Initialize(
		&BalanceStorageHelper{asserter},
		&BalanceStorageHandler{},
	)

	s := &Storage{
		logger:  logger,
		db:      db,
		block:   blockStorage,
		balance: balanceStorage,
		coin:    coinStorage,
		workers: []modules.BlockWorker{coinStorage, balanceStorage},
	}

	return s, nil
}

func (s *Storage) Initialize() {
	s.logger.Debug("starting block workers")

	s.block.Initialize(s.workers)
}

func (s *Storage) Shutdown(ctx context.Context) {
	err := s.db.Close(ctx)
	if err != nil {
		s.logger.Fatalw("unable to close indexer database", "error", err)
	}

	s.logger.Info("database closed successfully")
}

func (s *Storage) Block() *modules.BlockStorage {
	return s.block
}

func (s *Storage) Balance() *modules.BalanceStorage {
	return s.balance
}

func (s *Storage) Coin() *modules.CoinStorage {
	return s.coin
}

func (s *Storage) Db() database.Database {
	return s.db
}
