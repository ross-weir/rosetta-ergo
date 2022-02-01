package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	serverutil "github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/pkg/config"
	"github.com/ross-weir/rosetta-ergo/pkg/ergo"
	"github.com/ross-weir/rosetta-ergo/pkg/errutil"
	"github.com/ross-weir/rosetta-ergo/pkg/indexer"
	"github.com/ross-weir/rosetta-ergo/pkg/logger"
	"github.com/ross-weir/rosetta-ergo/pkg/server"
	"github.com/ross-weir/rosetta-ergo/pkg/storage"
	"github.com/ross-weir/rosetta-ergo/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	// readTimeout is the maximum duration for reading the entire
	// request, including the body.
	readTimeout = 5 * time.Second

	// writeTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read.
	writeTimeout = 15 * time.Second

	// idleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled.
	idleTimeout = 30 * time.Second
)

func initRunCmd() *cobra.Command {
	var startNode bool

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run rosetta-ergo",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCmdHandler(startNode)
		},
	}

	runCmd.Flags().BoolVar(&startNode, "start-node", false, "Launch a Ergo node")

	return runCmd
}

func startOnlineDependencies(
	ctx context.Context,
	cancel context.CancelFunc,
	cfg *config.Configuration,
	g *errgroup.Group,
	l *zap.Logger,
) (*ergo.Client, *indexer.Indexer, *storage.Storage, error) {
	client := ergo.NewClient(
		ergo.LocalNodeURL(cfg.NodePort),
		l,
	)

	onlineAsserter, err := asserter.NewClientWithOptions(
		cfg.Network,
		cfg.GenesisBlockIdentifier,
		ergo.OperationTypes,
		ergo.OperationStatuses,
		errutil.Errors,
		nil,
		new(asserter.Validations),
	)
	if err != nil {
		return nil, nil, nil, err
	}

	storage, err := storage.NewStorage(ctx, cfg.IndexerPath, l, onlineAsserter)
	if err != nil {
		return nil, nil, nil, err
	}

	i, err := indexer.InitIndexer(
		cancel,
		cfg,
		client,
		l,
		onlineAsserter,
		storage,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: unable to initialize indexer", err)
	}

	g.Go(func() error {
		return i.Sync(ctx)
	})

	return client, i, storage, nil
}

func runCmdHandler(startNode bool) error {
	zapLogger, err := zap.NewDevelopment()

	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
	}

	defer func() {
		err := zapLogger.Sync()
		if err != nil {
			log.Fatal(err)
		}
	}()

	l := zapLogger.Sugar().Named("main")

	l.Debug("rosetta-ergo starting...")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	go handleSignals([]context.CancelFunc{cancel})

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return util.MonitorMemoryUsage(ctx, -1, zapLogger)
	})

	cfg, err := config.LoadConfiguration()
	if err != nil {
		l.Fatalw("unable to load configuration", "error", err)
	}

	l.Infow("loaded configuration", "configuration", types.PrintStruct(cfg))

	var client *ergo.Client
	var i *indexer.Indexer
	var storage *storage.Storage
	if cfg.Mode == config.Online {
		if startNode {
			g.Go(func() error {
				return ergo.StartErgoNode(ctx, "", zapLogger, g)
			})
		}

		client, i, storage, err = startOnlineDependencies(ctx, cancel, cfg, g, zapLogger)
		if err != nil {
			l.Fatalw("unable to start online dependencies", "error", err)
		}
	}

	l.Info("loaded ergo node client")

	// The asserter automatically rejects incorrectly formatted requests
	asserter, err := asserter.NewServer(
		ergo.OperationTypes,
		true,
		[]*types.NetworkIdentifier{cfg.Network},
		nil,
		false,
		"",
	)

	if err != nil {
		l.Fatalw("unable to create new asserter", "error", err)
	}

	router := server.NewBlockchainRouter(cfg, client, asserter, storage)
	loggedRouter := logger.NewMiddleware(zapLogger, router)
	corsRouter := serverutil.CorsMiddleware(loggedRouter)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.RosettaPort),
		Handler:      corsRouter,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	g.Go(func() error {
		l.Infow("rosetta server listening", "port", cfg.RosettaPort)

		return server.ListenAndServe()
	})

	g.Go(func() error {
		// If we don't shutdown server in errgroup, it will
		// never stop because server.ListenAndServe doesn't
		// take any context.
		<-ctx.Done()

		return server.Shutdown(ctx)
	})

	err = g.Wait()

	// Attempt to close the database gracefully after all indexer goroutines have stopped.
	if i != nil {
		storage.Shutdown(ctx)
	}

	if err != nil {
		l.Errorw("%w: error running rosetta-ergo", err)
	}

	if SignalReceived {
		l.Info("rosetta-ergo halted")

		return nil
	}

	return err
}
