# MASQUE Server Deployment Example

Status: planning template.

## Assumptions

- Target OS: Ubuntu 24.04
- Server port: 4433/udp
- Example domain: vpn.example.com
- Authentication model: mTLS
- API and metrics endpoints are disabled for MVP or bound to localhost only

## Deployment outline

1. Prepare a VPS and verify basic connectivity
2. Choose runtime model: native binary + systemd or container-based deployment
3. Create a working directory such as `/opt/masque`
4. Place the server binary and a sanitized `config.server.toml`
5. Prepare TLS materials for mTLS
6. Open the required UDP port in the firewall
7. Start the service and verify logs
8. Test client connectivity with placeholder-based profiles
9. Keep deployment reproducible with example scripts and systemd units

## Notes

- Do not publish real VPS IP addresses
- Do not publish real domains
- Do not publish certificates or private keys
- Do not publish secrets or internal access notes
