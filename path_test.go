package pubsub

import (
	"testing"
)

func TestParsePath(t *testing.T) {
	var tests = []struct {
		path string

		out Path
	}{
		{"", []string{}},
		{"/", []string{}},
		{"//", []string{}},

		{"a", []string{"a"}},
		{"/a", []string{"a"}},
		{"/a/", []string{"a"}},
		{"//a//", []string{"a"}},

		{"a/b", []string{"a", "b"}},
		{"/a/b", []string{"a", "b"}},
		{"/a//b/", []string{"a", "b"}},

		{"/abc/bcd/def", []string{"abc", "bcd", "def"}},
		{"/ä/ö/ü/···", []string{"ä", "ö", "ü", "···"}},
	}

	for _, test := range tests {
		path := ParsePath(test.path)
		if len(test.out) != len(path) {
			t.Errorf("invalid path length, want=%d (%v), got=%d (%v)", len(test.out), test.out, len(path), path)
			continue
		}
		for i, want := range test.out {
			if got := path[i]; got != want {
				t.Errorf("path=%q, i=%d, got=%q, want=%q", test.path, i, got, want)
			}
		}
	}
}
