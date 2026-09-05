package logger

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"

	"nexis.run/nexa/kit/configure"
	"nexis.run/nexa/pkg/clara"
)

type blockedKafkaTransport struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	err     error
}

func (transport *blockedKafkaTransport) RoundTrip(ctx context.Context, _ net.Addr, _ kafka.Request) (response kafka.Response, err error) {
	transport.once.Do(func() { close(transport.started) })

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-transport.release:
		err = transport.err
	}

	return
}

func TestKafkaWriterCancellationAndClose(t *testing.T) {
	transport := &blockedKafkaTransport{
		started: make(chan struct{}),
		release: make(chan struct{}),
		err:     errors.New("模拟发送失败"),
	}
	writer := NewKafkaWriter([]string{"mock:9092"}, "logs")
	writer.With(func(w *kafka.Writer) { w.Transport = transport })

	_, err := writer.Write([]byte("message"))
	require.NoError(t, err)
	<-transport.started

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err = writer.SyncContext(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	err = writer.CloseContext(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	_, err = writer.Write([]byte("after-close"))
	require.ErrorIs(t, err, io.ErrClosedPipe)

	close(transport.release)

	err = writer.Close()
	require.ErrorIs(t, err, transport.err)
}

func TestKafkaWriterPayloadBudget(t *testing.T) {
	transport := &blockedKafkaTransport{
		started: make(chan struct{}),
		release: make(chan struct{}),
		err:     errors.New("模拟发送失败"),
	}
	writer := NewKafkaWriter([]string{"mock:9092"}, "logs")
	writer.Writer = clara.NewWriter([]string{"mock:9092"}, "logs", clara.WithTimeout(time.Millisecond))
	writer.With(func(w *kafka.Writer) { w.Transport = transport })

	n, err := writer.Write(make([]byte, kafkaMaxLogBytes+1))
	require.ErrorIs(t, err, ErrKafkaLogTooLarge)
	require.Zero(t, n)
	require.Zero(t, writer.buffered.Load())

	payload := make([]byte, kafkaMaxLogBytes)

	_, err = writer.Write(payload)
	require.NoError(t, err)
	<-transport.started

	for range kafkaBufferBytes/kafkaMaxLogBytes - 1 {
		_, err = writer.Write(payload)
		require.NoError(t, err)
	}

	_, err = writer.Write(payload)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err = writer.SyncContext(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.EqualValues(t, kafkaBufferBytes, writer.buffered.Load())
	require.EqualValues(t, 2, writer.Dropped())

	close(transport.release)

	err = writer.Close()
	require.ErrorIs(t, err, transport.err)
	require.Zero(t, writer.buffered.Load())
}

func TestCloseWaitsForReplacedWriter(t *testing.T) {
	transport := &blockedKafkaTransport{
		started: make(chan struct{}),
		release: make(chan struct{}),
		err:     errors.New("模拟发送失败"),
	}
	err := Setup(&configure.Logger{Kafka: &configure.LoggerKafka{Brokers: []string{"mock:9092"}, Topic: "logs"}})
	require.NoError(t, err)

	setupMu.Lock()
	writer := activeKafkaWriter
	setupMu.Unlock()
	writer.With(func(w *kafka.Writer) { w.Transport = transport })

	_, err = writer.Write([]byte("before-reload"))
	require.NoError(t, err)
	<-transport.started

	err = Setup(&configure.Logger{Stdout: true})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err = Close(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	close(transport.release)

	err = writer.Close()
	require.ErrorIs(t, err, transport.err)
}
