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
// - a 'leaf' level to receive a subset of messages, filtered for the specific path
// Published messages are delivered to the root and all leafs of the publishing path.
type TopicTree interface {
	// Publish publishes a new message to the tree.
	// Publish panics if the supplied path is not valid.
	Publish(path string, msg interface{})

	// PublishPath publishes a new message to the tree.
	PublishPath(path []string, msg interface{})

	// Subscribe creates a new Subscription on the tree.
	// Subscribe panics if the supplied path is not valid.
	Subscribe(path string) Subscription

	// SubscribePath creates a new Subscription on the tree.
	SubscribePath(path []string) Subscription

	//TODO: per level/subscriber/global? scrap altogether ?
	//SetHWM(path string, hwm int, recursive bool)
	//GetHWM(path string) (hwm int, recursive bool)

	//TODO: subscriptions where the path of the received message is communicated

	// List the entire topic tree and the amount of subscribers currently on a specific level.
	// Intermediate levels with no active subscribers are also listed (with a value of 0).
	List() map[string]int
}

type tt struct {
	// the topic which receives messages for this level
	topic Topic

	rwmu sync.RWMutex
	// the number of subscriptions on this level and all child trees
	refs int64
	// all child nodes
	leafs map[string]*tt
}

var _ (TopicTree) = (*tt)(nil)

func NewTopicTree() TopicTree { return newtt() }

func (t *tt) Publish(treePath string, msg interface{}) {
	path := path.Split(treePath)
	t.publish(path, 0, msg)
}
func (t *tt) PublishPath(path []string, msg interface{}) {
	t.publish(path, 0, msg)
}

// publishing:
// - walk the tree until the destination leaf is reached
// - publish the message on the topics of all leafs on the way
func (t *tt) publish(p []string, depth int, msg interface{}) {

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

func (t *tt) Subscribe(treePath string) Subscription {
	path := path.Split(treePath)
	return t.subscribe(t, path, 0)
}
func (t *tt) SubscribePath(path []string) Subscription {
	// defensive copy for externally supplied paths on subscriptions
	p := make([]string, len(path))
	copy(p, path)

	return t.subscribe(t, p, 0)
}

// subscription:
// - walk the tree until the requested leaf is reached
// - create non-existent leafs on the way
// - register on the leafs topic
func (t *tt) subscribe(root *tt, path []string, depth int) Subscription {
	t.rwmu.Lock()
	defer t.rwmu.Unlock()

	t.refs++

	if len(path) > depth {
		// subscription is not for this level, propagate to the next level in the hierarchy
		elem := path[depth]
		if t.leafs == nil {
			t.leafs = make(map[string]*tt)
		}
		leaf, ok := t.leafs[elem]
		if !ok {
			leaf = newtt()
			t.leafs[elem] = leaf
		}
		return leaf.subscribe(root, path, depth+1)
	}

	s := t.topic.Subscribe()
	return &ttsub{
		root: root,
		p:    path,
		s:    s,
	}
}

// unsubscribe:
// - walk the tree until the final leaf is reached
// - unsubscribe from the leaf
// - remove all unused leafs
func (t *tt) unsubscribe(path []string, depth int) int64 {
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
	if t.leafs != nil {
		for leafPath, leaf := range t.leafs {
			leaf.list(m, path+leafPath)
		}
	}
}

type ttsub struct {
	root *tt
	p    []string
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
		//leafs: make(map[string]*tt),
	}
}
