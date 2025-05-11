package pubsub_test

import (
	"context"
	"pubsub/internal/pubsub"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSubscribeAndPublish(t *testing.T) {
	b := pubsub.NewBroker(10)
	defer b.Close(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	var received string
	_, err := b.Subscribe("topic", func(msg interface{}) {
		received = msg.(string)
		wg.Done()
	})
	assert.NoError(t, err)

	err = b.Publish("topic", "hello")
	assert.NoError(t, err)

	wg.Wait()
	assert.Equal(t, "hello", received)
}

func TestUnsubscribe(t *testing.T) {
	b := pubsub.NewBroker(10)
	defer b.Close(context.Background())

	sub, err := b.Subscribe("topic", func(msg interface{}) {})
	assert.NoError(t, err)

	sub.Unsubscribe()
	err = b.Publish("topic", "message")
	assert.NoError(t, err)
}

func TestPublishToNonexistentTopic(t *testing.T) {
	b := pubsub.NewBroker(10)
	defer b.Close(context.Background())

	err := b.Publish("unknown", "message")
	assert.ErrorIs(t, err, pubsub.ErrSubjectNotFound)
}

func TestCloseBroker(t *testing.T) {
	b := pubsub.NewBroker(10)

	var called bool
	_, err := b.Subscribe("topic", func(msg interface{}) {
		called = true
	})
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = b.Close(ctx)
	assert.NoError(t, err)

	err = b.Publish("topic", "message")
	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, called)
}

func TestDroppedMessages(t *testing.T) {
	b := pubsub.NewBroker(1)
	defer b.Close(context.Background())

	// slow subscriber
	_, err := b.Subscribe("topic", func(msg interface{}) {
		time.Sleep(50 * time.Millisecond)
	})
	assert.NoError(t, err)

	b.Publish("topic", "msg1")       // goes through
	err = b.Publish("topic", "msg2") // likely dropped
	assert.Error(t, err)
}
