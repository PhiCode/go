// Copyright 2015 Philipp Meinen. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

// Package pubsub implements a fan-out message distribution pattern in the form of a topic.
package pubsub

import (
	"sync"
)

// A Topic is used to distribute messages to multiple consumers.
// Published messages are delivered in publishing order.
// If an optional "high water mark" (HWM) is set, messages for slow consumsers may be dropped.
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

// Subscription is the topic subscription of one consumer.
// Messages which are published on the subscribed Topic are delivered at most once but may be consumed from any goroutine.
type Subscription interface {
	// C returns the receive-only consumer channel on which newly published messages are received.
	// Unsubscribing from a Subscription will close this consumer channel.
	// The consumer channel can therefore be used in a for-range construct.
	C() <-chan interface{}

	// Unsubscribe frees the resources which are associated with this Subscription.
	// Consumers must call Unsubscribe to prevent memory/goroutine leaks.
	Unsubscribe()
}

// NewTopic creates a new message topic.
// No high water mark is set, consumers which are lagging behind will no loose messages.
func NewTopic() Topic { return &topic{head: newMsg()} }

type topic struct {
	mu   sync.Mutex
	head *msg // head of message chain
	nsub int  // number of subscribers
}

var _ Topic = (*topic)(nil)

func (t *topic) Publish(msg interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.nsub == 0 {
		return
	}

	oldHead, newHead := t.head, newMsg()

	newHead.seq = oldHead.seq + 1
	newHead.hwm = oldHead.hwm

	oldHead.msg = msg
	oldHead.next = newHead

	t.head = newHead

	close(oldHead.ready)
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
		t:    t,
		recv: make(chan interface{}),
		stop: make(chan struct{}),
	}
	go sub.loop(head)
	return sub
}

func (t *topic) SetHWM(hwm int) {
	t.mu.Lock()
	t.head.hwm = int64(hwm)
	t.mu.Unlock()
}

// forward referencing message chain
// the ready channel is closed upon message readyness
// consumers select upon the ready channel
type msg struct {
	seq   int64         // message sequence number
	ready chan struct{} // guards the fields below, a closed ready channel is indicative of message readiness
	hwm   int64         // high water mark at message creation time
	msg   interface{}   // the published message
	next  *msg          // the next message holder object in the chain of published messages
}

func newMsg() *msg { return &msg{ready: make(chan struct{})} }

type sub struct {
	t    *topic
	recv chan interface{} // consumer receive channel
	stop chan struct{}    // unsubscribe and end subscriber loop notification channel
}

var _ Subscription = (*sub)(nil)

func (s *sub) C() <-chan interface{} { return s.recv }

func (s *sub) Unsubscribe() {
	s.t.mu.Lock()
	s.t.nsub--
	s.t.mu.Unlock()

	// shut down the subscriptions loop goroutine
	close(s.stop)
}

func (s *sub) loop(head *msg) {
	defer close(s.recv)

	var cur *msg
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
func hwmCheck(cur, newest *msg) *msg {
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
