// Copyright 2015 Philipp Meinen. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package pubsub

import (
	"sync"
	"sync/atomic"
	"time"
)

// A Topic is used to distribute messages to multiple consumers/subscriptions
type Topic interface {
	// Publish publishes a new message to all subscribers.
	Publish(msg interface{})

	// Subscribe creates a new Subscription on which newly published messages can be received.
	Subscribe() Subscription

	// NumSubscribers returns the current number of subscribers.
	NumSubscribers() int

	// SetHWM sets a high water mark in number of messages.
	// Subscribers which lag behind more messages than the high water mark will have messages discarded.
	// A high water mark of zero means no messages will be discarded (default).
	// Changes to the high water mark only apply to newly published messages.
	SetHWM(hwm int)
}

// Subscription is a topic subscription of one consumer.
// Messages which are published on the subscribed Topic are delivered at most once but may be consumed from any goroutine.
type Subscription interface {
	// C returns the receive-only consumer channel on which newly published messages are received.
	// Unsubscribing from a subscription will close this consumer channel.
	// The consumer channel can therefore be used in a for-range construct.
	C() <-chan interface{}

	// Unsubscribe frees the resources which are associated with this Subscription.
	// Consumers must call Unsubscribe to prevent memory/goroutine leaks.
	Unsubscribe()
}

func NewTopic() Topic { return &topic{head: newMsg()} }

type topic struct {
	mu      sync.Mutex
	head    *msg  // head of message chain
	headSeq int64 // sequence number of head message, read by consumers to check for high water mark checks
	hwm     int64 // high water mark for newly published messages
	nsub    int   // number of subscribers
}

var _ Topic = (*topic)(nil)

func (t *topic) Publish(msg interface{}) {
	next := newMsg()

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.nsub == 0 {
		return
	}
	t.headSeq++
	next.seq = t.headSeq

	t.head.msg = msg
	t.head.next = next
	t.head.hwm = t.hwm
	close(t.head.ready)
	t.head = next
}

func (t *topic) NumSubscribers() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.nsub
}

func (t *topic) Subscribe() Subscription {
	t.mu.Lock()
	head := t.head
	t.nsub++
	t.mu.Unlock()

	sub := &sub{
		t:         t,
		current:   head,
		recv:      make(chan interface{}),
		stop:      make(chan struct{}),
		hwmTicker: time.NewTicker(hwmCheckInterval),
	}
	go sub.subscriberLoop()
	return sub
}

func (t *topic) SetHWM(hwm int) {
	t.mu.Lock()
	t.hwm = int64(hwm)
	t.mu.Unlock()
}

// forward referencing message chain
// the ready channel is closed upon message readyness
// consumers select upon the ready channel
type msg struct {
	ready chan struct{} // guards the fields below, a closed ready channel is indicative of message readiness
	msg   interface{}   // the published message
	next  *msg          // the next message holder object in the chain of published messages
	seq   int64         // message sequence number
	hwm   int64         // high water mark at message creation time
}

func newMsg() *msg { return &msg{ready: make(chan struct{})} }

type sub struct {
	t       *topic
	current *msg
	recv    chan interface{} // consumer receive channel
	stop    chan struct{}    // unsubscribe and end subscriber loop notification channel

	hwmTicker *time.Ticker // ticker for periodic high water mark checks
}

var _ Subscription = (*sub)(nil)

func (s *sub) C() <-chan interface{} { return s.recv }

func (s *sub) Unsubscribe() {
	s.t.mu.Lock()
	s.t.nsub--
	s.t.mu.Unlock()
	close(s.stop)
}

// TODO: improve this "constant"
// TODO: expose through setter ?
const hwmCheckInterval = 10 * time.Second

func (s *sub) subscriberLoop() {
	defer func() {
		close(s.recv)
		s.hwmTicker.Stop()
		s.current = nil
	}()

	var ready bool
	for {
		if !ready {
			// receive next message or stop
			select {
			case <-s.stop:
				return
			case <-s.current.ready:
				ready = true
			case <-s.hwmTicker.C:
				// no message is waiting to be consumed
				// => no high water mark check required
				// => do nothing
			}
		} else {
			// deliver message or stop
			select {
			case <-s.stop:
				return
			case s.recv <- s.current.msg:
				ready = false
				s.current = s.current.next
			case <-s.hwmTicker.C:
				s.hwmCheck()
			}
		}
	}
}

// discards messages which have a message lag higher than the HWM.
// the subscribers current message must be ready.
func (s *sub) hwmCheck() {
	hwm := s.current.hwm
	if hwm < 1 {
		return
	}
	seq := s.current.seq
	headSeq := atomic.LoadInt64(&s.t.headSeq)
	ready := headSeq - seq
	discard := ready - hwm
	for discard > 0 {
		discard--
		s.current = s.current.next
		<-s.current.ready // synchronization on the current message holder
	}
}
