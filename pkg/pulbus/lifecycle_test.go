package pulbus

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/stretchr/testify/require"
)

type lifecycleClient struct {
	pulsar.Client

	subscribed      chan struct{}
	release         chan struct{}
	consumer        *lifecycleConsumer
	closed          atomic.Int32
	producerStarted chan struct{}
	producerRelease chan struct{}
	producer        *lifecycleProducer
	producerCount   atomic.Int32
}

func (client *lifecycleClient) CreateProducer(_ pulsar.ProducerOptions) (pulsar.Producer, error) {
	if client.producerCount.Add(1) == 1 {
		close(client.producerStarted)
	}

	if client.producerRelease != nil {
		<-client.producerRelease
	}

	return client.producer, nil
}

func (client *lifecycleClient) Subscribe(_ pulsar.ConsumerOptions) (pulsar.Consumer, error) {
	close(client.subscribed)

	if client.release != nil {
		<-client.release
	}

	return client.consumer, nil
}

func (client *lifecycleClient) Close() {
	client.closed.Add(1)
}

type lifecycleConsumer struct {
	pulsar.Consumer

	closed atomic.Int32
}

func (consumer *lifecycleConsumer) Close() {
	consumer.closed.Add(1)
}

type lifecycleProducer struct {
	pulsar.Producer
	closed atomic.Int32
}

func (producer *lifecycleProducer) Close() {
	producer.closed.Add(1)
}

func TestConcurrentProducerCreation(t *testing.T) {
	client := &lifecycleClient{
		producerStarted: make(chan struct{}),
		producerRelease: make(chan struct{}),
		producer:        &lifecycleProducer{},
	}
	bus := &Pulbus{client: client, closing: make(chan struct{}), closeDone: make(chan struct{})}
	results := make(chan error, 16)
	var waiting sync.WaitGroup
	waiting.Add(cap(results))

	for range cap(results) {
		go func() {
			waiting.Done()

			producer, err := bus.getProducer(context.Background(), "topic")
			if err == nil && producer != client.producer {
				t.Error("并发请求未共享同一个 Producer")
			}

			results <- err
		}()
	}

	<-client.producerStarted
	waiting.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := bus.getProducer(ctx, "topic")
	require.ErrorIs(t, err, context.Canceled)
	close(client.producerRelease)

	for range cap(results) {
		require.NoError(t, <-results)
	}

	bus.Close()
	require.EqualValues(t, 1, client.producerCount.Load())
	require.EqualValues(t, 1, client.producer.closed.Load())
}

func TestProducerCancellationAndClose(t *testing.T) {
	client := &lifecycleClient{
		producerStarted: make(chan struct{}),
		producerRelease: make(chan struct{}),
		producer:        &lifecycleProducer{},
	}
	bus := &Pulbus{client: client, closing: make(chan struct{}), closeDone: make(chan struct{})}
	result := make(chan error, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_, err := bus.getProducer(ctx, "topic")
		result <- err
	}()
	<-client.producerStarted

	cancel()
	require.ErrorIs(t, <-result, context.Canceled)

	err := bus.CloseContext(ctx)
	require.ErrorIs(t, err, context.Canceled)
	close(client.producerRelease)
	bus.Close()

	require.EqualValues(t, 1, client.producer.closed.Load())
	require.EqualValues(t, 1, client.closed.Load())
}

func TestCloseStopsConsumption(t *testing.T) {
	client := &lifecycleClient{subscribed: make(chan struct{}), consumer: &lifecycleConsumer{}}
	bus := &Pulbus{client: client, closing: make(chan struct{}), closeDone: make(chan struct{})}
	result := make(chan error, 1)

	go func() {
		result <- bus.Consume(context.Background(), "topic", "subscription", func(pulsar.Message) error { return nil })
	}()
	<-client.subscribed

	bus.Close()
	require.ErrorIs(t, <-result, ErrClosed)
	require.EqualValues(t, 1, client.consumer.closed.Load())
	require.EqualValues(t, 1, client.closed.Load())

	bus.Close()

	err := bus.SendBytes(context.Background(), "topic", []byte("after-close"))
	require.ErrorIs(t, err, ErrClosed)
	require.EqualValues(t, 1, client.closed.Load())
}

func TestCloseWaitsForConsumerCreation(t *testing.T) {
	client := &lifecycleClient{
		subscribed: make(chan struct{}),
		release:    make(chan struct{}),
		consumer:   &lifecycleConsumer{},
	}
	bus := &Pulbus{client: client, closing: make(chan struct{}), closeDone: make(chan struct{})}
	result := make(chan error, 1)

	go func() {
		result <- bus.ConsumeWithLoop(context.Background(), "topic", "subscription", func(pulsar.Message) error { return nil })
	}()
	<-client.subscribed

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := bus.CloseContext(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Zero(t, client.closed.Load())

	close(client.release)

	bus.Close()
	require.ErrorIs(t, <-result, ErrClosed)
	require.EqualValues(t, 1, client.consumer.closed.Load())
	require.EqualValues(t, 1, client.closed.Load())
}
