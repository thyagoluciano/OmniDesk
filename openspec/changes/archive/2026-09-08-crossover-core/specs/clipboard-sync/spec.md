## Purpose

Sincroniza o conteúdo da área de transferência de texto de forma bidirecional e em tempo real entre todos os dispositivos pareados e conectados na rede local.

## ADDED Requirements

### Requirement: Real-Time Bidirectional Text Synchronization
The system SHALL / O sistema DEVE detectar alterações no clipboard de texto local e transmiti-las imediatamente para todos os nós pareados e online na rede local.

#### Scenario: Cópia de texto no nó emissor
- **WHEN** o usuário copia um texto na máquina local
- **THEN** o conteúdo é transmitido para todos os dispositivos remotos pareados e injetado na área de transferência desses nós em menos de 1 segundo

#### Scenario: Recepção de texto remoto
- **WHEN** um texto é recebido de um nó autenticado
- **THEN** a área de transferência do sistema operacional local é atualizada com o novo texto recebido

### Requirement: Anti-Echo Loop Prevention
The system SHALL / O sistema DEVE evitar loops infinitos de propagação de texto (quando a injeção do texto recebido dispara um novo evento de cópia).

#### Scenario: Injeção de texto remoto sem eco
- **WHEN** o agente local escreve no clipboard um texto recebido de um nó remoto
- **THEN** o evento subsequente de alteração de clipboard não gera uma nova transmissão de volta para a rede

### Requirement: Clipboard Sync Pause and Resume
The system SHALL / O sistema DEVE permitir que o usuário pause temporariamente a sincronização da área de transferência para proteger senhas ou dados confidenciais.

#### Scenario: Sincronização pausada
- **WHEN** a sincronização de clipboard estiver pausada no nó local
- **THEN** nenhum texto copiado localmente é transmitido para os pares e nenhum texto remoto recebido altera o clipboard local
