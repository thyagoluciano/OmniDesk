# cli-control Specification

## Purpose
Disponibiliza uma interface de linha de comando para operação do serviço em background, pareamento de dispositivos, envio de arquivos e verificação de status.

## Requirements

### Requirement: Daemon Lifecycle Management via CLI
The system SHALL / O binário do OmniDesk DEVE permitir iniciar e inspecionar o daemon em segundo plano através de comandos de terminal.

#### Scenario: Início do daemon
- **WHEN** o usuário executa o comando `omnidesk daemon`
- **THEN** o nó inicializa seus serviços de rede, monitoramento de clipboard e ícone de bandeja (quando disponível)

#### Scenario: Consulta de status do nó
- **WHEN** o usuário executa o comando `omnidesk status`
- **THEN** o terminal exibe o estado atual do daemon, portas ativas e resumo de conexões locais

### Requirement: Device Listing and Pairing via CLI
The system SHALL / O usuário DEVE conseguir listar nós descobertos e iniciar ou responder a solicitações de pareamento diretamente pelo terminal.

#### Scenario: Listagem de dispositivos na rede
- **WHEN** o usuário executa o comando `omnidesk devices`
- **THEN** o sistema exibe a lista de nós visíveis na rede local, indicando o status de cada um (pareado ou não pareado)

#### Scenario: Início de pareamento por CLI
- **WHEN** o usuário executa `omnidesk pair <device-name-or-id>`
- **THEN** a requisição de pareamento é disparada e o PIN de 6 dígitos é impresso no terminal aguardando a confirmação do nó remoto

### Requirement: Direct CLI File Dispatch
The system SHALL / O usuário DEVE conseguir disparar o envio de um arquivo para um dispositivo remoto especificando o caminho local e o destino.

#### Scenario: Envio de arquivo com sucesso via terminal
- **WHEN** o usuário executa `omnidesk send <caminho-arquivo> <dispositivo>`
- **THEN** o arquivo é transmitido ao nó de destino e uma barra de progresso ou confirmação de envio é apresentada no terminal
