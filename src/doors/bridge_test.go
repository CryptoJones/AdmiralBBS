package doors

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestSanitizeVersion(t *testing.T) {
	cases := map[string]string{
		"1.0.0":            "1.0.0",
		"2.3.4-rc.1+build": "2.3.4-rc.1+build",
		"1.0.0\x1b[31m":    "1.0.031m",  // ESC and [ stripped, "31m" kept
		"bad\x00\x07ctrl":  "badctrl",   // NUL/BEL stripped
		"with space":       "withspace", // space stripped
		"a/b;c":            "abc",       // punctuation outside the set stripped
	}
	for in, want := range cases {
		if got := sanitizeVersion([]byte(in)); got != want {
			t.Errorf("sanitizeVersion(%q) = %q, want %q", in, got, want)
		}
	}
	// Length is capped at 32 (truncate before filtering).
	long := bytes.Repeat([]byte("a"), 50)
	if got := sanitizeVersion(long); len(got) != 32 {
		t.Errorf("sanitizeVersion(50×'a') len = %d, want 32", len(got))
	}
}

// serveOnce accepts one connection, writes payload, holds it open for keepOpen,
// then closes. Returns the dial address.
func serveOnce(t *testing.T, payload []byte, keepOpen time.Duration) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		_, _ = conn.Write(payload)
		if keepOpen > 0 {
			time.Sleep(keepOpen)
		}
		conn.Close()
	}()
	return ln.Addr().String()
}

func TestDialResidentReadsVersionHandshake(t *testing.T) {
	// Sentinel + greeting in one write; readHandshake must parse the version
	// regardless of how the bytes are framed.
	addr := serveOnce(t, []byte("\x1b]ABBS;version=1.2.3\x07Welcome, runner.\r\n"), 0)
	rc, err := DialResident("tcp", addr, 2*time.Second, time.Second)
	if err != nil {
		t.Fatalf("DialResident: %v", err)
	}
	defer rc.Close()
	if rc.Version != "1.2.3" {
		t.Fatalf("Version = %q, want %q", rc.Version, "1.2.3")
	}
}

func TestDialResidentNoHandshakeForwardsBytes(t *testing.T) {
	// A door that sends no sentinel: Version stays empty and every byte it sent
	// is relayed verbatim to the caller (nothing is eaten by the peek).
	addr := serveOnce(t, []byte("plain greeting\r\n"), 200*time.Millisecond)
	rc, err := DialResident("tcp", addr, 2*time.Second, 300*time.Millisecond)
	if err != nil {
		t.Fatalf("DialResident: %v", err)
	}
	if rc.Version != "" {
		t.Fatalf("Version = %q, want empty", rc.Version)
	}

	client, server := net.Pipe()
	go func() { _ = rc.Relay(server) }()
	defer client.Close()

	_ = client.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 64)
	n, err := client.Read(buf)
	if err != nil {
		t.Fatalf("read relayed bytes: %v", err)
	}
	if !bytes.Contains(buf[:n], []byte("plain greeting")) {
		t.Fatalf("relayed %q, want it to contain %q", buf[:n], "plain greeting")
	}
}
