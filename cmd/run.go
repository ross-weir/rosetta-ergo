package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ross-weir/rosetta-ergo/configuration"
	"github.com/ross-weir/rosetta-ergo/ergo"
	"github.com/ross-weir/rosetta-ergo/indexer"
	"github.com/ross-weir/rosetta-ergo/services"
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

var (
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run rosetta-ergo",
		RunE:  runCmdHandler,
	}
)

func startOnlineDependencies(
	ctx context.Context,
	cancel context.CancelFunc,
	cfg *configuration.Configuration,
	g *errgroup.Group,
	l *zap.Logger,
) (*ergo.Client, *indexer.Indexer, error) {
	client := ergo.NewClient(
		ergo.LocalNodeURL(cfg.NodePort),
		cfg.GenesisBlockIdentifier,
		cfg.Currency,
		l,
	)

	i, err := indexer.InitIndexer(
		ctx,
		cancel,
		cfg,
		client,
		l,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: unable to initialize indexer", err)
	}

	return client, i, nil
}

func runCmdHandler(cmd *cobra.Command, args []string) error {
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

	logger := zapLogger.Sugar().Named("main")

	logger.Debug("rosetta-ergo starting...")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	go handleSignals([]context.CancelFunc{cancel})

	g, ctx := errgroup.WithContext(ctx)

	cfg, err := configuration.LoadConfiguration()
	if err != nil {
		logger.Fatalw("unable to load configuration", "error", err)
	}

	logger.Infow("loaded configuration", "configuration", types.PrintStruct(cfg))

	// The asserter automatically rejects incorrectly formatted requests
	asserter, err := asserter.NewServer(
		ergo.OperationTypes,
		false,
		[]*types.NetworkIdentifier{cfg.Network},
		nil,
		false,
		"",
	)

	if err != nil {
		logger.Fatalw("unable to create new server asserter", "error", err)
	}

	logger.Info("loaded asserter server")

	var client *ergo.Client
	var i *indexer.Indexer
	if cfg.Mode == configuration.Online {
		client, i, err = startOnlineDependencies(ctx, cancel, cfg, g, zapLogger)
		if err != nil {
			logger.Fatalw("unable to start online dependencies", "error", err)
		}
	}

	logger.Info("loaded ergo node client")

	router := services.NewBlockchainRouter(cfg, client, asserter)
	loggedRouter := services.LoggerMiddleware(zapLogger, router)
	corsRouter := server.CorsMiddleware(loggedRouter)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.RosettaPort),
		Handler:      corsRouter,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	g.Go(func() error {
		logger.Infow("rosetta server listening", "port", cfg.RosettaPort)

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

	// Attempt to close the database gracefullly after all indexer goroutines have stopped.
	if i != nil {
		i.CloseDatabase(ctx)
	}

	if SignalReceived {
		logger.Info("rosetta-ergo halted")

		return nil
	}

	return err
}
