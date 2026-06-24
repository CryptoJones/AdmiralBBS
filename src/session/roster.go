package session

import (
	"sort"
	"sync"
	"time"
)

// Online is a snapshot of one connected caller, for the who's-online list.
type Online struct {
	Node      int
	Handle    string
	IP        string
	Transport string
	Since     time.Time
}

// Roster is the live registry of who is currently connected, keyed by node
// number. It powers the who's-online view. Safe for concurrent use.
type Roster struct {
	mu    sync.Mutex
	byNS  map[int]*Online
	clock Clock
}

// NewRoster builds an empty roster. clock may be nil (defaults to time.Now).
func NewRoster(clock Clock) *Roster {
	if clock == nil {
		clock = time.Now
	}
	return &Roster{byNS: make(map[int]*Online), clock: clock}
}

// Join records a caller as online on the given node.
func (r *Roster) Join(node int, handle, ip, transport string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byNS[node] = &Online{Node: node, Handle: handle, IP: ip, Transport: transport, Since: r.clock()}
}

// Leave removes a node from the roster.
func (r *Roster) Leave(node int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byNS, node)
}

// List returns a snapshot of everyone online, sorted by node number.
func (r *Roster) List() []Online {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Online, 0, len(r.byNS))
	for _, o := range r.byNS {
		out = append(out, *o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Node < out[j].Node })
	return out
}

// Count returns how many callers are online.
func (r *Roster) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byNS)
}
