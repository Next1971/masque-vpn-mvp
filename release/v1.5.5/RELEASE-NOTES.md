v1.5.5 is a GitHub **pre-release**. Server install packaging only: public `install.sh`, `SHA256SUMS`, and Sigstore (GitHub artifact) attestations for the Linux server files. Same CONNECT-IP protocol as v1.5.4; existing profiles keep working. **v1.5.3** stays **Latest**.

## What this tag is

- `install.sh` on the VPS: downloads `vpn-server-linux-*` + `gen-config.sh` from this release, checks `SHA256SUMS`, then lays down `/opt/masque` (same layout as `masque-setup.exe`).
- `SHA256SUMS` for release files. Linux server binaries, `install.sh`, and `gen-config.sh` also have GitHub **attestations** (Sigstore). Verify with `gh attestation verify <file> --repo Next1971/masque-vpn`.
- Clients are unchanged: use the **v1.5.4** MSI/APKs (`connect-ip-go` v0.3.0, optional `alt_port`).
- New `vpn-server-linux-amd64` / `vpn-server-linux-arm64` (same protocol as v1.5.4).

## Who should update

- Operators who want a curl-able Linux install and checksums / provenance on the server binary.
- Stay on v1.5.3 for a daily driver. Stay on v1.5.4 if you already soaked dual-port and do not need `install.sh`.

## Compatibility

- Same as v1.5.4. Old clients reach a v1.5.5 server. Prefer pairing 1.5.4+ clients with this server if you use `alt_port`.

## iOS

No iOS build is attached. TestFlight stays **v1.7**.

## Notes

- Experimental, self-hosted software. No third-party security audit.
- DoH/DoT is not in this tag.
- Next on main (no extra GitHub pre-release): Android AGP **9.4.0** + Gradle **9.6.0** (deferred Dependabot #75). **v1.5.6** is QUIC to the server over IPv6; that changelog will mention the AGP pair.
- `masque-setup.exe` still uploads a local Linux binary. Add `alt_port` by hand or via `gen-config.sh --alt-port`.

## Files

- `vpn-server-linux-amd64` / `vpn-server-linux-arm64` — server
- `install.sh` / `gen-config.sh` — VPS install
- `SHA256SUMS`
- Windows/Android clients: [v1.5.4](https://github.com/Next1971/masque-vpn/releases/tag/v1.5.4)

## Linux install

```bash
# as root, Ubuntu/Debian
export MASQUE_HOST=vpn.example.com
export MASQUE_PORT=443
curl -fsSL -o install.sh https://github.com/Next1971/masque-vpn/releases/download/v1.5.5/install.sh
curl -fsSL -o SHA256SUMS https://github.com/Next1971/masque-vpn/releases/download/v1.5.5/SHA256SUMS
sha256sum -c SHA256SUMS --ignore-missing
bash install.sh
```
