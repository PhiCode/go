// Copyright 2015 Philipp Meinen. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package pubsub

import (
	"sync"
	"testing"
	"time"
)

func TestReceive(t *testing.T) {
	topic := NewTopic()
	sub := topic.Subscribe()
	topic.Publish("a")
	topic.Publish("b")
	a, ok := <-sub.C()
	if a != "a" || !ok {
		t.Errorf("subscription receive got (%v, %v), want: (a, true)", a, ok)
	}
	sub.Unsubscribe()
	// give the sender goroutine the opportunity to shut down
	time.Sleep(25 * time.Millisecond)
	b, ok := <-sub.C()
	if b != nil || ok {
		t.Errorf("unsubscribed subscription receive got (%v, %v), want: (nil, false)", b, ok)
	}
}

func TestNumSubscribers(t *testing.T) {
	topic := NewTopic()
	topic.Publish("stuff")
	sub := topic.Subscribe()
	if n := topic.NumSubscribers(); n != 1 {
		t.Fatalf("number of subscribers missmatch, got: %d, want: 1", n)
	}
	select {
	case x := <-sub.C():
		t.Fatalf("received unexpected: %v", x)
	default:
	}
	sub.Unsubscribe()
	if n := topic.NumSubscribers(); n != 0 {
		t.Fatalf("number of subscribers missmatch, got: %d, want: 0", n)
	}
}

func BenchmarkPublishReceive1kSub(b *testing.B)   { benchPublishReceive(b, 1000) }
func BenchmarkPublishReceive10kSub(b *testing.B)  { benchPublishReceive(b, 10000) }
func BenchmarkPublishReceive100kSub(b *testing.B) { benchPublishReceive(b, 100000) }

func benchPublishReceive(b *testing.B, consumers int) {
	topic := NewTopic()
	var wg sync.WaitGroup
	wg.Add(consumers)
	for j := 0; j < consumers; j++ {
		sub := topic.Subscribe()
		go benchConsumer(b, sub, &wg, b.N)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 1; i <= b.N; i++ {
		topic.Publish(i)
	}

	wg.Wait()
}

func benchConsumer(b *testing.B, sub Subscription, wg *sync.WaitGroup, numMsg int) {
	defer sub.Unsubscribe()
	var sum int
	for i := 0; i < numMsg; i++ {
		sum += (<-sub.C()).(int)
	}
	var wsum = (numMsg * (numMsg + 1)) / 2
	if sum != wsum {
		b.Errorf("invalid sum - got: %d, want: %d", sum, wsum)
	}
	wg.Done()
}
