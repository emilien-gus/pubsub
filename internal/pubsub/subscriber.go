package pubsub

import (
	"sync"
	"sync/atomic"
)

type Subscriber struct {
	cb        MessageHandler
	msgQueue  chan interface{}
	subject   string
	unsubFunc func()
	active    atomic.Bool
	once      sync.Once
}

func (s *Subscriber) Unsubscribe() {
	s.once.Do(func() {
		s.active.Store(false)
		close(s.msgQueue)
		s.unsubFunc()
	})
}
