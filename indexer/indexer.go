package indexer

import (
	"context"
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/storage/database"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/configuration"
	"github.com/ross-weir/rosetta-ergo/ergo"
	"github.com/ross-weir/rosetta-ergo/services"
	"go.uber.org/zap"
)

type Indexer struct {
	cancel context.CancelFunc

	network *types.NetworkIdentifier

	client   *ergo.Client
	asserter *asserter.Asserter
	db       database.Database
	logger   *zap.SugaredLogger
}

func InitIndexer(
	ctx context.Context,
	cancel context.CancelFunc,
	cfg *configuration.Configuration,
	client *ergo.Client,
	logger *zap.Logger,
) (*Indexer, error) {
	localDb, err := database.NewBadgerDatabase(ctx, cfg.IndexerPath)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to initialize storage", err)
	}

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
		cancel:   cancel,
		network:  cfg.Network,
		client:   client,
		asserter: asserter,
		db:       localDb,
		logger:   logger.Sugar().Named("indexer"),
	}

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
