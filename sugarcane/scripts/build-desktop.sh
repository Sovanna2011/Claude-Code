#!/usr/bin/env bash
#
# Build the Windows desktop client.
#
# The window is Win32, but building it needs no Windows and no C compiler, so this runs on the same
# machine that builds the service. The result is a real PE executable that can be copied to a
# Windows machine and double-clicked.
#
#   ./scripts/build-desktop.sh              build for 64-bit Windows into desktop/dist
#   ./scripts/build-desktop.sh 386          build for 32-bit Windows instead
#
set -euo pipefail

arch="${1:-amd64}"
here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out="$here/desktop/dist"

mkdir -p "$out"

echo "Building the Farm Area desktop client for windows/$arch"
(
    cd "$here/desktop"

    # -H=windowsgui stops a console window opening behind the form; -s -w drop the symbol table,
    # which is a third of the size and nothing a user of the programme needs.
    GOOS=windows GOARCH="$arch" CGO_ENABLED=0 \
        go build -trimpath -ldflags "-H=windowsgui -s -w" \
        -o "$out/farmarea-desktop.exe" ./cmd/farmarea-desktop
)

# Windows reads the manifest from a file of exactly this name beside the executable. Without it the
# form opens with comctl32 version 5 controls and no idea of the monitor's scaling.
cp "$here/desktop/cmd/farmarea-desktop/farmarea-desktop.exe.manifest" "$out/"

echo
echo "  $out/farmarea-desktop.exe"
echo "  $out/farmarea-desktop.exe.manifest"
echo
echo "Copy both to the Windows machine, keeping them side by side, and run the executable."
echo "It opens a sign-in box pointed at http://127.0.0.1:8080; give it the address of the service"
echo "on your network, or start it with -server http://<host>:8080."
