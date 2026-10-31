#!/usr/bin/env bash
set -e

# Project root directory
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

BIN_DIR="$ROOT_DIR/bin"
mkdir -p "$BIN_DIR"

COMMAND="${1:-app}"

build_macos_app() {
    echo "==> Building macOS Application Bundle (otool.app)..."
    APP_NAME="otool.app"
    APP_PATH="$BIN_DIR/$APP_NAME"
    rm -rf "$APP_PATH"

    mkdir -p "$APP_PATH/Contents/MacOS"
    mkdir -p "$APP_PATH/Contents/Resources"

    # 1. Compile binary
    echo "  -> Compiling binary (Metal GPU accelerated, pure Go)..."
    go build -ldflags "-s -w" -o "$APP_PATH/Contents/MacOS/otool" .

    # 2. Copy App Icon
    if [ -f "build/icon.icns" ]; then
        echo "  -> Embedding icon (build/icon.icns)..."
        cp "build/icon.icns" "$APP_PATH/Contents/Resources/icon.icns"
    fi

    # 3. Copy Info.plist
    if [ -f "build/darwin/Info.plist" ]; then
        echo "  -> Injecting metadata (build/darwin/Info.plist)..."
        cp "build/darwin/Info.plist" "$APP_PATH/Contents/Info.plist"
    fi

    # 4. Ad-hoc code signing to avoid Gatekeeper warning
    if command -v codesign >/dev/null 2>&1; then
        echo "  -> Applying ad-hoc codesign..."
        codesign --force --deep --sign - "$APP_PATH"
    fi

    echo "✅ Success! macOS app bundle created at:"
    echo "   $APP_PATH"
    echo "👉 You can double-click it in Finder, run 'open \"$APP_PATH\"', or drag it to /Applications"
}

build_windows_exe() {
    echo "==> Building Windows executable with embedded icon (otool.exe)..."
    # Temporarily copy .syso to root for go build linking
    if [ -f "build/windows/rsrc_windows_amd64.syso" ]; then
        cp "build/windows/rsrc_windows_amd64.syso" "./rsrc_windows_amd64.syso"
    fi

    GOOS=windows GOARCH=amd64 go build -ldflags "-H windowsgui -s -w" -o "$BIN_DIR/otool.exe" .

    # Cleanup temporary syso
    rm -f "./rsrc_windows_amd64.syso"

    echo "✅ Success! Windows 64-bit executable created at:"
    echo "   $BIN_DIR/otool.exe"
}

case "$COMMAND" in
    app)
        build_macos_app
        ;;
    windows|win)
        build_windows_exe
        ;;
    all)
        build_macos_app
        build_windows_exe
        ;;
    clean)
        echo "Cleaning bin/..."
        rm -rf "$BIN_DIR"
        ;;
    *)
        echo "Usage: $0 [app|windows|all|clean]"
        exit 1
        ;;
esac

