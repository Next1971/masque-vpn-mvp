package clientcore

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"sync"

	connectip "github.com/quic-go/connect-ip-go"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/yosida95/uritemplate/v3"
	"golang.zx2c4.com/wireguard/tun"
)

// tunOffset is the headroom reserved before packet data for wireguard/tun
// (device.MessageTransportHeaderSize from wireguard-go). It must match
// the server-side value.
const tunOffset = 16

// Verbose enables detailed diagnostic logs (per-packet tracing,
// TTL normalization, etc.). Disabled by default because these logs
// are too noisy for normal use. Wrappers set it via the -verbose flag.
var Verbose bool

// vlog prints a diagnostic message only when Verbose=true.
func vlog(format string, args ...any) {
	if Verbose {
		log.Printf(format, args...)
	}
}

// Session is an active VPN connection. It owns QUIC/UDP resources and TUN,
// and supports graceful shutdown. Created via Connect.
type Session struct {
	udpConn *net.UDPConn
	qconn   *quic.Conn
	ipconn  *connectip.Conn
	dev     tun.Device

	// AssignedPrefixes are the addresses assigned by the server to the client
	// (used by the platform wrapper to configure TUN and routes).
	AssignedPrefixes []netip.Prefix
	// Routes are the routes advertised by the server (typically 0.0.0.0/0).
	Routes []connectip.IPRoute

	closeOnce sync.Once
	done      chan struct{}
}

// buildTLSConfig constructs tls.Config from the profile: server certificate
// verification using the configured CA (required if CA is set) + optional
// client certificate for mTLS.
func buildTLSConfig(p *Profile) (*tls.Config, error) {
	tlsConf := &tls.Config{
		ServerName: p.ServerName,
		NextProtos: []string{http3.NextProtoH3},
	}

	if p.CA != "" {
		caPEM, err := os.ReadFile(p.CA)
		if err != nil {
			return nil, fmt.Errorf("read CA %q: %w", p.CA, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("failed to parse CA %q", p.CA)
		}
		tlsConf.RootCAs = pool
	} else {
		// Without a CA, the server certificate is not verified.
		// Unsafe, intended for debugging only.
		tlsConf.InsecureSkipVerify = true
	}

	if p.Cert != "" && p.Key != "" {
		clientCert, err := tls.LoadX509KeyPair(p.Cert, p.Key)
		if err != nil {
			return nil, fmt.Errorf("load client keypair: %w", err)
		}
		tlsConf.Certificates = []tls.Certificate{clientCert}
	}
	return tlsConf, nil
}

// Connect establishes a MASQUE CONNECT-IP session using the provided profile.
// dev is a ready-to-use TUN interface created by the platform wrapper outside
// the core (the core does not create TUN devices or modify routes).
// After a successful Connect, the caller configures addresses/routes using
// s.AssignedPrefixes / s.Routes, then starts s.Run(ctx).
func Connect(ctx context.Context, p *Profile, dev tun.Device) (*Session, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	udpAddr, err := net.ResolveUDPAddr("udp", p.Server)
	if err != nil {
		return nil, fmt.Errorf("resolve server %q: %w", p.Server, err)
	}
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero})
	if err != nil {
		return nil, fmt.Errorf("listen UDP: %w", err)
	}

	tlsConf, err := buildTLSConfig(p)
	if err != nil {
		udpConn.Close()
		return nil, err
	}

	qconn, err := quic.Dial(ctx, udpConn, udpAddr, tlsConf, &quic.Config{
		EnableDatagrams:   true,
		InitialPacketSize: 1350,
	})
	if err != nil {
		udpConn.Close()
		return nil, fmt.Errorf("QUIC dial: %w", err)
	}
	log.Printf("QUIC connection established to %s", p.Server)

	tr := &http3.Transport{EnableDatagrams: true}
	hconn := tr.NewClientConn(qconn)

	template := uritemplate.MustNew(fmt.Sprintf("https://%s/vpn", p.ServerName))
	ipconn, rsp, err := connectip.Dial(ctx, hconn, template)
	if err != nil {
		qconn.CloseWithError(0, "")
		udpConn.Close()
		return nil, fmt.Errorf("connect-ip dial: %w", err)
	}
	if rsp.StatusCode != http.StatusOK {
		ipconn.Close()
		qconn.CloseWithError(0, "")
		udpConn.Close()
		return nil, fmt.Errorf("unexpected CONNECT-IP status: %d", rsp.StatusCode)
	}
	log.Printf("CONNECT-IP session established (HTTP %d)", rsp.StatusCode)

	prefixes, err := ipconn.LocalPrefixes(ctx)
	if err != nil {
		ipconn.Close()
		qconn.CloseWithError(0, "")
		udpConn.Close()
		return nil, fmt.Errorf("get local prefixes: %w", err)
	}
	if len(prefixes) == 0 {
		ipconn.Close()
		qconn.CloseWithError(0, "")
		udpConn.Close()
		return nil, fmt.Errorf("server assigned no prefixes")
	}
	log.Printf("server assigned prefixes: %v", prefixes)

	routes, err := ipconn.Routes(ctx)
	if err != nil {
		ipconn.Close()
		qconn.CloseWithError(0, "")
		udpConn.Close()
		return nil, fmt.Errorf("get routes: %w", err)
	}
	for _, r := range routes {
		log.Printf("server advertised route: %s - %s (proto %d)", r.StartIP, r.EndIP, r.IPProtocol)
	}

	return &Session{
		udpConn:          udpConn,
		qconn:            qconn,
		ipconn:           ipconn,
		dev:              dev,
		AssignedPrefixes: prefixes,
		Routes:           routes,
		done:             make(chan struct{}),
	}, nil
}

