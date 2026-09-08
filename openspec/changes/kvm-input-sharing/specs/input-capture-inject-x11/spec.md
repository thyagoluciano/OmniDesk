## ADDED Requirements

### Requirement: Captura global de eventos de mouse e teclado via X11
O sistema SHALL, em um nó Linux rodando sessão X11, capturar globalmente eventos de movimento de mouse, cliques, scroll e teclas pressionadas/liberadas, independentemente de qual janela está em foco.

#### Scenario: Captura funciona com qualquer janela em foco
- **WHEN** o nó está em posse local de input e o usuário move o mouse ou pressiona uma tecla, com qualquer aplicação em foco
- **THEN** o evento correspondente é capturado pelo sistema

### Requirement: Injeção de eventos de mouse e teclado via X11
O sistema SHALL, em um nó Linux rodando sessão X11, injetar eventos de movimento de mouse, cliques, scroll e teclas pressionadas/liberadas recebidos de um nó remoto, de forma indistinguível de input físico local para as aplicações em execução.

#### Scenario: Evento remoto de clique chega a uma aplicação
- **WHEN** o nó recebe um evento de clique esquerdo do nó remoto em posse do input
- **THEN** o clique é injetado na posição atual do cursor e a aplicação sob o cursor recebe o evento normalmente

### Requirement: Tradução de código de tecla nativo para HID Usage Code na captura
O sistema SHALL traduzir cada keycode nativo do X11 capturado para o Usage Code HID canônico correspondente antes de transmitir o evento de tecla, e SHALL descartar (sem transmitir) qualquer tecla capturada cujo keycode não tenha mapeamento conhecido, registrando um aviso.

#### Scenario: Tecla mapeada é traduzida corretamente
- **WHEN** o usuário pressiona a tecla física correspondente a "A" no layout local
- **THEN** o evento transmitido carrega o Usage Code HID de "A", não o keycode X11 nativo nem o caractere interpretado pelo layout

### Requirement: Tradução de HID Usage Code para código de tecla nativo na injeção
O sistema SHALL traduzir cada Usage Code HID recebido para o keycode nativo do X11 correspondente ao layout de teclado configurado localmente antes de injetar o evento de tecla.

#### Scenario: Tecla recebida é injetada no layout local correto
- **WHEN** o nó recebe um evento de tecla com o Usage Code HID de "A", e o layout de teclado local é ABNT2
- **THEN** o evento é injetado usando o keycode X11 que produz "A" no layout ABNT2 local

### Requirement: Nó não reinjeta localmente eventos que está enviando como origem
O sistema SHALL, enquanto um nó estiver atuando como origem de input para um nó remoto (posse de input transferida), suprimir a injeção local dos eventos que ele próprio está capturando e enviando, evitando duplicação de input na máquina de origem.

#### Scenario: Digitar enquanto controla remotamente não afeta a máquina local
- **WHEN** o nó local está em posse remota de input (controlando outro nó) e o usuário pressiona uma tecla
- **THEN** a tecla é capturada e enviada ao nó remoto, e nenhuma tecla é injetada na máquina local
