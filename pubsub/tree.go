package pubsub

// subscription:
// - register a subscriber on the highest possible node
// - create tree nodes if necessary
//
// publishing:
// - walk the publishing path
// - publish on all tree nodes with non-empty list of subscribers

// A Topic is used to distribute messages to multiple consumers.
// Published messages are delivered in publishing order.
// If an optional "high water mark" (HWM) is set, messages for slow consumsers may be dropped.
type TopicTree interface {
	// Publish publishes a new message to all subscribers.
	Publish(path string, msg interface{})

	// Subscribe creates a new Subscription on which newly published messages can be received.
	Subscribe(path string) Subscription

	// NumSubscribers returns the current number of subscribers.
	NumSubscribers(path string) int

	// SetHWM sets a high water mark in number of messages.
	// Subscribers which lag behind more messages than the high water mark will have messages discarded.
	// A high water mark of zero means no messages will be discarded (default).
	// Changes to the high water mark only apply to newly published messages.
	SetHWM(path string, hwm int)
}

type path struct {
	path    []byte  //24b
	slashes []int   //24b
	scratch [10]int //80b
	//total: 64/128 b => 1/2 cachelines
}

const pathSeparator byte = '/'

func (p *path) initString(s string) bool {
	p.path = []byte(s)
	return p.init()
}

func (p *path) initbyte(b []byte) bool {
	p.path = b
	return p.init()
}


func (p *path) init() bool {
	path := p.path
	l := len(path)

	// at least the root path => '/'
	if l < 1 || path[0] != pathSeparator {
		return false
	}

	p.slashes = p.scratch[:0]
	last := -2

	for i, c := range path {
		if c != pathSeparator {
			continue
		}
		if i == last+1 {
			// double slash
			return false
		}
		last = i
		p.slashes = append(p.slashes, i)
	}
	return l == 1 || l > last+1
}

//func (p *path) head() ([]byte, bool) {
//	p.pos = 0
//	return []byte{}, len(p.path) > 1
//}
//
//func (p *path) tail() ([]byte, bool) {
//	p.pos = bytes.LastIndexByte(p.path[:len(p.path)-1], pathSeparator)
//	if p.pos == -1 {
//		return []byte{}, false
//	}
//	return p.path[p.pos+1:], true
//}
//
//func (p *path) resetStart() {
//	p.pos = -1
//}
//func (p *path) next() ([]byte, bool) {
//	if p.pos < 0 {
//		p.pos = 0
//		return []byte{}, len(p.path) > 1
//	}
//	start := p.pos + 1
//	p.pos = bytes.IndexByte(p.path[start:], pathSeparator)
//	if p.pos == -1 {
//		return p.path[start:], false
//	}
//	return p.path[start:p.pos], true
//}

func (p *path) elem(i int) ([]byte, bool) {
	s := len(p.slashes)
	if i >= s {
		panic("invalid element index")
	}

	if i+1 == s { // last element
		return p.path[p.slashes[i]+1:], false
	}

	return p.path[p.slashes[i]+1 : p.slashes[i+1]], true
}
