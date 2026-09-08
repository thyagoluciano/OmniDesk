#!/usr/bin/env bash
set -e

VERSION="0.1.0"
TARGET_ARCH="${1:-arm64}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
APP_DIR="${DIST_DIR}/Crossover.app"
STAGING_DIR="${DIST_DIR}/dmg_staging"
DMG_PATH="${DIST_DIR}/Crossover_${VERSION}_${TARGET_ARCH}.dmg"
ZIP_PATH="${DIST_DIR}/Crossover_${VERSION}_${TARGET_ARCH}.zip"

# 1. Build macOS application bundle first
"${ROOT_DIR}/scripts/build-macos-app.sh" "${TARGET_ARCH}"

echo "==> Preparando pasta temporária para empacotamento..."
rm -rf "${STAGING_DIR}"
mkdir -p "${STAGING_DIR}"

cp -R "${APP_DIR}" "${STAGING_DIR}/"
# Create standard macOS drag-and-drop link to Applications
ln -s /Applications "${STAGING_DIR}/Applications"

# Copy 1-click installer script
cp "${ROOT_DIR}/scripts/install-mac.command" "${STAGING_DIR}/Instalar Crossover.command"
chmod +x "${STAGING_DIR}/Instalar Crossover.command"

# 2. If running on macOS (hdiutil is available), build true Apple Disk Image (DMG)
if command -v hdiutil >/dev/null 2>&1; then
    echo "==> Gerando imagem Apple Disk Image (.dmg) nativa via hdiutil..."
    rm -f "${DMG_PATH}"
    hdiutil create -volname "Crossover" \
        -srcfolder "${STAGING_DIR}" \
        -ov -format UDZO \
        "${DMG_PATH}"
    echo "✓ DMG criado com sucesso em: ${DMG_PATH}"
fi

# 3. Create universal zip package preserving symlinks (-y) and executable permissions
echo "==> Gerando arquivo distribuível compactado (.zip) com atalho /Applications e instalador..."
rm -f "${ZIP_PATH}"
cd "${STAGING_DIR}"
zip -r -y -q "${ZIP_PATH}" "Crossover.app" "Applications" "Instalar Crossover.command"
cd "${ROOT_DIR}"
rm -rf "${STAGING_DIR}"

echo "✓ Pacote macOS gerado com sucesso em: ${ZIP_PATH}"
if [ -f "${DMG_PATH}" ]; then
    echo "✓ Imagem DMG disponível em: ${DMG_PATH}"
fi
