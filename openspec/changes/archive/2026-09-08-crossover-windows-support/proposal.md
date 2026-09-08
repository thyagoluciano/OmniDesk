## Why

Com o Crossover já operacional e empacotado para Linux e macOS, a inclusão do suporte oficial ao Microsoft Windows permite que desenvolvedores e usuários com ambientes híbridos (PC Windows, MacBooks e Linux Desktops) sincronizem a área de transferência e transfiram arquivos instantaneamente na mesma rede local sem depender da nuvem.

## What Changes

- Compilação do executável nativo do Windows (`crossover.exe`) com `-H=windowsgui` para execução silenciosa sem prompt de comando.
- Suporte a notificações de sistema nativas no Windows via PowerShell Toast.
- Suporte a abertura do Dashboard na janela nativa do Microsoft Edge (`msedge --app=...`) pré-instalado no Windows.
- Implementação de comandos CLI `crossover install` e `crossover uninstall` para Windows, manipulando chaves de inicialização automática no Registro (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`).
- Criação de script NSIS (`scripts/crossover.nsi`) e automação (`scripts/build-windows.sh`) para compilar o assistente instalador tradicional `Crossover-Setup-0.1.0-x64.exe` com atalhos no Menu Iniciar, Área de Trabalho e desinstalador no Painel de Controle.

## Capabilities

### New Capabilities
- `windows-installer`: Instalação automática, persistência de autostart no Registro do Windows, atalhos do sistema e empacotador de assistente executável (.exe) via NSIS.

### Modified Capabilities
<!-- Nenhuma especificação existente modificada em suas regras fundamentais -->

## Impact

- **Compatibilidade Multiplataforma**: Crossover agora cobre os 3 principais sistemas operacionais desktop (Linux, macOS e Windows).
- **Subprocessos do Sistema**: Integração com Registro do Windows (HKCU), Microsoft Edge app-mode e PowerShell para notificações.
- **Empacotamento**: Adição de scripts de build para geração de executável Windows e instalador NSIS.
