# MASQUE VPN — Windows Client

Windows client for the MASQUE VPN MVP, built on top of a portable Go core (`clientcore`) shared with the Android and Linux variants. It speaks **QUIC + HTTP/3 CONNECT-IP (MASQUE, RFC 9484)** with mutual TLS to a MASQUE server and tunnels traffic through a Wintun adapter.

This directory contains the public Windows client sources, build scripts, Web UI assets, and example configuration materials for the MVP.

## Status (July 2026)

- Stable operation for 8 consecutive days.
- Web UI mode is available for daily use through a local browser interface.
- CLI mode remains supported for advanced usage and scripting.
- Recent fixes improved QUIC connection stability, TLS handling with self-signed certificates, and Wintun loading when launched as Administrator.

## Repository layout

```text
windows/
├─ cmd/vpn-client/            # platform wrapper (TUN + routing + DNS + Web UI entrypoint)
│  ├─ main.go                 # flags, profile loading, run loop
│  ├─ iface_windows.go        # Windows: Wintun, routes, DNS (build tag)
│  ├─ iface_linux.go          # Linux equivalent (build tag)
│  └─ index.html              # local Web UI served by the client
├─ internal/clientcore/       # shared, platform-independent core
│  ├─ client.go               # QUIC dial, mTLS, CONNECT-IP session, forwarding
│  ├─ iphdr.go                # IPv4/IPv6 header helpers (TTL normalization)
│  └─ profile.go              # TOML profile parsing + validation
├─ scripts/
│  ├─ build.bat               # build via cmd.exe
│  └─ build.ps1               # build via PowerShell
├─ profile.client.toml.example
├─ go.mod / go.sum
└─ README.md
```

## Prerequisites

- **Go 1.21+** for Windows
- Windows 10/11 x64
- Administrator rights to run the VPN client

No CGO or external C toolchain is required.

## Build

### Option A — script

From `cmd.exe`:

```bat
scripts\build.bat
```

Or from PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\build.ps1
```

### Option B — manual

```bat
go mod download
set GOOS=windows
set GOARCH=amd64
go build -trimpath -ldflags "-s -w" -o dist\vpn-client.exe .\cmd\vpn-client
```

Expected output: `dist\vpn-client.exe`.

## Configure

1. Copy the example profile and edit it:

   ```bat
   copy profile.client.toml.example dist\profile.client.toml
   ```

2. Set `server`, `server_name`, and DNS values appropriate for your deployment.
3. Place your mTLS materials next to the runtime profile as referenced by the profile paths.
4. Keep all real certificates, keys, and production profiles out of the public repository.

## Web UI mode

Running `vpn-client.exe` without the `-profile` flag starts the local Web UI.

### How to use

1. Extract or build the client into a working folder.
2. Right-click `vpn-client.exe` and run it as Administrator.
3. Open your browser and go to `http://localhost:8080`.
4. Upload your `profile.client.toml` file.
5. Use the connect button to start the tunnel and the disconnect button to stop it.

The Web UI is intended to remove the need for daily CLI usage in normal operation.

## CLI mode

Running the client with the `-profile` flag keeps the traditional console workflow.

```bat
cd dist
vpn-client.exe -profile profile.client.toml -full-route -timeout 60s
```

### Command-line flags

| Flag | Default | Meaning |
|---|---|---|
| `-profile` | optional in Web UI mode | Path to the client profile TOML |
| `-full-route` | false | Route all traffic via the tunnel |
| `-timeout` | 25s | Overall timeout; use `0` to run until Ctrl+C |
| `-verbose` | false | Verbose diagnostics |
| `-test` | true | Test mode: route only `-test-dst` via TUN |
| `-test-dst` | 1.1.1.1 | Test-mode destination |
| `-ping` | 3 | Test-mode ICMP echo count |

For a normal full VPN session in CLI mode, use `-profile`, `-full-route`, and `-timeout 0`.

## How it works

- `internal/clientcore` is platform-independent: it dials QUIC, performs mTLS, opens a CONNECT-IP session, and forwards packets between the tunnel and the client runtime.
- `cmd/vpn-client/iface_windows.go` handles Wintun setup, routing, and DNS on Windows.
- `cmd/vpn-client/index.html` provides the local browser-based control panel.
- The same `main.go` can also build for Linux through build tags.

## Troubleshooting

- **`wintun.dll` load errors** — ensure the runtime package includes the required Wintun library next to the EXE.
- **Access denied / adapter not created** — run the client as Administrator.
- **No connectivity** — verify `server`, `server_name`, certificate paths, and profile values.
- **Timeouts during startup** — retry with updated dependencies and confirm the server is reachable over UDP.

## Security

- Never commit real `*.crt`, `*.key`, or production-ready profiles.
- Treat all client certificate material as sensitive.

## Limitations

- IPv4 tunnel only.
- In-tunnel DNS is plaintext UDP/53.
- Single server/profile workflow for the MVP.
