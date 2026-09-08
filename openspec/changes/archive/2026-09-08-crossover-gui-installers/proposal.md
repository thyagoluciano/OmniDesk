## Why

Com a sincronização de clipboard e o envio de arquivos validados e operacionais entre Linux e macOS, a operação via terminal e menu de bandeja atende desenvolvedores, mas carece de uma experiência visual moderna (visualização de dispositivos em tempo real, envio de arquivos arrastando e soltando) e de instalação simplificada que inicie o Crossover automaticamente com o sistema operacional sem exigir abertura manual de terminais.

## What Changes

- Criação de uma interface gráfica de desktop (GUI) leve e moderna em Webview nativo (WebKitGTK no Linux e WKWebView no macOS).
- Adição de zona de arrastar e soltar (Drag and Drop) na interface gráfica para envio direto de arquivos para qualquer máquina selecionada.
- Adição de visualizador e gestor de pareamento interativo por PIN na interface gráfica.
- Criação de instalador nativo para Ubuntu Linux (`crossover install`, integração com `.desktop`, ícones hicolor e serviço systemd de usuário para autostart).
- Criação de instalador e empacotador para macOS (bundle `Crossover.app`, ícone Apple `.icns`, `LaunchAgent` para inicialização automática no boot e geração de `.dmg`).

## Capabilities

### New Capabilities
- `desktop-gui`: Janela visual de desktop em Webview nativo exibindo dashboard de nós ativos, dropzone para envio de arquivos com drag-and-drop, visualizador de status do clipboard e pareamento por PIN.
- `linux-installer`: Rotinas de instalação e empacotamento para Ubuntu Linux com registro no lançador de aplicativos do GNOME, ícones do sistema e inicialização automática via systemd de usuário.
- `macos-installer`: Empacotamento em bundle oficial `Crossover.app` para macOS, com ícone de sistema, autostart via `LaunchAgent` e gerador de imagem `.dmg`.

### Modified Capabilities
<!-- Nenhuma especificação existente modificada em suas regras fundamentais -->

## Impact

- **Frontend & Visual**: Interface HTML5/CSS moderna servida pelo próprio nó Go e renderizada pelo WebKit nativo do sistema (WebKitGTK no Ubuntu e WebKit no macOS), com consumo levíssimo de memória (~20-30MB).
- **Sistema e Inicialização**: Criação de arquivos de serviço e autostart tanto em `~/.config/systemd/user/` (Linux) quanto em `~/Library/LaunchAgents/` (macOS).
- **Builds e Distribuição**: Scripts de automação para empacotar `.deb` no Linux e `.app`/`.dmg` no Mac.
