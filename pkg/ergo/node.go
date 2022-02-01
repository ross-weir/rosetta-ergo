package ergo

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	ergoLogger       = "ergo-node"
	ergoStdErrLogger = "ergo-node stderr"
)

func logPipe(ctx context.Context, pipe io.ReadCloser, identifier string, l *zap.Logger) error {
	logger := l.Sugar().Named(identifier)
	reader := bufio.NewReader(pipe)
	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			logger.Warnw("closing logger", "error", err)
			return err
		}

		message := strings.ReplaceAll(str, "\n", "")
		messages := strings.SplitAfterN(message, " ", 2)

		// Trim the timestamp from the log if it exists
		if len(messages) > 1 {
			message = messages[1]
		}

		// Print debug log if from bitcoindLogger
		if identifier == ergoLogger {
			logger.Debugw(message)
			continue
		}

		logger.Warnw(message)
	}
}

// StartErgoNode starts a ergo node in another goroutine
// and logs the results to the console.
func StartErgoNode(ctx context.Context, configPath string, l *zap.Logger, g *errgroup.Group) error {
	logger := l.Sugar().Named("ergo-node")
	cmd := exec.Command(
		"java -jar /app/ergo.jar",
		"--testnet",
		fmt.Sprintf("-c=%s", configPath),
	) // #nosec G204

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	g.Go(func() error {
		return logPipe(ctx, stdout, ergoLogger, l)
	})

	g.Go(func() error {
		return logPipe(ctx, stderr, ergoStdErrLogger, l)
	})

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%w: unable to start ergo node", err)
	}

	g.Go(func() error {
		<-ctx.Done()

		logger.Warnw("sending interrupt to ergo node")
		return cmd.Process.Signal(os.Interrupt)
	})

	return cmd.Wait()
}
