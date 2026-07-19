// ippool.go — thread-safe IPv4 address pool for client assignment (E1).
// From pool_cidr we allocate one /32 per client, skipping the server address,
// the network address, and the broadcast address.
package main

import (
	"fmt"
	"net/netip"
	"sync"
)

// IPPool allocates addresses from a given CIDR, excluding reserved ones.
type IPPool struct {
	mu       sync.Mutex
	free     []netip.Addr        // free addresses (stack)
	inUse    map[netip.Addr]bool // currently allocated
	prefixOf map[netip.Addr]netip.Prefix
}

// NewIPPool builds a pool from poolCIDR, reserving serverAddr (server tunnel address),
// as well as the network and broadcast addresses of the range.
func NewIPPool(poolCIDR string, serverAddr netip.Addr) (*IPPool, error) {
	prefix, err := netip.ParsePrefix(poolCIDR)
	if err != nil {
		return nil, fmt.Errorf("parse pool_cidr %q: %w", poolCIDR, err)
	}
	prefix = prefix.Masked()
	if !prefix.Addr().Is4() {
		return nil, fmt.Errorf("only IPv4 pools supported, got %q", poolCIDR)
	}

	network := prefix.Addr()
	broadcast := lastIPOfPrefix(prefix)

	p := &IPPool{
		inUse:    make(map[netip.Addr]bool),
		prefixOf: make(map[netip.Addr]netip.Prefix),
	}

	addr := network.Next() // first host
	for addr.Is4() && addr.Less(broadcast) {
		if addr != serverAddr {
			p.free = append(p.free, addr)
		}
		addr = addr.Next()
	}
	if len(p.free) == 0 {
		return nil, fmt.Errorf("pool %q has no assignable addresses", poolCIDR)
	}
	return p, nil
}

// Allocate returns a free address as a /32 prefix. It fails if the pool is exhausted.
func (p *IPPool) Allocate() (netip.Prefix, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.free) == 0 {
		return netip.Prefix{}, fmt.Errorf("IP pool exhausted")
	}
	addr := p.free[len(p.free)-1]
	p.free = p.free[:len(p.free)-1]
	p.inUse[addr] = true
	pfx := netip.PrefixFrom(addr, 32)
	p.prefixOf[addr] = pfx
	return pfx, nil
}

// Release returns an address back to the pool.
func (p *IPPool) Release(pfx netip.Prefix) {
	addr := pfx.Addr()
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.inUse[addr] {
		return
	}
	delete(p.inUse, addr)
	delete(p.prefixOf, addr)
	p.free = append(p.free, addr)
}

// Available returns the number of free addresses (for logging/metrics).
func (p *IPPool) Available() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.free)
}
