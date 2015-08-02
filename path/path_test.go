package path

import (
	"testing"
)

func TestParse(t *testing.T) {
	var tests = []struct {
		path string
		ok   bool
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
		_, ok := Parse(test.path)
		if ok != test.ok {
			t.Errorf("path=%q, got=%v, want=%v", test.path, ok, test.ok)
		}
	}
}

func TestElem(t *testing.T) {
	var tests = []struct {
		path string

		next []string
	}{
		{"/", []string{}},
		{"/a", []string{"a"}},
		{"/a/b", []string{"a", "b"}},
		{"/abc/bcd/def", []string{"abc", "bcd", "def"}},
		{"/ä/ö/ü/···", []string{"ä", "ö", "ü", "···"}},
	}

	for _, test := range tests {
		path, ok := Parse(test.path)
		if !ok {
			t.Errorf("invalid path: %q", test.path)
		}
		for i, wantElem := range test.next {
			if gotElem, ok := path.Elem(i); gotElem != wantElem || !ok {
				t.Errorf("path=%q, i=%d, got-elem=%q, want-elem=%q, got-ok=%v, want-ok=true",
					test.path, i, gotElem, wantElem, ok)
			}
		}
		if elem, ok := path.Elem(len(test.next)); elem != "" || ok {
			t.Errorf("read past path end succeeded, got=(%v, %v), want=(nil, false)", elem, ok)
		}
	}
}

func BenchmarkParseFitInScratch(b *testing.B) {
	b.ReportAllocs()
	ok := nTimesParse(b.N, "/path/with/six/elements/uses/scratchpad")
	if !ok {
		b.Fatal("path init failed")
	}
}

func BenchmarkParseTooLongForScratch(b *testing.B) {
	b.ReportAllocs()
	ok := nTimesParse(b.N, "/path/with/more/than/six/elements/requires/additional/allocations")
	if !ok {
		b.Fatal("path init failed")
	}
}

func nTimesParse(n int, s string) bool {
	allok := true
	for i := 0; i < n; i++ {
		_, ok := Parse(s)
		allok = allok && ok
	}
	return allok
}
