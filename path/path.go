// Copyright 2015 Philipp Meinen. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in the LICENSE file.

package path

import (
	"path"
	"strings"
)

var emptyStringSlice = []string{}

const pathSeparator = '/'
const pathSeparatorString = "/"

func Split(p string) []string {
	if p == "" {
		return emptyStringSlice
	}
	p = path.Clean(p)
	l := len(p)

	if p == pathSeparatorString {
		return emptyStringSlice
	}
	if p[0] == pathSeparator {
		p = p[1:]
		l--
	}

	elems := strings.Count(p, pathSeparatorString) + 1
	var path = make([]string, elems)

	for i := 0; i < elems; i++ {
		end := strings.IndexByte(p, pathSeparator)
		if end == -1 {
			path[i] = p
		} else {
			path[i] = p[:end]
			p = p[end+1:]
		}
	}
	return path
}
