// Copyright 2015 Philipp Meinen. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package pubsub

import (
	"sync"
)

//TODO: optional high water mark
// - sequence number on message
// - atomic head sequence number on topic
// - subscription sender checks current-to-head delay against the HWM
// - discard messages

// Topic
type Topic interface {
	// Publish publishes a new message to all subscribers.
	Publish(msg interface{})

	// Subscribe creates a new Subscription on which newly published messages can be received.
	Subscribe() Subscription

	// NumSubscribers returns the current number of subscribers.
	NumSubscribers() int
}

type Subscription interface {
	// C returns the receive-only channel on which newly published messages are received.
	C() <-chan interface{}

	// Unsubscribe frees the resources which are associated with this Subscription.
	// Consumers must call Unsubscribe to prevent memory/goroutine leaks.
	Unsubscribe()
}

func NewTopic() Topic { return &ct{head: newm()} }

type ct struct {
	mu     sync.Mutex
	head   *m
	numSub int
}
type cs struct {
	t       *ct
	current *m
	recv    chan interface{}
	stop    chan struct{}
}
type m struct {
	ready chan struct{} // guards the fields below, a closed ready channel is indicative of message readiness
	msg   interface{}   // the published message
	next  *m            // the next message holder object in the chain of published messages
}

func newm() *m { return &m{ready: make(chan struct{})} }

var _ Topic = (*ct)(nil)
var _ Subscription = (*cs)(nil)

func (t *ct) Publish(msg interface{}) {
	next := newm()

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.numSub == 0 {
		return
	}

	t.head.msg = msg
	t.head.next = next
	close(t.head.ready)
	t.head = next
}

func (t *ct) NumSubscribers() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.numSub
}

func (t *ct) Subscribe() Subscription {
	t.mu.Lock()
	head := t.head
	t.numSub++
	t.mu.Unlock()

	sub := &cs{
		t:       t,
		current: head,
		recv:    make(chan interface{}),
		stop:    make(chan struct{}),
	}
	go sub.subscriberLoop()
	return sub
}

func (s *cs) C() <-chan interface{} { return s.recv }

func (s *cs) Unsubscribe() {
	s.t.mu.Lock()
	s.t.numSub--
	s.t.mu.Unlock()
	close(s.stop)
}

func (s *cs) subscriberLoop() {
	defer func() {
		s.current = nil
		close(s.recv)
	}()

	var msg interface{}
	for {
		if msg == nil {
			// receive next message or stop
			select {
			case <-s.stop:
				return
			case <-s.current.ready:
				msg = s.current.msg
				s.current = s.current.next
			}
		} else {
			// send message or stop
			select {
			case <-s.stop:
				return
			case s.recv <- msg:
				msg = nil
			}
		}
	}
}
