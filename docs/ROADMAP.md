# Roadmap

> Status snapshot: last updated 2026-09-09. See [CHANGELOG.md](../CHANGELOG.md) for release history.

## Current status

MASQUE VPN has been operational and tested end-to-end since **July 15, 2026** (server, Windows, Android). **v1.0** shipped **14 August 2026**. **v1.5.0** is GitHub **Latest** (optional IPv6 inside the tunnel). **v1.5.1** is a technical pre-release (CN denylist, TUN MTU **1369**, UDP 443 redirect). **v1.5.2** is a technical pre-release: optional **kill switch** (default off).

**v1.6** (iOS TestFlight) is scheduled **not later than 12 October 2026**. Client/installer **design work can happen in any branch** and is not gated on a version number. **Store listings** (Play / F-Droid) start **after** that visual snapshot, not in 1.5.x.

| Component | Status | Notes |
|---|---|---|
| Server | Stable | Same protocol as v1.5.0/1.5.1; no server binary change in v1.5.2 |
| Windows client | Stable | v1.5.2 MSI: optional kill switch |
| Windows VPS installer | **Experimental (v1.5.1)** | `masque-setup.exe`; not required for v1.5.2 |
| Android client | Stable | v1.5.2 APKs: optional kill switch (phone + TV) |
| iOS client | **In progress (v1.6)** | Source in `ios/`; TestFlight planned for v1.6 |

This is experimental software and has not received an independent security audit.

## Release slices (near term)

| Tag | Focus | Notes |
|---|---|---|
| **v1.5.2** | Kill switch | Client-only. Default **off**. |
| **v1.5.3** | Dual-port, DoH/DoT, library bump | Bump `quic-go` / `connect-ip-go` **first** and soak, then dual-port and DoH. Do not land all three in one night. |
| **v1.5.4** | Server install packaging | `install.sh`, SHA256SUMS, Sigstore attestations. Little/no protocol change. |
| **v1.5.5** | QUIC to the server over IPv6 | AAAA + host-route / exclude so the UDP socket does not loop into TUN. Separate from dual-port. |
| **v1.6** | iOS TestFlight | UI refresh ships in whichever build is ready; stores follow the designed UI. |

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

## Completed (v1.5.2)

- [x] Optional kill switch on Android (`setBlocking` + held TUN) and Windows (routes before dial, held until Disconnect). Default off.

## Known limitations (all platforms)

- Connecting to the VPN server is still **IPv4 QUIC** (no AAAA / UDP 443 on IPv6 until **v1.5.5**).
- Some networks drop outbound **UDP 443**. Use an alternate UDP port with VPS DNAT (see [server README](../server/README.md#udp-443-blocked-on-the-client-path)). Simultaneous dual-port is **v1.5.3**.
- In-tunnel DNS is plaintext UDP:53 — hidden from the local ISP but visible to the server operator. DoH/DoT is **v1.5.3**.
- Kill switch (v1.5.2) does not survive a killed VPN process. On Android, system Always-on VPN / “Block connections without VPN” is the extra layer.
- Single server/profile per client — no profile list or automatic failover.
- No independent security audit yet.
- Some OEM battery savers ignore the exemption dialog; a killed process still needs a manual Connect.
- NAT64/DNS64 is not included: AAAA destinations need WAN IPv6 on the VPS.
- CN denylist is not a CRL/OCSP PKI: it is a server-side name list reloaded on process start.

## Planned for v1.5.3

- [ ] Bump and soak `quic-go` / `connect-ip-go` (first commit of the line; e2e on Android, Windows, Linux).
- [ ] Android dual-port: attempt UDP 443 and the alternate listen port at the same time.
- [ ] DNS over HTTPS/TLS (DoH/DoT) inside the tunnel.

## Planned for v1.5.4

- [ ] Server `install.sh`, SHA256SUMS, Sigstore attestations on release artifacts.

## Planned for v1.5.5

- [ ] QUIC to the server over IPv6 (AAAA + host-route bypass). Do not mix with dual-port in the same drop.

## Planned for v1.6 (not later than 12 October 2026)

- [ ] iOS TestFlight access (source already in `ios/`; first on-device connect succeeded; no GitHub IPA in 1.5.x).
- Visual refresh of clients and `masque-setup.exe` can merge whenever it is ready. **App stores after that UI**, not before.

## Dependency notes

- `quic-go` and `connect-ip-go` versions are pinned deliberately; `connect-ip-go` releases historically lag behind `quic-go` API changes (e.g. `http3.ParseCapsule`). Upgrades are tested end-to-end on all three platforms before bumping either dependency.

## Planned for future releases (no committed dates)

- [ ] Expanded troubleshooting and platform-specific FAQ.

## Explicitly out of scope for now

- GUI-based certificate management beyond the experimental Windows VPS installer (certs are generated/distributed out-of-band by design).
- Multi-hop / chained proxy support.
