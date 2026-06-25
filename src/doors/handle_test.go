package doors

import (
	"net"
	"testing"
	"time"
)

// A door advertising caps=handle gets the caller's handle pushed back.
func TestHandshakeCapsAndSendHandle(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	got := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		_, _ = c.Write([]byte("\x1b]ABBS;version=1.0.2;caps=handle\x07\r\nHandle: "))
		buf := make([]byte, 64)
		_ = c.SetReadDeadline(time.Now().Add(time.Second))
		n, _ := c.Read(buf)
		got <- string(buf[:n])
	}()

	rc, err := DialResident("tcp", ln.Addr().String(), 2*time.Second, time.Second)
	if err != nil {
		t.Fatalf("DialResident: %v", err)
	}
	if rc.Version != "1.0.2" || !rc.caps["handle"] {
		t.Fatalf("parsed version=%q caps=%v (want 1.0.2 + handle)", rc.Version, rc.caps)
	}
	rc.SendHandle("Crypto.Jones-9")
	rc.Close()

	select {
	case s := <-got:
		if s != "\x1b]ABBS;handle=Crypto.Jones-9\x07" {
			t.Fatalf("door received %q", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("door never received the handle sentinel")
	}
}

// A door that did NOT advertise the capability gets nothing (no injected bytes).
func TestSendHandleNoCapIsNoOp(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	sawBytes := make(chan bool, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		_, _ = c.Write([]byte("\x1b]ABBS;version=1.0.1\x07")) // no caps
		buf := make([]byte, 16)
		_ = c.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
		n, _ := c.Read(buf)
		sawBytes <- n > 0
	}()

	rc, err := DialResident("tcp", ln.Addr().String(), 2*time.Second, time.Second)
	if err != nil {
		t.Fatalf("DialResident: %v", err)
	}
	if rc.caps["handle"] {
		t.Fatal("door advertised no caps but handle cap is set")
	}
	rc.SendHandle("Whoever") // must write nothing
	defer rc.Close()

	if <-sawBytes {
		t.Fatal("SendHandle wrote bytes to a door that didn't ask for the handle")
	}
}
