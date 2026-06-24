package session

import "sync"

// Presence tracks how many concurrent sessions each handle has, and caps it.
// Without this, one user could open unlimited simultaneous sessions — wasting
// resources and, critically, multiplying their daily time budget (each session
// independently grants the full allowance). Default cap is one session per user
// (classic BBS "one node per caller").
type Presence struct {
	mu  sync.Mutex
	n   map[string]int
	max int
}

// NewPresence caps each handle to max concurrent sessions (max <= 0 => 1).
func NewPresence(max int) *Presence {
	if max <= 0 {
		max = 1
	}
	return &Presence{n: make(map[string]int), max: max}
}

// Max reports the per-user limit.
func (p *Presence) Max() int { return p.max }

// Enter reserves a session slot for handle, returning false if the user is
// already at the limit.
func (p *Presence) Enter(handle string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.n[handle] >= p.max {
		return false
	}
	p.n[handle]++
	return true
}

// Leave releases a session slot for handle.
func (p *Presence) Leave(handle string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.n[handle] > 0 {
		p.n[handle]--
		if p.n[handle] == 0 {
			delete(p.n, handle)
		}
	}
}
