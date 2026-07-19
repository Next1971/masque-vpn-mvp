# MASQUE — Windows CLI client (example)

Minimal MASQUE VPN command-line client for Windows. This repository contains
only public documentation and example configuration. Real binaries and
certificates are not included.

## Package layout (local setup)

When testing locally, place files in a directory such as `C:\masque\`:

- `vpn-client.exe`        — MASQUE VPN client (amd64), **not part of this repo**.
- `wintun.dll`            — Wintun TUN driver (from wintun.net), **not part of this repo**.
- `profile.example.toml`  — example MASQUE client profile (this repo).
- `certs\`                — your own certificates and keys (not tracked in git).

The client expects `vpn-client.exe`, `wintun.dll`, `profile.example.toml` and
the `certs\` directory to be in the same folder.

## Requirements

- Windows 10/11 x64.
- Run PowerShell or CMD as Administrator (to create the TUN adapter and edit routes).
- `wintun.dll` must be located next to `vpn-client.exe`.

## Example usage (test mode)

Test mode tunnels only a single destination IP (for example `1.1.1.1`) through
the MASQUE tunnel while leaving the rest of your traffic untouched.

```powershell
cd C:\masque
vpn-client.exe -profile profile.example.toml -test -test-dst 1.1.1.1 -ping 3 -timeout 35s
```

On success, the client reports that ping through the tunnel succeeded. On
failure, capture the full console output for diagnostics.

## Example usage (full route)

Full-route mode sends all IPv4 traffic through the MASQUE tunnel. Use this only
after test mode works as expected.

```powershell
cd C:\masque
vpn-client.exe -profile profile.example.toml -full-route -timeout 60s
```

To verify, open a site like `https://ifconfig.me` in a browser and check that
your external IP matches the MASQUE server you have configured.

## Client flags

The CLI client supports flags similar to:

```text
-profile <path>    path to the MASQUE profile (required)
-test              test mode: tunnel only -test-dst
-test-dst <ip>     test destination IP (default 1.1.1.1)
-ping <n>          how many ICMP echo requests to send in test mode (0 = no ping)
-full-route        full VPN: route all traffic into the tunnel
-timeout <dur>     overall session timeout (e.g. 35s, 60s, 5m)
-verbose           verbose diagnostics (packet-level trace) for debugging only
```

## Security

This repository does **not** include any real certificates or private keys.

- `certs\ca.crt`, `certs\client.crt`, `certs\client.key` must be created and
  managed privately.
- Never commit real `client.key` or other secrets to a public repository.
- Example profiles use placeholder values only (see `profile.example.toml`).
