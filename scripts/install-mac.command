#!/usr/bin/env bash
cd "$(dirname "$0")"

echo "======================================================="
echo "        Instalador do OmniDesk para macOS"
echo "======================================================="

# 1. Check if OmniDesk.app is next to this script
if [ -d "OmniDesk.app" ]; then
    echo "-> Instalando OmniDesk.app em /Applications..."
    rm -rf /Applications/OmniDesk.app
    cp -R "OmniDesk.app" /Applications/
fi

TARGET_APP="/Applications/OmniDesk.app"
if [ ! -d "$TARGET_APP" ]; then
    TARGET_APP="$(pwd)/OmniDesk.app"
fi

if [ ! -d "$TARGET_APP" ]; then
    echo "Erro: OmniDesk.app não encontrado!"
    exit 1
fi

echo "-> Removendo quarentena do Gatekeeper (Apple)..."
xattr -cr "$TARGET_APP" 2>/dev/null || sudo xattr -rd com.apple.quarantine "$TARGET_APP" 2>/dev/null

echo "-> Aplicando assinatura ad-hoc local para Apple Silicon..."
codesign --force --deep -s - "$TARGET_APP" 2>/dev/null || true

echo "-> Ajustando permissões de execução..."
chmod +x "$TARGET_APP/Contents/MacOS/omnidesk" 2>/dev/null || true

echo "-> Configurando inicialização automática (LaunchAgent)..."
mkdir -p "$HOME/Library/LaunchAgents"
cat <<'EOF' > "$HOME/Library/LaunchAgents/com.omnidesk.app.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.omnidesk.app</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Applications/OmniDesk.app/Contents/MacOS/omnidesk</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>ProcessType</key>
    <string>Interactive</string>
</dict>
</plist>
EOF

launchctl unload "$HOME/Library/LaunchAgents/com.omnidesk.app.plist" 2>/dev/null || true
launchctl load -w "$HOME/Library/LaunchAgents/com.omnidesk.app.plist" 2>/dev/null || true

echo "======================================================="
echo "✓ Instalação e desbloqueio concluídos com sucesso!"
echo "-> Abrindo o OmniDesk..."
open "$TARGET_APP"
echo "======================================================="
sleep 2
