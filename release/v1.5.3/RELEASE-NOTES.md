v1.5.3 is GitHub **Latest**. It is the v1.5.2 client line plus an Android TV **Connect crash** fix. Optional **kill switch** (default off) and optional IPv6 inside the tunnel are included. The CONNECT-IP protocol is unchanged; existing profiles keep working. Linux server binaries are the same protocol as v1.5.0 / v1.5.1. `masque-setup.exe` is not in this release; use [v1.5.1](https://github.com/Next1971/masque-vpn/releases/tag/v1.5.1) if you need the VPS helper.

## What this tag is

- Android TV: Connect no longer crashes (leanback notification intent; skip battery-exemption on TV; IPv6 TUN skipped if the device rejects it).
- Optional kill switch, **off** until the user enables it (from v1.5.2).
- Android `1.5.3` (`versionCode` 20); Windows product **1.5.3**.
- Same CONNECT-IP server as v1.5.0 / v1.5.1.

## Who should update

- **Android TV** users on v1.5.0–v1.5.2 (Connect crash).
- Anyone who wants Latest with kill switch + IPv6 clients in one download.

## iOS

No iOS build is attached. TestFlight is **v1.7** (not later than **12 October 2026**).

## Notes

- Experimental, self-hosted software. No third-party security audit.
- Android APKs are **signed** when built with the project keystore.
- Linux server binaries contain **no certificates**.
- Next technical slices: **v1.5.4** dual-port + DoH + library bump; **v1.5.5** install.sh / attestations; **v1.5.6** QUIC over IPv6.

## Files

- `masque-1.5.3.msi` — Windows VPN client
- `masque-phone-1.5.3.apk` — Android phone
- `masque-tv-1.5.3.apk` — Android TV
- `vpn-server-linux-amd64` / `vpn-server-linux-arm64` — server