// Run starts bidirectional forwarding between conn↔TUN and blocks until
// completion (an error on either side, s.Close(), or ctx cancellation).
// It returns the first stop reason.
func (s *Session) Run(ctx context.Context) error {
	errCh := make(chan error, 2)
	mtu, err := s.dev.MTU()
	if err != nil || mtu <= 0 {
		mtu = 1400
	}

	// conn → TUN: packets from the server (reply traffic) are written into TUN,
	// where the OS can deliver them to local applications.
	go func() {
		buf := make([]byte, tunOffset+mtu+64)
		var inCount int // diagnostics: number of packets received from conn into TUN
		for {
			n, err := s.ipconn.ReadPacket(buf[tunOffset:])
			if err != nil {
				errCh <- fmt.Errorf("conn read: %w", err)
				return
			}
			inCount++
			if Verbose && inCount <= 6 {
				vlog("conn→TUN packet #%d: %s (%d bytes)", inCount, describePkt(buf[tunOffset:tunOffset+n]), n)
			}
			if _, err := s.dev.Write([][]byte{buf[:tunOffset+n]}, tunOffset); err != nil {
				errCh <- fmt.Errorf("tun write: %w", err)
				return
			}
		}
	}()

	// TUN → conn: packets from local applications (read from TUN) are sent to the server.
	go func() {
		batch := s.dev.BatchSize()
		if batch < 1 {
			batch = 1
		}
		bufs := make([][]byte, batch)
		sizes := make([]int, batch)
		for i := range bufs {
			bufs[i] = make([]byte, tunOffset+mtu+64)
		}
		var fixedCount int // diagnostics: number of packets with raised TTL/Hop Limit
		for {
			k, err := s.dev.Read(bufs, sizes, tunOffset)
			if err != nil {
				errCh <- fmt.Errorf("tun read: %w", err)
				return
			}
			select {
			case <-s.done:
				return
			default:
			}
			for i := 0; i < k; i++ {
				pkt := bufs[i][tunOffset : tunOffset+sizes[i]]
				// Raise a too-small TTL / Hop Limit, otherwise connect-ip
				// may drop the packet ("Hop Limit too small"). This is a shared
				// fix for all platforms, especially useful with Windows TUN routing.
				if orig, fixed := normalizeTTL(pkt); fixed {
					fixedCount++
					if fixedCount <= 3 {
						vlog("raised low TTL/HopLimit %d→%d on outgoing packet (%d bytes)", orig, fixTTL, len(pkt))
					}
				}
				if _, err := s.ipconn.WritePacket(pkt); err != nil {
					errCh <- fmt.Errorf("conn write: %w", err)
					return
				}
			}
		}
	}()

	log.Printf("forwarding started (conn↔TUN)")

	var runErr error
	select {
	case runErr = <-errCh:
	case <-ctx.Done():
		runErr = ctx.Err()
	}
	s.Close()
	return runErr
}

// Close gracefully shuts the session down: first closes CONNECT-IP
// (so the server can release the address back to the pool immediately),
// then QUIC and UDP. TUN is NOT closed here — its lifecycle is controlled
// by the platform wrapper that created it and will also clean up routes.
func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
		if s.ipconn != nil {
			s.ipconn.Close() // sends CONNECT-IP close → server does Release
		}
		if s.qconn != nil {
			s.qconn.CloseWithError(0, "client shutdown")
		}
		if s.udpConn != nil {
			s.udpConn.Close()
		}
		log.Printf("session closed gracefully")
	})
	return nil
}
