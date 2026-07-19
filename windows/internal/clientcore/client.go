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

// tunOffset — запас в буфере перед данными пакета для wireguard/tun
// (device.MessageTransportHeaderSize из wireguard-go). Должен совпадать
// с серверным значением.
const tunOffset = 16

// Verbose включает подробные диагностические логи (per-packet трассировка
// conn→TUN, нормализация TTL и т.п.). По умолчанию выключено — в проде
// эти логи спамят. Обёртки выставляют его из флага -verbose.
var Verbose bool

// vlog печатает диагностическое сообщение только при Verbose=true.
func vlog(format string, args ...any) {
	if Verbose {
		log.Printf(format, args...)
	}
}

// Session — активное VPN-подключение. Держит QUIC/UDP-ресурсы и TUN,
// умеет грациозно закрываться. Создаётся через Connect.
type Session struct {
	udpConn *net.UDPConn
	qconn   *quic.Conn
	ipconn  *connectip.Conn
	dev     tun.Device

	// AssignedPrefixes — адреса, выданные сервером клиенту (для настройки
	// TUN и маршрутов платформенной обёрткой).
	AssignedPrefixes []netip.Prefix
	// Routes — маршруты, объявленные сервером (обычно 0.0.0.0/0).
	Routes []connectip.IPRoute

	closeOnce sync.Once
	done      chan struct{}
}

// buildTLSConfig собирает tls.Config из профиля: проверка серверного серта
// по CA (обязательна, если CA задан) + клиентский серт для mTLS.
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
		// Без CA сервер не проверяется — небезопасно, только для отладки.
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

// Connect устанавливает MASQUE CONNECT-IP сессию по профилю.
// dev — готовый TUN-интерфейс, созданный платформенной обёрткой снаружи
// (ядро не создаёт TUN и не трогает маршруты). После успешного Connect
// вызывающая сторона настраивает адрес/маршруты по s.AssignedPrefixes/s.Routes,
// затем запускает s.Run(ctx).
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

// Run запускает двунаправленный форвардинг conn↔TUN и блокируется до
// завершения (ошибка одной из сторон, s.Close() или отмена ctx).
// Возвращает первую причину остановки.
func (s *Session) Run(ctx context.Context) error {
	errCh := make(chan error, 2)
	mtu, err := s.dev.MTU()
	if err != nil || mtu <= 0 {
		mtu = 1400
	}

	// conn → TUN: пакеты от сервера (ответы из интернета) пишем в TUN,
	// откуда их читает ОС и отдаёт приложениям.
	go func() {
		buf := make([]byte, tunOffset+mtu+64)
		var inCount int // диагностика: сколько пакетов пришло из conn в TUN
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

	// TUN → conn: пакеты от приложений (из TUN) отправляем на сервер.
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
		var fixedCount int // сколько пакетов с поднятым TTL (диагностика)
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
				// Поднимаем слишком маленький TTL/Hop Limit, иначе connect-ip
				// отбросит пакет ("Hop Limit too small"). Общий фикс для всех
				// платформ (актуально для Windows-маршрутизации в TUN).
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

// Close грациозно завершает сессию: закрывает CONNECT-IP (сервер сразу
// освобождает адрес в пул), затем QUIC и UDP. TUN НЕ закрывается здесь —
// его жизненным циклом управляет платформенная обёртка (создала — она и
// закроет, вместе с откатом маршрутов).
func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
		if s.ipconn != nil {
			s.ipconn.Close() // шлёт CONNECT-IP close → сервер делает Release
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
