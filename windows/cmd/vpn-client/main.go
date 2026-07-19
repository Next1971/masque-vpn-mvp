// vpn-client — Linux-обёртка вокруг общего клиентского ядра clientcore.
//
// Ядро (internal/clientcore) платформо-независимо и НЕ трогает TUN/маршруты.
// Эта обёртка делает платформенную часть для Linux:
//   - создаёт TUN (wireguard/tun, CreateTUN по имени);
//   - поднимает интерфейс с выданным сервером адресом (ip addr/link);
//   - настраивает маршрутизацию;
//   - по сигналу грациозно закрывает сессию и откатывает интерфейс.
//
// Два режима маршрутизации:
//   test (по умолчанию): маршрут только до -test-dst через клиентский TUN;
//     НЕ трогает default route хоста → безопасно для loopback-теста на VPS.
//   full (-full-route):  заворачивает весь трафик (0.0.0.0/0) в TUN, добавляя
//     host-route до VPS-сервера через прежний шлюз (чтобы QUIC не зациклился).
//     Для реального устройства (E3), НЕ запускать на самом VPS.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zavodovskii/masque-b2/internal/clientcore"
	"golang.zx2c4.com/wireguard/tun"
)

func main() {
	var (
		profilePath = flag.String("profile", "", "path to client profile TOML (required)")
		testMode    = flag.Bool("test", true, "test mode: route only -test-dst via TUN (safe on VPS)")
		fullRoute   = flag.Bool("full-route", false, "full mode: route all traffic via TUN (real device only)")
		testDst     = flag.String("test-dst", "1.1.1.1", "test-mode: destination to route through tunnel")
		pingCount   = flag.Int("ping", 3, "test-mode: number of ICMP echo requests to send")
		timeout     = flag.Duration("timeout", 25*time.Second, "overall timeout")
		verbose     = flag.Bool("verbose", false, "verbose diagnostics (per-packet conn→TUN trace, TTL fixes)")
	)
	flag.Parse()
	clientcore.Verbose = *verbose

	if *profilePath == "" {
		log.Fatalf("FAIL: -profile is required")
	}

	prof, err := clientcore.LoadProfile(*profilePath)
	if err != nil {
		log.Fatalf("FAIL: load profile: %v", err)
	}
	log.Printf("loaded profile: server=%s server_name=%s tun=%s mtu=%d",
		prof.Server, prof.ServerName, prof.TUNName, prof.MTU)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	if err := run(ctx, prof, *testMode, *fullRoute, *testDst, *pingCount); err != nil {
		log.Fatalf("FAIL: %v", err)
	}
}

func run(ctx context.Context, prof *clientcore.Profile, testMode, fullRoute bool, testDst string, pingCount int) error {
	// 1. Создаём TUN-интерфейс (платформенная деталь Linux).
	dev, err := tun.CreateTUN(prof.TUNName, prof.MTU)
	if err != nil {
		return fmt.Errorf("create TUN %q: %w", prof.TUNName, err)
	}
	name, _ := dev.Name()
	log.Printf("TUN %s created (mtu %d)", name, prof.MTU)
	defer func() {
		dev.Close()
		log.Printf("TUN %s closed", name)
	}()

	// 2. Подключаемся ядром (QUIC + mTLS + CONNECT-IP).
	sess, err := clientcore.Connect(ctx, prof, dev)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	// 3. Поднимаем интерфейс с выданным адресом.
	if len(sess.AssignedPrefixes) == 0 {
		sess.Close()
		return fmt.Errorf("no address assigned")
	}
	clientAddr := sess.AssignedPrefixes[0]
	if err := ifUp(name, clientAddr); err != nil {
		sess.Close()
		return fmt.Errorf("bring up %s: %w", name, err)
	}
	log.Printf("interface %s up with %s", name, clientAddr)

	// 4. Маршрутизация.
	var cleanup func()
	if fullRoute {
		cleanup, err = setupFullRoute(name, prof.Server, clientAddr.Addr(), prof.DNS)
		if err != nil {
			sess.Close()
			return fmt.Errorf("setup full route: %w", err)
		}
		log.Printf("full-route mode: all traffic via %s", name)
	} else {
		dst, perr := netip.ParseAddr(testDst)
		if perr != nil {
			sess.Close()
			return fmt.Errorf("parse test-dst %q: %w", testDst, perr)
		}
		cleanup, err = setupTestRoute(name, dst, clientAddr.Addr())
		if err != nil {
			sess.Close()
			return fmt.Errorf("setup test route: %w", err)
		}
		log.Printf("test mode: routing %s/32 via %s (default route untouched)", dst, name)
	}
	defer cleanup()

	// 5. Запускаем форвардинг в фоне.
	runErr := make(chan error, 1)
	go func() { runErr <- sess.Run(ctx) }()

	// 6a. Тест-режим: шлём ICMP echo и ждём reply через TUN хоста.
	if testMode && !fullRoute {
		if err := runPingTest(ctx, testDst, name, pingCount); err != nil {
			log.Printf("ping test note: %v", err)
		}
		sess.Close()
		<-runErr
		return nil
	}

	// 6b. Полный режим: работаем до сигнала.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
		log.Printf("signal received, shutting down")
	case err := <-runErr:
		sess.Close()
		return err
	case <-ctx.Done():
	}
	sess.Close()
	<-runErr
	return nil
}
