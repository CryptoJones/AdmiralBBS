package tests

import (
	"fmt"
	"sync"
	"testing"
)

// Simulate many callers online at once, all reading and writing — the BBS must
// not return "database is locked". SQLite is single-writer, so the store must
// serialize its connection pool.
func TestConcurrentUsersHammerDB(t *testing.T) {
	s, _ := openTestStore(t)
	if err := s.EnsureSeedAreas(); err != nil {
		t.Fatal(err)
	}
	areas, _ := s.MessageAreas().Visible(50)
	area := areas[0]

	const users = 40
	var wg sync.WaitGroup
	errs := make(chan error, users*8)
	wg.Add(users)
	for i := 0; i < users; i++ {
		go func(i int) {
			defer wg.Done()
			u, err := s.Users().Create(fmt.Sprintf("user%03d", i), "h", "", "")
			if err != nil {
				errs <- fmt.Errorf("create: %w", err)
				return
			}
			for j := 0; j < 5; j++ {
				if _, err := s.Messages().Post(area.ID, u.ID, nil, "subj", "body"); err != nil {
					errs <- fmt.Errorf("post: %w", err)
					return
				}
				if _, err := s.Messages().Thread(area.ID); err != nil {
					errs <- fmt.Errorf("read: %w", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatalf("concurrent DB op failed (multi-user breaks): %v", e)
	}
}
