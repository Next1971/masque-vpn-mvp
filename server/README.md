# Server

Sanitized server-side part of the MASQUE VPN MVP.

## Notes

This directory will contain the public server implementation prepared for publication.
All private deployment values, infrastructure-specific endpoints, domains, VPS addresses, certificates, keys and secrets are removed before upload.

## Planned contents

- Basic deployment structure
- Configuration templates
- Setup notes
- Example values with placeholders only

## Source layout

The `server` directory contains both deployment examples and Go source code
for the MASQUE MVP:

- `config.server.example.toml` — example server configuration (TOML).
- `install.example.sh` — example systemd install/update script.
- `systemd/masque.service.example` — example systemd unit for the server.
- `DEPLOYMENT.example.md` — high-level deployment outline.

Go source code lives under `server/src`:

- `src/cmd/poc-server/` — PoC/MVP MASQUE CONNECT-IP server:
  - `main.go` — entrypoint, flag/config handling, QUIC + HTTP/3 + CONNECT-IP.
  - `config.go` — TOML config loader and validation.
  - `iface_linux.go` — Linux-specific TUN setup using `ip`.
  - `ippool.go` — thread-safe IPv4 address pool used by the server.

- `src/internal/clientcore/` — shared MASQUE client core used by
  multiple platforms (Linux/Windows/Android):
  - `profile.go` — client profile format and TOML loader.
  - `client.go` — QUIC + HTTP/3 + CONNECT-IP client session and
    conn↔TUN forwarding.
  - `iphdr.go` — IP header helpers (TTL/Hop Limit normalization,
    logging helpers).

The client core does not create TUN devices or modify routes directly.
Platform-specific wrappers are expected to provide a TUN interface and
apply routes based on the information from `clientcore.Session`.
