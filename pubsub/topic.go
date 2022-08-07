// Copyright 2015 Philipp Meinen. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

// Package pubsub implements a fan-out message distribution pattern in the form of a topic.
package pubsub

import (
	"sync"
	"sync/atomic"
)

// A Topic is used to distribute messages to multiple consumers.
// Published messages are delivered in publishing order.
// If an optional "high water mark" (HWM) is set, messages for slow consumers may be dropped.
type Topic[M any] interface {
	// Publish publishes a new message to all subscribers.
	Publish(msg M)

	// Subscribe creates a new Subscription on which newly published messages can be received.
	Subscribe() Subscription[M]

	// NumSubscribers returns the current number of subscribers.
	NumSubscribers() int

	// SetHWM sets a high water mark in number of messages.
	// Subscribers which lag behind more messages than the high water mark will have messages discarded.
	// A high water mark of zero means no messages will be discarded (default).
	// Changes to the high water mark only apply to newly published messages.
	SetHWM(hwm int)
}

// Subscription is the topic subscription of one consumer.
// Messages which are published on the subscribed Topic are delivered at most once but may be consumed from any goroutine.
type Subscription[M any] interface {
	// C returns the receive-only consumer channel on which newly published messages are received.
	// Unsubscribing from a Subscription will close this consumer channel.
	// The consumer channel can therefore be used in a for-range construct.
	C() <-chan M

	// Unsubscribe frees the resources which are associated with this Subscription.
	// Consumers must call Unsubscribe to prevent memory/goroutine leaks.
	Unsubscribe() bool
}

// NewTopic creates a new message topic.
// No high water mark is set, consumers which are lagging behind will not loose messages.
func NewTopic[M any]() Topic[M] { return &topic[M]{head: newMsg[M]()} }

type topic[M any] struct {
	// number of subscribers - atomically mutated
	nsub int64

	mu sync.Mutex

	// head of message chain
	head *msg[M]
}

var _ Topic[int] = (*topic[int])(nil)

func (t *topic[M]) Publish(msg M) {
	if atomic.LoadInt64(&t.nsub) <= 0 {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	oldHead, newHead := t.head, newMsg[M]()

	newHead.seq = oldHead.seq + 1
	newHead.hwm = oldHead.hwm

	oldHead.msg = msg
	oldHead.next = newHead

	t.head = newHead

	close(oldHead.ready)
}

func (t *topic[M]) NumSubscribers() int {
	return int(atomic.LoadInt64(&t.nsub))
}

func (t *topic[M]) Subscribe() Subscription[M] {
	t.mu.Lock()
	head := t.head
	atomic.AddInt64(&t.nsub, 1)
	t.mu.Unlock()

	sub := &sub[M]{
		subscribed: 1,
		t:          t,
		recv:       make(chan M),
		stop:       make(chan struct{}),
	}
	go sub.loop(head)
	return sub
}

func (t *topic[M]) SetHWM(hwm int) {
	t.mu.Lock()
	t.head.hwm = int64(hwm)
	t.mu.Unlock()
}

// forward referencing message chain
// the ready channel is closed upon message readyness
// consumers select upon the ready channel
type msg[M any] struct {
	seq   int64         // message sequence number
	ready chan struct{} // guards the fields below, a closed ready channel is indicative of message readiness
	hwm   int64         // high water mark at message creation time
	msg   M             // the published message
	next  *msg[M]       // the next message holder object in the chain of published messages
}

func newMsg[M any]() *msg[M] { return &msg[M]{ready: make(chan struct{})} }

type sub[M any] struct {
	subscribed int32
	t          *topic[M]
	recv       chan M        // consumer receive channel
	stop       chan struct{} // unsubscribe and end subscriber loop notification channel
}

var _ Subscription[int] = (*sub[int])(nil)

func (s *sub[M]) C() <-chan M { return s.recv }

func (s *sub[M]) Unsubscribe() bool {
	if !atomic.CompareAndSwapInt32(&s.subscribed, 1, 0) {
		return false
	}
	atomic.AddInt64(&s.t.nsub, -1)
	// shut down the subscriptions loop goroutine
	close(s.stop)
	return true
}

func (s *sub[M]) loop(head *msg[M]) {
	defer close(s.recv)

	var cur *msg[M]
	for {
		if cur == nil {
			// receive next message or stop
			select {
			case <-s.stop:
				return
			case <-head.ready:
				cur = head
				head = head.next
			}
		} else {
			// deliver message or stop
			select {
			case <-s.stop:
				return
			case s.recv <- cur.msg:
				cur = cur.next
				if cur == head {
					cur = nil
				}
			case <-head.ready:
				newest := head
				head = head.next
				cur = hwmCheck(cur, newest)
			}
		}
	}
}

// discards messages which have a message lag higher than the HWM.
// the subscribers current message must be ready.
func hwmCheck[M any](cur, newest *msg[M]) *msg[M] {
	if newest.hwm < 1 {
		return cur
	}
	nready := newest.seq - cur.seq + 1
	discard := nready - newest.hwm
	for discard > 0 {
		discard--
		cur = cur.next
	}
	return cur
}
