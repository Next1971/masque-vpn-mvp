# Masque VPN MVP

MASQUE VPN MVP is an experimental end‑to‑end MASQUE‑based VPN prototype: Android client, Windows client and server. It uses QUIC + HTTP/3 CONNECT‑IP (RFC 9484) with mutual TLS and a Wintun adapter on Windows. This repository contains only sanitized example profiles and public client/server code — no production certificates or keys.

## Status

This repository is an early public project space for a working MVP.
The Android client and server side are already in progress.
A Windows build also exists as an executable prototype.

## Status (July 2026)

After 15 days of testing:

- **Server**: stable, no issues observed.
- **Android client**: stable, no issues observed.
- **Windows client**: stable operation for 4 consecutive days.

## Upcoming release

A v2.0 release is planned for August 2026, no later than 19.08.2026.
It will include a full-featured Android APK and a Windows EXE build.

## Goals

- Build a practical MASQUE-based VPN stack
- Keep the project lightweight and testable
- Publish reusable server and client components step by step
- Prepare a cleaner public codebase for further development

## Current scope

- Android client
- Server side
- Windows prototype build

## Repository plan

This repository will be filled gradually.
The first public materials will include a short project teaser and a sanitized server-side implementation.
Android sources will be added after removing infrastructure-specific references and private endpoints.

## Repository structure

- [`server/`](./server) — sanitized server-side materials
- [`android/`](./android) — sanitized Android-side materials
- [`windows/`](./windows) — sanitized Windows-side materials
- [`docs/ROADMAP.md`](./docs/ROADMAP.md) — publication and MVP roadmap
  
## Privacy note

Sensitive infrastructure details, hostnames, VPS addresses, domains and deployment-specific values are intentionally excluded from the public version of this project.

## Research & DPI Resistance

This project is built on top of peer-reviewed standards and recent academic research on active censorship evasion.

- **RFC 9484** — CONNECT-IP: [IP Proxying Support for HTTP](https://datatracker.ietf.org/doc/html/rfc9484)  
  The core protocol we implement: IP tunneling over HTTP/3 via QUIC.

- **RFC 9298** — MASQUE: [Proxying UDP in HTTP](https://datatracker.ietf.org/doc/html/rfc9298)  
  Foundation for UDP proxying and datagram transport inside HTTP/3.

- **GFW Report, USENIX Security '25** — [How the Great Firewall Blocks QUIC](https://gfw.report/publications/usenixsecurity25/en/)  
  Reverse-engineering of GFW's QUIC SNI blocking revealed implementation bugs:  
  – Flow tracking only activates when `src-port &gt; dst-port`  
  – No reassembly of fragmented Client Hellos across QUIC CRYPTO frames  
  – A single decoy UDP packet before the real Initial confuses the state machine  
  – Degradation under load: up to 80% of connections slip through during daytime flooding  
  Our stack benefits from these findings natively: we run on high ports, rely on QUIC datagrams, and build on `quic-go` ≥ v0.52.0 which ships SNI-slicing by default.

- **IETF MASQUE Working Group** — [datatracker.ietf.org/wg/masque](https://datatracker.ietf.org/wg/masque/about/)  
  Official standardization track for MASQUE protocols.

### Related Implementations

- [tmasque](https://github.com/quangtrieu1312/masque-vpn) by quangtrieu1312 — Linux-only MASQUE server with AF_XDP/eBPF datapath.
- [usque](https://github.com/Diniboy1123/usque) by Diniboy1123 — CLI reimplementation of Cloudflare WARP's MASQUE client.

## Publication approach

This repository is being published gradually.
Each public component is reviewed and sanitized before upload.

## Disclaimer

This project is published for research and development purposes.
