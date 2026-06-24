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
	"os"
	"os/signal"
	"sync"
	"syscall"
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
		autosave := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		defer autosave.Stop()
		for {
			select {
			case ev := <-events:
				if ev.typ == evShutdown {
					world.SaveAll() // flush everyone before the process exits
					close(ev.done)
					return
				}
				handle(world, ev)
			case <-ticker.C:
				world.Tick()
				// Re-show the prompt ONLY for players who got output this tick
				// (combat/chat/room events). Idle players keep their single
				// prompt instead of it repeating every tick.
				for _, c := range activeConns() {
					if c.player != nil {
						world.PromptIfDirty(c.player)
					}
				}
			case <-autosave.C:
				world.SaveAll() // periodic persistence so a crash loses < 30s
			}
		}
	}()

	// Graceful shutdown: on SIGINT/SIGTERM (e.g. systemctl restart), flush all
	// connected players via the world goroutine before exiting.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Print("shutting down — saving players")
		done := make(chan struct{})
		events <- event{typ: evShutdown, done: done}
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
		os.Exit(0)
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

	mu     sync.Mutex // guards inLine/prompt so output redraws don't race input echo
	inLine []byte     // the caller's in-progress (un-submitted) input
	prompt string     // the current status prompt (managed-prompt redraw)
}

// raw enqueues bytes for the writer. Non-blocking (drop on overflow) so a stalled
// client can't block the world goroutine.
func (c *conn) raw(s string) {
	select {
	case c.outCh <- s:
	default:
	}
}

// out is the simple pre-world sink (login/creation prompts; no input to preserve).
func (c *conn) out(s string) { c.raw(s) }

const clrLine = "\r\x1b[K" // CR + erase-to-end-of-line: wipe the current row

// emit writes async/response content while PRESERVING the caller's in-progress
// input: wipe the current prompt+input row, print the content, then redraw the
// prompt with whatever they'd typed. This is what stops combat lines from
// scrambling text the player is mid-typing.
func (c *conn) emit(s string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.raw(clrLine + s + c.prompt + string(c.inLine))
}

// setPrompt updates the status prompt and redraws it (with current input).
func (c *conn) setPrompt(p string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.prompt = p
	c.raw(clrLine + c.prompt + string(c.inLine))
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

	// In-world input loop with a MANAGED prompt: we own the input buffer here so
	// async output (combat/chat) can wipe-and-redraw it via conn.emit/setPrompt
	// without garbling what the caller is typing.
	for {
		b, err := r.ReadByte()
		if err != nil {
			events <- event{typ: evDisconnect, c: c}
			return
		}
		switch b {
		case '\r', '\n':
			if r.Buffered() > 0 { // swallow a buffered CRLF/LFCR partner (never block)
				if nb, e := r.ReadByte(); e == nil {
					if !((b == '\r' && nb == '\n') || (b == '\n' && nb == '\r')) {
						_ = r.UnreadByte()
					}
				}
			}
			c.mu.Lock()
			line := string(c.inLine)
			c.inLine = c.inLine[:0]
			c.raw("\r\n")
			c.mu.Unlock()
			events <- event{typ: evLine, c: c, line: line}
		case 0x08, 0x7f: // backspace / DEL
			c.mu.Lock()
			if len(c.inLine) > 0 {
				c.inLine = c.inLine[:len(c.inLine)-1]
				c.raw("\b \b")
			}
			c.mu.Unlock()
		case 0x00:
			// ignore
		case 0xff: // telnet IAC — skip it and its command byte
			_, _ = r.ReadByte()
		default:
			if b >= 0x20 && b < 0x7f {
				c.mu.Lock()
				c.inLine = append(c.inLine, b)
				c.raw(string(b)) // echo
				c.mu.Unlock()
			}
		}
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
	evShutdown // flush all players, then signal done (graceful shutdown)
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
	done  chan struct{} // evShutdown: closed once all players are saved
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
		p := world.Connect(ev.name, ev.c.emit)
		world.SetPrompter(p, ev.c.setPrompt)
		ev.c.player = p
		world.Prompt(p)
		ev.reply <- connectResult{}
	case evCreate:
		if world.Online(ev.name) { // lost a race to another connection
			ev.c.out("\r\nThat runner just jacked in elsewhere.\r\n")
			ev.reply <- connectResult{rejected: true}
			return
		}
		p := world.CreateCharacter(ev.name, ev.spec, ev.c.emit)
		world.SetPrompter(p, ev.c.setPrompt)
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
