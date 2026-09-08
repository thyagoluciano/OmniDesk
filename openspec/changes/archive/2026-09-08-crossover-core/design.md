## Context

Ver `proposal.md` para motivação e escopo geral. O sistema operará em rede local (LAN) composto por dois computadores macOS e um Desktop Linux Ubuntu 24.04 (X11). A solução será construída em Go gerando um binário executável único capaz de atuar tanto como daemon/tray quanto como cliente de linha de comando (CLI).

## Goals / Non-Goals

**Goals:**
- Arquitetura totalmente descentralizada (P2P mesh), sem depender de servidores em nuvem ou nós mestres fixos.
- Descoberta automática de nós na mesma sub-rede via mDNS (`_crossover._tcp`).
- Pareamento mútuo com PIN de 6 dígitos e persistência de credenciais em arquivo local (`trusted_devices.json`).
- Sincronização em tempo real do clipboard de texto com prevenção de eco (anti-loop) via hashes SHA-256.
- Transferência direta de arquivos por streaming HTTP com salvamento automático em pasta isolada e notificações nativas.
- Suporte a menu de bandeja (Systray) e comandos CLI completos.

**Non-Goals:**
- Controle compartilhado de teclado e mouse (KVM Virtual) está explicitamente delimitado para a Fase 2.
- Acesso fora da rede local (sem suporte a relay na nuvem ou NAT traversal complexo no escopo inicial).
- Suporte a dispositivos móveis (Android/iOS) nesta fase inicial.

## Decisions

### 1. Linguagem e Runtime: Go (Golang)
- **Decisão**: Utilizar Go como linguagem principal do projeto.
- **Racional**: Compilação para binários nativos estáticos leves (~20MB), facilidade de cross-compilação entre macOS (arm64/amd64) e Linux (amd64), e modelo de concorrência com goroutines ideal para listeners de rede e streams de arquivos.
- **Alternativas consideradas**:
  - *Rust*: Excelente performance e menor consumo de RAM, porém curva de desenvolvimento e tempo de compilação maiores.
  - *Electron/Node*: Fácil para UI rica, porém consumo proibitivo de memória (100MB+ por nó em segundo plano).

### 2. Descoberta Local: mDNS / DNS-SD
- **Decisão**: Anunciar o serviço `_crossover._tcp` na rede local utilizando mDNS (compatível com Bonjour no Mac e Avahi no Linux).
- **Racional**: Não requer configuração manual de endereços IP pelos usuários quando os laptops mudam de endereço DHCP na rede doméstica.
- **Alternativas consideradas**: Servidor de registro centralizado (cria ponto único de falha quando o desktop estiver desligado).

### 3. Pareamento e Segurança: PIN de 6 dígitos com Autenticação de Tokens Mútuos
- **Decisão**: Na primeira tentativa de conexão entre dois nós, gera-se um PIN efêmero de 6 dígitos. Após a confirmação em ambas as telas, um segredo criptográfico compartilhado é gerado e armazenado em `~/.config/crossover/trusted_devices.json`.
- **Racional**: Protege o clipboard contra invasores na mesma rede Wi-Fi sem a complexidade de gerenciar autoridades certificadoras externas.
- **Alternativas consideradas**: Chave estática única manual no `config.yaml` (mais simples, porém menos amigável).

### 4. Sincronização de Clipboard: Anti-Echo com SHA-256
- **Decisão**: Ao receber um texto da rede e injetá-lo no clipboard local do sistema operacional, o agente armazena o hash SHA-256 desse texto em uma lista recente de exclusão por 3 segundos.
- **Racional**: Previne loops de transmissão infinita de eventos (ping-pong entre máquinas).

### 5. Arquitetura em Camadas do Nó Go

```
+-------------------------------------------------------------------+
|                        CROSSOVER AGENT                            |
+-------------------------------------------------------------------+
|  [Entrypoint] main.go / cmd/                                      |
|    - CLI Parser (daemon, send, devices, pair, status)             |
|    - Systray Menu Lifecycle                                       |
+---------------------------------+---------------------------------+
                                  |
                                  v
+---------------------------------+---------------------------------+
|  [Core Services]                                                  |
|    - PeerRegistry: gerencia nós online e dispositivos confiáveis  |
|    - ClipboardEngine: watcher local + anti-echo + broadcast       |
|    - FileTransferEngine: HTTP server streaming + upload client    |
|    - DiscoveryEngine: mDNS Announcer / Querier                    |
+---------------------------------+---------------------------------+
                                  |
                                  v
+---------------------------------+---------------------------------+
|  [Platform Adapters]                                              |
|    - clipboard: Cocoa (macOS) / X11 (Linux)                       |
|    - notifier: osascript (macOS) / notify-send/dbus (Linux)       |
+-------------------------------------------------------------------+
```

## Risks / Trade-offs

- **[Isolamento de clientes no roteador Wi-Fi (AP Isolation)]** → *Mitigação*: Se o roteador bloquear tráfego mDNS ou P2P, o agente suportará especificação manual de IP/porta como fallback na CLI.
- **[Conflito de escrita concorrente no clipboard]** → *Mitigação*: Uso de timestamps e IDs de sequência para que apenas o evento mais recente prevaleça.
- **[Dados confidenciais no clipboard (ex: gerenciadores de senhas)]** → *Mitigação*: Botão de pausa rápida no Systray e CLI; suporte futuro a flags de clipboard protegido (`concealed-type`).
- **[Transferência de arquivos volumosos saturando RAM]** → *Mitigação*: Streaming via `io.Copy` do corpo da requisição HTTP diretamente para o disco, sem carregar o arquivo em memória.
