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

// tunnelSubnetBits — маска подсети туннеля (пул 10.8.0.0/24 на сервере).
// Адрес клиента назначается на интерфейс с этой маской, чтобы вся подсеть
// была on-link и next-hop 10.8.0.1 был достижим как обычный шлюз.
const tunnelSubnetBits = 24

// tunnelGateway возвращает адрес сервера в туннеле = первый адрес подсети
// клиента (напр. для 10.8.0.254/24 → 10.8.0.1). Сервер резервирует .1 за собой.
func tunnelGateway(client netip.Addr) netip.Addr {
	if !client.Is4() {
		return client
	}
	p := netip.PrefixFrom(client, tunnelSubnetBits).Masked()
	// первый хост в подсети = network + 1
	b := p.Addr().As4()
	b[3]++
	return netip.AddrFrom4(b)
}

// runCmd выполняет команду и возвращает ошибку с выводом при неудаче.
func runCmd(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ifUp назначает адрес клиента на Wintun-адаптер и поднимает его.
// На Windows используем netsh; маска для /32-адреса из туннеля — 255.255.255.255,
// но такой адрес netsh принимает как host-адрес интерфейса. Чтобы система
// корректно строила on-link маршруты в TUN, ставим адрес с маской из назначенного
// префикса (обычно /32) и явно задаём саму подсеть маршрутами позже.
func ifUp(iface string, addr netip.Prefix) error {
	ip := addr.Addr().String()
	// ВАЖНО: сервер выдаёт адрес как /32. Если назначить /32 на интерфейс,
	// у него нет on-link подсети, и маршрут до dst через "gateway = свой же
	// адрес" Windows трактует как single-hop → TTL=1 → connect-ip-go
	// отбрасывает пакеты ("Hop Limit too small: 1"). Поэтому назначаем адрес
	// с маской /24 (весь пул 10.8.0.0/24 становится on-link), а трафик до dst
	// направляем через шлюз 10.8.0.1 (адрес сервера в туннеле) — это обычный
	// gateway-хоп с нормальным TTL.
	mask := prefixToMask(tunnelSubnetBits)
	// netsh interface ip set address name="<iface>" static <ip> <mask>
	if err := runCmd("netsh", "interface", "ip", "set", "address",
		"name="+iface, "static", ip, mask); err != nil {
		return err
	}
	// MTU уже задан при CreateTUN; на всякий случай выставим через netsh.
	// (не критично, ошибку не считаем фатальной)
	_ = runCmd("netsh", "interface", "ipv4", "set", "subinterface", iface, "mtu=1400", "store=active")
	return nil
}

// setupTestRoute добавляет маршрут ТОЛЬКО до dst через TUN, не трогая default.
// На Windows: route add <dst> mask 255.255.255.255 <gateway> if <ifindex>.
// Gateway = адрес сервера в туннеле (первый адрес подсети клиента, напр.
// 10.8.0.1). Это НАСТОЯЩИЙ next-hop внутри on-link подсети /24, поэтому
// Windows формирует пакеты с нормальным TTL (не single-hop), и connect-ip-go
// их проксирует, а не отбрасывает как "Hop Limit too small: 1".
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

// setupFullRoute заворачивает весь трафик в TUN. Чтобы QUIC-пакеты до VPS
// не зациклились, добавляет host-route до сервера через текущий default-шлюз.
// Затем добавляет две половинки /1, перекрывающие default (легко откатываются).
func setupFullRoute(iface, server string, client netip.Addr, dns []string) (func(), error) {
	host := server
	if i := strings.LastIndex(server, ":"); i > 0 {
		host = server[:i]
	}
	serverIP, err := netip.ParseAddr(host)
	if err != nil {
		// host — имя, а не IP: резолвим в IPv4 для bypass-маршрута до VPS.
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
	// next-hop в туннеле (10.8.0.1) — чтобы пакеты шли с нормальным TTL,
	// а не single-hop TTL=1 (см. комментарий в setupTestRoute).
	// Адрес берём из назначенного сервером (client), а не опрашиваем
	// интерфейс — иначе гонка с Windows/Wintun (адрес ещё не применён).
	tunGW := tunnelGateway(client)

	// 1. Host-route до VPS через прежний шлюз (иначе петля).
	// Если gwIdx==0 — не указываем "if", route выберёт интерфейс по gateway.
	srvStr := serverIP.String()
	srvArgs := []string{"add", srvStr, "mask", "255.255.255.255", gw.String(), "metric", "1"}
	if gwIdx > 0 {
		srvArgs = append(srvArgs, "if", strconv.Itoa(gwIdx))
	}
	if err := runCmd("route", srvArgs...); err != nil {
		return nil, fmt.Errorf("add server bypass route: %w", err)
	}

	// 2. Две половинки default в TUN.
	added := [][2]string{} // {network, mask}
	halves := [][2]string{{"0.0.0.0", "128.0.0.0"}, {"128.0.0.0", "128.0.0.0"}}
	for _, h := range halves {
		if err := runCmd("route", "add", h[0], "mask", h[1],
			tunGW.String(), "metric", "1", "if", strconv.Itoa(idx)); err != nil {
			for _, a := range added {
				_ = runCmd("route", "delete", a[0], "mask", a[1])
			}
			_ = runCmd("route", "delete", srvStr)
			return nil, fmt.Errorf("add default-half %s: %w", h[0], err)
		}
		added = append(added, h)
	}

	// 3. DNS на туннельном интерфейсе. CONNECT-IP не анонсирует DNS
	// (нет в RFC 9484), поэтому берём из профиля и ставим на masque0.
	// Трафик к DNS:53 пойдёт в туннель (default уже завёрнут).
	dnsSet := false
	for i, d := range dns {
		if i == 0 {
			if err := runCmd("netsh", "interface", "ip", "set", "dns",
				"name="+iface, "static", d, "primary"); err != nil {
				log.Printf("warn: set primary DNS %s on %s: %v", d, iface, err)
			} else {
				dnsSet = true
				log.Printf("DNS %s set on %s (tunnel)", d, iface)
			}
		} else {
			if err := runCmd("netsh", "interface", "ip", "add", "dns",
				"name="+iface, d, "index="+strconv.Itoa(i+1)); err != nil {
				log.Printf("warn: add DNS %s on %s: %v", d, iface, err)
			}
		}
	}

	return func() {
		if dnsSet {
			// Вернём DNS в авто (DHCP) на туннельном интерфейсе.
			// (Сам masque0 всё равно удаляется при закрытии TUN.)
			if err := runCmd("netsh", "interface", "ip", "set", "dns",
				"name="+iface, "dhcp"); err != nil {
				log.Printf("cleanup: reset DNS on %s: %v", iface, err)
			}
		}
		for _, a := range added {
			if err := runCmd("route", "delete", a[0], "mask", a[1]); err != nil {
				log.Printf("cleanup: del %s: %v", a[0], err)
			}
		}
		if err := runCmd("route", "delete", srvStr); err != nil {
			log.Printf("cleanup: del server route: %v", err)
		}
	}, nil
}

// runPingTest шлёт ICMP echo через уже настроенный маршрут. На Windows
// нельзя привязать ping к интерфейсу (-I), но маршрут до dst уже направлен
// в TUN с нужным src, поэтому пакеты пойдут через туннель.
func runPingTest(ctx context.Context, dst, iface string, count int) error {
	log.Printf("sending %d ICMP echo(s) to %s via tunnel...", count, dst)
	// ping -n <count> -w 5000 <dst>
	out, err := exec.CommandContext(ctx, "ping", "-n", strconv.Itoa(count), "-w", "5000", dst).CombinedOutput()
	// Windows ping выводит в OEM-кодировке; печатаем как есть.
	log.Printf("ping output:\n%s", strings.TrimSpace(string(out)))
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	// На Windows ping возвращает 0 даже при частичных потерях в некоторых версиях —
	// дополнительно проверим наличие "TTL=" (признак успешного ответа).
	if !strings.Contains(string(out), "TTL=") && !strings.Contains(string(out), "ttl=") {
		return fmt.Errorf("no reply (no TTL in output)")
	}
	log.Printf("ping through tunnel SUCCEEDED — client core data-plane WORKS")
	return nil
}

// --- вспомогательные функции Windows ---

// prefixToMask преобразует длину префикса в маску вида 255.255.255.255.
func prefixToMask(bits int) string {
	if bits < 0 || bits > 32 {
		bits = 32
	}
	var m uint32 = 0xffffffff << (32 - bits)
	if bits == 0 {
		m = 0
	}
	return fmt.Sprintf("%d.%d.%d.%d", byte(m>>24), byte(m>>16), byte(m>>8), byte(m))
}

// ifIndex возвращает числовой индекс интерфейса по имени через
// `netsh interface ipv4 show interfaces`.
func ifIndex(name string) (int, error) {
	out, err := exec.Command("netsh", "interface", "ipv4", "show", "interfaces").CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("show interfaces: %w", err)
	}
	// Строки вида: "  Idx     Met         MTU          State                Name"
	// затем: "   23        25        1400  connected            masque0"
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 5 {
			continue
		}
		// Имя может содержать пробелы — берём хвост после 4-го поля.
		// Собираем имя как всё, начиная с 5-го поля.
		idxStr := f[0]
		nm := strings.TrimSpace(strings.Join(f[4:], " "))
		if nm == name {
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				continue
			}
			return idx, nil
		}
	}
	return 0, fmt.Errorf("interface %q not found in netsh output", name)
}

