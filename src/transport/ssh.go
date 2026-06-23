package transport

import (
	"crypto/ed25519"
	"encoding/pem"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// ServeSSH listens on addr with an encrypted SSH transport. The host key is
// loaded from hostKeyPath, or generated and persisted there on first run.
// Authentication is open (NoClientAuth) for Sprint 002 — the ssh username is
// captured for the audit trail; real accounts arrive in Sprint 003.
func ServeSSH(addr, hostKeyPath string, limits Limits, handle func(Conn)) error {
	signer, err := loadOrCreateHostKey(hostKeyPath)
	if err != nil {
		return err
	}
	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	lm := newLimiter(limits)
	for {
		raw, err := ln.Accept()
		if err != nil {
			return err
		}
		ip := hostOf(raw.RemoteAddr())
		if !lm.acquire(ip) {
			raw.Close()
			continue
		}
		go func(raw net.Conn, ip string) {
			defer lm.release(ip)
			if limits.HandshakeTimeout > 0 {
				raw.SetDeadline(time.Now().Add(limits.HandshakeTimeout))
			}
			serveSSHConn(raw, cfg, handle)
		}(raw, ip)
	}
}

func serveSSHConn(raw net.Conn, cfg *ssh.ServerConfig, handle func(Conn)) {
	sconn, chans, reqs, err := ssh.NewServerConn(raw, cfg)
	if err != nil {
		raw.Close()
		return
	}
	raw.SetDeadline(time.Time{}) // handshake done — clear the slow-loris deadline
	go ssh.DiscardRequests(reqs)

	for nc := range chans {
		if nc.ChannelType() != "session" {
			nc.Reject(ssh.UnknownChannelType, "only session channels are supported")
			continue
		}
		ch, chReqs, err := nc.Accept()
		if err != nil {
			continue
		}
		c := &sshConn{
			ch:       ch,
			user:     sconn.User(),
			remote:   sconn.RemoteAddr(),
			termType: "",
			winCh:    make(chan WindowSize, 8),
		}
		go c.serviceRequests(chReqs, handle)
	}
	sconn.Close()
}

type ptyReq struct {
	Term       string
	Cols, Rows uint32
	Wpx, Hpx   uint32
	Modes      string
}

type winChReq struct {
	Cols, Rows uint32
	Wpx, Hpx   uint32
}

type sshConn struct {
	ch     ssh.Channel
	user   string
	remote net.Addr

	mu       sync.Mutex
	termType string
	win      WindowSize

	winCh   chan WindowSize
	started bool
}

func (c *sshConn) serviceRequests(reqs <-chan *ssh.Request, handle func(Conn)) {
	for req := range reqs {
		switch req.Type {
		case "pty-req":
			var p ptyReq
			if err := ssh.Unmarshal(req.Payload, &p); err == nil {
				c.mu.Lock()
				c.termType = p.Term
				c.win = WindowSize{Cols: int(p.Cols), Rows: int(p.Rows)}
				c.mu.Unlock()
			}
			req.Reply(true, nil)
		case "window-change":
			var w winChReq
			if err := ssh.Unmarshal(req.Payload, &w); err == nil {
				ws := WindowSize{Cols: int(w.Cols), Rows: int(w.Rows)}
				c.mu.Lock()
				c.win = ws
				c.mu.Unlock()
				select {
				case c.winCh <- ws:
				default:
				}
			}
			if req.WantReply {
				req.Reply(true, nil)
			}
		case "shell":
			req.Reply(true, nil)
			c.mu.Lock()
			start := !c.started
			c.started = true
			c.mu.Unlock()
			if start {
				go func() {
					handle(c)
					c.ch.Close()
				}()
			}
		case "exec":
			req.Reply(false, nil) // interactive only
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

func (c *sshConn) Read(p []byte) (int, error)  { return c.ch.Read(p) }
func (c *sshConn) Write(p []byte) (int, error) { return c.ch.Write(p) }
func (c *sshConn) Close() error                { return c.ch.Close() }
func (c *sshConn) RemoteAddr() net.Addr        { return c.remote }
func (c *sshConn) Transport() string           { return "ssh" }
func (c *sshConn) Username() string            { return c.user }

func (c *sshConn) TermType() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.termType
}

func (c *sshConn) WindowSize() WindowSize {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.win
}

func (c *sshConn) WindowChanges() <-chan WindowSize { return c.winCh }

// loadOrCreateHostKey reads an ed25519 host key from path, generating and
// persisting one (0600) if absent.
func loadOrCreateHostKey(path string) (ssh.Signer, error) {
	if data, err := os.ReadFile(path); err == nil {
		return ssh.ParsePrivateKey(data)
	}
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(priv)
}
