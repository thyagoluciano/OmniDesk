## Purpose

Transfere arquivos diretamente entre dispositivos pareados na rede local de forma rápida, segura e com notificações nativas do sistema operacional.

## ADDED Requirements

### Requirement: Direct Peer-to-Peer File Streaming
The system SHALL / O sistema DEVE realizar o envio de arquivos diretamente entre o nó emissor e o nó receptor na rede local através de streaming contínuo.

#### Scenario: Envio de arquivo para nó pareado
- **WHEN** o usuário seleciona um ou mais arquivos para envio a um dispositivo remoto pareado
- **THEN** o fluxo de dados binários é transmitido diretamente ao nó de destino preservando o nome original e a integridade do arquivo

#### Scenario: Tentativa de envio para nó não pareado ou offline
- **WHEN** um usuário tenta enviar um arquivo para um nó que não está autenticado ou que se encontra inacessível
- **THEN** a transferência é rejeitada e uma mensagem de erro clara é apresentada ao usuário

### Requirement: Automatic Inbound Storage
The system SHALL / O sistema DEVE salvar arquivos recebidos em um diretório configurado de forma segura, evitando sobreescritas acidentais.

#### Scenario: Salvamento de arquivo com nome conflitante
- **WHEN** um arquivo recebido possui o mesmo nome de um arquivo já existente na pasta de destino
- **THEN** o sistema renomeia automaticamente o novo arquivo adicionando um sufixo numérico (ex: `arquivo (1).pdf`) sem sobreescrever o arquivo prévio

### Requirement: Desktop Notifications for File Events
The system SHALL / O sistema DEVE notificar o usuário através do sistema nativo de notificações do sistema operacional sobre a chegada e conclusão de arquivos.

#### Scenario: Notificação de conclusão de recebimento
- **WHEN** a transferência de um arquivo é concluída com sucesso
- **THEN** uma notificação nativa é exibida no sistema operacional contendo o nome do arquivo recebido e o nome do dispositivo remetente
