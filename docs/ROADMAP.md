# Roadmap

## Current status

- Server: stable after 15 days of testing, with no issues observed.
- Android client: stable after 15 days of testing together with the server.
- Windows client: stable operation for 4 consecutive days.
- Windows Web UI is available for daily use through a local browser interface.

## Completed

- Published repository structure
- Published sanitized server materials
- Published sanitized Android materials
- Published Windows client materials
- Added platform-specific build and usage documentation
- Added sanitized deployment examples and configuration templates

## In progress

- Improve documentation consistency across repository sections
- Continue sanitizing platform-specific materials before broader public release
- Keep dependency and build instructions aligned across Android, Windows, and server components

## Planned for v2.0

- Release a full-featured Android APK
- Release a Windows EXE package
- Finalize the public-facing MVP release documentation
- Target release date: no later than 19.08.2026

## Rules for public releases

- No real VPS IP addresses
- No private domains or subdomains
- No embedded client certificates
- No private keys
- No production secrets

## Next steps

- Update cryptographic and QUIC dependencies to patched versions
- Upgrade `golang.org/x/crypto` and related libraries to the latest secure releases
- Rebuild and re-test the Windows client after dependency updates
- Prepare the v2.0 release package for Android APK and Windows EXE publication
