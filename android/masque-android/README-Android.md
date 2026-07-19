# MASQUE VPN — Android client (Android Studio build)

Minimal Android VPN client built on top of the same Go core (`clientcore`)
used by the Windows/Linux versions. It shares the profile format and the
QUIC/CONNECT-IP (MASQUE) client logic. The core is exposed via gomobile
(Go → `.aar`), with a thin Kotlin layer on top: `VpnService` +
minimal UI + profile import.

---

## Project layout

```text
masque-android/
├─ app/
│  ├─ src/main/
│  │  ├─ java/com/zavodovskii/masque/
│  │  │  ├─ MainActivity.kt        UI: profile import, connect button, status
│  │  │  ├─ MasqueVpnService.kt    VpnService: builds TUN, passes fd to Go core
│  │  │  └─ ProfileStore.kt        .masque parsing, certificate storage
│  │  ├─ res/…                     layouts + strings
│  │  └─ AndroidManifest.xml
│  ├─ libs/                        masque.aar goes here (step 2)
│  └─ build.gradle.kts
├─ go-src/masque-b2/               Go core sources + gomobile bridge
│  ├─ mobile/masque.go             gomobile bridge (Connect/Tunnel/Config/Callback)
│  ├─ internal/clientcore/         shared core (same as Windows/Linux)
│  ├─ cmd/…                        desktop wrappers (not needed for AAR build)
│  ├─ go.mod / go.sum
├─ scripts/build-aar.bat           gomobile build script for masque.aar (Windows)
├─ profile.masque                  example client profile with mTLS material
└─ README-Android.md               this file
```

---

## Prerequisites

Install the latest stable Android Studio and via SDK Manager:

- Android SDK Platform 34  
- NDK (Side by side) — required for gomobile  
- CMake (usually installed together with NDK)  

On your Windows machine:

- Go 1.21+  
- Internet access for the initial Gradle plugins download and `gomobile`
  tool installation.[web:137][web:138]

---

## Step 1 — Build `masque.aar` (Go core)

`gomobile` turns the Go `mobile` package into an Android `.aar` library.

### Option A — using the script (recommended)

Set environment variables (Control Panel → “Edit environment variables”):

- `ANDROID_HOME` → Android SDK path, e.g.  
  `C:\Users\YOU\AppData\Local\Android\Sdk`
- `ANDROID_NDK_HOME` → NDK path, e.g.  
  `C:\Users\YOU\AppData\Local\Android\Sdk\ndk\<version>`

Open `cmd` in the project directory and run:

```bat
scripts\build-aar.bat
```

The script will install `gomobile`/`gobind`, run `gomobile init` and build
`app\libs\masque.aar`.

### Option B — manual gomobile invocation

```bat
go install golang.org/x/mobile/cmd/gomobile@latest
go install golang.org/x/mobile/cmd/gobind@latest

set PATH=%GOPATH%\bin;%PATH%     REM see `go env GOPATH`
gomobile init

cd go-src\masque-b2
gomobile bind -target=android -androidapi 24 -o ..\..\app\libs\masque.aar .\mobile
```

After a successful run you should see `app/libs/masque.aar` (a few MB,
containing native `.so` for arm64/arm/x86_64).[web:137][web:140]

> If `gomobile bind` complains about NDK, verify `ANDROID_NDK_HOME` and that
> NDK (side by side) is actually installed via SDK Manager (not just
> “NDK (obsolete)”).

---

## Step 2 — Open and build APK in Android Studio

1. In Android Studio: **File → Open** → select the `masque-android` folder.  
2. Wait for Gradle Sync. On first launch, Studio may suggest creating
   `local.properties` with `sdk.dir` — accept this (or it will generate
   the file automatically).[web:149]
3. Make sure `app/libs/masque.aar` exists; otherwise the build will fail
   on `implementation(files("libs/masque.aar"))`.
4. Build the APK:  
   **Build → Build Bundle(s) / APK(s) → Build APK(s)**.

The resulting file will be:

```text
app/build/outputs/apk/debug/app-debug.apk
```

For installation on a test device, the debug APK is sufficient. For a
release build and signing, use  
**Build → Generate Signed Bundle / APK** with your own keystore.

---

## Step 3 — Install and run on device

1. Copy `app-debug.apk` to your phone and install it  
   (enable “install from unknown sources” if needed).
2. Copy `profile.masque` to the device storage.
3. Open the MASQUE VPN app → tap “Import profile” → select `profile.masque`.  
   You should see a “Profile imported” message.
4. Tap “Connect”. Android will show the system VPN permission dialog —
   allow it. A key icon will appear in the status bar (VPN active).[web:149]
5. To verify tunneling, open a browser and visit e.g. `https://ifconfig.me`
   — it should show the MASQUE server’s IP, not your local ISP address.

Disconnect via the “Disconnect” button in the app.

---

## How it works (high level)

- `MasqueVpnService` (Kotlin) builds a TUN interface via
  `VpnService.Builder`: address `10.8.0.254/24`, route `0.0.0.0/0`
  (full tunnel), DNS from the profile. It obtains the TUN file descriptor.[web:149]
- The Go core receives this `fd`
  (`Mobile.connect(cfg, fd, cb)`), wraps it into a `tun.Device`
  (`CreateUnmonitoredTUNFromFD`) and forwards packets between TUN and the
  QUIC/CONNECT-IP tunnel — using the same code as on Windows/Linux.
- Addresses/routes/DNS are applied on Android via `VpnService.Builder`; the
  Go core does not touch them, which keeps the bridge portable and clean.
- mTLS certificates are stored in the app’s internal storage
  (`files/certs/`), not accessible to other apps.

---

## Security notes

- The `profile.masque` file contains the client’s private key and other
  sensitive material. Treat it as a secret: do not commit it to git and
  do not publish it in public repositories.
- `.gitignore` excludes build artifacts and `.aar` binaries; only sources
  and example configs are intended to be versioned.[web:148][web:150]

---

## Known limitations (MVP)

- Single server/profile (no server list). UI is intentionally minimal.
- IPv4 tunnel only (server is IPv4).
- DNS in the tunnel uses plaintext UDP/53 (hidden from the local ISP but
  visible to the MASQUE server). DoH/DoT is a potential future enhancement.
- The real client address is assigned from the server-side pool
  (e.g. `10.8.0.0/24`); the Android interface uses a fixed `.254` address
  in the same subnet for this single-client MVP.
