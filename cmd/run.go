package cmd

import (
	"context"
	"errors"
	"log"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var (
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run rosetta-ergo",
		RunE:  runCmdHandler,
	}
)

func runCmdHandler(cmd *cobra.Command, args []string) error {
	zapLogger, err := zap.NewDevelopment()

	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
	}

	defer zapLogger.Sync()

	logger := zapLogger.Sugar().Named("main")

	// TODO: start asserter

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	go handleSignals([]context.CancelFunc{cancel})

	g, ctx := errgroup.WithContext(ctx)

	// TODO: create ergo client

	logger.Info("HELLO WORLD")

	err = g.Wait()

	if SignalReceived {
		return errors.New("rosetta-ergo halted")
	}

	return err
}
