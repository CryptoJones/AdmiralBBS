package tests

import "testing"

func TestReadPointersNewCount(t *testing.T) {
	s, _ := openTestStore(t)
	if err := s.EnsureSeedAreas(); err != nil {
		t.Fatal(err)
	}
	areas, _ := s.MessageAreas().Visible(50)
	area := areas[0]
	author, _ := s.Users().Create("author", "", "", "")
	visitor, _ := s.Users().Create("visitor", "", "", "")
	rp := s.ReadPointers()

	// Never-visited: everything is new.
	for i := 0; i < 3; i++ {
		s.Messages().Post(area.ID, author.ID, nil, "m", "b")
	}
	if n, _ := rp.NewCount(visitor.ID, area.ID); n != 3 {
		t.Fatalf("new count before any visit = %d, want 3", n)
	}

	// Mark the area read up to its newest post → nothing new.
	maxID, _ := rp.MaxMessageID(area.ID)
	if err := rp.Mark(visitor.ID, area.ID, maxID); err != nil {
		t.Fatal(err)
	}
	if n, _ := rp.NewCount(visitor.ID, area.ID); n != 0 {
		t.Fatalf("new count after marking read = %d, want 0", n)
	}

	// Two more posts arrive → exactly two new.
	s.Messages().Post(area.ID, author.ID, nil, "m4", "b")
	s.Messages().Post(area.ID, author.ID, nil, "m5", "b")
	if n, _ := rp.NewCount(visitor.ID, area.ID); n != 2 {
		t.Fatalf("new count after 2 more posts = %d, want 2", n)
	}

	// Mark never moves backward: a stale low value can't resurrect read posts.
	if err := rp.Mark(visitor.ID, area.ID, 1); err != nil {
		t.Fatal(err)
	}
	if got, _ := rp.LastSeen(visitor.ID, area.ID); got != maxID {
		t.Fatalf("pointer moved backward to %d, want %d", got, maxID)
	}
}
