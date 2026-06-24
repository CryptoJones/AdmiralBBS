package tests

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"admiralbbs/src/store"
	"admiralbbs/src/xfer"
)

func TestFileDuplicateNameRejected(t *testing.T) {
	s, _ := openTestStore(t)
	a, _ := s.FileAreas().Create("General", 0)
	u, _ := s.Users().Create("alice", "", "", "")
	if _, err := s.Files().Add(a.ID, u.ID, "map.zip", "v1", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Files().Add(a.ID, u.ID, "MAP.ZIP", "again", []byte("y")); err != store.ErrDuplicateName {
		t.Fatalf("duplicate name: want ErrDuplicateName, got %v", err)
	}
}

func TestFileQuotaRaceHonored(t *testing.T) {
	s, _ := openTestStore(t)
	a, _ := s.FileAreas().Create("General", 0)
	u, _ := s.Users().Create("alice", "", "", "")
	chunk := make([]byte, store.MaxFileBytes) // 10 MiB each; quota is 100 MiB
	var ok int32
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ { // 30*10MiB = 300MiB attempted, only ~10 should fit
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := s.Files().Add(a.ID, u.ID, fmtName(i), "", chunk); err == nil {
				atomic.AddInt32(&ok, 1)
			}
		}(i)
	}
	wg.Wait()
	if ok > store.MaxUserBytes/store.MaxFileBytes {
		t.Fatalf("quota race: admitted %d files (%d MiB), over the %d MiB cap",
			ok, ok*store.MaxFileBytes/(1<<20), store.MaxUserBytes/(1<<20))
	}
}

func fmtName(i int) string { return "f" + string(rune('a'+i)) }

func TestFileSearchSortFilterDelete(t *testing.T) {
	s, _ := openTestStore(t)
	a, _ := s.FileAreas().Create("General", 0)
	alice, _ := s.Users().Create("alice", "", "", "")
	bob, _ := s.Users().Create("bob", "", "", "")
	f1, _ := s.Files().Add(a.ID, alice.ID, "cats.txt", "about felines", []byte("a"))
	s.Files().Add(a.ID, bob.ID, "dogs.txt", "about canines", []byte("b"))

	if r, _ := s.Files().Search(a.ID, "feline"); len(r) != 1 || r[0].Filename != "cats.txt" {
		t.Fatalf("search by description failed: %+v", r)
	}
	if r, _ := s.Files().Search(a.ID, "dogs"); len(r) != 1 {
		t.Fatalf("search by filename failed: %+v", r)
	}
	if r, _ := s.Files().ByUploader(a.ID, bob.ID); len(r) != 1 || r[0].UploaderID != bob.ID {
		t.Fatalf("filter by uploader failed: %+v", r)
	}
	if r, _ := s.Files().ListSorted(a.ID, "name"); len(r) != 2 || r[0].Filename != "cats.txt" {
		t.Fatalf("sort by name failed: %+v", r)
	}
	if err := s.Files().Delete(f1.ID); err != nil {
		t.Fatal(err)
	}
	if r, _ := s.Files().ListByArea(a.ID); len(r) != 1 {
		t.Fatalf("delete failed: %d files remain", len(r))
	}
}

func TestXmodemReceiveCap(t *testing.T) {
	// Sender pushes 256B (2 blocks); receiver caps at 100B -> ErrTooBig.
	// Use a buffered TCP loopback rather than net.Pipe: when the receiver
	// aborts (CAN CAN) and returns, the sender quits after the first CAN, and
	// a synchronous pipe would deadlock the receiver's 2-byte abort write. A
	// real socket buffers it, which is the production shape (SSH/Telnet stream).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		_ = xfer.Send(c, make([]byte, 256))
	}()
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	rc := make(chan error, 1)
	go func() {
		_, err := xfer.Receive(conn, 100)
		rc <- err
	}()
	select {
	case err := <-rc:
		if err != xfer.ErrTooBig {
			t.Fatalf("oversized upload: want ErrTooBig, got %v", err)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("timed out")
	}
}
