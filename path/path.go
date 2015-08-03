// Copyright 2015 Philipp Meinen. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package path

type Path struct {
	elems   []string  // 24b
	scratch [6]string // 96b
	// total: 120 byte => slightly less than 2 cachelines
}

func Parse(p string) (Path, bool) {
	l := len(p)

	// at least the root path => '/'
	if l < 1 || p[0] != pathSeparator {
		return Path{}, false
	}
	if l == 1 {
		return Path{}, true
	}

	var path Path
	path.elems = path.scratch[:0]
	last := 0

	var i int
	var c rune
	for i, c = range p {
		if c != pathSeparator || i == 0 {
			continue
		}
		if i == last+1 {
			// double slash
			return Path{}, false
		}
		path.elems = append(path.elems, p[last+1:i])
		last = i
	}
	if c == pathSeparator {
		// no trailing slash
		return Path{}, false
	}
	path.elems = append(path.elems, p[last+1:])
	return path, true
}

const pathSeparator = '/'

func (p *Path) Elem(i int) (string, bool) {
	s := len(p.elems)
	if i >= s {
		return "", false
	}
	return p.elems[i], true
}
