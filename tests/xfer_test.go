package tests

import (
	"bytes"
	"net"
	"testing"
	"time"

	"admiralbbs/src/xfer"
)

// Send and Receive interoperate over a pipe, across block boundaries.
func TestXmodemRoundTrip(t *testing.T) {
	for _, size := range []int{1, 127, 128, 129, 500, 4096} {
		payload := make([]byte, size)
		for i := range payload {
			payload[i] = byte(i*7 + 3)
		}

		a, b := net.Pipe()
		got := make(chan []byte, 1)
		errc := make(chan error, 2)
		go func() {
			data, err := xfer.Receive(b)
			if err != nil {
				errc <- err
				return
			}
			got <- data
		}()
		go func() {
			if err := xfer.Send(a, payload); err != nil {
				errc <- err
			}
		}()

		select {
		case data := <-got:
			// XMODEM pads to 128-byte blocks; the payload is a prefix.
			if !bytes.HasPrefix(data, payload) {
				t.Fatalf("size %d: received data does not match payload", size)
			}
			if len(data) > size+128 {
				t.Fatalf("size %d: received %d bytes (too much padding)", size, len(data))
			}
		case err := <-errc:
			t.Fatalf("size %d: transfer error: %v", size, err)
		case <-time.After(10 * time.Second):
			t.Fatalf("size %d: transfer timed out", size)
		}
		a.Close()
		b.Close()
	}
}
