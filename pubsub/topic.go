package pubsub

import (
	"sync"
	"time"
)

type Topic interface {
	// Publish publishes a new message to all subscribers.
	Publish(msg interface{})

	// Subscribe creates a new Subscription on which newly published messages can be received.
	Subscribe() Subscription
}

type Subscription interface {
	// Unsubscribe frees the resources which are associated with this Receiver.
	//
	// This implementation of Receiver does not require Unregister to be called.
	// Garbage collecting the Receiver is enough to 'free' the resources.
	Unsubscribe()

	// Receive returns the next message or blocks until one is available.
	Receive() (msg interface{})

	// ReceiveNow returns the next message immediately, if there is any.
	// The ok return value indicates if a message was available.
	ReceiveNow() (msg interface{}, ok bool)

	// ReceiveTimeout returns the next message if one is available.
	// Otherwise it will block until either a message becomes available or the timeout has elapsed.
	ReceiveTimeout(timeout time.Duration) (msg interface{}, ok bool)

	// Latest advances to the head of the receive queue and returns the head message.
	// If this receiver has already received the head message Latest will block until a new message becomes available.
	// The messages leading up to the head are no longer accessible by this Receiver.
	Latest() (msg interface{})

	// LatestNow advances to the head of the receive queue and returns the head message.
	// If this receiver has already received the head message LatestNow will return (nil, false).
	// The messages leading up to the head are no longer accessible by this Receiver.
	LatestNow() (msg interface{}, ok bool)

	// LatestTimeout advances to the head of the receive queue and returns the head message if one is available.
	// Otherwise it will block until either a message becomes available or the timeout has elapsed.
	// The messages leading up to the head are no longer accessible by this Receiver.
	LatestTimeout(timeout time.Duration) (msg interface{}, ok bool)
}

type topic struct {
	mu   sync.Mutex
	head *node
}

var _ Topic = (*topic)(nil)

func NewTopic() Topic {
	return &topic{head: newNode()}
}

func (t *topic) Publish(msg interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.head = t.head.set(msg)
}

func (t *topic) Subscribe() Subscription {
	t.mu.Lock()
	defer t.mu.Unlock()

	return &sub{current: t.head}
}

type node struct {
	next chan *node
	msg  interface{}
}

func newNode() *node {
	return &node{next: make(chan *node, 1)}
}

func (n *node) set(msg interface{}) *node {
	next := newNode()
	n.msg = msg
	n.next <- next
	return next
}

func (n *node) get() (interface{}, *node) {
	next := <-n.next
	msg := n.msg
	n.next <- next
	return msg, next
}

func (n *node) getNow() (interface{}, *node) {
	select {
	case next := <-n.next:
		msg := n.msg
		n.next <- next
		return msg, next
	default:
		return nil, nil
	}
}

func (n *node) getTimeout(to time.Duration) (interface{}, *node) {
	select {
	case next := <-n.next:
		msg := n.msg
		n.next <- next
		return msg, next
	case <-time.After(to):
		return nil, nil
	}
}

//
// SUBSCRIPTION
//

type sub struct {
	current *node
}

var _ Subscription = (*sub)(nil)

func (s *sub) Unsubscribe() {
	// only unreference the current node so that the GC can clean up
	s.current = nil
}

func (s *sub) Receive() interface{} {
	msg, next := s.current.get()
	s.current = next
	return msg
}

func (s *sub) ReceiveNow() (interface{}, bool) {
	msg, next := s.current.getNow()
	if next == nil {
		return nil, false
	}
	s.current = next
	return msg, true
}

func (s *sub) ReceiveTimeout(to time.Duration) (interface{}, bool) {
	msg, next := s.current.getNow()
	if next == nil {
		msg, next = s.current.getTimeout(to)
	}
	if next == nil {
		return nil, false
	}
	s.current = next
	return msg, true
}

func (s *sub) Latest() interface{} {
	if msg, ok := s.LatestNow(); ok {
		return msg
	}
	return s.Receive()
}

func (s *sub) LatestNow() (interface{}, bool) {
	msg, next := s.current.getNow()
	if next == nil {
		return nil, false
	}
	for {
		v, n := next.getNow()
		if n == nil {
			s.current = next
			return msg, true
		}
		msg, next = v, n
	}
}

func (s *sub) LatestTimeout(to time.Duration) (msg interface{}, ok bool) {
	if msg, ok = s.LatestNow(); ok {
		return
	}
	return s.ReceiveTimeout(to)
}
