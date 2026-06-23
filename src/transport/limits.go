package transport

import (
	"net"
	"sync"
	"time"
)

// Limits bound resource use to blunt DoS / slow-loris attacks (RISKS SEC-3).
type Limits struct {
	MaxSessions      int           // total concurrent callers (0 = default 100)
	PerIP            int           // concurrent callers per remote IP (0 = unlimited)
	HandshakeTimeout time.Duration // deadline for the SSH handshake (0 = none)
}

// DefaultLimits returns sensible defaults.
func DefaultLimits() Limits {
	return Limits{MaxSessions: 100, PerIP: 5, HandshakeTimeout: 10 * time.Second}
}

// limiter enforces a global session cap and a per-IP cap.
type limiter struct {
	sem   chan struct{}
	mu    sync.Mutex
	perIP map[string]int
	maxIP int
}

func newLimiter(l Limits) *limiter {
	max := l.MaxSessions
	if max <= 0 {
		max = 100
	}
	return &limiter{sem: make(chan struct{}, max), perIP: map[string]int{}, maxIP: l.PerIP}
}

// acquire reserves a slot for ip, or returns false if a cap is hit.
func (lm *limiter) acquire(ip string) bool {
	select {
	case lm.sem <- struct{}{}:
	default:
		return false // global cap reached
	}
	lm.mu.Lock()
	if lm.maxIP > 0 && lm.perIP[ip] >= lm.maxIP {
		lm.mu.Unlock()
		<-lm.sem
		return false // per-IP cap reached
	}
	lm.perIP[ip]++
	lm.mu.Unlock()
	return true
}

func (lm *limiter) release(ip string) {
	lm.mu.Lock()
	if lm.perIP[ip] > 0 {
		lm.perIP[ip]--
		if lm.perIP[ip] == 0 {
			delete(lm.perIP, ip)
		}
	}
	lm.mu.Unlock()
	<-lm.sem
}

func hostOf(a net.Addr) string {
	if a == nil {
		return ""
	}
	if h, _, err := net.SplitHostPort(a.String()); err == nil {
		return h
	}
	return a.String()
}
