package tests

import (
	"path/filepath"
	"strings"
	"testing"

	"admiralbbs/src/audit"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
	"admiralbbs/src/transport"
)

// TestFullMemberJourney exercises the real end-to-end path through actual
// session-driven menus: telnet apply -> SysOp approve + token -> first SSH login
// onboarding -> post to a board -> send mail -> upload a file. Asserts the store
// side-effects at each step. This is the integration coverage that was missing.
func TestFullMemberJourney(t *testing.T) {
	s, v := openTestStore(t)
	if err := s.EnsureSeedAreas(); err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureSeedFileAreas(); err != nil {
		t.Fatal(err)
	}
	lg, err := audit.New(filepath.Join(t.TempDir(), "audit.jsonl"), v, s.SessionLog())
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	run := func(handle string, telnet bool, input string, fn func(*session.Session)) {
		c := newFakeConn(input, "ansi", transport.WindowSize{Cols: 80, Rows: 25})
		if !telnet {
			c.user = handle
			c.tr = "ssh"
		}
		sess := session.New("s-"+handle, c, lg, nil)
		fn(sess)
		sess.Close()
	}

	key := strings.TrimSpace(genSSHKey(t))

	// 1) Telnet apply.
	run("", true, "alice\n"+key+"\n\nalice@pgp.example\njust joining\n", func(sess *session.Session) {
		if err := menu.RunApply(sess, s.Users(), s.Memberships(), s.Keys()); err != nil {
			t.Fatalf("apply: %v", err)
		}
	})
	alice, err := s.Users().ByHandle("alice")
	if err != nil || alice.Status != store.StatusApproved && alice.Status != store.StatusPending {
		t.Fatalf("applicant not created: %+v err=%v", alice, err)
	}
	if act, _ := s.Keys().Active(alice.ID); len(act) != 1 {
		t.Fatalf("applicant key not registered: %d", len(act))
	}

	// 2) SysOp approves (simulating the control panel / sysopctl) + issues token.
	if err := s.Users().Approve(alice.ID, 50); err != nil {
		t.Fatal(err)
	}
	tok, err := s.Tokens().Issue(alice.ID)
	if err != nil {
		t.Fatal(err)
	}

	// 3) First SSH login: redeem token, set password (onboarding).
	run("alice", false, tok+"\nhunter2pw\nhunter2pw\n", func(sess *session.Session) {
		u, ok := menu.RunLogin(sess, s)
		if !ok || u == nil {
			t.Fatal("onboarding login failed")
		}
	})
	alice, _ = s.Users().ByHandle("alice")
	if alice.PasswordHash == "" {
		t.Fatal("password not set after onboarding")
	}

	// 4) Post to the first board: pick area 1, [P]ost subject+body, quit out.
	run("alice", false, "1\np\nHello BBS\nfirst post\n.\nq\nq\n", func(sess *session.Session) {
		if err := menu.RunBoards(sess, s, alice); err != nil {
			t.Fatalf("boards: %v", err)
		}
	})
	areas, _ := s.MessageAreas().Visible(alice.AccessLevel)
	thread, _ := s.Messages().Thread(areas[0].ID)
	if len(thread) != 1 || thread[0].Subject != "Hello BBS" || thread[0].Body != "first post" {
		t.Fatalf("board post not stored: %+v", thread)
	}

	// 5) Send private mail to the seeded SysOp recipient.
	sysop, _ := s.Users().Create("sysop", "x", "", "")
	run("alice", false, "c\nsysop\nHi Sysop\nplease review\n.\nq\n", func(sess *session.Session) {
		if err := menu.RunMail(sess, s, alice); err != nil {
			t.Fatalf("mail: %v", err)
		}
	})
	if n, _ := s.Mail().UnreadCount(sysop.ID); n != 1 {
		t.Fatalf("mail not delivered: unread=%d", n)
	}

	// 6) Upload a text file via paste.
	run("alice", false, "1\nu\nnotes.txt\nmy notes\nphello file body\n.\nq\nq\n", func(sess *session.Session) {
		if err := menu.RunFiles(sess, s, alice); err != nil {
			t.Fatalf("files: %v", err)
		}
	})
	fa, _ := s.FileAreas().Visible(alice.AccessLevel)
	files, _ := s.Files().ListByArea(fa[0].ID)
	if len(files) != 1 || files[0].Filename != "notes.txt" {
		t.Fatalf("file not uploaded: %+v", files)
	}
}
