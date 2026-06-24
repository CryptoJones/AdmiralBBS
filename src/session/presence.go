package session

import "sync"

// Presence enforces one live session per handle with NEWEST-WINS semantics: a
// new login for a handle that is already connected DISPLACES the older session
// rather than being rejected. This is the intuitive behavior — log in at work
// and your forgotten home session drops — and it self-heals stale/ghost
// sessions, since a fresh login always kicks a lingering dead one. It also still
// prevents one user from multiplying their daily time budget across many
// simultaneous sessions (there's only ever one).
type Presence struct {
	mu  sync.Mutex
	cur map[string]*token
	seq int64
}

type token struct {
	id      int64
	closeFn func()
}

// NewPresence builds an empty registry. The int argument is accepted for
// call-site compatibility but ignored — presence is always one-per-user.
func NewPresence(int) *Presence { return &Presence{cur: map[string]*token{}} }

// Enter registers a new session for handle and displaces any existing one by
// invoking its close function (called outside the lock so it can block on I/O).
// It returns a token id; pass it to Leave so a session that was itself kicked
// can't later evict the session that replaced it.
func (p *Presence) Enter(handle string, closeFn func()) int64 {
	p.mu.Lock()
	p.seq++
	id := p.seq
	prev := p.cur[handle]
	p.cur[handle] = &token{id: id, closeFn: closeFn}
	p.mu.Unlock()

	if prev != nil && prev.closeFn != nil {
		prev.closeFn() // disconnect the displaced session
	}
	return id
}

// Leave releases the slot only if id is still the current session for handle.
func (p *Presence) Leave(handle string, id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if t := p.cur[handle]; t != nil && t.id == id {
		delete(p.cur, handle)
	}
}
