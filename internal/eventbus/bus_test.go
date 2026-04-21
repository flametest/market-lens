package eventbus

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"log/slog"
)

func TestInMemoryBus_PublishSubscribe(t *testing.T) {
	bus := NewInMemoryBus(slog.Default())
	defer bus.Close()

	ch := bus.Subscribe("test.topic")
	bus.Publish("test.topic", "hello")

	select {
	case msg := <-ch:
		assert.Equal(t, "hello", msg)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestInMemoryBus_MultipleSubscribers(t *testing.T) {
	bus := NewInMemoryBus(slog.Default())
	defer bus.Close()

	ch1 := bus.Subscribe("topic")
	ch2 := bus.Subscribe("topic")
	bus.Publish("topic", 42)

	assert.Equal(t, 42, <-ch1)
	assert.Equal(t, 42, <-ch2)
}

func TestInMemoryBus_DifferentTopics(t *testing.T) {
	bus := NewInMemoryBus(slog.Default())
	defer bus.Close()

	chA := bus.Subscribe("topic.a")
	chB := bus.Subscribe("topic.b")
	bus.Publish("topic.a", "only-a")

	select {
	case msg := <-chA:
		assert.Equal(t, "only-a", msg)
	case <-time.After(time.Second):
		t.Fatal("topic.a should receive")
	}

	select {
	case <-chB:
		t.Fatal("topic.b should not receive")
	case <-time.After(100 * time.Millisecond):
		// expected
	}
}

func TestInMemoryBus_SubscribeFunc(t *testing.T) {
	bus := NewInMemoryBus(slog.Default())
	defer bus.Close()

	var received atomic.Value
	bus.SubscribeFunc("fn.topic", func(data any) {
		received.Store(data)
	})
	bus.Publish("fn.topic", "func-msg")

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "func-msg", received.Load())
}

func TestInMemoryBus_Close(t *testing.T) {
	bus := NewInMemoryBus(slog.Default())
	ch := bus.Subscribe("close.topic")
	bus.Close()

	_, ok := <-ch
	assert.False(t, ok, "channel should be closed")
}

func TestInMemoryBus_PublishAfterClose(t *testing.T) {
	bus := NewInMemoryBus(slog.Default())
	bus.Close()
	bus.Publish("any.topic", "data") // should not panic
}
