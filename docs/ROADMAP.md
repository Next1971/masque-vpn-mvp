# Roadmap

## Phase 1

- Publish repository structure
- Add project teaser
- Add sanitized server examples
- Add sanitized Android examples

## Phase 2

- Publish minimal server implementation
- Publish Android source code without private infrastructure references
- Improve configuration templates
- Add setup notes

## Phase 3

- Add Windows-related materials
- Improve documentation
- Add build instructions
- Prepare a cleaner public MVP release

## Rules for public releases

- No real VPS IP addresses
- No private domains or subdomains
- No embedded client certificates
- No private keys
- No production secrets

## Next steps
- Update cryptographic and QUIC dependencies to patched versions
  - Upgrade golang.org/x/crypto and related libraries to the latest secure releases
  - Rebuild and re-test the Windows client to ensure stability after dependency updates
