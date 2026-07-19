// Package clientcore — общее клиентское ядро MASQUE для всех платформ
// (Linux/Windows/Android). Ядро НЕ создаёт TUN само и НЕ трогает маршруты —
// эти платформенные детали инжектируются снаружи тонкими обёртками:
//   - Linux:   cmd/vpn-client (CreateTUN по имени + ip route)
//   - Windows: обёртка на wintun + netsh (следующий этап)
//   - Android: TUN fd от VpnService + CreateTUNFromFile (следующий этап)
//
// Так один и тот же код подключения/форвардинга/закрытия переиспользуется
// на всех платформах — это и есть «единое ядро» из PROJECT.md.
package clientcore

import (
	"fmt"
	"net/netip"

	"github.com/BurntSushi/toml"
)

// Profile — серверный профиль клиента. Один и тот же набор параметров
// для Android и Windows (требование PROJECT.md). Читается из TOML-файла,
// который на устройстве редактируется через UI.
//
// Секреты (приватный ключ клиента) в профиле хранятся как ПУТЬ к файлу,
// а не инлайн — чтобы профиль можно было показывать/логировать без утечки.
// (На Android/Windows UI позже может хранить ключ в защищённом хранилище.)
type Profile struct {
	// [server]
	Server     string `toml:"server"`      // host:port MASQUE-прокси (UDP), напр. "80.85.241.127:4433"
	ServerName string `toml:"server_name"` // TLS SNI / URI-template host, напр. "masque.zavodovskii.com"

	// [tls] — mTLS-материал (пути к PEM-файлам, НЕ инлайн-секреты)
	CA   string `toml:"ca"`   // CA для проверки серверного сертификата
	Cert string `toml:"cert"` // клиентский сертификат (mTLS)
	Key  string `toml:"key"`  // приватный ключ клиента (mTLS)

	// [tun]
	TUNName string   `toml:"tun_name"` // имя интерфейса (Linux/Windows), напр. "masque0"
	MTU     int      `toml:"mtu"`      // MTU туннеля, напр. 1400
	DNS     []string `toml:"dns"`      // DNS-серверы для туннеля (full-route), по умолчанию ["1.1.1.1"]
}

// tomlProfile — промежуточная структура для секций TOML.
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

// LoadProfile читает и валидирует TOML-профиль клиента.
// Неизвестные ключи считаются ошибкой (защита от опечаток в профиле).
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

// Validate проверяет обязательные поля профиля.
func (p *Profile) Validate() error {
	if p.Server == "" {
		return fmt.Errorf("profile: [server].server is required (host:port)")
	}
	if _, err := netip.ParseAddrPort(p.Server); err != nil {
		// Допускаем hostname:port — ParseAddrPort требует IP, поэтому
		// строгую проверку оставляем на этап Dial (ResolveUDPAddr).
		if !hasPort(p.Server) {
			return fmt.Errorf("profile: [server].server %q must be host:port", p.Server)
		}
	}
	if p.ServerName == "" {
		return fmt.Errorf("profile: [server].server_name is required (TLS SNI)")
	}
	if p.MTU == 0 {
		p.MTU = 1400 // разумное значение по умолчанию для QUIC/MASQUE
	}
	if p.MTU < 576 || p.MTU > 9000 {
		return fmt.Errorf("profile: [tun].mtu %d out of range (576..9000)", p.MTU)
	}
	if p.TUNName == "" {
		p.TUNName = "masque0"
	}
	if len(p.DNS) == 0 {
		p.DNS = []string{"1.1.1.1"} // разумный дефолт для туннеля
	}
	// Валидация DNS-адресов.
	for _, d := range p.DNS {
		if _, err := netip.ParseAddr(d); err != nil {
			return fmt.Errorf("profile: [tun].dns %q is not a valid IP: %w", d, err)
		}
	}
	return nil
}

// hasPort грубо проверяет наличие ":port" в конце строки.
func hasPort(s string) bool {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i < len(s)-1
		}
	}
	return false
}
