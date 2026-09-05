package graceful

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type lifecycleServer struct {
	start func()
	stop  func(context.Context)
}

func (server lifecycleServer) Start() { server.start() }

func (server lifecycleServer) Stop(ctx context.Context) { server.stop(ctx) }

func TestRunContextOrdersStartupAndShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialized := false
	stopped := false
	server := lifecycleServer{
		start: func() {
			cancel()

			initialized = true
		},
		stop: func(shutdown context.Context) {
			require.True(t, initialized)
			require.NoError(t, shutdown.Err())
			deadline, exists := shutdown.Deadline()
			require.True(t, exists)
			require.InDelta(t, 1, time.Until(deadline).Seconds(), 0.1)

			stopped = true
		},
	}
	RunContext(ctx, server, WithTimeout(time.Second))
	require.True(t, stopped)

	server.start = func() { t.Fatal("已取消时不应启动") }
	server.stop = func(context.Context) { t.Fatal("未启动时不应关闭") }
	RunContext(ctx, server)
}
