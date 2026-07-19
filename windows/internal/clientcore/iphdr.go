package clientcore

import (
	"fmt"
	"net/netip"
)

// Обработка IP-заголовка исходящих пакетов перед проксированием.
//
// Проблема: некоторые ОС (в частности Windows при определённой маршрутизации
// в TUN) формируют пакеты с TTL=1 (IPv4) / Hop Limit=1 (IPv6). Библиотека
// connect-ip-go по RFC 9484 при проксировании IP декрементирует TTL и, если
// он становится 0, ОБЯЗАНА отбросить пакет ("datagram Hop Limit too small: 1").
// Из-за этого весь трафик клиента дропается ещё до отправки на сервер.
//
// Решение: перед отправкой поднимаем слишком маленький TTL до безопасного
// значения (minTTL→64) и пересчитываем контрольную сумму IPv4-заголовка.
// Это делаем в клиентском ядре, поэтому фикс общий для всех платформ
// (Linux/Windows/Android). На корректных пакетах (TTL уже нормальный) функция
// ничего не меняет.

const (
	// minTTL — если TTL/Hop Limit пакета меньше этого, поднимаем до fixTTL.
	// Порог 2, т.к. connect-ip декрементирует и требует результат ≥ 1.
	minTTL = 2
	// fixTTL — значение, до которого поднимаем слишком маленький TTL.
	fixTTL = 64
)

// normalizeTTL проверяет IP-версию пакета и, если TTL/Hop Limit < minTTL,
// поднимает его до fixTTL. Для IPv4 пересчитывает контрольную сумму заголовка.
// Возвращает исходный TTL (для диагностики) и признак того, что пакет правился.
// pkt — полный IP-пакет (начиная с версии/IHL).
func normalizeTTL(pkt []byte) (origTTL int, fixed bool) {
	if len(pkt) < 1 {
		return -1, false
	}
	version := pkt[0] >> 4
	switch version {
	case 4:
		// IPv4: минимальный заголовок 20 байт. TTL — байт 8. Чек-сумма — байты 10-11.
		if len(pkt) < 20 {
			return -1, false
		}
		origTTL = int(pkt[8])
		if origTTL >= minTTL {
			return origTTL, false
		}
		pkt[8] = fixTTL
		// Пересчёт контрольной суммы заголовка (по IHL).
		ihl := int(pkt[0]&0x0f) * 4
		if ihl < 20 || ihl > len(pkt) {
			ihl = 20
		}
		pkt[10] = 0
		pkt[11] = 0
		csum := ipv4Checksum(pkt[:ihl])
		pkt[10] = byte(csum >> 8)
		pkt[11] = byte(csum & 0xff)
		return origTTL, true
	case 6:
		// IPv6: фиксированный заголовок 40 байт. Hop Limit — байт 7.
		// Контрольной суммы в IPv6-заголовке нет.
		if len(pkt) < 40 {
			return -1, false
		}
		origTTL = int(pkt[7])
		if origTTL >= minTTL {
			return origTTL, false
		}
		pkt[7] = fixTTL
		return origTTL, true
	default:
		return -1, false
	}
}

// ipv4Checksum считает контрольную сумму IPv4-заголовка (RFC 791):
// one's complement суммы 16-битных слов.
func ipv4Checksum(hdr []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(hdr); i += 2 {
		sum += uint32(hdr[i])<<8 | uint32(hdr[i+1])
	}
	if len(hdr)%2 == 1 {
		sum += uint32(hdr[len(hdr)-1]) << 8
	}
	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}

// describePkt возвращает краткое человекочитаемое описание IP-пакета для логов:
// версия, src→dst, протокол и TTL/Hop Limit. Используется для диагностики
// входящего пути conn→TUN.
func describePkt(pkt []byte) string {
	if len(pkt) < 1 {
		return "empty"
	}
	switch pkt[0] >> 4 {
	case 4:
		if len(pkt) < 20 {
			return "short-ipv4"
		}
		src := netip.AddrFrom4([4]byte{pkt[12], pkt[13], pkt[14], pkt[15]})
		dst := netip.AddrFrom4([4]byte{pkt[16], pkt[17], pkt[18], pkt[19]})
		return fmt.Sprintf("IPv4 %s→%s proto=%d ttl=%d", src, dst, pkt[9], pkt[8])
	case 6:
		if len(pkt) < 40 {
			return "short-ipv6"
		}
		var s, d [16]byte
		copy(s[:], pkt[8:24])
		copy(d[:], pkt[24:40])
		return fmt.Sprintf("IPv6 %s→%s next=%d hlim=%d",
			netip.AddrFrom16(s), netip.AddrFrom16(d), pkt[6], pkt[7])
	default:
		return fmt.Sprintf("unknown-version %d", pkt[0]>>4)
	}
}
