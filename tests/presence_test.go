package tests

import (
	"sync"
	"sync/atomic"
	"testing"

	"admiralbbs/src/session"
)

// Newest login wins: a second login for the same handle kicks the first, and a
// kicked session's later Leave must not evict its replacement.
func TestPresenceNewestWins(t *testing.T) {
	p := session.NewPresence(1)
	aKicked, bKicked, cKicked := false, false, false

	idA := p.Enter("alice", func() { aKicked = true })
	if aKicked {
		t.Fatal("the first login should not be kicked")
	}
	idB := p.Enter("alice", func() { bKicked = true })
	if !aKicked {
		t.Fatal("the second login should have kicked the first")
	}
	if bKicked {
		t.Fatal("the second login should not be kicked")
	}

	// Alice's first (kicked) session now unwinds and calls Leave with its OLD id.
	// That must NOT remove the current (second) session.
	p.Leave("alice", idA)

	// Proof B is still current: a third login kicks B.
	idC := p.Enter("alice", func() { cKicked = true })
	if !bKicked {
		t.Fatal("third login should kick the second — A's stale Leave wrongly evicted B")
	}
	_, _, _ = idB, idC, cKicked
	p.Leave("alice", idC)
}

// Under a race of concurrent logins for one handle, exactly one ends up current
// and every other is kicked exactly once (N-1 kicks).
func TestPresenceConcurrentRace(t *testing.T) {
	p := session.NewPresence(1)
	const n = 50
	var kicks int32
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Enter("u", func() { atomic.AddInt32(&kicks, 1) })
		}()
	}
	wg.Wait()
	if kicks != n-1 {
		t.Fatalf("kicks = %d, want %d (exactly one session survives)", kicks, n-1)
	}
}

func TestNodePoolUniqueAndBounded(t *testing.T) {
	np := session.NewNodePool(2)
	a, b := np.Acquire(), np.Acquire()
	if a == 0 || b == 0 || a == b {
		t.Fatalf("expected two distinct nodes, got %d,%d", a, b)
	}
	if c := np.Acquire(); c != 0 {
		t.Fatalf("over capacity: allocated node %d", c)
	}
	np.Release(a)
	if d := np.Acquire(); d != a {
		t.Fatalf("freed node not reused: got %d want %d", d, a)
	}
}
