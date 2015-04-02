package pubsub

import (
    "testing"
    "runtime"
    "sync"
)

func TestReceive(t *testing.T) {
	topic := NewChanTopic()
	sub := topic.Subscribe()
	topic.Publish("a")
	topic.Publish("b")
	a, ok := <-sub.C()
	if a != "a" || !ok {
		t.Errorf("subscription receive got (%v, %v), want: (a, true)", a, ok)
	}
	sub.Unsubscribe()
    runtime.Gosched()
	b, ok := <-sub.C()
	if b != nil || ok {
		t.Errorf("unsubscribed subscription receive got (%v, %v), want: (nil, false)", b, ok)
	}
}


func BenchmarkChanPublishReceive1kSub(b *testing.B)   { benchChanPublishReceive(b, 1000) }
func BenchmarkChanPublishReceive10kSub(b *testing.B)  { benchChanPublishReceive(b, 10000) }
func BenchmarkChanPublishReceive100kSub(b *testing.B) { benchChanPublishReceive(b, 100000) }

func benchChanPublishReceive(b *testing.B, consumers int) {
    topic := NewChanTopic()
    var wg sync.WaitGroup
    wg.Add(consumers)
    for j := 0; j < consumers; j++ {
        sub := topic.Subscribe()
        go benchChanConsumer(b, sub, &wg, b.N)
    }

    b.ReportAllocs()
    b.ResetTimer()

    for i := 1; i <= b.N; i++ {
        topic.Publish(i)
    }

    wg.Wait()
}

func benchChanConsumer(b *testing.B, sub ChanSub, wg *sync.WaitGroup, numMsg int) {
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
