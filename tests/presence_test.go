package tests

import (
	"sync"
	"sync/atomic"
	"testing"

	"admiralbbs/src/session"
)

// A user can't exceed their concurrent-session cap (the 50-logins fix).
func TestPresenceCapsConcurrentLogins(t *testing.T) {
	p := session.NewPresence(1)
	if !p.Enter("alice") {
		t.Fatal("first login rejected")
	}
	if p.Enter("alice") {
		t.Fatal("second concurrent login allowed — budget bypass")
	}
	p.Leave("alice")
	if !p.Enter("alice") {
		t.Fatal("login rejected after leave")
	}
}

func TestPresenceConcurrentRace(t *testing.T) {
	p := session.NewPresence(3)
	var got int32
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if p.Enter("u") {
				atomic.AddInt32(&got, 1)
			}
		}()
	}
	wg.Wait()
	if got != 3 {
		t.Fatalf("admitted %d concurrent sessions, want 3", got)
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
