#!/usr/bin/env bash
set -e

VERSION="0.1.0"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
TARGET_EXE="${DIST_DIR}/crossover.exe"
ZIP_PATH="${DIST_DIR}/crossover_${VERSION}_windows_x64.zip"

echo "==> Limpando builds anteriores do Windows..."
mkdir -p "${DIST_DIR}"
rm -f "${TARGET_EXE}" "${ZIP_PATH}"

echo "==> Compilando crossover.exe para Windows (amd64, -H=windowsgui)..."
cd "${ROOT_DIR}"
GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui -s -w" -o "${TARGET_EXE}" ./cmd/crossover

echo "==> Criando pacote portátil ZIP..."
STAGING_ZIP="${DIST_DIR}/win_staging"
rm -rf "${STAGING_ZIP}"
mkdir -p "${STAGING_ZIP}"

cp "${TARGET_EXE}" "${STAGING_ZIP}/"
cp "${ROOT_DIR}/assets/crossover.ico" "${STAGING_ZIP}/"

cat <<'EOF' > "${STAGING_ZIP}/LEIA-ME.txt"
Crossover para Windows v0.1.0
==================================================

COMO INSTALAR E USAR:
1. Para instalar com inicialização automática no Windows:
   Abra o Terminal (PowerShell ou Prompt de Comando) nesta pasta e execute:
     .\crossover.exe install

2. Para abrir a interface gráfica de controle:
     .\crossover.exe gui

3. Para desinstalar e remover o autostart:
     .\crossover.exe uninstall

==================================================
EOF

cd "${STAGING_ZIP}"
zip -q -r "${ZIP_PATH}" .
cd "${ROOT_DIR}"
rm -rf "${STAGING_ZIP}"
echo "✓ Pacote ZIP portátil gerado em: ${ZIP_PATH}"

# Check if makensis is available to build Setup Wizard .exe
if command -v makensis >/dev/null 2>&1; then
    echo "==> Compilando assistente de instalação NSIS (.exe)..."
    cd "${ROOT_DIR}/scripts"
    makensis crossover.nsi
    cd "${ROOT_DIR}"
    echo "✓ Instalador Setup gerado com sucesso: ${DIST_DIR}/Crossover-Setup-${VERSION}-x64.exe"
else
    echo "Aviso: 'makensis' não encontrado no PATH. O executável standalone e o ZIP foram gerados."
    echo "Para gerar o Crossover-Setup.exe no Ubuntu, instale o compilador NSIS: sudo apt install -y nsis"
fi

echo "=================================================="
echo "✓ Build do Windows concluído!"
echo "=================================================="
