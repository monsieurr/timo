# Install Timo

Timo needs no Go, Node.js, browser extension, or online account to run. It uses a local Go HTTP server so both the graphical interface and the REPL can safely modify the same JSON file. **Do not open `web/index.html` directly.**

## Windows 10/11 (x64) — easy install

1. Extract **the complete** `timo-v1.0.4-windows-install.zip`.
2. Close any old `timo web` terminal/server from v1.0.3 or earlier.
3. Double-click **`Install Timo (Windows).cmd`** in the extracted folder.
4. Find **Timo** in the Start Menu. Click to open the timer in your default browser. You do not need to open PowerShell.

Also installed in the Start Menu: **Timo terminal** (interactive REPL) and **Quit Timo** (stops its background web server). Reopening Timo reuses a running Timo server instead of creating a second one.

The installer copies the executables to `%LOCALAPPDATA%\Programs\Timo`, needs no administrator permissions, and creates shortcuts. Run the same installer again to update it. The optional `Uninstall Timo (Windows).cmd` removes binaries and Start Menu shortcuts but deliberately keeps your data. For a conventional installer/uninstaller `.exe`, the repository includes an Inno Setup script and automated packaging workflow; **that `.exe` was not built in this environment**.

You can also skip installation and directly double-click `dist\timo-app-windows-amd64.exe` in the extracted ZIP to launch it (portable mode). To launch the terminal, double-click `dist\timo-windows-amd64.exe`.

Windows may display a Microsoft Defender SmartScreen warning for unsigned, unfamiliar binaries. Do not disable Defender or other system protections. Only run downloaded builds that you trust; inspect the source and build it yourself if desired.

## macOS — install the app

1. Download the `.app.zip` matching your Mac (`arm64` for Apple Silicon M-series, `amd64` for Intel).
2. Extract it and move `Timo.app` to `~/Applications` or `/Applications` using Finder.
3. Open **Timo** using Finder or Spotlight. It starts the local Go server and opens the dashboard in your default browser.

Timo is a menu-bar-less background launcher; the browser remains the main interface. For a terminal command, run the optional `packaging/macos/install-cli.sh` script from the source distribution after moving the `.app`. It links the app's CLI binary to `~/.local/bin/timo` (add that directory to your shell PATH if necessary).

**Note:** This prototype `.app` is not Apple Developer signed or notarized. Gatekeeper may block it when downloaded. Signing and notarization are required for a frictionless public macOS distribution. On an untrusted build, do not bypass the warning. A trusted local build from source is an alternative.

## Data, updates and background process

The JSON remains at the same location as previous versions; installation does not delete or migrate it:

- Windows: `%APPDATA%\timo\data.json` (or the legacy Pomo location, if previously used).
- macOS: `~/Library/Application Support/timo/data.json` (or the legacy Pomo location).

Environment variable `TIMO_DATA_FILE` is still supported. Uninstalling leaves the JSON intact so your settings, sessions and current timer survive upgrades. To delete everything, back up the JSON and then delete its directory explicitly.

The graphical launcher opens a browser tab, not a custom native window. The local server continues running after closing that tab; its timer uses timestamps rather than high-frequency background writes. To stop it, select **Quit Timo** on Windows or run `timo stop-desktop` from a terminal where the CLI binary is available. A `timo web` process launched separately from a terminal is instead stopped in that terminal with Ctrl+C.

Only one listener uses port `127.0.0.1:8765`. Close older Timo instances before upgrading to avoid opening the previous UI. The API remains loopback-only; it is not exposed to other devices.

## Maintainer: build a conventional Windows Setup EXE

On a Windows machine with Inno Setup 6/7 and Go:

```powershell
mkdir dist -ErrorAction SilentlyContinue
go build -trimpath -o dist/timo-windows-amd64.exe .
go build -trimpath -ldflags '-H=windowsgui -X main.desktopBuild=true' -o dist/timo-app-windows-amd64.exe .
& 'C:\Program Files (x86)\Inno Setup 6\ISCC.exe' packaging\windows\Timo.iss
```

This generates `release/timo-setup-windows-amd64.exe` (a normal installer with a Start Menu entry and standard Windows uninstall registration). The GitHub Actions packaging workflow automates it on Windows runners. Distribution signing is not configured: set up code signing before publicly publishing.

For macOS, package after cross-compiling the two architecture-specific binaries:

```sh
sh packaging/macos/make-app.sh arm64
sh packaging/macos/make-app.sh amd64
```

Unsigned apps/installer artefacts should not be represented as trusted or notarized.
