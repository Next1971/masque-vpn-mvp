# MASQUE VPN — Android client (Android Studio build)

Android client for the MASQUE VPN MVP, built on top of the same Go core (`clientcore`) used by the Windows and Linux variants. It shares the profile format and QUIC/CONNECT-IP client logic. The core is exposed through gomobile (Go to `.aar`), with a thin Kotlin layer on top: `VpnService`, minimal UI, and profile import.

## Status (July 2026)

- Android client and server have shown stable operation for 15 days of testing.
- A packaged Android APK is planned for the v2.0 release.
- Target release date: no later than 19.08.2026.

## Current public scope

This public repository currently includes sanitized Android Studio materials for the MVP. Keep all infrastructure-specific values, real certificates, private keys, and production profiles out of the public tree.

## Project layout

```text
masque-android/
├─ app/
│  ├─ src/main/
│  │  └─ AndroidManifest.xml
│  ├─ build.gradle.kts
│  └─ proguard-rules.pro
└─ README-Android.md
```

Additional Android sources, generated AAR files, signing assets, and private deployment materials are intentionally excluded from the public repository when they contain infrastructure-specific or sensitive data.

## Build notes

- Open the `masque-android` project in Android Studio.
- Review Gradle settings and local SDK configuration before building.
- Keep gomobile-generated artifacts and local signing materials outside the public repository unless they have been sanitized for release.
- Use your own local test profile and certificate materials for device testing.

## Usage notes

- Import a sanitized `.masque` profile on the device.
- Grant Android VPN permission when prompted.
- Verify that the observed exit IP matches the MASQUE server instead of the local ISP.

## Security notes

- Treat `.masque` profiles and all client certificate material as sensitive.
- Never commit real private keys, production certificates, or filled-in end-user profiles.
- Keep signing configs and keystore paths local.

## Known limitations (MVP)

- Single server/profile workflow.
- IPv4 tunnel only.
- In-tunnel DNS currently uses plaintext UDP/53.
