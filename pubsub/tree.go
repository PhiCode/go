package pubsub

import (
	"sync"

	"github.com/phicode/go/path"
)

// subscription:
// - register a subscriber on the highest possible node
// - create tree nodes if necessary
//
// publishing:
// - walk the publishing path
// - publish on all tree nodes with non-empty list of subscribers

// A TopicTree is used to distribute messages to multiple consumers.
// Published messages are delivered in publishing order.
// If an optional "high water mark" (HWM) is set, messages for slow consumsers may be dropped.
type TopicTree interface {
	// Publish publishes a new message to all subscribers.
	Publish(path string, msg interface{})

	// Subscribe creates a new Subscription on which newly published messages can be received.
	Subscribe(path string) Subscription

	//TODO: per level/subscriber/global? scrap altogether ?
	// NumSubscribers returns the current number of subscribers.
	//	NumSubscribers(path string) int

	//TODO: per level/subscriber/global? scrap altogether ?
	// SetHWM sets a high water mark in number of messages.
	// Subscribers which lag behind more messages than the high water mark will have messages discarded.
	// A high water mark of zero means no messages will be discarded (default).
	// Changes to the high water mark only apply to newly published messages.
	//	SetHWM(path string, hwm int, recursive bool)
}

type tt struct {
	topic Topic

	mu    sync.Mutex
	refs  int64
	leafs map[string]*tt
}

var _ (TopicTree) = (*tt)(nil)

func NewTopicTree() TopicTree { return newtt() }

func (t *tt) Publish(treePath string, msg interface{}) {
	path, ok := path.Parse(treePath)
	if !ok {
		//TODO: return an error for invalid paths?
		return
	}

	t.publish(&path, 0, msg)
}

func (t *tt) publish(p *path.Path, depth int, msg interface{}) {
	t.topic.Publish(msg)

	t.mu.Lock()
	defer t.mu.Unlock()

	if next, ok := p.Elem(depth); ok {
		if leaf, ok := t.leafs[next]; ok {
			leaf.publish(p, depth+1, msg)
		}
	}
}

func (t *tt) Subscribe(treePath string) Subscription {
	path, ok := path.Parse(treePath)
	if !ok {
		//TODO: return an error for invalid paths?
		return nil
	}

	return t.subscribe(t, &path, 0)
}

func (t *tt) subscribe(root *tt, p *path.Path, depth int) Subscription {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.refs++

	if next, ok := p.Elem(depth); ok {
		// subscription is not for this level, propagate to the next level in the hierarchy
		leaf, ok := t.leafs[next]
		if !ok {
			leaf = newtt()
			t.leafs[next] = leaf
		}
		return leaf.subscribe(root, p, depth+1)
	}

	s := t.topic.Subscribe()
	return &ttsub{
		root: root,
		p:    p,
		s:    s,
	}
}

func (t *tt) unsubscribe(p *path.Path, depth int) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.refs--

	if next, ok := p.Elem(depth); ok {
		// subscription is not for this level, propagate to the next level in the hierarchy
		leaf, ok := t.leafs[next]
		if !ok {
			panic("invalid unsubscribe")
		}
		if leaf.unsubscribe(p, depth+1) == 0 {
			delete(t.leafs, next)
		}
	}

	return t.refs
}

type ttsub struct {
	root *tt
	p    *path.Path
	s    Subscription
}

var _ Subscription = (*ttsub)(nil)

func (s *ttsub) C() <-chan interface{} {
	return s.s.C()
}

func (s *ttsub) Unsubscribe() bool {
	if !s.s.Unsubscribe() {
		return false
	}
	s.root.unsubscribe(s.p, 0)
	return true
}

func newtt() *tt {
	return &tt{
		topic: NewTopic(),
		leafs: make(map[string]*tt),
	}
}
