# Android endpoint sanitization

Before publishing Android sources, replace or remove all infrastructure-specific values.

## Must not be published

- Real VPS IP addresses
- Real domains and subdomains
- Production API endpoints
- Embedded CA certificates
- Client certificates
- Private keys
- Hardcoded update URLs
- Analytics or crash-reporting endpoints
- Internal test hostnames

## Public repository rule

All public Android examples must use placeholders such as:

- `vpn.example.com`
- `https://example.com/api`
- `EXAMPLE_VALUE`
- empty strings where sensitive values are optional
