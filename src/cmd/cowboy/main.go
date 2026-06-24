// Command cowboy is the persistent game server for Console Cowboy 2026 — a
// multiplayer cyberpunk MUD. It listens on TCP; AdmiralBBS bridges each caller
// in as a "resident" door, so everyone shares one live world. All game state is
// mutated on a single goroutine (events + ticks serialized), so the engine
// stays lock-free and deterministic; this process only owns I/O.
package main

import (
	"bufio"
	"flag"
	"log"
	"net"
	"sync"
	"time"

	"admiralbbs/src/game/cowboy"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:4000", "TCP listen address for BBS bridge")
	dbPath := flag.String("db", "cowboy.db", "character database path (SQLite)")
	tick := flag.Duration("tick", 2*time.Second, "combat/world tick interval")
	flag.Parse()

	store, err := cowboy.OpenSQLite(*dbPath)
	if err != nil {
		log.Fatalf("open character db: %v", err)
	}
	defer store.Close()

	world := cowboy.NewWorld(store)
	events := make(chan event, 256)

	// The single world goroutine: every mutation happens here.
	go func() {
		ticker := time.NewTicker(*tick)
		defer ticker.Stop()
		for {
			select {
			case ev := <-events:
				handle(world, ev)
			case <-ticker.C:
				world.Tick()
				// Re-show a prompt so combat output doesn't leave a bare line.
				for _, c := range activeConns() {
					if c.player != nil {
						world.Prompt(c.player)
					}
				}
			}
		}
	}()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("Console Cowboy 2026 listening on %s (tick %s)", *addr, *tick)
	for {
		nc, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go serve(nc, events)
	}
}

// ---- per-connection plumbing ----

type conn struct {
	nc     net.Conn
	outCh  chan string
	player *cowboy.Player
	closed bool // set on the world goroutine during teardown
}

// out enqueues text for the writer. Called only from the world goroutine, so the
// non-blocking send (drop on overflow) protects the world from a stalled client.
func (c *conn) out(s string) {
	select {
	case c.outCh <- s:
	default:
	}
}

var (
	connMu  sync.Mutex
	connSet = map[*conn]struct{}{}
)

func activeConns() []*conn {
	connMu.Lock()
	defer connMu.Unlock()
	out := make([]*conn, 0, len(connSet))
	for c := range connSet {
		out = append(out, c)
	}
	return out
}

func serve(nc net.Conn, events chan event) {
	c := &conn{nc: nc, outCh: make(chan string, 512)}
	connMu.Lock()
	connSet[c] = struct{}{}
	connMu.Unlock()

	// Writer goroutine: drains outCh to the socket, then closes the socket.
	go func() {
		for s := range c.outCh {
			if _, err := nc.Write([]byte(s)); err != nil {
				break
			}
		}
		nc.Close()
	}()

	r := bufio.NewReader(nc)
	c.out("\r\n" + "Handle (your runner name): ")
	name, err := cowboy.ReadLine(r, c.out)
	if err != nil || len(name) == 0 {
		events <- event{typ: evClose, c: c}
		return
	}

	reply := make(chan connectResult, 1)
	events <- event{typ: evConnect, c: c, name: name, reply: reply}
	res := <-reply
	if res.rejected {
		// name already online — give the writer a beat to flush, then close.
		time.Sleep(200 * time.Millisecond)
		events <- event{typ: evClose, c: c}
		return
	}
	if res.needCreate {
		// New runner: run the creation screen on this goroutine (the I/O side),
		// then hand the chosen loadout to the world to build the character.
		spec, err := cowboy.RunCharacterCreation(r, c.out)
		if err != nil {
			events <- event{typ: evClose, c: c}
			return
		}
		reply2 := make(chan connectResult, 1)
		events <- event{typ: evCreate, c: c, name: name, spec: spec, reply: reply2}
		<-reply2
	}

	for {
		line, err := cowboy.ReadLine(r, c.out)
		if err != nil {
			events <- event{typ: evDisconnect, c: c}
			return
		}
		events <- event{typ: evLine, c: c, line: line}
	}
}

// ---- events (all handled on the world goroutine) ----

type evType int

const (
	evConnect evType = iota
	evCreate
	evLine
	evDisconnect
	evClose
)

// connectResult tells the connection goroutine how the world handled a connect:
// rejected (name online), needCreate (new runner — run the creation screen), or
// otherwise an existing character was placed in the world.
type connectResult struct {
	rejected   bool
	needCreate bool
}

type event struct {
	typ   evType
	c     *conn
	name  string
	line  string
	spec  cowboy.CharSpec
	reply chan connectResult
}

func handle(world *cowboy.World, ev event) {
	switch ev.typ {
	case evConnect:
		if world.Online(ev.name) {
			ev.c.out("\r\nThat runner is already jacked in. Try another handle.\r\n")
			ev.reply <- connectResult{rejected: true}
			return
		}
		if !world.HasCharacter(ev.name) {
			ev.reply <- connectResult{needCreate: true}
			return
		}
		p := world.Connect(ev.name, ev.c.out)
		ev.c.player = p
		world.Prompt(p)
		ev.reply <- connectResult{}
	case evCreate:
		if world.Online(ev.name) { // lost a race to another connection
			ev.c.out("\r\nThat runner just jacked in elsewhere.\r\n")
			ev.reply <- connectResult{rejected: true}
			return
		}
		p := world.CreateCharacter(ev.name, ev.spec, ev.c.out)
		ev.c.player = p
		world.Prompt(p)
		ev.reply <- connectResult{}
	case evLine:
		if ev.c.closed || ev.c.player == nil {
			return
		}
		if world.Command(ev.c.player, ev.line) {
			teardown(world, ev.c)
		}
	case evDisconnect, evClose:
		teardown(world, ev.c)
	}
}

func teardown(world *cowboy.World, c *conn) {
	if c.closed {
		return
	}
	c.closed = true
	if c.player != nil {
		world.Disconnect(c.player)
		c.player = nil
	}
	connMu.Lock()
	delete(connSet, c)
	connMu.Unlock()
	close(c.outCh) // ends the writer, which flushes remaining output then closes the socket
}
