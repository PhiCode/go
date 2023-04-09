// Copyright 2015 Philipp Meinen. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package pubsub

import (
	"sync"

	"github.com/phicode/go/path"
)

// A TopicTree is a hierarchical structure of topics.
// Subscriber can subscribe on:
// - the root level to receive all messages
// - a sub-level to receive a subset of messages, filtered for the specific path
// Published messages are delivered to the root and all path elements of the publishing path.
type TopicTree[M any] interface {
	// Publish publishes a new message to the tree.
	// Publish panics if the supplied path is not valid.
	Publish(path string, msg M)

	// PublishPath publishes a new message to the tree.
	PublishPath(path []string, msg M)

	// Subscribe creates a new Subscription on the tree.
	// Subscribe panics if the supplied path is not valid.
	Subscribe(path string) Subscription[M]

	// SubscribePath creates a new Subscription on the tree.
	SubscribePath(path []string) Subscription[M]

	//TODO: per level/subscriber/global? scrap altogether ?
	//SetHWM(path string, hwm int, recursive bool)
	//GetHWM(path string) (hwm int, recursive bool)

	//TODO: subscriptions where the path of the received message is communicated

	// List the entire topic tree and the amount of subscribers currently on a specific level.
	// Intermediate levels with no active subscribers are also listed (with a value of 0).
	List() map[string]int
}

type tt[M any] struct {
	// the topic which receives messages for this level
	topic Topic[M]

	rwmu sync.RWMutex
	// the number of subscriptions on this level and all child trees
	refs int64
	// all child nodes
	leafs map[string]*tt[M]
}

var _ TopicTree[any] = (*tt[any])(nil)

func NewTopicTree[M any]() TopicTree[M] { return newtt[M]() }

func (t *tt[M]) Publish(treePath string, msg M) {
	p := path.Split(treePath)
	t.publish(p, 0, msg)
}
func (t *tt[M]) PublishPath(path []string, msg M) {
	t.publish(path, 0, msg)
}

// publishing:
// - walk the tree until the destination leaf is reached
// - publish the message on the topics of all leafs on the way
func (t *tt[M]) publish(p []string, depth int, msg M) {

	// publish message on the current tree level
	t.topic.Publish(msg)

	t.rwmu.RLock()
	defer t.rwmu.RUnlock()

	if len(p) > depth && t.leafs != nil {
		elem := p[depth]
		if leaf, ok := t.leafs[elem]; ok {
			leaf.publish(p, depth+1, msg)
		}
	}
}

func (t *tt[M]) Subscribe(treePath string) Subscription[M] {
	p := path.Split(treePath)
	return t.subscribe(t, p, 0)
}
func (t *tt[M]) SubscribePath(path []string) Subscription[M] {
	// defensive copy for externally supplied paths on subscriptions
	p := make([]string, len(path))
	copy(p, path)

	return t.subscribe(t, p, 0)
}

// subscription:
// - walk the tree until the requested leaf is reached
// - create non-existent leafs on the way
// - register on the leafs topic
func (t *tt[M]) subscribe(root *tt[M], path []string, depth int) Subscription[M] {
	t.rwmu.Lock()
	defer t.rwmu.Unlock()

	t.refs++

	if len(path) > depth {
		// subscription is not for this level, propagate to the next level in the hierarchy
		elem := path[depth]
		if t.leafs == nil {
			t.leafs = make(map[string]*tt[M])
		}
		leaf, ok := t.leafs[elem]
		if !ok {
			leaf = newtt[M]()
			t.leafs[elem] = leaf
		}
		return leaf.subscribe(root, path, depth+1)
	}

	s := t.topic.Subscribe()
	return &ttsub[M]{
		root: root,
		p:    path,
		s:    s,
	}
}

// unsubscribe:
// - walk the tree until the final leaf is reached
// - unsubscribe from the leaf
// - remove all unused leafs
func (t *tt[M]) unsubscribe(path []string, depth int) int64 {
	t.rwmu.Lock()
	defer t.rwmu.Unlock()

	t.refs--

	if len(path) > depth && t.leafs != nil {
		// subscription is not for this level, propagate to the next level in the hierarchy
		elem := path[depth]
		leaf, ok := t.leafs[elem]
		if !ok {
			panic("invalid unsubscribe")
		}
		if leaf.unsubscribe(path, depth+1) == 0 {
			delete(t.leafs, elem)
		}
		if len(t.leafs) == 0 {
			t.leafs = nil
		}
	}

	return t.refs
}

func (t *tt[M]) List() map[string]int {
	m := make(map[string]int)
	t.list(m, "/")
	return m
}

func (t *tt[M]) list(m map[string]int, path string) {
	t.rwmu.RLock()
	defer t.rwmu.RUnlock()

	m[path] = t.topic.NumSubscribers()
	if len(path) > 1 {
		path += "/"
	}
	if t.leafs != nil {
		for leafPath, leaf := range t.leafs {
			leaf.list(m, path+leafPath)
		}
	}
}

type ttsub[M any] struct {
	root *tt[M]
	p    []string
	s    Subscription[M]
}

var _ Subscription[any] = (*ttsub[any])(nil)

func (s *ttsub[M]) C() <-chan M {
	return s.s.C()
}

func (s *ttsub[M]) Unsubscribe() bool {
	if !s.s.Unsubscribe() {
		return false
	}
	s.root.unsubscribe(s.p, 0)
	return true
}

func newtt[M any]() *tt[M] {
	return &tt[M]{
		topic: NewTopic[M](),
		//leafs: make(map[string]*tt),
	}
}
