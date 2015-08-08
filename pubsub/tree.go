// Copyright 2015 Philipp Meinen. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package pubsub

import (
	"fmt"
	"sync"

	"github.com/phicode/go/path"
)

// A TopicTree is a hierarchical structure of topics.
// Subscriber can subscribe on:
// - the root level to receive all messages
// - a 'leaf' level to receive a subset of messages, filtered for the specific path
// Published messages delivered to the root and all leafs of the publishing path.
type TopicTree interface {
	// Publish publishes a new message to the tree.
	// Publish panics if the supplied path is not valid.
	Publish(path string, msg interface{})

	// PublishPath publishes a new message to the tree.
	PublishPath(path *path.Path, msg interface{})

	// Subscribe creates a new Subscription on the tree.
	// Subscribe panics if the supplied path is not valid.
	Subscribe(path string) Subscription

	// SubscribePath creates a new Subscription on the tree.
	SubscribePath(path *path.Path) Subscription

	//TODO: per level/subscriber/global? scrap altogether ?
	//NumSubscribers(path string) int

	//TODO: per level/subscriber/global? scrap altogether ?
	//SetHWM(path string, hwm int, recursive bool)

	//TODO: subscriptions where the path of the received message is communicated

	List() map[string]int
}

type tt struct {
	topic Topic

	rwmu    sync.RWMutex
	refs  int64 // the number of subscriptions on this level or on child trees
	leafs map[string]*tt
}

var _ (TopicTree) = (*tt)(nil)

func NewTopicTree() TopicTree { return newtt() }

func (t *tt) Publish(treePath string, msg interface{}) {
	path, ok := path.Parse(treePath)
	if !ok {
		panic(fmt.Errorf("invalid path: %q", treePath))
	}
	t.PublishPath(&path, msg)
}
func (t *tt) PublishPath(path *path.Path, msg interface{}) {
	t.publish(path, 0, msg)
}

// publishing:
// - walk the tree until the final leaf is reached
// - publish the message on the topics of all leafs on the way
func (t *tt) publish(p *path.Path, depth int, msg interface{}) {
	t.topic.Publish(msg)

	t.rwmu.RLock()
	defer t.rwmu.RUnlock()

	if next, ok := p.Elem(depth); ok {
		if leaf, ok := t.leafs[next]; ok {
			leaf.publish(p, depth+1, msg)
		}
	}
}

func (t *tt) Subscribe(treePath string) Subscription {
	path, ok := path.Parse(treePath)
	if !ok {
		panic(fmt.Errorf("invalid path: %q", treePath))
	}
	return t.SubscribePath(&path)
}
func (t *tt) SubscribePath(path *path.Path) Subscription {
	return t.subscribe(t, path, 0)
}

// subscription:
// - walk the tree until the requested leaf is reached
// - create non-existent leafs on the way
// - register on the leafs topic
func (t *tt) subscribe(root *tt, p *path.Path, depth int) Subscription {
	t.rwmu.Lock()
	defer t.rwmu.Unlock()

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

// unsubscribe:
// - walk the tree until the final leaf is reached
// - unsubscribe from the leaf
// - remove all unused leafs
func (t *tt) unsubscribe(p *path.Path, depth int) int64 {
	t.rwmu.Lock()
	defer t.rwmu.Unlock()

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

func (t *tt) List() map[string]int {
	m := make(map[string]int)
	t.list(m, "/")
	return m
}

func (t *tt) list(m map[string]int, path string) {
	t.rwmu.RLock()
	defer t.rwmu.RUnlock()

	m[path] = t.topic.NumSubscribers()
	if len(path) > 1 {
		path += "/"
	}
	for leafPath, leaf := range t.leafs {
		leaf.list(m, path+leafPath)
	}
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
