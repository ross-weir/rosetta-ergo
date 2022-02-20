package ergo

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type NodeConfiguration struct {
	Network        string
	ConfigFilePath string
}

const (
	ergoLogger       = "ergo-node"
	ergoStdErrLogger = "ergo-node stderr"
)

func logPipe(ctx context.Context, pipe io.ReadCloser, identifier string) error {
	sc := bufio.NewScanner(pipe)

	for sc.Scan() {
		text := sc.Text()

		fmt.Printf("%s: %s\n", identifier, text)
	}

	return sc.Err()
}

// StartErgoNode starts a ergo node in another goroutine
// and logs the results to the console.
func StartErgoNode(ctx context.Context, cfg *NodeConfiguration, l *zap.Logger, g *errgroup.Group) error {
	logger := l.Sugar().Named("ergo-node")
	javaHome := os.Getenv("JAVA_HOME")
	javaPath := path.Join(javaHome, "bin", "java")

	logger.Debugf("javaPath: %s | network: %s | configPath: %s", javaPath, cfg.Network, cfg.ConfigFilePath)

	cmd := exec.Command(
		javaPath,
		"-Dfile.encoding=UTF-8",
		"-jar",
		"/app/ergo.jar",
		fmt.Sprintf("--%s", cfg.Network),
		fmt.Sprintf("-c=%s", cfg.ConfigFilePath),
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
		return logPipe(ctx, stdout, ergoLogger)
	})

	g.Go(func() error {
		return logPipe(ctx, stderr, ergoStdErrLogger)
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
