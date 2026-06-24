package tests

import (
	"testing"
)

func TestBlocksMuteAndList(t *testing.T) {
	s, _ := openTestStore(t)
	alice, _ := s.Users().Create("alice", "", "", "")
	bob, _ := s.Users().Create("bob", "", "", "")
	carol, _ := s.Users().Create("carol", "", "", "")
	blocks := s.Blocks()

	if err := blocks.Block(alice.ID, bob.ID); err != nil {
		t.Fatal(err)
	}
	// Idempotent — re-blocking is a no-op, not an error or a dup row.
	if err := blocks.Block(alice.ID, bob.ID); err != nil {
		t.Fatal(err)
	}
	// Can't block yourself.
	if err := blocks.Block(alice.ID, alice.ID); err != nil {
		t.Fatal(err)
	}

	if ok, _ := blocks.IsBlocked(alice.ID, bob.ID); !ok {
		t.Fatal("alice should have bob blocked")
	}
	if ok, _ := blocks.IsBlocked(alice.ID, carol.ID); ok {
		t.Fatal("carol should not be blocked")
	}
	if ok, _ := blocks.IsBlocked(alice.ID, alice.ID); ok {
		t.Fatal("self-block should not have been recorded")
	}

	set, _ := blocks.BlockedSet(alice.ID)
	if len(set) != 1 || !set[bob.ID] {
		t.Fatalf("blocked set = %v, want {bob}", set)
	}
	list, _ := blocks.List(alice.ID)
	if len(list) != 1 || list[0] != bob.ID {
		t.Fatalf("list = %v, want [bob]", list)
	}

	// Block is one-directional: bob does not have alice blocked.
	if ok, _ := blocks.IsBlocked(bob.ID, alice.ID); ok {
		t.Fatal("block should be one-directional")
	}

	if err := blocks.Unblock(alice.ID, bob.ID); err != nil {
		t.Fatal(err)
	}
	if ok, _ := blocks.IsBlocked(alice.ID, bob.ID); ok {
		t.Fatal("unblock did not take effect")
	}
}

func TestReportsQueueLifecycle(t *testing.T) {
	s, _ := openTestStore(t)
	alice, _ := s.Users().Create("alice", "", "", "")
	bob, _ := s.Users().Create("bob", "", "", "")
	sysop, _ := s.Users().Create("sysop", "", "", "")
	reports := s.Reports()

	r, err := reports.File(alice.ID, bob.ID, "mail #5", "bob is sending threats")
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := reports.OpenCount(); n != 1 {
		t.Fatalf("open count = %d, want 1", n)
	}
	open, _ := reports.Open()
	if len(open) != 1 || open[0].TargetID != bob.ID || open[0].Context != "mail #5" {
		t.Fatalf("open reports unexpected: %+v", open)
	}

	if err := reports.Resolve(r.ID, sysop.ID); err != nil {
		t.Fatal(err)
	}
	if n, _ := reports.OpenCount(); n != 0 {
		t.Fatalf("open count after resolve = %d, want 0", n)
	}
	// Resolving again is a harmless no-op.
	if err := reports.Resolve(r.ID, sysop.ID); err != nil {
		t.Fatal(err)
	}
}
