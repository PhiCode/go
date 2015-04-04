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
		t.Fatalf("number of subscribers missmatch, got: %v, want: 1", n)
	}
	select {
	case x := <-sub.C():
		t.Fatalf("received unexpected message: %v", x)
	default:
	}
	sub.Unsubscribe()
	if n := topic.NumSubscribers(); n != 0 {
		t.Fatalf("number of subscribers missmatch, got: %v, want: 0", n)
	}
}

func TestMessageOrder(t *testing.T) {
	const N = 1000
	topic := NewTopic()
	sub := topic.Subscribe()
	go func() {
		for i := 0; i < N; i++ {
			topic.Publish(i)
		}
	}()
	verifyNextMessages(t, sub, 0, N)
}

func TestHighWaterMark(t *testing.T) {
	topic := NewTopic()
	topic.SetHWM(1)

	s := topic.Subscribe()
	// "tune" the high water mark check so that the test runs faster
	s.(*sub).hwmTicker = time.NewTicker(time.Millisecond)

	topic.Publish(1)
	topic.Publish(2)

	// wait for the high water mark check
	time.Sleep(5 * time.Millisecond)

	if got := <-s.C(); got != 2 {
		t.Errorf("high water mark discard failed - got: %v, want: 2", got)
	}

	topic.SetHWM(10)
	for i := 0; i < 100; i++ {
		topic.Publish(i)
	}

	// wait for the high water mark check
	time.Sleep(5 * time.Millisecond)
	verifyNextMessages(t, s, 90, 100)
}

func verifyNextMessages(t *testing.T, sub Subscription, from, to int) {
	c := sub.C()
	for i := from; i < to; i++ {
		select {
		case got := <-c:
			if got != i {
				t.Fatalf("wrong delivery order - got: %v, want: %v", got, i)
			}
		case <-time.After(time.Second):
			t.Fatalf("missing message: %v", i)
		}
	}
	select {
	case x := <-c:
		t.Fatalf("received unexpected message: %v", x)
	default:
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
	c := sub.C()
	for i := 0; i < numMsg; i++ {
		sum += (<-c).(int)
	}
	var wsum = (numMsg * (numMsg + 1)) / 2
	if sum != wsum {
		b.Errorf("invalid sum - got: %v, want: %v", sum, wsum)
	}
	wg.Done()
}
