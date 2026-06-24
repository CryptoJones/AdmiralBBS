package tests

import (
	"bufio"
	"io"
	"strings"
	"testing"
	"time"

	"admiralbbs/src/game/cowboy"
)

func TestCowboyReadLine(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"case\r\n", []string{"case"}},
		{"hi\r\nyo\r\n", []string{"hi", "yo"}},          // CRLF splits cleanly
		{"ab\x08c\r\n", []string{"ac"}},                 // backspace erases
		{"x\x00y\r\n", []string{"xy"}},                  // NUL ignored
		{"\xff\xf9z\r\n", []string{"z"}},                // telnet IAC + cmd skipped
	}
	for _, c := range cases {
		r := bufio.NewReader(strings.NewReader(c.in))
		for _, want := range c.want {
			got, err := cowboy.ReadLine(r, nil)
			if err != nil {
				t.Fatalf("ReadLine(%q): %v", c.in, err)
			}
			if got != want {
				t.Errorf("ReadLine(%q) = %q, want %q", c.in, got, want)
			}
		}
	}
}

// Interactive Enter sends a lone CR (or LF) with nothing after it. ReadLine
// must return immediately on it — NOT block peeking for a CRLF partner (the bug
// that produced a hang / a spurious extra prompt in the door game).
func TestCowboyReadLineLoneCRDoesNotBlock(t *testing.T) {
	pr, pw := io.Pipe()
	r := bufio.NewReader(pr)
	go func() { pw.Write([]byte("l\r")); /* deliberately write nothing more */ }()

	done := make(chan string, 1)
	go func() {
		line, _ := cowboy.ReadLine(r, nil)
		done <- line
	}()
	select {
	case got := <-done:
		if got != "l" {
			t.Fatalf("lone-CR line = %q, want %q", got, "l")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ReadLine blocked on a lone CR (peeked for a CRLF partner that never came)")
	}
	pw.Close()
}

func TestCowboyReadLineEcho(t *testing.T) {
	var echoed strings.Builder
	r := bufio.NewReader(strings.NewReader("hi\r\n"))
	if _, err := cowboy.ReadLine(r, func(s string) { echoed.WriteString(s) }); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(echoed.String(), "h") || !strings.Contains(echoed.String(), "i") {
		t.Errorf("echo missing typed chars: %q", echoed.String())
	}
}
