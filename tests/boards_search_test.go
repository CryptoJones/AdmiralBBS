package tests

import (
	"testing"
)

// Board search (over encrypted content), sort-by-date, and filter-by-user.
func TestBoardSearchSortFilter(t *testing.T) {
	s, _ := openTestStore(t)
	area, _ := s.MessageAreas().Create("General", "", 0)
	alice, _ := s.Users().Create("alice", "", "", "")
	bob, _ := s.Users().Create("bob", "", "", "")

	s.Messages().Post(area.ID, alice.ID, nil, "Hello world", "I love cats")
	s.Messages().Post(area.ID, bob.ID, nil, "Goodbye", "I prefer DOGS")

	// Search decrypts + scans (subject/body are ciphertext at rest).
	if r, _ := s.Messages().Search(area.ID, "dogs"); len(r) != 1 || r[0].AuthorID != bob.ID {
		t.Fatalf("search 'dogs' = %+v", r)
	}
	if r, _ := s.Messages().Search(area.ID, "love"); len(r) != 1 || r[0].AuthorID != alice.ID {
		t.Fatalf("search 'love' = %+v", r)
	}

	// Sort by date.
	asc, _ := s.Messages().ThreadSorted(area.ID, false)
	desc, _ := s.Messages().ThreadSorted(area.ID, true)
	if len(asc) != 2 || asc[0].Subject != "Hello world" {
		t.Fatalf("oldest-first wrong: %+v", asc)
	}
	if len(desc) != 2 || desc[0].Subject != "Goodbye" {
		t.Fatalf("newest-first wrong: %+v", desc)
	}

	// Filter by user.
	byAlice, _ := s.Messages().ByAuthor(area.ID, alice.ID)
	if len(byAlice) != 1 || byAlice[0].AuthorID != alice.ID {
		t.Fatalf("filter by alice = %+v", byAlice)
	}
}
