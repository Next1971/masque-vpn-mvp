# Masque VPN MVP

MASQUE VPN MVP is an experimental end‑to‑end MASQUE‑based VPN prototype: Android client, Windows client and server. It uses QUIC + HTTP/3 CONNECT‑IP (RFC 9484) with mutual TLS and a Wintun adapter on Windows. This repository contains only sanitized example profiles and public client/server code — no production certificates or keys.

## Status

This repository is an early public project space for a working MVP.
The Android client and server side are already in progress.
A Windows build also exists as an executable prototype.

## Status (July 2026)

After three days of testing:

- **Android client + server**: stable in everyday use. No critical issues observed so far.
- **Server**: tested on two different VPS providers and works reliably in both setups.
- **Windows client**: experimental and currently unstable. It may randomly lose connection or hang. Use at your own risk until further fixes are made.

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
- [`docs/ROADMAP.md`](./docs/ROADMAP.md) — publication and MVP roadmap
  
## Privacy note

Sensitive infrastructure details, hostnames, VPS addresses, domains and deployment-specific values are intentionally excluded from the public version of this project.

## Publication approach

This repository is being published gradually.
Each public component is reviewed and sanitized before upload.

## Disclaimer

This project is published for research and development purposes.
