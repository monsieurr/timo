#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
arch=${1:-arm64}
case "$arch" in arm64|amd64) ;; *) echo 'arch must be arm64 or amd64' >&2; exit 2;; esac
app="$root/release/macos-$arch/Timo.app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
cp "$root/dist/timo-app-darwin-$arch" "$app/Contents/MacOS/timo-app"
cp "$root/dist/timo-darwin-$arch" "$app/Contents/Resources/timo"
cp "$root/packaging/macos/Info.plist" "$app/Contents/Info.plist"
chmod 755 "$app/Contents/MacOS/timo-app" "$app/Contents/Resources/timo"
(cd "$root/release/macos-$arch" && zip -qr "$root/release/timo-macos-$arch.app.zip" Timo.app)
echo "Created $root/release/timo-macos-$arch.app.zip"
