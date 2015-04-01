package pubsub

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSendReceive(t *testing.T) {
	topic := NewTopic()
	sub := topic.Subscribe()
	topic.Publish("test")
	if m := sub.Receive(); m != "test" {
		t.Errorf("Received: %q", m)
	}
}

func TestSendReceiveNow(t *testing.T) {
	topic := NewTopic()
	sub := topic.Subscribe()
	for i := 0; i < 100000; i++ {
		topic.Publish(i)
	}
	for i := 0; i < 100000; i++ {
		v, ok := sub.ReceiveNow()
		if v != i || !ok {
			t.Errorf("got: (%v, %v), want: (%v,%v)", v, ok, i, true)
		}
	}
	v, ok := sub.ReceiveNow()
	if v != nil || ok {
		t.Errorf("got: (%v, %v), want: (%v,%v)", v, ok, nil, false)
	}
}

func TestLatest(t *testing.T) {
	topic := NewTopic()
	sub := topic.Subscribe()
	topic.Publish("1")
	topic.Publish("2")
	if v := sub.Latest(); v != "2" {
		fmt.Errorf("got %q, want %q", v, "2")
	}
	v, ok := sub.LatestNow()
	if v != nil || ok {
		t.Errorf("got: (%v, %v), want: (%v,%v)", v, ok, nil, false)
	}
}

func TestReceiveTimeout(t *testing.T) {
	topic := NewTopic()
	a := topic.Subscribe()
	ready := make(chan bool, 1)
	done := make(chan bool, 1)
	go func() {
		ready <- true
		x, okx := a.ReceiveTimeout(time.Hour)
		y, oky := a.ReceiveTimeout(100 * time.Millisecond)
		done <- (x == 31415 && okx == true && y == nil && oky == false)
	}()
	<-ready
	time.Sleep(10 * time.Millisecond)
	topic.Publish(31415)
	if !<-done {
		t.Errorf("ReceiveTimeout is not done")
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
	var sum int
	for i := 0; i < numMsg; i++ {
		sum += sub.Receive().(int)
	}
	var wsum = (numMsg * (numMsg + 1)) / 2
	if sum != wsum {
		b.Errorf("invalid sum - got: %d, want: %d", sum, wsum)
	}
	wg.Done()
}
