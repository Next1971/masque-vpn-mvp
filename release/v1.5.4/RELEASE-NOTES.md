v1.5.4 is a GitHub **pre-release**. It bumps `connect-ip-go` to **v0.3.0** and races two QUIC ports so a path that drops UDP 443 can still connect. The CONNECT-IP protocol is unchanged; existing profiles keep working. **v1.5.3** stays **Latest** until this soaks. This is the first new Linux server binary since v1.5.1.

## What this tag is

- `connect-ip-go` **v0.2.0 → v0.3.0** (server + Android/Windows/Linux clients). `quic-go` stays **v0.62.0**.
- Dual-port: optional `[server].alt_port` (any second UDP port on the same host). Clients race it with the profile port. Omit the field for a single port. `gen-config.sh --alt-port N` writes it.
- Android `1.5.4` (`versionCode` 21); Windows product **1.5.4**.
- New `vpn-server-linux-amd64` / `vpn-server-linux-arm64`.

## Who should update

- Anyone whose path drops one of two UDP ports (put the other in `alt_port`, and open it on the VPS).
- Operators who will soak the new `connect-ip-go` on a machine they can roll back.

Stay on v1.5.3 if you only need a daily driver.

## Compatibility

- Old **v1.5.3** clients should still reach a **v1.5.4** server (same RFC 9484 capsules).
- New clients should still reach a **v1.5.1** server. Prefer pairing new client + new server for soak.
- Dual-port without `alt_port` behaves like v1.5.3 (one port). A dead `alt_port` fails; the profile port can still win.

## iOS

No iOS build is attached. The shared Go core is updated; TestFlight stays **v1.7**.

## Notes

- Experimental, self-hosted software. No third-party security audit.
- DoH/DoT is not in this tag (own VPS; in-tunnel UDP:53 is visible to the operator by design).
- Next technical slices: **v1.5.5** install.sh / attestations; **v1.5.6** QUIC over IPv6.

## Files

- `masque-1.5.4.msi` — Windows VPN client
- `masque-phone-1.5.4.apk` — Android phone
- `masque-tv-1.5.4.apk` — Android TV
- `vpn-server-linux-amd64` / `vpn-server-linux-arm64` — server
