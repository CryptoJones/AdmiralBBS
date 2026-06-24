package tests

import (
	"errors"
	"testing"

	"admiralbbs/src/store"
)

func TestMessageEditOwnershipAndDelete(t *testing.T) {
	s, _ := openTestStore(t)
	if err := s.EnsureSeedAreas(); err != nil {
		t.Fatal(err)
	}
	areas, _ := s.MessageAreas().Visible(50)
	area := areas[0]
	alice, _ := s.Users().Create("alice", "", "", "")
	bob, _ := s.Users().Create("bob", "", "", "")

	post, _ := s.Messages().Post(area.ID, alice.ID, nil, "orig", "body")

	// A non-owner can't edit.
	if err := s.Messages().Edit(post.ID, bob.ID, "hax", "x"); !errors.Is(err, store.ErrNotOwner) {
		t.Fatalf("non-owner edit: want ErrNotOwner, got %v", err)
	}
	// The owner can, and it persists (decrypts back to the new content).
	if err := s.Messages().Edit(post.ID, alice.ID, "edited", "new body"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Messages().ByID(post.ID)
	if got.Subject != "edited" || got.Body != "new body" {
		t.Fatalf("edit didn't persist: %+v", got)
	}

	// Delete removes the post and its replies.
	reply, _ := s.Messages().Post(area.ID, bob.ID, &post.ID, "re", "reply body")
	if err := s.Messages().Delete(post.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Messages().ByID(post.ID); err == nil {
		t.Error("deleted post still present")
	}
	if _, err := s.Messages().ByID(reply.ID); err == nil {
		t.Error("reply to deleted post should be gone too")
	}
}

func TestMailDeleteRecipientOnly(t *testing.T) {
	s, _ := openTestStore(t)
	alice, _ := s.Users().Create("alice", "", "", "")
	bob, _ := s.Users().Create("bob", "", "", "")

	m, _ := s.Mail().Send(alice.ID, bob.ID, "hi", "body") // alice -> bob

	// The sender (alice) cannot delete it from bob's mailbox.
	if err := s.Mail().Delete(m.ID, alice.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("sender delete: want ErrNotFound, got %v", err)
	}
	// The recipient (bob) can.
	if err := s.Mail().Delete(m.ID, bob.ID); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.Mail().UnreadCount(bob.ID); n != 0 {
		t.Fatalf("mail not deleted: unread=%d", n)
	}
}
