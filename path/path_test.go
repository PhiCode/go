// Copyright 2015 Philipp Meinen. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package path

import (
	"testing"
)

func TestSplit(t *testing.T) {
	var tests = []struct {
		path string

		out []string
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
		path := Split(test.path)
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
