#!/usr/bin/env bash
cd "$(dirname "$0")"

echo "======================================================="
echo "        Instalador do Crossover para macOS"
echo "======================================================="

# 1. Check if Crossover.app is next to this script
if [ -d "Crossover.app" ]; then
    echo "-> Instalando Crossover.app em /Applications..."
    rm -rf /Applications/Crossover.app
    cp -R "Crossover.app" /Applications/
fi

TARGET_APP="/Applications/Crossover.app"
if [ ! -d "$TARGET_APP" ]; then
    TARGET_APP="$(pwd)/Crossover.app"
fi

if [ ! -d "$TARGET_APP" ]; then
    echo "Erro: Crossover.app não encontrado!"
    exit 1
fi

echo "-> Removendo quarentena do Gatekeeper (Apple)..."
xattr -cr "$TARGET_APP" 2>/dev/null || sudo xattr -rd com.apple.quarantine "$TARGET_APP" 2>/dev/null

echo "-> Aplicando assinatura ad-hoc local para Apple Silicon..."
codesign --force --deep -s - "$TARGET_APP" 2>/dev/null || true

echo "-> Ajustando permissões de execução..."
chmod +x "$TARGET_APP/Contents/MacOS/crossover" 2>/dev/null || true

echo "-> Configurando inicialização automática (LaunchAgent)..."
mkdir -p "$HOME/Library/LaunchAgents"
cat <<'EOF' > "$HOME/Library/LaunchAgents/com.crossover.app.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.crossover.app</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Applications/Crossover.app/Contents/MacOS/crossover</string>
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

launchctl unload "$HOME/Library/LaunchAgents/com.crossover.app.plist" 2>/dev/null || true
launchctl load -w "$HOME/Library/LaunchAgents/com.crossover.app.plist" 2>/dev/null || true

echo "======================================================="
echo "✓ Instalação e desbloqueio concluídos com sucesso!"
echo "-> Abrindo o Crossover..."
open "$TARGET_APP"
echo "======================================================="
sleep 2
