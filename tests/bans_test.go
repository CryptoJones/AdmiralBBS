package tests

import (
	"net"
	"testing"
	"time"

	"admiralbbs/src/store"
	"admiralbbs/src/transport"
)

func TestBansExactAndCIDR(t *testing.T) {
	s, _ := openTestStore(t)
	bans := s.Bans()

	if _, err := bans.Add("203.0.113.7", "spam", 0); err != nil {
		t.Fatalf("add exact: %v", err)
	}
	if _, err := bans.Add("198.51.100.0/24", "abusive netblock", 0); err != nil {
		t.Fatalf("add cidr: %v", err)
	}

	cases := map[string]bool{
		"203.0.113.7":   true,  // exact
		"203.0.113.8":   false, // neighbour, not banned
		"198.51.100.42": true,  // inside CIDR
		"198.51.100.0":  true,  // network address inside CIDR
		"198.51.101.1":  false, // outside CIDR
		"":              false, // garbage
		"not-an-ip":     false,
	}
	for ip, want := range cases {
		if got := bans.IsBanned(ip); got != want {
			t.Errorf("IsBanned(%q) = %v, want %v", ip, got, want)
		}
	}

	// Lifting a ban frees the IP.
	active, _ := bans.Active()
	if len(active) != 2 {
		t.Fatalf("active bans = %d, want 2", len(active))
	}
	for _, b := range active {
		if b.Pattern == "203.0.113.7" {
			if err := bans.Lift(b.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	if bans.IsBanned("203.0.113.7") {
		t.Error("lifted ban still blocks")
	}
}

func TestBanPatternValidation(t *testing.T) {
	if _, err := store.NormalizeBanPattern("not an ip"); err == nil {
		t.Error("garbage pattern accepted")
	}
	if _, err := store.NormalizeBanPattern("10.0.0.0/8"); err != nil {
		t.Errorf("valid CIDR rejected: %v", err)
	}
	if got, _ := store.NormalizeBanPattern(" 203.0.113.7 "); got != "203.0.113.7" {
		t.Errorf("normalize trims: got %q", got)
	}
}

// End-to-end: a banned source is dropped by the real Telnet listener at accept
// time — the handler is never invoked and the connection closes immediately.
func TestTelnetRejectsBannedSource(t *testing.T) {
	addr := freeAddr(t)
	handled := make(chan struct{}, 1)
	banned := func(ip string) bool { return ip == "127.0.0.1" }
	go func() {
		_ = transport.ServeTelnet(addr, transport.Limits{}, banned, func(c transport.Conn) {
			handled <- struct{}{}
			c.Close()
		})
	}()
	waitListening(t, addr)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	// A dropped connection yields EOF on the first read.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1)
	if _, err := conn.Read(buf); err == nil {
		t.Fatal("expected the banned connection to be closed, but it stayed open")
	}
	select {
	case <-handled:
		t.Fatal("handler ran for a banned source")
	case <-time.After(200 * time.Millisecond):
		// good — handler never fired
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

func waitListening(t *testing.T, addr string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			c.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server never came up on %s", addr)
}
