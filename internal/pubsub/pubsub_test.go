package pubsub_test

import (
	"context"
	"pubsub/internal/pubsub"
	"sync"
	"testing"

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
