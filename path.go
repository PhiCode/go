package pubsub

import (
	"path"
	"strings"
)

var emptyStringSlice = []string{}

const pathSeparator = '/'
const pathSeparatorString = "/"

type Path []string

func ParsePath(p string) Path {
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
	parsed := make(Path, elems)

	for i := 0; i < elems; i++ {
		end := strings.IndexByte(p, pathSeparator)
		if end == -1 {
			parsed[i] = p
		} else {
			parsed[i] = p[:end]
			p = p[end+1:]
		}
	}
	return parsed
}
