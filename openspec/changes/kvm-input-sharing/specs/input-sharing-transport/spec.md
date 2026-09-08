## ADDED Requirements

### Requirement: Canal WebSocket autenticado por par de dispositivos
O sistema SHALL expor um endpoint de upgrade WebSocket, protegido pelo mesmo mecanismo de autenticação por device ID + token usado nos demais endpoints protegidos, para transmitir eventos de input entre dois nós pareados que tenham a permissão de controle de input concedida em pelo menos uma direção.

#### Scenario: Estabelecer canal entre dois nós autorizados
- **WHEN** o nó A inicia uma conexão WebSocket de input para o nó B enviando seu device ID e token válidos
- **THEN** o nó B aceita o upgrade e o canal fica disponível para troca de eventos de input

#### Scenario: Rejeitar canal sem token válido
- **WHEN** um dispositivo tenta abrir o canal WebSocket de input sem um token válido ou com token de dispositivo não pareado
- **THEN** o nó SHALL rejeitar o upgrade da conexão com um erro de autenticação, sem estabelecer o canal

#### Scenario: Rejeitar canal sem permissão de input concedida
- **WHEN** um dispositivo pareado e autenticado tenta abrir o canal WebSocket de input, mas não possui a permissão de controle de input concedida (ver capability `input-control-permission`) em nenhuma direção com o nó de destino
- **THEN** o nó SHALL rejeitar o upgrade da conexão, sem estabelecer o canal

### Requirement: Coalescing de eventos de movimento de mouse
O sistema SHALL manter no máximo um evento de movimento de mouse pendente de envio por canal ativo, descartando eventos de movimento anteriores ainda não transmitidos quando um novo evento de movimento mais recente estiver disponível.

#### Scenario: Múltiplos movimentos rápidos geram apenas o último envio
- **WHEN** o driver de captura local gera três eventos de movimento de mouse em sequência antes que o primeiro tenha sido escrito no socket
- **THEN** apenas o evento de movimento mais recente é efetivamente transmitido; os dois anteriores são descartados sem erro

### Requirement: Entrega ordenada e sem perdas de eventos de tecla e clique
O sistema SHALL transmitir todos os eventos de tecla (key down/up) e de clique de mouse em ordem estrita, sem descarte, mesmo quando eventos de movimento estiverem sendo coalescidos concorrentemente no mesmo canal.

#### Scenario: Sequência de teclas preservada sob carga de movimento
- **WHEN** eventos de movimento de mouse de alta frequência estão sendo gerados ao mesmo tempo que uma sequência de teclas é pressionada
- **THEN** todos os eventos de tecla chegam ao destino na mesma ordem em que foram capturados, sem nenhum ser descartado

### Requirement: Liberação automática de teclas na desconexão
O sistema SHALL, ao detectar o encerramento ou a falha do canal WebSocket de input enquanto uma sessão de controle está ativa, emitir eventos de liberação (key-up) para todas as teclas que o nó de origem registrou como atualmente pressionadas na sessão.

#### Scenario: Canal cai com tecla modificadora pressionada
- **WHEN** o canal WebSocket de input é encerrado abruptamente enquanto a tecla Ctrl está registrada como pressionada na sessão ativa
- **THEN** o sistema emite um evento de liberação (key-up) para Ctrl imediatamente após detectar a queda do canal

### Requirement: Formato de evento binário e compacto
O sistema SHALL codificar eventos de input em um formato binário de tamanho fixo por tipo de evento (movimento, clique, scroll, tecla), evitando serialização textual (como JSON) no caminho de alta frequência do canal de input.

#### Scenario: Evento de movimento cabe em um único frame sem fragmentação
- **WHEN** um evento de movimento de mouse é serializado para envio
- **THEN** o payload resultante cabe em um único frame WebSocket, sem necessidade de fragmentação em múltiplos frames

### Requirement: Retorno de posse de input por hotkey global
O sistema SHALL permitir configurar uma combinação de teclas que, quando pressionada no nó atualmente em posse do input, devolve imediatamente a posse ao nó físico de origem, independentemente da posição do cursor ou do grafo de layout configurado.

#### Scenario: Usuário aciona a hotkey configurada
- **WHEN** o usuário pressiona a combinação de teclas configurada como hotkey de retorno enquanto um nó remoto está em posse do input
- **THEN** a posse de input retorna imediatamente ao nó físico de origem

### Requirement: Retorno de posse de input por hot corner fixo
O sistema SHALL reservar um canto de tela fixo e configurável que, ao ser atingido pelo cursor, devolve a posse de input ao nó físico de origem, independentemente das adjacências de borda configuradas no grafo de layout.

#### Scenario: Cursor atinge o hot corner de retorno
- **WHEN** o cursor atinge o canto reservado como hot corner de retorno enquanto um nó remoto está em posse do input
- **THEN** a posse de input retorna imediatamente ao nó físico de origem, mesmo que aquele canto também faça parte de uma borda com adjacência configurada

### Requirement: Pausa de compartilhamento de input via bandeja/dashboard
O sistema SHALL permitir que o usuário pause e retome o compartilhamento de input por dispositivo através de um controle na bandeja do sistema ou no dashboard web, sem depender de interação com o mouse ou teclado físico compartilhado.

#### Scenario: Pausar pelo ícone da bandeja
- **WHEN** o usuário seleciona "pausar compartilhamento de input" no ícone da bandeja para um dispositivo específico
- **THEN** o canal de input com aquele dispositivo é suspenso e nenhum evento de input é mais transmitido até ser retomado

#### Scenario: Pausa não afeta outros dispositivos pareados
- **WHEN** o usuário pausa o compartilhamento de input com o dispositivo A
- **THEN** o compartilhamento de input com quaisquer outros dispositivos pareados e autorizados permanece ativo e inalterado
