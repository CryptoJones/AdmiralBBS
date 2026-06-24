package doors

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// Opts tune the sandbox.
type Opts struct {
	Timeout  time.Duration // wall-clock kill (default 15m)
	CPULimit int           // CPU-seconds rlimit (default 120)
	Term     string        // TERM handed to the door

	// Deploy-time isolation (opt-in; needs privilege). Zero values = disabled,
	// so the default non-root run is unaffected.
	RunAsUID  int    // drop to this uid before exec (0 = no drop)
	RunAsGID  int    // gid to pair with RunAsUID
	Chroot    string // chroot the door into this dir (Linux; needs /bin/sh inside)
	NoNetwork bool   // Linux: run in a fresh empty network namespace (no net)
	Isolate   bool   // Linux: fresh mount/pid/ipc/uts namespaces
}

// Launch runs a door game as a sandboxed subprocess wired to the caller's
// session I/O. Hardening (SEC-1):
//   - a fully SCRUBBED environment — the door inherits NONE of the daemon's env,
//     so the master key (ADMIRALBBS_KEY) and every other secret are invisible;
//   - a throwaway jail working dir holding only the door32.sys dropfile;
//   - a CPU rlimit + a wall-clock timeout;
//   - its own process group, SIGKILL'd as a group on timeout/disconnect so the
//     door and any children it spawned all die.
//
// Full uid-drop / chroot / namespaces / seccomp require running door execution
// under a dedicated unprivileged uid or a container (deploy-time, S009); this
// launcher is the in-process layer beneath that.
func Launch(sess io.ReadWriter, command string, args []string, drop DropInfo, opts Opts) error {
	if command == "" {
		return errors.New("doors: empty command")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Minute
	}
	if opts.CPULimit <= 0 {
		opts.CPULimit = 120
	}

	jail, err := os.MkdirTemp("", "bbsdoor-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(jail)
	if err := os.Chmod(jail, 0o700); err != nil {
		return err
	}
	if err := WriteDoor32(filepath.Join(jail, "door32.sys"), drop); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// Wrap in /bin/sh to set rlimits in the child, then exec the door. The
	// "$@" passes command+args through without re-quoting.
	sh := fmt.Sprintf("ulimit -t %d 2>/dev/null; exec \"$@\"", opts.CPULimit)
	argv := append([]string{"-c", sh, "sh", command}, args...)
	cmd := exec.CommandContext(ctx, "/bin/sh", argv...)
	cmd.Dir = jail
	cmd.Env = scrubbedEnv(jail, opts.Term)
	cmd.Stdin = sess
	cmd.Stdout = sess
	cmd.Stderr = io.Discard

	spa := &syscall.SysProcAttr{Setpgid: true}
	if opts.RunAsUID > 0 {
		spa.Credential = &syscall.Credential{Uid: uint32(opts.RunAsUID), Gid: uint32(opts.RunAsGID)}
	}
	applyOSIsolation(spa, opts) // platform-specific: Linux adds chroot + namespaces
	cmd.SysProcAttr = spa
	// On cancel (timeout/disconnect) kill the whole process group, not just sh.
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
	cmd.WaitDelay = 3 * time.Second

	return cmd.Run()
}

// scrubbedEnv builds a minimal env from scratch — nothing inherited, so no
// secret in the daemon's environment can reach a door.
func scrubbedEnv(home, term string) []string {
	if term == "" {
		term = "ansi"
	}
	return []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + home,
		"TERM=" + term,
		"DOORFILE=" + filepath.Join(home, "door32.sys"),
	}
}
