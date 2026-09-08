## Why

Trabalhar simultaneamente com múltiplos dispositivos (dois MacBooks e um Desktop Linux Ubuntu) em uma mesma bancada gera fricção diária ao compartilhar textos, links e arquivos entre as máquinas sem depender de serviços externos de nuvem. Esta proposta estabelece a fundação do Crossover: um agente P2P leve em Go que opera em rede local (LAN) com descoberta automática, pareamento seguro por PIN, sincronização de área de transferência em tempo real e transferência direta de arquivos, deixando a base pronta para a futura expansão para KVM virtual (controle unificado de teclado e mouse).

## What Changes

- Criação de um daemon P2P em Go (`crossover`) com ícone na barra de menu / bandeja (Systray) para macOS e Linux.
- Implementação de descoberta automática de nós locais via mDNS (`_crossover._tcp`).
- Sistema de segurança e pareamento mútuo baseado em PIN de 6 dígitos para autorização inicial de novos dispositivos na LAN.
- Sincronização contínua e bidirecional da área de transferência de texto (clipboard) com algoritmo anti-eco baseado em hash SHA-256.
- Transferência P2P de arquivos via streaming local direto para pasta dedicada (`~/Downloads/Crossover`) com notificações nativas do sistema.
- Interface de linha de comando (CLI) para iniciar o daemon, listar dispositivos, efetuar pareamento e disparar envio de arquivos.

## Capabilities

### New Capabilities
- `peer-discovery`: Descoberta automática de nós na rede local via mDNS e mecanismo de pareamento seguro por PIN de 6 dígitos com persistência de dispositivos confiáveis.
- `clipboard-sync`: Monitoramento e propagação em tempo real do clipboard de texto entre nós pareados, com prevenção de loops infinitos (anti-echo) e suporte a pausa.
- `file-transfer`: Streaming direto de arquivos máquina a máquina via HTTP/TLS local, com salvamento automático em pasta de recebidos e notificações nativas no desktop.
- `cli-control`: Comandos de terminal (`daemon`, `devices`, `pair`, `send`, `status`) para automação e controle do agente Crossover.

### Modified Capabilities
<!-- Nenhuma capacidade existente modificada (projeto novo) -->

## Impact

- **Código e Dependências**: Nova base de código em Go com bibliotecas para Systray (`fyne.io/systray`), Clipboard (`golang.design/x/clipboard` ou nativa X11/Cocoa) e mDNS (`grandcat/zeroconf`).
- **Sistema Operacional**: Suporte a macOS (Apple Silicon / Intel via APIs Cocoa) e Linux Ubuntu (X11 com suporte a notificações desktop).
- **Rede**: Abertura de porta local para tráfego HTTP/TLS e uso da porta padrão mDNS (5353/UDP).
