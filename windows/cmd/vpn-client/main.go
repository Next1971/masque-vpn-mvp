package main

import (
    "context"
   
    "flag"
    "fmt"
    "io"
    "log"
    "net/http"
    "net/netip"
    "os"
    "os/signal"
    "syscall"
    "time"

    "masque-client"
    "golang.zx2c4.com/wireguard/tun"
)



var (
    globalCancel context.CancelFunc
    globalProf   *clientcore.Profile
)

func main() {
    // 1. Восстанавливаем флаги для командной строки
    profilePath := flag.String("profile", "", "path to client profile TOML (required)")
    testMode := flag.Bool("test", true, "test mode: route only -test-dst via TUN")
    fullRoute := flag.Bool("full-route", false, "full mode: route all traffic via TUN")
    testDst := flag.String("test-dst", "1.1.1.1", "test-mode: destination to route through tunnel")
    pingCount := flag.Int("ping", 3, "test-mode: number of ICMP echo requests to send")
    timeout := flag.Duration("timeout", 25*time.Second, "overall timeout")
    flag.Parse()

    // 2. Если передали -profile, работаем в консоли
    if *profilePath != "" {
        prof, err := clientcore.LoadProfile(*profilePath)
        if err != nil {
            log.Fatalf("FAIL: load profile: %v", err)
        }
        ctx, cancel := context.WithTimeout(context.Background(), *timeout)
        defer cancel()

        err = run(ctx, prof, *testMode, *fullRoute, *testDst, *pingCount)
        if err != nil {
            log.Fatalf("FAIL: %v", err)
        }
        return
    }

    // 3. Если флагов нет — запускаем Веб-интерфейс
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        data, err := os.ReadFile("index.html")
        if err != nil {
            http.Error(w, "Не удалось прочитать index.html", 500)
            return
        }
        w.Write(data)
    })

    http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
            http.Error(w, "Method not allowed", 405)
            return
        }
        file, _, err := r.FormFile("file")
        if err != nil {
            http.Error(w, "Ошибка чтения файла", 400)
            return
        }
        defer file.Close()

        tempPath := "uploaded_profile.toml"
        out, err := os.Create(tempPath)
        if err != nil {
            http.Error(w, "Не удалось сохранить файл", 500)
            return
        }
        defer out.Close()
        io.Copy(out, file)

        prof, err := clientcore.LoadProfile(tempPath)
        if err != nil {
            http.Error(w, "Ошибка парсинга TOML: "+err.Error(), 400)
            return
        }
        globalProf = prof
        w.WriteHeader(200)
    })

    http.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
        if globalProf == nil {
            w.Write([]byte(`{"status":"error", "error":"Сначала загрузите конфиг"}`))
            return
        }
        if globalCancel != nil {
            w.Write([]byte(`{"status":"error", "error":"Уже включено"}`))
            return
        }

        ctx, cancel := context.WithCancel(context.Background())
        globalCancel = cancel

        go func() {
            err := run(ctx, globalProf, false, true, "1.1.1.1", 3)
            if err != nil {
                log.Printf("VPN Error: %v", err)
            }
            globalCancel = nil
        }()

        w.Write([]byte(`{"status":"ok", "ip":"Подключено"}`))
    })

    http.HandleFunc("/disconnect", func(w http.ResponseWriter, r *http.Request) {
        if globalCancel != nil {
            globalCancel()
            globalCancel = nil
        }
        w.Write([]byte(`{"status":"ok"}`))
    })

    log.Println("Веб-интерфейс запущен! Откройте в браузере: http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
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
