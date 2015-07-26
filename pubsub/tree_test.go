package pubsub

import (
	"testing"
)

func TestPathInit(t *testing.T) {
	var tests = []struct {
		path  string
		valid bool
	}{
		{"/", true},
		{"a", false},
		{"", false},
		{"a", false},
		{"/a", true},
		{"/abc", true},
		{"//", false},
		{"//abc", false},
		{"/a/", false},
		{"/a/b", true},
		{"/a/abc", true},
		{"/a/abc/", false},
		{"/a/abc//", false},
	}

	for _, test := range tests {
		p := path{path: []byte(test.path)}
		if gotValid := p.init(); gotValid != test.valid {
			t.Errorf("path=%q, got=%v, want=%v", test.path, gotValid, test.valid)
		}
	}
}

//
//func TestPathHead(t *testing.T) {
//	var tests = []struct {
//		path string
//		more bool
//	}{
//		{"/", false},
//		{"/a", true},
//		{"/a/b", true},
//	}
//
//	for _, test := range tests {
//		p := path{path: []byte(test.path)}
//		if gotHead, gotMore := p.head(); string(gotHead) != "" || gotMore != test.more {
//			t.Errorf("path=%q, got-head=%q, want-head=%q, got-more=%v, want-more=%v",
//				test.path, gotHead, "", gotMore, test.more)
//		}
//	}
//}
//
//func TestPathTail(t *testing.T) {
//	var tests = []struct {
//		path string
//
//		tail string
//		more bool
//	}{
//		{"/", "", false},
//		{"/a", "a", true},
//		{"/a/b", "b", true},
//	}
//
//	for _, test := range tests {
//		p := path{path: []byte(test.path)}
//		if gotTail, gotMore := p.tail(); string(gotTail) != test.tail || gotMore != test.more {
//			t.Errorf("path=%q, got-tail=%q, want-tail=%q, got-more=%v, want-more=%v",
//				test.path, gotTail, test.tail, gotMore, test.more)
//		}
//	}
//}
//
func TestPathElem(t *testing.T) {
	var tests = []struct {
		path string

		next []string
	}{
		{"/a", []string{"a"}},
		{"/a/b", []string{"a", "b"}},
		{"/abc/bcd/def", []string{"abc", "bcd", "def"}},
	}

	for _, test := range tests {
		p := path{path: []byte(test.path)}
		if p.init() == false {
			t.Errorf("invalid path: %q", test.path)
		}
		for i, wantNext := range test.next {
			wantMore := i+1 < len(test.next)
			if gotNext, gotMore := p.elem(i); string(gotNext) != wantNext || gotMore != wantMore {
				t.Errorf("path=%q, i=%d, got-next=%q, want-next=%q, got-more=%v, want-more=%v",
					test.path, i, gotNext, wantNext, gotMore, wantMore)
			}
		}
	}
}

func BenchmarkPathInitFitInScratchString(b *testing.B) {
	b.ReportAllocs()
	ok := pathInitString(b.N, "/some/path/with/less/than/10/slashes")
	if !ok {
		b.Fatal("path init failed")
	}
}

func BenchmarkPathInitFitInBytes(b *testing.B) {
	b.ReportAllocs()
	ok := pathInitByte(b.N, []byte("/some/path/with/less/than/10/slashes"))
	if !ok {
		b.Fatal("path init failed")
	}
}

func pathInitString(n int, s string) bool {
	ok := true
	var  p path
	for i := 0; i < n; i++ {
		ok =  p.initString(s) && ok
	}
	return ok
}
func pathInitByte(n int, b []byte) bool {
	ok := true
	var  p path
	for i := 0; i < n; i++ {
		ok =  p.initbyte(b) && ok
	}
	return ok
}
