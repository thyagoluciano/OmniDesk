#!/usr/bin/env bash
set -e

VERSION="0.1.0"
ARCH="amd64"
PKG_NAME="crossover_${VERSION}_${ARCH}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
PKG_DIR="${DIST_DIR}/${PKG_NAME}"

echo "==> Limpando builds anteriores..."
rm -rf "${PKG_DIR}" "${DIST_DIR}/${PKG_NAME}.deb"
mkdir -p "${PKG_DIR}/DEBIAN"
mkdir -p "${PKG_DIR}/usr/bin"
mkdir -p "${PKG_DIR}/usr/share/applications"
mkdir -p "${PKG_DIR}/usr/share/icons/hicolor/256x256/apps"
mkdir -p "${PKG_DIR}/usr/share/icons/hicolor/scalable/apps"
mkdir -p "${PKG_DIR}/usr/lib/systemd/user"

echo "==> Compilando binário Linux amd64..."
cd "${ROOT_DIR}"
go build -ldflags="-s -w" -o "${PKG_DIR}/usr/bin/crossover" ./cmd/crossover

echo "==> Copiando recursos e assets..."
cp "${ROOT_DIR}/assets/crossover.desktop" "${PKG_DIR}/usr/share/applications/"
cp "${ROOT_DIR}/assets/crossover.png" "${PKG_DIR}/usr/share/icons/hicolor/256x256/apps/"
cp "${ROOT_DIR}/assets/crossover.svg" "${PKG_DIR}/usr/share/icons/hicolor/scalable/apps/"
cp "${ROOT_DIR}/assets/crossover.service" "${PKG_DIR}/usr/lib/systemd/user/"

echo "==> Gerando DEBIAN/control..."
cat <<EOF > "${PKG_DIR}/DEBIAN/control"
Package: crossover
Version: ${VERSION}
Section: net
Priority: optional
Architecture: ${ARCH}
Maintainer: Crossover Team <thyagoluciano@gmail.com>
Description: Sincronizacao P2P de Clipboard e Arquivos em Rede Local
 Crossover e uma ferramenta ultraleve para sincronizacao em tempo real de
 area de transferencia e transferencia de arquivos direta entre Linux e macOS.
EOF

echo "==> Gerando DEBIAN/postinst..."
cat <<'EOF' > "${PKG_DIR}/DEBIAN/postinst"
#!/bin/sh
set -e
if [ -x /usr/bin/update-desktop-database ]; then
    /usr/bin/update-desktop-database /usr/share/applications || true
fi
if [ -x /usr/bin/gtk-update-icon-cache ]; then
    /usr/bin/gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
fi
exit 0
EOF
chmod 755 "${PKG_DIR}/DEBIAN/postinst"

echo "==> Empacotando com dpkg-deb..."
dpkg-deb --build --root-owner-group "${PKG_DIR}" "${DIST_DIR}/${PKG_NAME}.deb"

echo "==> Pacote Debian gerado com sucesso: ${DIST_DIR}/${PKG_NAME}.deb"
