package pubsub

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"
)

func verifyTopicTree(t *testing.T, tt TopicTree, m map[string]int) {
	got := tt.List()
	//t.Logf("%+v",got)
	if len(got) != len(m) {
		t.Errorf("topic tree length missmatch, got=%d, want=%d", len(got), len(m))
	}

	for k, v := range m {
		if gotV, ok := got[k]; ok {
			if gotV != v {
				t.Errorf("topic tree subscribers missmatch, path=%q, got=%d, want=%d", k, gotV, v)
			}
		} else {
			t.Errorf("topic tree subscribers missing for path=%q, want=%d", k, v)
		}
	}
}

func TestTopicTreeRoot(t *testing.T) {
	tt := NewTopicTree()
	sub := tt.Subscribe("/")
	if sub == nil {
		t.Fatal("subscribe failed")
	}
	verifyTopicTree(t, tt, map[string]int{"/": 1})

	tt.Publish("/", 1)
	tt.Publish("/a", 2)
	tt.Publish("/b", 3)

	verifyNextMessages(t, sub, 1, 2, 3)

	sub.Unsubscribe()

	verifyTopicTree(t, tt, map[string]int{"/": 0})

	tt.Publish("/", "stuff")

	select {
	case got, ok := <-sub.C():
		if ok {
			t.Errorf("received message on unsubscribed subscriber: %v", got)
		}
	case <-time.After(25 * time.Millisecond):
	}
}

func TestTopicTreeSubNode(t *testing.T) {
	tt := NewTopicTree()
	sub := tt.Subscribe("/a")
	if sub == nil {
		t.Fatal("subscribe failed")
	}
	defer sub.Unsubscribe()
	verifyTopicTree(t, tt, map[string]int{"/": 0, "/a": 1})

	tt.Publish("/", "root")
	tt.Publish("/a", "a")
	tt.Publish("/b", "b")

	c := sub.C()
	select {
	case got := <-c:
		if got != "a" {
			t.Fatalf("wrong message, got=%q, want=\"a\"", got)
		}
	case <-time.After(time.Second):
		t.Fatalf("missing message: \"a\"")
	}
}

func TestTopicDoubleUnsubscribe(t *testing.T) {
	tt := NewTopicTree()
	sub1 := tt.Subscribe("/a")
	sub2 := tt.Subscribe("/a")
	if sub1 == nil || sub2 == nil {
		t.Fatal("subscribe failed")
	}
	verifyTopicTree(t, tt, map[string]int{"/": 0, "/a": 2})

	tt.Publish("/a", "a")

	if !sub1.Unsubscribe() {
		t.Error("unsubscribe failed")
	}
	verifyTopicTree(t, tt, map[string]int{"/": 0, "/a": 1})
	if sub1.Unsubscribe() {
		t.Error("double unsubscribe succeeded")
	}
	defer sub2.Unsubscribe()
	verifyTopicTree(t, tt, map[string]int{"/": 0, "/a": 1})

	c := sub2.C()
	select {
	case got := <-c:
		if got != "a" {
			t.Fatalf("wrong message, got=%q, want=\"a\"", got)
		}
	case <-time.After(time.Second):
		t.Fatalf("missing message: \"a\"")
	}
}

func TestTopicTreeList(t *testing.T) {
	tt := NewTopicTree()
	verifyTopicTree(t, tt, map[string]int{"/": 0})

	abc := tt.Subscribe("/a/b/c")
	verifyTopicTree(t, tt, map[string]int{
		"/":      0,
		"/a":     0,
		"/a/b":   0,
		"/a/b/c": 1,
	},
	)

	bc := tt.Subscribe("/b/c")
	verifyTopicTree(t, tt, map[string]int{
		"/":      0,
		"/a":     0,
		"/a/b":   0,
		"/a/b/c": 1,
		"/b":     0,
		"/b/c":   1,
	},
	)

	b := tt.Subscribe("/b")
	verifyTopicTree(t, tt, map[string]int{
		"/":      0,
		"/a":     0,
		"/a/b":   0,
		"/a/b/c": 1,
		"/b":     1,
		"/b/c":   1,
	},
	)

	abc.Unsubscribe()
	verifyTopicTree(t, tt, map[string]int{
		"/":    0,
		"/b":   1,
		"/b/c": 1,
	},
	)

	bc.Unsubscribe()

	verifyTopicTree(t, tt, map[string]int{
		"/":  0,
		"/b": 1,
	},
	)

	b.Unsubscribe()
	verifyTopicTree(t, tt, map[string]int{"/": 0})
}

func TestPublishOnAllLevels(t *testing.T) {
	tt := NewTopicTree()
	verifyTopicTree(t, tt, map[string]int{"/": 0})

	a := tt.Subscribe("/a")
	ab := tt.Subscribe("/a/b")
	abc := tt.Subscribe("/a/b/c")

	tt.Publish("a/b/c/d", 1)
	tt.Publish("a/b/c", 2)
	tt.Publish("a/b", 3)
	tt.Publish("a", 4)

	verifyNextMessages(t, a, 1, 2, 3, 4)
	verifyNextMessages(t, ab, 1, 2, 3)
	verifyNextMessages(t, abc, 1, 2)

	a.Unsubscribe()
	ab.Unsubscribe()
	abc.Unsubscribe()
}

func TestPublishSubscribeWithPath(t *testing.T) {
	tt := NewTopicTree()
	verifyTopicTree(t, tt, map[string]int{"/": 0})

	ab := tt.SubscribePath([]string{"a", "b"})
	xy := tt.SubscribePath([]string{"x", "y"})

	all1 := tt.SubscribePath(nil)
	all2 := tt.SubscribePath([]string{})

	verifyTopicTree(t, tt, map[string]int{
		"/":    2,
		"/a":   0,
		"/a/b": 1,
		"/x":   0,
		"/x/y": 1,
	})

	tt.PublishPath(nil, 1)
	tt.PublishPath([]string{"no-listeners"}, 2)
	tt.PublishPath([]string{"a", "b"}, 3)
	tt.PublishPath([]string{"x", "y"}, 4)

	verifyNextMessages(t, ab, 3)
	verifyNextMessages(t, xy, 4)

	verifyNextMessages(t, all1, 1, 2, 3, 4)
	verifyNextMessages(t, all2, 1, 2, 3, 4)
}

const debugGoroutines = false

func TestMain(m *testing.M) {
	rv := m.Run()
	if debugGoroutines {
		time.Sleep(50 * time.Millisecond)
		printGoroutines()
	}
	os.Exit(rv)
}

func printGoroutines() {
	buf := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, len(buf)*2)
	}
	fmt.Printf("goroutines:\n%s\n", buf)
}
