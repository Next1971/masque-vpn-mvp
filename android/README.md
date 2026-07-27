# Android

Public Android part of the MASQUE VPN MVP.

This directory contains the sanitized Android client materials for the public MVP. Private infrastructure references, production endpoints, domains, hardcoded VPS addresses, signing data, and deployment-specific values must not be committed here.

## Current status

- Android client and server have worked stably for 15 days of testing.
- A full-featured Android APK is planned for the v2.0 release.
- Target release date: no later than 19.08.2026.

## Contents

- `README-Android.md` — Android build and usage notes.
- `profile.example.masque` — sanitized example profile format.
- `ENDPOINTS.md` — notes about endpoint sanitization and placeholder values.
- `masque-android/` — Android Studio project and public Android sources for the MVP.

## Publication notes

Before publishing additional Android sources, verify that the following have been removed or replaced:

- Real VPS IP addresses and domains.
- Production certificates, private keys, and client profiles.
- Signing configs, keystore paths, and local machine settings.
- Hardcoded deployment-specific API or tunnel values.

Use `profile.example.masque` as a template only. Real user profiles should be created locally and must never be committed to a public repository.
