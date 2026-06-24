package session

import "sync"

// NodePool hands out unique node numbers (1..max) to concurrent sessions — the
// classic BBS "node" / line number. A unique node per session keeps concurrent
// players of the same door from colliding (door data is namespaced per node) and
// bounds the total number of simultaneous callers.
type NodePool struct {
	mu   sync.Mutex
	used map[int]bool
	max  int
}

// NewNodePool allows max concurrent nodes (max <= 0 => 64).
func NewNodePool(max int) *NodePool {
	if max <= 0 {
		max = 64
	}
	return &NodePool{used: make(map[int]bool), max: max}
}

// Acquire returns a free node number, or 0 if all nodes are busy.
func (p *NodePool) Acquire() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	for n := 1; n <= p.max; n++ {
		if !p.used[n] {
			p.used[n] = true
			return n
		}
	}
	return 0
}

// Release frees a node number.
func (p *NodePool) Release(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.used, n)
}
