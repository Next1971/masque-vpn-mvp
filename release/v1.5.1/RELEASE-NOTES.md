v1.5.1 is a **technical pre-release** after v1.5.0. It closes small operational gaps. The CONNECT-IP protocol is unchanged; existing profiles keep working. **v1.5.0 remains GitHub Latest.**

## What this tag is

- Server: refuse listed client certificate CNs (`/opt/masque/blocked_cns`, optional `tls.blocked_cns`). Reload on process start.
- Windows `masque-setup.exe`: **Revoke certificate** (append CN + restart `masque.service`).
- Clients: default TUN **MTU 1369** from path tests (MTS mobile data DF-safe at 1370, loss at 1380; one-byte margin). Details: repository `docs/benchmarks/mtu.md`.
- Android `1.5.1` (`versionCode` 18); Windows product **1.5.1**.

## UDP 443 unusable on the client path

Some networks drop outbound **UDP 443**. Keep the server bound to 443. On the VPS:

```bash
ufw allow 2053/udp
iptables -t nat -C PREROUTING -p udp --dport 2053 -j REDIRECT --to-ports 443 2>/dev/null \
  || iptables -t nat -A PREROUTING -p udp --dport 2053 -j REDIRECT --to-ports 443
netfilter-persistent save   # after installing iptables-persistent
```

Also allow UDP 2053 in the cloud security group. Set the client profile host to `:2053`. TLS server name is unchanged. Android still dials **one** port in this release.

## iOS

First successful on-device result. **No iOS build is attached here.** TestFlight access is planned for **v1.6** (not later than **12 October 2026**).

## v1.6 (committed)

- TestFlight access for the iOS client.
- Android dual-port: UDP 443 and the alternate port at the same time.

## Who should update

- Operators who need to disable a leaked `profile.masque` without rotating the CA.
- Clients on paths where inner DF probes failed above ~1370 (use the new APK/MSI).
- Anyone hitting UDP 443 blackholes (server redirect + profile port; no client upgrade strictly required for the redirect).

## Notes

- Experimental, self-hosted software. No third-party security audit.
- Android APKs are **signed**. Linux server binaries contain **no certificates**.
- Replace the **server binary** for the denylist. Client updates are required for MTU 1369 on Android (hardcoded). Windows uses 1369 for new/default profiles and the adapter MTU in this MSI.

## Files

- `masque-1.5.1.msi` — Windows VPN client
- `masque-setup.exe` — experimental VPS installer (revoke + issue); place `vpn-server-linux-*` next to it
- `masque-phone-1.5.1.apk` — Android phone
- `masque-tv-1.5.1.apk` — Android TV
- `vpn-server-linux-amd64` / `vpn-server-linux-arm64` — server
