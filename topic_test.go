package pubsub

import (
	"sync"
	"testing"
	"time"
)

func TestReceive(t *testing.T) {
	topic := NewTopic[string]()
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
	if ok {
		t.Errorf("unsubscribed subscription receive got (%v, %v), want: (nil, false)", b, ok)
	}
}

func TestNumSubscribers(t *testing.T) {
	topic := NewTopic[string]()
	topic.Publish("stuff")
	sub1 := topic.Subscribe()
	if n := topic.NumSubscribers(); n != 1 {
		t.Fatalf("number of subscribers missmatch, got: %v, want: 1", n)
	}
	select {
	case x := <-sub1.C():
		t.Fatalf("received unexpected message: %v", x)
	default:
	}

	sub2 := topic.Subscribe()
	if n := topic.NumSubscribers(); n != 2 {
		t.Fatalf("number of subscribers missmatch, got: %v, want: 2", n)
	}

	sub1.Unsubscribe()
	sub1.Unsubscribe()
	if n := topic.NumSubscribers(); n != 1 {
		t.Fatalf("number of subscribers missmatch, got: %v, want: 1", n)
	}

	sub2.Unsubscribe()
	if n := topic.NumSubscribers(); n != 0 {
		t.Fatalf("number of subscribers missmatch, got: %v, want: 0", n)
	}
}

func TestMessageOrder(t *testing.T) {
	const N = 1000
	topic := NewTopic[int]()
	sub := topic.Subscribe()
	defer sub.Unsubscribe()
	go func() {
		for i := 0; i < N; i++ {
			topic.Publish(i)
		}
	}()
	verifyNextMessagesRange(t, sub, 0, N)
}

func TestHighWaterMark(t *testing.T) {
	topic := NewTopic[int]()
	topic.SetHWM(1)

	sub := topic.Subscribe()
	defer sub.Unsubscribe()

	topic.Publish(1)
	topic.Publish(2)

	// ensure that the subscriber goroutine has time to discard messages over the HWM
	time.Sleep(5 * time.Millisecond)

	if got := <-sub.C(); got != 2 {
		t.Errorf("high water mark discard failed - got: %v, want: 2", got)
	}

	topic.SetHWM(10)
	for i := 0; i < 100; i++ {
		topic.Publish(i)
	}

	// ensure that the subscriber goroutine has time to discard messages over the HWM
	time.Sleep(5 * time.Millisecond)

	verifyNextMessagesRange(t, sub, 90, 100)
}

func verifyNextMessagesRange(t *testing.T, sub Subscription[int], from, to int) {
	t.Helper()
	for msg := from; msg < to; msg++ {
		verifyNextMessage(t, sub, msg)
	}
	select {
	case x := <-sub.C():
		t.Fatalf("received unexpected message: %v", x)
	default:
	}
}

func verifyNextMessages(t *testing.T, sub Subscription[int], messages ...int) {
	t.Helper()
	for _, msg := range messages {
		verifyNextMessage(t, sub, msg)
	}
	select {
	case x := <-sub.C():
		t.Fatalf("received unexpected message: %v", x)
	default:
	}
}

func verifyNextMessage(t *testing.T, sub Subscription[int], msg int) {
	t.Helper()
	select {
	case got := <-sub.C():
		if got != msg {
			t.Fatalf("wrong delivery order - got: %v, want: %v", got, msg)
		}
	case <-time.After(time.Second):
		t.Fatalf("missing message: %v", msg)
	}
}

func BenchmarkPublishReceive1kSub(b *testing.B)   { benchPublishReceive(b, 1000) }
func BenchmarkPublishReceive10kSub(b *testing.B)  { benchPublishReceive(b, 10000) }
func BenchmarkPublishReceive50kSub(b *testing.B)  { benchPublishReceive(b, 50000) }
func BenchmarkPublishReceive100kSub(b *testing.B) { benchPublishReceive(b, 100000) }

func benchPublishReceive(b *testing.B, consumers int) {
	topic := NewTopic[int]()
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

func benchConsumer(b *testing.B, sub Subscription[int], wg *sync.WaitGroup, numMsg int) {
	defer sub.Unsubscribe()
	var sum int
	c := sub.C()
	for i := 0; i < numMsg; i++ {
		sum += <-c
	}
	var wsum = (numMsg * (numMsg + 1)) / 2
	if sum != wsum {
		b.Errorf("invalid sum - got: %v, want: %v", sum, wsum)
	}
	wg.Done()
}
