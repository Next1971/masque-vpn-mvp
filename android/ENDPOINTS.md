# Android endpoint sanitization

Before publishing Android sources, replace or remove all infrastructure-specific
values.

## Must not be published

The public Android tree must not contain:

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
- Empty strings where sensitive values are optional

## Recommended practice

For public commits:

- Keep only sanitized example profiles and example configs
- Store real secrets outside the repository
- Replace deployment-specific values before commit
- Re-check commit history if any sensitive data was previously added

Use `profile.example.masque` as the public template. Real user profiles,
certificates, and private keys must be created and stored locally only.
