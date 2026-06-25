package doors

// Release-install: fetch a resident-door binary from a forge RELEASE and run it
// under BBS supervision. This is how a SysOp adds a door by pointing the BBS at
// a release URL. It is forge-agnostic — it works against any releases JSON that
// returns the GitHub/Codeberg/Forgejo shape {tag_name, assets:[{name,
// browser_download_url}]} — and platform-aware: it picks the asset built for the
// host the BBS itself runs on (Linux, Windows, or macOS; amd64 or arm64).
//
// Running a downloaded binary is a trust decision, so installing is SysOp-gated.

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ReleaseAsset is one downloadable file attached to a release.
type ReleaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// Release is the subset of a forge release JSON the installer needs.
type Release struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
}

// FetchRelease GETs a forge releases JSON endpoint and parses it. The endpoint
// is whatever the SysOp pasted (e.g. a GitHub/Forgejo ".../releases/latest" API
// URL); all three forges return the same {tag_name, assets[]} shape.
func FetchRelease(apiURL string, timeout time.Duration) (*Release, error) {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release endpoint returned %s", resp.Status)
	}
	var rel Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rel); err != nil {
		return nil, fmt.Errorf("parse release json: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("response had no tag_name (is this a releases endpoint?)")
	}
	return &rel, nil
}

// PickAsset selects the release asset built for the BBS host's OS+arch, matching
// the platform tokens (and common aliases) in the asset filename. This is what
// makes a door installable on a Linux *or* Windows *or* macOS BBS from the same
// release, as long as the door publishes a per-platform binary.
func PickAsset(r *Release) (ReleaseAsset, error) {
	oss, arches := platformTokens()
	for _, a := range r.Assets {
		n := strings.ToLower(a.Name)
		if containsAny(n, oss) && containsAny(n, arches) {
			return a, nil
		}
	}
	return ReleaseAsset{}, fmt.Errorf("no %s/%s binary in release %s — assets: [%s]",
		runtime.GOOS, runtime.GOARCH, r.TagName, assetNames(r))
}

// platformTokens returns the OS and arch name fragments (with common aliases) to
// look for in an asset filename, for the host the BBS is running on.
func platformTokens() (oss, arches []string) {
	switch runtime.GOOS {
	case "windows":
		oss = []string{"windows", "win"}
	case "darwin":
		oss = []string{"darwin", "macos", "osx"}
	default:
		oss = []string{runtime.GOOS} // linux, freebsd, ...
	}
	switch runtime.GOARCH {
	case "amd64":
		arches = []string{"amd64", "x86_64", "x64"}
	case "arm64":
		arches = []string{"arm64", "aarch64"}
	case "386":
		arches = []string{"386", "i386"}
	default:
		arches = []string{runtime.GOARCH}
	}
	return oss, arches
}

func containsAny(s string, subs []string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

func assetNames(r *Release) string {
	names := make([]string, 0, len(r.Assets))
	for _, a := range r.Assets {
		names = append(names, a.Name)
	}
	return strings.Join(names, ", ")
}

// DownloadBinary fetches url to destPath (atomic temp+rename) and marks it
// executable. Capped at 512 MiB to bound a hostile/broken download.
func DownloadBinary(url, destPath string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 3 * time.Minute
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	tmp := destPath + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 512<<20)); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, destPath)
}

// FreeLocalPort asks the OS for an unused localhost TCP port. There's an
// inherent race between releasing and re-binding it, but installed doors get a
// stable port persisted at install time, so it's only used once per door.
func FreeLocalPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// ---- Supervised resident-door processes ----

// Supervisor launches BBS-installed resident-door binaries and keeps them
// running. Each door starts as `<bin> -addr <addr>` with its working directory
// set to a per-door data dir; if it exits it is relaunched with capped backoff
// until StopAll. Cross-platform (os/exec) so it works on a Windows BBS too.
type Supervisor struct {
	mu    sync.Mutex
	procs map[string]*superProc
}

type superProc struct {
	cmd  *exec.Cmd
	stop chan struct{}
}

// NewSupervisor builds an empty supervisor.
func NewSupervisor() *Supervisor { return &Supervisor{procs: map[string]*superProc{}} }

// Launch starts and keeps alive the door named name (`binPath -addr addr`, cwd =
// dataDir). Idempotent: a door already supervised under this name is left as-is.
func (s *Supervisor) Launch(name, binPath, dataDir, addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.procs[name]; ok {
		return nil
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	sp := &superProc{stop: make(chan struct{})}
	s.procs[name] = sp
	go s.supervise(name, binPath, dataDir, addr, sp)
	return nil
}

func (s *Supervisor) supervise(name, binPath, dataDir, addr string, sp *superProc) {
	backoff := time.Second
	for {
		select {
		case <-sp.stop:
			return
		default:
		}
		cmd := exec.Command(binPath, "-addr", addr)
		cmd.Dir = dataDir
		s.mu.Lock()
		sp.cmd = cmd
		s.mu.Unlock()
		start := time.Now()
		if err := cmd.Start(); err != nil {
			log.Printf("door %q failed to start (%s): %v", name, binPath, err)
		} else {
			log.Printf("door %q up: %s -addr %s (cwd %s)", name, binPath, addr, dataDir)
			_ = cmd.Wait()
		}
		select {
		case <-sp.stop:
			return
		default:
		}
		if time.Since(start) > 30*time.Second {
			backoff = time.Second // a healthy run resets the backoff
		}
		log.Printf("door %q exited; relaunching in %s", name, backoff)
		select {
		case <-sp.stop:
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// Running reports whether a door is currently supervised.
func (s *Supervisor) Running(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.procs[name]
	return ok
}

// Stop ends one supervised door (kills the process and stops relaunching).
func (s *Supervisor) Stop(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sp, ok := s.procs[name]; ok {
		close(sp.stop)
		if sp.cmd != nil && sp.cmd.Process != nil {
			_ = sp.cmd.Process.Kill()
		}
		delete(s.procs, name)
	}
}

// StopAll tears down every supervised door (called on BBS shutdown).
func (s *Supervisor) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, sp := range s.procs {
		close(sp.stop)
		if sp.cmd != nil && sp.cmd.Process != nil {
			_ = sp.cmd.Process.Kill()
		}
		delete(s.procs, name)
	}
}
