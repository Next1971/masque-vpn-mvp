//go:build linux

// iface_linux.go — bring up a TUN interface on Linux using the `ip` utility.
// Isolated via a build tag so that the core MASQUE logic stays cross-platform:
// on Windows/Android, interface setup will be specific (wintun / VpnService).
package main

import (
	"fmt"
	"os/exec"
)

// bringUpTUN assigns an address to the TUN interface and brings it up.
// addr is a CIDR string, for example "10.8.0.1/24".
func bringUpTUN(name, addr string) error {
	// ip addr add <addr> dev <name>
	if out, err := exec.Command("ip", "addr", "add", addr, "dev", name).CombinedOutput(); err != nil {
		return fmt.Errorf("ip addr add: %w: %s", err, out)
	}
	// ip link set <name> up
	if out, err := exec.Command("ip", "link", "set", name, "up").CombinedOutput(); err != nil {
		return fmt.Errorf("ip link set up: %w: %s", err, out)
	}
	return nil
}
