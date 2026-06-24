package tests

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"admiralbbs/src/doors"
)

type pipeRW struct {
	r io.Reader
	w io.Writer
}

func (p pipeRW) Read(b []byte) (int, error)  { return p.r.Read(b) }
func (p pipeRW) Write(b []byte) (int, error) { return p.w.Write(b) }

// The bundled demo door actually plays when launched THROUGH the sandbox
// launcher (reads its handle from the dropfile, greets, takes input).
func TestDemoDoorPlaysThroughLauncher(t *testing.T) {
	doorPath, _ := filepath.Abs("../doors/numguess.sh")
	if _, err := os.Stat(doorPath); err != nil {
		t.Skipf("demo door not found: %v", err)
	}

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()

	var sb strings.Builder
	readDone := make(chan struct{})
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := outR.Read(buf)
			if n > 0 {
				sb.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		close(readDone)
	}()

	done := make(chan error, 1)
	go func() {
		err := doors.Launch(pipeRW{inR, outW}, doorPath, nil,
			doors.DropInfo{BBSName: "AdmiralBBS", Handle: "tester", AccessLevel: 50, ANSI: true},
			doors.Opts{Timeout: 10 * time.Second})
		outW.Close()
		done <- err
	}()

	inW.Write([]byte("q\n")) // quit promptly
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("door did not exit")
	}
	inW.Close()
	<-readDone

	out := sb.String()
	if !strings.Contains(out, "NUMBER GUESS") || !strings.Contains(out, "Welcome, tester") {
		t.Fatalf("demo door did not play through the launcher:\n%s", out)
	}
}
