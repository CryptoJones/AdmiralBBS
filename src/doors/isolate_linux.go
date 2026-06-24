//go:build linux

package doors

import "syscall"

// applyOSIsolation adds Linux-kernel isolation when requested (needs privilege):
// a chroot, fresh namespaces (mount/pid/ipc/uts, and an empty network namespace
// so the door has NO network), and SIGKILL-on-parent-death.
func applyOSIsolation(spa *syscall.SysProcAttr, opts Opts) {
	if opts.Chroot != "" {
		spa.Chroot = opts.Chroot
	}
	var flags uintptr
	if opts.Isolate {
		flags |= syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWIPC | syscall.CLONE_NEWUTS
	}
	if opts.NoNetwork {
		flags |= syscall.CLONE_NEWNET // empty net namespace == no network for the door
	}
	spa.Cloneflags |= flags
	spa.Pdeathsig = syscall.SIGKILL
}
