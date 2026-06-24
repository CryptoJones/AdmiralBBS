package doors

import (
	"io"
	"net"
	"time"
)

// Bridge connects a caller's session to a RESIDENT door — a persistent,
// real-time multiplayer game server (MajorMUD / Worldgroup style) already
// running and listening at network+address. It relays bytes both ways until
// either side closes, so every caller shares the one live game world. The BBS
// spawns nothing per player here; it's a relay.
func Bridge(sess io.ReadWriter, network, address string, dialTimeout time.Duration) error {
	if dialTimeout <= 0 {
		dialTimeout = 10 * time.Second
	}
	conn, err := net.DialTimeout(network, address, dialTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	done := make(chan struct{}, 2)
	go func() { io.Copy(conn, sess); done <- struct{}{} }() // caller -> game
	go func() { io.Copy(sess, conn); done <- struct{}{} }() // game -> caller
	<-done                                                  // one side hung up
	return nil
}
