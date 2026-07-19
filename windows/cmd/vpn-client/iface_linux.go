//go:build linux

package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"os/exec"
	"strings"
)

// runCmd выполняет команду и возвращает ошибку с выводом при неудаче.
func runCmd(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ifUp поднимает интерфейс с адресом клиента (маску берём /32, адрес точечный).
func ifUp(iface string, addr netip.Prefix) error {
	// Адрес выдан как /32 — назначаем его на интерфейс с /32 и поднимаем линк.
	if err := runCmd("ip", "addr", "add", addr.String(), "dev", iface); err != nil {
		return err
	}
	if err := runCmd("ip", "link", "set", "dev", iface, "up"); err != nil {
		return err
	}
	return nil
}

// setupTestRoute добавляет маршрут ТОЛЬКО до dst через TUN, не трогая default.
// Указываем src = адрес клиента в туннеле, чтобы исходящие пакеты имели
// правильный source (иначе ядро NAT-ит чужой src и ответ не вернётся).
// Безопасно на VPS: SSH и серверный трафик идут прежним путём.
// Возвращает cleanup, удаляющий маршрут.
func setupTestRoute(iface string, dst netip.Addr, src netip.Addr) (func(), error) {
	route := dst.String() + "/32"
	if err := runCmd("ip", "route", "add", route, "dev", iface, "src", src.String()); err != nil {
		return nil, err
	}
	return func() {
		if err := runCmd("ip", "route", "del", route, "dev", iface); err != nil {
			log.Printf("cleanup: del route %s: %v", route, err)
		}
	}, nil
}

// setupFullRoute заворачивает весь трафик в TUN. Чтобы QUIC-пакеты до VPS
// не зациклились в туннель, добавляет host-route до сервера через текущий
// default-шлюз. Только для реального устройства (E3), НЕ на VPS.
func setupFullRoute(iface, server string, _ netip.Addr, _ []string) (func(), error) {
	// Извлекаем IP сервера (host:port).
	host := server
	if i := strings.LastIndex(server, ":"); i > 0 {
		host = server[:i]
	}
	serverIP, err := netip.ParseAddr(host)
	if err != nil {
		return nil, fmt.Errorf("server host %q is not an IP (test mode expects literal IP): %w", host, err)
	}

	// Определяем текущий default-шлюз.
	gw, dev, err := defaultGateway()
	if err != nil {
		return nil, fmt.Errorf("detect default gateway: %w", err)
	}
	log.Printf("current default gateway: %s dev %s", gw, dev)

	// 1. Host-route до VPS через прежний шлюз (иначе петля).
	srvRoute := serverIP.String() + "/32"
	if err := runCmd("ip", "route", "add", srvRoute, "via", gw.String(), "dev", dev); err != nil {
		return nil, fmt.Errorf("add server bypass route: %w", err)
	}

	// 2. Заворачиваем весь трафик в TUN двумя половинками /1 (перекрывают default,
	//    но не удаляют исходный — легко откатить).
	added := []string{}
	for _, half := range []string{"0.0.0.0/1", "128.0.0.0/1"} {
		if err := runCmd("ip", "route", "add", half, "dev", iface); err != nil {
			// откат уже добавленного
			for _, h := range added {
				_ = runCmd("ip", "route", "del", h, "dev", iface)
			}
			_ = runCmd("ip", "route", "del", srvRoute, "via", gw.String(), "dev", dev)
			return nil, fmt.Errorf("add default-half %s: %w", half, err)
		}
		added = append(added, half)
	}

	return func() {
		for _, h := range added {
			if err := runCmd("ip", "route", "del", h, "dev", iface); err != nil {
				log.Printf("cleanup: del %s: %v", h, err)
			}
		}
		if err := runCmd("ip", "route", "del", srvRoute, "via", gw.String(), "dev", dev); err != nil {
			log.Printf("cleanup: del server route: %v", err)
		}
	}, nil
}

// defaultGateway парсит `ip route show default` → (gateway, dev).
func defaultGateway() (netip.Addr, string, error) {
	out, err := exec.Command("ip", "route", "show", "default").CombinedOutput()
	if err != nil {
		return netip.Addr{}, "", fmt.Errorf("ip route show default: %w", err)
	}
	// пример: "default via 203.0.113.1 dev ens3 proto static"
	fields := strings.Fields(string(out))
	var gw, dev string
	for i := 0; i < len(fields)-1; i++ {
		switch fields[i] {
		case "via":
			gw = fields[i+1]
		case "dev":
			dev = fields[i+1]
		}
	}
	if gw == "" || dev == "" {
		return netip.Addr{}, "", fmt.Errorf("could not parse default route: %q", strings.TrimSpace(string(out)))
	}
	addr, err := netip.ParseAddr(gw)
	if err != nil {
		return netip.Addr{}, "", fmt.Errorf("parse gateway %q: %w", gw, err)
	}
	return addr, dev, nil
}

// runPingTest шлёт ICMP echo системным ping'ом через уже настроенный маршрут,
// привязываясь к клиентскому TUN (-I iface), чтобы source был адресом в туннеле.
// Пакеты идут TUN → ядро → сервер → интернет → назад. Проверяет data-plane.
func runPingTest(ctx context.Context, dst, iface string, count int) error {
	log.Printf("sending %d ICMP echo(s) to %s via tunnel (bind %s)...", count, dst, iface)
	out, err := exec.CommandContext(ctx, "ping", "-c", fmt.Sprint(count), "-W", "5", "-I", iface, dst).CombinedOutput()
	log.Printf("ping output:\n%s", strings.TrimSpace(string(out)))
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	log.Printf("✅ ping through tunnel SUCCEEDED — client core data-plane WORKS")
	return nil
}
