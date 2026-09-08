## 1. Setup do Projeto e Estrutura Base

- [x] 1.1 Inicializar o módulo Go (`go mod init crossover`) e estruturar diretórios (`cmd/`, `internal/core/`, `internal/clipboard/`, `internal/discovery/`, `internal/transfer/`, `internal/ui/`).
- [x] 1.2 Implementar o modelo de configuração (`config.go`) e persistência de dispositivos confiáveis (`trusted_devices.json`).
- [x] 1.3 Adicionar dependências essenciais (`zeroconf`, `systray`, `clipboard`).

## 2. Descoberta mDNS e Pareamento com PIN

- [x] 2.1 Implementar o serviço de anúncio e descoberta mDNS para `_crossover._tcp`.
- [x] 2.2 Criar o servidor HTTP local seguro para troca de mensagens e endpoints de pareamento.
- [x] 2.3 Implementar o fluxo de handshake com geração de PIN de 6 dígitos e validação de tokens mútua.

## 3. Sincronização de Clipboard (Área de Transferência)

- [x] 3.1 Implementar o watcher de eventos de cópia do sistema operacional (Cocoa para macOS e X11 para Linux).
- [x] 3.2 Criar o motor anti-echo com cache de hash SHA-256 e TTL de descarte de retransmissão.
- [x] 3.3 Implementar o broadcast de eventos de texto entre os nós pareados conectados.

## 4. Transferência de Arquivos P2P

- [x] 4.1 Implementar o handler HTTP para streaming de upload/download de arquivos com gravação direta em disco (`~/Downloads/Crossover`).
- [x] 4.2 Adicionar lógica de resolução de nomes duplicados e preservação de metadados básicos.
- [x] 4.3 Integrar serviço de notificações nativas do sistema operacional (osascript no macOS e notify-send/dbus no Linux).

## 5. Interfaces: CLI e Systray

- [x] 5.1 Implementar comandos da CLI (`crossover daemon`, `crossover devices`, `crossover pair`, `crossover send`, `crossover status`).
- [x] 5.2 Integrar ícone de bandeja (Systray) com status de conexões, atalho para envio de arquivos e opção de pausar clipboard.

## 6. Verificação e Testes de Integração

- [x] 6.1 Executar testes unitários para verificação de anti-echo e resolução de conflito de arquivos.
- [x] 6.2 Validar o fluxo completo de pareamento, sincronização de texto e transferência de arquivos em ambiente local.
