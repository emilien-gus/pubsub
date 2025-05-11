package pubsub

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
)

var ErrSubjectNotFound = errors.New("subject not found")

type Broker struct {
	subscribers    map[string][]*Subscriber
	chanBufferSize int
	mu             sync.RWMutex
	closed         chan struct{}
}

func NewBroker(bufferSize int) *Broker {
	return &Broker{
		subscribers:    make(map[string][]*Subscriber),
		chanBufferSize: bufferSize,
		closed:         make(chan struct{}),
	}
}

func (b *Broker) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	select {
	case <-b.closed:
		return nil, context.Canceled
	default:
	}

	subs, ok := b.subscribers[subject]
	if !ok {
		subs = []*Subscriber{}
	}

	sub := &Subscriber{
		cb:       cb,
		msgQueue: make(chan interface{}, b.chanBufferSize),
		subject:  subject,
		once:     sync.Once{},
	}
	sub.active.Store(true)
	sub.unsubFunc = func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		subs := b.subscribers[subject]
		for i, s := range subs {
			if s == sub {
				b.subscribers[subject] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}

	subs = append(subs, sub)
	b.subscribers[subject] = subs

	go func() {
		for msg := range sub.msgQueue {
			sub.cb(msg)
		}
	}()

	return sub, nil
}

func (b *Broker) Publish(subject string, msg interface{}) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	select {
	case <-b.closed:
		return context.Canceled
	default:
	}

	subs, ok := b.subscribers[subject]
	if !ok {
		return ErrSubjectNotFound
	}
	var dropped int
	for _, sub := range subs {
		if !sub.active.Load() {
			continue
		}
		select {
		case sub.msgQueue <- msg:
		default:
			// chanel buffer overflow
			dropped++
			log.Printf("%s subscriber message queue full, dropping message", sub.subject)
		}
	}

	if dropped > 0 {
		return fmt.Errorf("message dropped for %d of %d subscribers", dropped, len(subs))
	}

	return nil
}

func (b *Broker) Close(ctx context.Context) error {
	b.mu.Lock()
	select {
	case <-b.closed:
		b.mu.Unlock()
		return nil
	default:
		close(b.closed)
	}

	allSubs := []*Subscriber{}
	for subject, subs := range b.subscribers {
		allSubs = append(allSubs, subs...)
		delete(b.subscribers, subject) // preventing from new
	}
	b.mu.Unlock()

	wg := sync.WaitGroup{}
	for _, sub := range allSubs {
		wg.Add(1)
		go func(s *Subscriber) {
			defer wg.Done()
			s.Unsubscribe()
		}(sub)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