// ifAddr возвращает первый IPv4-адрес интерфейса (для использования как on-link gw).
func ifAddr(name string) (netip.Addr, error) {
	out, err := exec.Command("netsh", "interface", "ipv4", "show", "addresses", "name="+name).CombinedOutput()
	if err != nil {
		return netip.Addr{}, fmt.Errorf("show addresses: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// "IP Address:                           10.8.0.254"
		if i := strings.LastIndex(line, ":"); i >= 0 && strings.Contains(strings.ToLower(line), "address") {
			cand := strings.TrimSpace(line[i+1:])
			if a, err := netip.ParseAddr(cand); err == nil && a.Is4() {
				return a, nil
			}
		}
	}
	return netip.Addr{}, fmt.Errorf("no IPv4 address on %q", name)
}

// defaultGatewayWin парсит default-шлюз и его if-индекс из `route print -4`.
func defaultGatewayWin() (netip.Addr, int, error) {
	out, err := exec.Command("route", "print", "-4").CombinedOutput()
	if err != nil {
		return netip.Addr{}, 0, fmt.Errorf("route print: %w", err)
	}
	// Ищем строку "0.0.0.0    0.0.0.0    <gateway>    <iface-ip>    <metric>"
	var bestGW netip.Addr
	bestMetric := int(^uint(0) >> 1)
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) >= 5 && f[0] == "0.0.0.0" && f[1] == "0.0.0.0" {
			gw, err := netip.ParseAddr(f[2])
			if err != nil {
				continue
			}
			metric, _ := strconv.Atoi(f[len(f)-1])
			if metric < bestMetric {
				bestMetric = metric
				bestGW = gw
			}
		}
	}
	if !bestGW.IsValid() {
		return netip.Addr{}, 0, fmt.Errorf("default gateway not found in route print")
	}
	// if-индекс шлюза определим по интерфейсу, через который он достижим:
	// используем `route print` interface list — но проще взять индекс по IP шлюза
	// через `netsh interface ipv4 show route`. Для bypass достаточно шлюза;
	// if-индекс возьмём как индекс интерфейса с тем же subnet. Упростим:
	// вернём 0 и не указываем "if" — route сам выберет по gateway.
	return bestGW, 0, nil
}
