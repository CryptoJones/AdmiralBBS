//go:build !linux

package doors

import "syscall"

// applyOSIsolation is a no-op off Linux: chroot/namespaces aren't available the
// same way (e.g. macOS dev box). uid-drop via Credential still applies (set in
// the portable path); full isolation is a Linux/container deploy concern.
func applyOSIsolation(spa *syscall.SysProcAttr, opts Opts) {}
