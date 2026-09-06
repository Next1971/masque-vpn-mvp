# Roadmap

> Status snapshot: last updated 2026-09-06. See [CHANGELOG.md](../CHANGELOG.md) for release history.

## Current status

MASQUE VPN has been operational and tested end-to-end since **July 15, 2026** across all three components (server, Windows client, Android client). **v1.0** was publicly released on **August 14, 2026**. **v1.3** is the Android reconnect release. **v1.4** added client polish (icon + on-screen ping). **v1.4.1** (pre-release) is a maintenance line: AGP 9 / Gradle 9, Docker packaging, graceful server shutdown, and Android IPv6-bypass hardening. **v1.5.0** adds optional dual-stack inside the tunnel (ULA + NAT66). **v1.5.1** (pre-release) is a technical follow-up: CN denylist / Windows revoke, TUN MTU **1369**, and a documented UDP 443 workaround.

**v1.6** is scheduled **not later than 12 October 2026**.

| Component | Status | Notes |
|---|---|---|
| Server | Stable | QUIC keepalive, sticky `/32` and optional sticky `/128`, mTLS, CN denylist (`blocked_cns`), systemd, optional Docker, graceful SIGTERM |
| Windows client | Stable | Signed EXE + Wintun DLL in release, GUI + tray icon, on-screen ping, IPv6 default via TUN when the server assigns v6, MTU 1369 |
| Windows VPS installer | **Experimental (v1.5.1)** | `masque-setup.exe` can issue bundles and **revoke** a CN; not a substitute for the documented SSH install |
| Android client | Stable | Phone + TV; dual-stack TUN when the server has a v6 pool; TUN MTU 1369; version label in UI; reconnect without tearing TUN |
| iOS client | **In progress (v1.6)** | First on-device success; not in a GitHub Release; TestFlight access planned for v1.6 |

This is experimental software and has not received an independent security audit.

## Completed (v1.0)

- QUIC + HTTP/3 transport via `quic-go`.
- CONNECT-IP (RFC 9484) tunneling via `connect-ip-go`.
- Mutual TLS with an internal EC (P-256) CA.
- Server config/certificate generator (`gen-config.sh`) producing ready-to-use client bundles.
- Windows client: signed EXE + Wintun DLL, console mode + local web UI.
- Android client: signed APK (all ABIs), Kotlin app + shared Go core (`clientcore`) via gomobile.
- systemd service unit with IP forwarding and NAT.
- CONTRIBUTING.md, SECURITY.md, issue templates.

## Completed (v1.2)

- Server multi-client return-path demux by destination IP.
- Android two-phase connect using the server-assigned `/32`.
- Android TV product flavor (leanback, paste-text import).

## Completed (v1.3)

- QUIC keepalive and idle timeout on client and server.
- Android battery-optimization exemption prompt.
- Reconnect in the gomobile bridge without closing the TUN.
- `NetworkCallback` + `setUnderlyingNetworks` + UDP `protect` for Wi-Fi → cellular.
- Sticky IP pool keyed by client certificate CN.
- `poc-client` requires a CA (no default `InsecureSkipVerify`).

## Completed (v1.4)

- App icon on Android and Windows (launcher, tray, MSI, notification, TV banner).
- On-screen ping: smoothed QUIC RTT to the MASQUE server.
- Android airplane-mode soak: after 8 hours offline the tunnel came back cleanly (reconnect + sticky `/32`).

## Completed (v1.4.1)

- Android toolchain: AGP 9.x + Gradle 9.x + built-in Kotlin (see `android/README.md` for current versions).
- Docker image / Compose for the server (host network, TUN, NAT).
- Server graceful shutdown on SIGTERM/SIGINT.
- Android IPv6 sink + TUN `/24` (block dual-stack bypass on IPv4-only servers).
- Version label in Android UI; high-contrast launcher icon.

## Completed (v1.5.0)

- [x] Full IPv6 in the tunnel (optional pool, NAT66/forwarding, client assigned v6).
- Android forwards assigned IPv6 instead of sinking it when the server has a v6 pool.
- Windows GUI/console and Linux console install IPv6 on TUN and `::/0`.

## Completed (v1.5.1)

- [x] CN denylist on the server (`/opt/masque/blocked_cns`) and revoke in `masque-setup.exe`.
- [x] Default TUN MTU **1369** from path MTU tests (`docs/benchmarks/mtu.md`).
- [x] Documented VPS port redirect when UDP 443 is unusable on the client path.

## Known limitations (all platforms)

- Connecting to the VPN server is still **IPv4 QUIC** (no AAAA / UDP 443 on IPv6 for the control plane in this release).
- Some networks drop outbound **UDP 443**. Use an alternate UDP port with VPS DNAT (see [server README](../server/README.md#udp-443-blocked-on-the-client-path)). Android will not dial two ports at once until v1.6.
- In-tunnel DNS is plaintext UDP:53 — hidden from the local ISP but visible to the server operator. DoH/DoT is planned.
- Single server/profile per client — no profile list or automatic failover.
- No independent security audit yet.
- Some OEM battery savers ignore the exemption dialog; a killed process still needs a manual Connect.
- NAT64/DNS64 is not included: AAAA destinations need WAN IPv6 on the VPS.
- CN denylist is not a CRL/OCSP PKI: it is a server-side name list reloaded on process start.

## Planned for v1.6 (not later than 12 October 2026)

- [ ] iOS TestFlight access (source already in `ios/`; first on-device connect succeeded; no GitHub IPA in 1.5.1).
- [ ] Android dual-port: attempt UDP 443 and the alternate listen port at the same time.
- [ ] DNS over HTTPS/TLS (DoH/DoT) inside the tunnel.
- [ ] Dependency review process for quic-go / connect-ip-go version pinning (see notes below).
- [ ] QUIC to the server over IPv6 (AAAA + host-route bypass).

## Dependency notes

- `quic-go` and `connect-ip-go` versions are pinned deliberately; `connect-ip-go` releases historically lag behind `quic-go` API changes (e.g. `http3.ParseCapsule`). Upgrades are tested end-to-end on all three platforms before bumping either dependency.

## Planned for future releases (no committed dates)

- [ ] Expanded troubleshooting and platform-specific FAQ.

## Explicitly out of scope for now

- GUI-based certificate management beyond the experimental Windows VPS installer (certs are generated/distributed out-of-band by design).
- Multi-hop / chained proxy support.
