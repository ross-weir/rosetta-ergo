package utils

import (
	"context"
	"time"

	sdkUtils "github.com/coinbase/rosetta-sdk-go/utils"
	"go.uber.org/zap"
)

const (
	// monitorMemorySleep is how long we should sleep
	// between checking memory stats.
	monitorMemorySleep = 50 * time.Millisecond
)

// MonitorMemoryUsage periodically logs memory usage
// stats and triggers garbage collection when heap allocations
// surpass maxHeapUsage.
func MonitorMemoryUsage(
	ctx context.Context,
	maxHeapUsage int,
	logger *zap.Logger,
) error {
	l := logger.Sugar().Named("memory")

	maxHeap := float64(0)
	garbageCollections := uint32(0)
	for ctx.Err() == nil {
		memUsage := sdkUtils.MonitorMemoryUsage(ctx, maxHeapUsage)
		if memUsage.Heap > maxHeap {
			maxHeap = memUsage.Heap
		}

		if memUsage.GarbageCollections > garbageCollections {
			garbageCollections = memUsage.GarbageCollections
			l.Debugw(
				"stats",
				"heap (MB)", memUsage.Heap,
				"max heap (MB)", maxHeap,
				"stack (MB)", memUsage.Stack,
				"system (MB)", memUsage.System,
				"garbage collections", memUsage.GarbageCollections,
			)
		}

		if err := sdkUtils.ContextSleep(ctx, monitorMemorySleep); err != nil {
			return err
		}
	}

	return ctx.Err()
}
