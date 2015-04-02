package pubsub

import "sync"

type ChanTopic interface {
	Publish(msg interface{})
	Subscribe() ChanSub
}
type ChanSub interface {
	C() <-chan interface{}
	Unsubscribe()
}

func NewChanTopic() ChanTopic { return &ct{head: newm()} }

type ct struct {
	mu   sync.Mutex
	head *m
}
type cs struct {
	current *m
	recv    chan interface{}
	stop    chan struct{}
}
type m struct {
	msg   interface{}
	next  *m
	ready chan struct{}
}

func newm() *m { return &m{ready: make(chan struct{})} }

var _ ChanTopic = (*ct)(nil)
var _ ChanSub = (*cs)(nil)

func (t *ct) Publish(msg interface{}) {
    next := newm()

    t.mu.Lock()

	t.head.msg = msg
	t.head.next = next
	close(t.head.ready)
	t.head = next

	t.mu.Unlock()
}

func (t *ct) Subscribe() ChanSub {
	t.mu.Lock()
	head := t.head
	t.mu.Unlock()

	sub := &cs{
		current: head,
		recv:    make(chan interface{}),
		stop:    make(chan struct{}),
	}
	go sub.sender()
	return sub
}

func (s *cs) C() <-chan interface{} { return s.recv }
func (s *cs) Unsubscribe()          { close(s.stop) }

func (s *cs) sender() {
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
