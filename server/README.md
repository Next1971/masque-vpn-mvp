# Server

Sanitized server-side part of the MASQUE VPN MVP.

## Current status

The public server materials in this directory correspond to a working MVP server that has remained stable after 15 days of testing. All private deployment values, infrastructure-specific endpoints, domains, VPS addresses, certificates, keys, and secrets are intentionally removed before publication.

## Start here

- `DEPLOYMENT.md` — practical deployment notes for the public MVP server.
- `DEPLOYMENT.example.md` — sanitized deployment outline with placeholder values.
- `config.server.example.toml` — example server configuration.
- `install.example.sh` — example install/update script.
- `systemd/masque.service.example` — example systemd unit.

## Source layout

The `server` directory contains both deployment examples and Go source code for the MASQUE MVP.

- `config.server.example.toml` — example server configuration (TOML).
- `install.example.sh` — example systemd install/update script.
- `systemd/masque.service.example` — example systemd unit for the server.
- `DEPLOYMENT.example.md` — high-level deployment outline.
- `DEPLOYMENT.md` — practical deployment notes.

Go source code lives under `server/src`:

- `src/cmd/poc-server/` — PoC/MVP MASQUE CONNECT-IP server:
  - `main.go` — entrypoint, flag/config handling, QUIC + HTTP/3 + CONNECT-IP.
  - `config.go` — TOML config loader and validation.
  - `iface_linux.go` — Linux-specific TUN setup using `ip`.
  - `ippool.go` — thread-safe IPv4 address pool used by the server.
- `src/internal/clientcore/` — shared MASQUE client core used by multiple platforms:
  - `profile.go` — client profile format and TOML loader.
  - `client.go` — QUIC + HTTP/3 + CONNECT-IP client session and tunnel forwarding.
  - `iphdr.go` — IP header helpers and logging helpers.

The shared client core does not create TUN devices or modify routes directly. Platform-specific wrappers are expected to provide a TUN interface and apply routes based on information from `clientcore.Session`.
