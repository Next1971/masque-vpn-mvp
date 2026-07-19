// Package clientcore — shared MASQUE client core for all platforms
// (Linux/Windows/Android). The core does NOT create a TUN device itself
// and does NOT modify routes — those platform-specific details are injected
// from thin wrappers outside the core:
//   - Linux:   cmd/vpn-client (CreateTUN by name + ip route)
//   - Windows: wintun + netsh wrapper (next stage)
//   - Android: TUN fd from VpnService + CreateTUNFromFile (next stage)
//
// This allows the same connection / forwarding / shutdown logic to be reused
// across all platforms — the "single core" model described in PROJECT.md.
package clientcore

import (
	"fmt"
	"net/netip"

	"github.com/BurntSushi/toml"
)

// Profile is the client-side server profile.
// The same parameter set is intended for both Android and Windows
// (as required by PROJECT.md). It is loaded from a TOML file,
// which is expected to be edited through the UI on the target device.
//
// Secrets (such as the client private key) are stored as FILE PATHS,
// not inline values, so the profile can be displayed or logged without leaking
// sensitive material. (On Android/Windows, a future UI may store the key
// in protected storage instead.)
type Profile struct {
	// [server]
	Server     string `toml:"server"`      // host:port of the MASQUE proxy (UDP), e.g. "80.85.241.127:4433"
	ServerName string `toml:"server_name"` // TLS SNI / URI-template host, e.g. "masque.example.com"

	// [tls] — mTLS material (paths to PEM files, NOT inline secrets)
	CA   string `toml:"ca"`   // CA used to verify the server certificate
	Cert string `toml:"cert"` // client certificate (mTLS)
	Key  string `toml:"key"`  // client private key (mTLS)

	// [tun]
	TUNName string   `toml:"tun_name"` // interface name (Linux/Windows), e.g. "masque0"
	MTU     int      `toml:"mtu"`      // tunnel MTU, e.g. 1400
	DNS     []string `toml:"dns"`      // DNS servers for the tunnel (full-route), default ["1.1.1.1"]
}

// tomlProfile is an intermediate structure for TOML sections.
type tomlProfile struct {
	Server struct {
		Server     string `toml:"server"`
		ServerName string `toml:"server_name"`
	} `toml:"server"`
	TLS struct {
		CA   string `toml:"ca"`
		Cert string `toml:"cert"`
		Key  string `toml:"key"`
	} `toml:"tls"`
	TUN struct {
		Name string   `toml:"tun_name"`
		MTU  int      `toml:"mtu"`
		DNS  []string `toml:"dns"`
	} `toml:"tun"`
}

// LoadProfile reads and validates a TOML client profile.
// Unknown keys are treated as an error to protect against typos.
func LoadProfile(path string) (*Profile, error) {
	var tp tomlProfile
	md, err := toml.DecodeFile(path, &tp)
	if err != nil {
		return nil, fmt.Errorf("decode profile %q: %w", path, err)
	}
	if undec := md.Undecoded(); len(undec) > 0 {
		return nil, fmt.Errorf("profile %q has unknown keys: %v", path, undec)
	}

	p := &Profile{
		Server:     tp.Server.Server,
		ServerName: tp.Server.ServerName,
		CA:         tp.TLS.CA,
		Cert:       tp.TLS.Cert,
		Key:        tp.TLS.Key,
		TUNName:    tp.TUN.Name,
		MTU:        tp.TUN.MTU,
		DNS:        tp.TUN.DNS,
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}

// Validate checks required profile fields.
func (p *Profile) Validate() error {
	if p.Server == "" {
		return fmt.Errorf("profile: [server].server is required (host:port)")
	}
	if _, err := netip.ParseAddrPort(p.Server); err != nil {
		// Hostname:port is allowed — ParseAddrPort requires a literal IP,
		// so strict validation is deferred to the dial/resolve stage.
		if !hasPort(p.Server) {
			return fmt.Errorf("profile: [server].server %q must be host:port", p.Server)
		}
	}
	if p.ServerName == "" {
		return fmt.Errorf("profile: [server].server_name is required (TLS SNI)")
	}
	if p.MTU == 0 {
		p.MTU = 1400 // reasonable default for QUIC/MASQUE
	}
	if p.MTU < 576 || p.MTU > 9000 {
		return fmt.Errorf("profile: [tun].mtu %d out of range (576..9000)", p.MTU)
	}
	if p.TUNName == "" {
		p.TUNName = "masque0"
	}
	if len(p.DNS) == 0 {
		p.DNS = []string{"1.1.1.1"} // reasonable default for the tunnel
	}
	// Validate DNS addresses.
	for _, d := range p.DNS {
		if _, err := netip.ParseAddr(d); err != nil {
			return fmt.Errorf("profile: [tun].dns %q is not a valid IP: %w", d, err)
		}
	}
	return nil
}

// hasPort performs a simple check for a trailing ":port".
func hasPort(s string) bool {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i < len(s)-1
		}
	}
	return false
}
