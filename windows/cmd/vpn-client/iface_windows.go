//go:build windows

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os/exec"
	"strconv"
	"strings"
)

// tunnelSubnetBits is the tunnel subnet mask (10.8.0.0/24 pool on the server).
// The client address is assigned to the interface with this mask so the whole subnet
// is on-link and next-hop 10.8.0.1 is reachable as a regular gateway.
const tunnelSubnetBits = 24

// tunnelGateway returns the server's tunnel address = the first address
// of the client's subnet (e.g. for 10.8.0.254/24 → 10.8.0.1). The server reserves .1 for itself.
func tunnelGateway(client netip.Addr) netip.Addr {
	if !client.Is4() {
		return client
	}
	p := netip.PrefixFrom(client, tunnelSubnetBits).Masked()
	// first host in subnet = network + 1
	b := p.Addr().As4()
	b[3]++
	return netip.AddrFrom4(b)
}

// runCmd runs a command and returns an error with its output on failure.
func runCmd(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ifUp assigns the client address to the Wintun adapter and brings it up.
// On Windows we use netsh; for a /32 address from the tunnel, the mask is 255.255.255.255,
// but netsh accepts such an address as the host address of the interface. To ensure the system
// correctly builds on-link routes in TUN, we assign the address with the mask of the assigned
// prefix (usually /32) and explicitly define the subnet routes later.
func ifUp(iface string, addr netip.Prefix) error {
	ip := addr.Addr().String()
	// IMPORTANT: the server assigns the address as /32. If /32 is assigned to the interface,
	// it has no on-link subnet, and a route to dst via "gateway = the interface's own address"
	// is treated by Windows as single-hop → TTL=1 → connect-ip-go
	// drops such packets ("Hop Limit too small: 1"). Therefore we assign the address
	// with a /24 mask (the entire 10.8.0.0/24 pool becomes on-link), and direct traffic to dst
	// via gateway 10.8.0.1 (the server's address in the tunnel) — this is a normal
	// gateway hop with a normal TTL.
	mask := prefixToMask(tunnelSubnetBits)
	// netsh interface ip set address name="<iface>" static <ip> <mask>
	if err := runCmd("netsh", "interface", "ip", "set", "address",
		"name="+iface, "static", ip, mask); err != nil {
		return err
	}
	// MTU is already set during CreateTUN; set it via netsh just in case.
	// (not critical, do not treat error as fatal)
	_ = runCmd("netsh", "interface", "ipv4", "set", "subinterface", iface, "mtu=1400", "store=active")
	return nil
}

// setupTestRoute adds a route ONLY to dst via TUN, leaving the default route untouched.
// On Windows: route add <dst> mask 255.255.255.255 <gateway> if <ifindex>.
// Gateway = server address in the tunnel (the first address in the client subnet, e.g.
// 10.8.0.1). This is a REAL next-hop inside the on-link /24 subnet, so
// Windows generates packets with a normal TTL (not single-hop), and connect-ip-go
// proxies them instead of dropping them as "Hop Limit too small: 1".
func setupTestRoute(iface string, dst netip.Addr, src netip.Addr) (func(), error) {
	idx, err := ifIndex(iface)
	if err != nil {
		return nil, err
	}
	gw := tunnelGateway(src)
	dstStr := dst.String()
	// route add 1.1.1.1 mask 255.255.255.255 <tunnel-gw=10.8.0.1> metric 1 if <idx>
	if err := runCmd("route", "add", dstStr, "mask", "255.255.255.255",
		gw.String(), "metric", "1", "if", strconv.Itoa(idx)); err != nil {
		return nil, err
	}
	return func() {
		if err := runCmd("route", "delete", dstStr); err != nil {
			log.Printf("cleanup: route delete %s: %v", dstStr, err)
		}
	}, nil
}

// setupFullRoute sends all traffic into TUN. To avoid looping QUIC packets to the VPS,
// it adds a host-route to the server via the current default gateway.
// Then it adds two /1 halves that override the default route (easy to roll back).
func setupFullRoute(iface, server string, client netip.Addr, dns []string) (func(), error) {
	host := server
	if i := strings.LastIndex(server, ":"); i > 0 {
		host = server[:i]
	}
	serverIP, err := netip.ParseAddr(host)
	if err != nil {
		// host is a name, not an IP: resolve it to IPv4 for the bypass route to the VPS.
		ips, rerr := net.LookupIP(host)
		if rerr != nil {
			return nil, fmt.Errorf("resolve server host %q: %w", host, rerr)
		}
		for _, ip := range ips {
			if v4 := ip.To4(); v4 != nil {
				if a, ok := netip.AddrFromSlice(v4); ok {
					serverIP = a
					break
				}
			}
		}
		if !serverIP.IsValid() {
			return nil, fmt.Errorf("no IPv4 address for server host %q", host)
		}
		log.Printf("resolved server %s → %s (for bypass route)", host, serverIP)
	}

	gw, gwIdx, err := defaultGatewayWin()
	if err != nil {
		return nil, fmt.Errorf("detect default gateway: %w", err)
	}
	log.Printf("current default gateway: %s (if %d)", gw, gwIdx)

	idx, err := ifIndex(iface)
	if err != nil {
		return nil, err
	}
	// next-hop in the tunnel (10.8.0.1) — so packets use a normal TTL,
	// not single-hop TTL=1 (see comment in setupTestRoute).
	// Take the address from the one assigned by the server (client), rather than querying
	// the interface — otherwise there is a race with Windows/Wintun (the address may not be applied yet).
	tunGW := tunnelGateway(client)

	// 1. Host-route to the VPS through the previous gateway (otherwise a loop).
	// If gwIdx==0 — do not specify "if", route will choose the interface by gateway.
	srvStr := serverIP.String()
	srvArgs := []string{"add", srvStr, "mask", "255.255.255.255", gw.String(), "metric", "1"}
	if gwIdx > 0 {
		srvArgs 
