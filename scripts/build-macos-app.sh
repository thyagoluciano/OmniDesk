#!/usr/bin/env bash
set -e

VERSION="0.1.0"
TARGET_ARCH="${1:-arm64}" # default to Apple Silicon arm64 (MacBook M1/M2/M3), can pass amd64
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
APP_DIR="${DIST_DIR}/OmniDesk.app"
CONTENTS_DIR="${APP_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"

echo "==> Limpando builds macOS anteriores..."
rm -rf "${APP_DIR}"
mkdir -p "${MACOS_DIR}"
mkdir -p "${RESOURCES_DIR}"

if [ "$(uname -s)" = "Darwin" ]; then
    CGO_ENABLED=1 GOOS=darwin GOARCH="${TARGET_ARCH}" go build -ldflags="-s -w" -o "${MACOS_DIR}/omnidesk" ./cmd/omnidesk
else
    CGO_ENABLED=0 GOOS=darwin GOARCH="${TARGET_ARCH}" go build -ldflags="-s -w" -o "${MACOS_DIR}/omnidesk" ./cmd/omnidesk
fi

echo "==> Gerando PkgInfo..."
echo -n "APPL????" > "${CONTENTS_DIR}/PkgInfo"

echo "==> Copiando ícones..."
cp "${ROOT_DIR}/assets/omnidesk.png" "${RESOURCES_DIR}/AppIcon.png"
cp "${ROOT_DIR}/assets/omnidesk.svg" "${RESOURCES_DIR}/AppIcon.svg"

echo "==> Gerando Info.plist..."
cat <<EOF > "${CONTENTS_DIR}/Info.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>omnidesk</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>com.omnidesk.app</string>
    <key>CFBundleName</key>
    <string>OmniDesk</string>
    <key>CFBundleDisplayName</key>
    <string>OmniDesk</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>LSMinimumSystemVersion</key>
    <string>11.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSSupportsAutomaticGraphicsSwitching</key>
    <true/>
</dict>
</plist>
EOF

echo "==> Bundle OmniDesk.app criado com sucesso em: ${APP_DIR}"
