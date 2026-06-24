package tests

import (
	"bytes"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"admiralbbs/src/doors"
)

type syncBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
func (b *syncBuf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// A resident (MajorMUD-style) door: the BBS bridges the caller to a running
// game server. Verify the relay carries bytes both ways.
func TestResidentDoorBridge(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	gotFromCaller := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 64)
		n, _ := c.Read(buf)
		gotFromCaller <- string(buf[:n])
		c.Write([]byte("WELCOME TO THE REALM"))
	}()

	// A live session's input doesn't EOF until the user disconnects, so use a
	// pipe kept open — the bridge should persist until the GAME closes.
	inR, inW := io.Pipe()
	go func() { inW.Write([]byte("look")) }()
	out := &syncBuf{}
	sess := pipeRW{r: inR, w: out}
	if err := doors.Bridge(sess, "tcp", ln.Addr().String(), 5*time.Second); err != nil {
		t.Fatalf("bridge: %v", err)
	}

	select {
	case got := <-gotFromCaller:
		if got != "look" {
			t.Fatalf("game received %q, want \"look\"", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("game never received caller input")
	}
	time.Sleep(50 * time.Millisecond) // let the game->caller copy settle
	if !strings.Contains(out.String(), "WELCOME TO THE REALM") {
		t.Fatalf("caller did not receive game output: %q", out.String())
	}
}
