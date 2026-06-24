package tests

import (
	"bufio"
	"strings"
	"testing"

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
