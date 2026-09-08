## Purpose

Provides the macOS-specific half of input-sharing: global capture of local mouse/keyboard events and injection of remote ones, so a Mac can act as either the sending or receiving node in a KVM-style input-sharing session on equal footing with an already-supported Linux/X11 node.

## ADDED Requirements

### Requirement: Captura global de eventos de mouse e teclado no macOS
O sistema SHALL, em um nó macOS, capturar globalmente eventos de movimento de mouse, cliques, scroll e teclas pressionadas/liberadas, independentemente de qual aplicação está em foco.

#### Scenario: Captura funciona com qualquer aplicação em foco
- **WHEN** o nó está em posse local de input e o usuário move o mouse ou pressiona uma tecla, com qualquer aplicação em foco
- **THEN** o evento correspondente é capturado pelo sistema

### Requirement: Injeção de eventos de mouse e teclado no macOS
O sistema SHALL, em um nó macOS, injetar eventos de movimento de mouse, cliques, scroll e teclas pressionadas/liberadas recebidos de um nó remoto, de forma indistinguível de input físico local para as aplicações em execução.

#### Scenario: Evento remoto de clique chega a uma aplicação
- **WHEN** o nó recebe um evento de clique esquerdo do nó remoto em posse do input
- **THEN** o clique é injetado na posição atual do cursor e a aplicação sob o cursor recebe o evento normalmente

### Requirement: Tradução de keycode nativo do macOS para HID Usage Code na captura
O sistema SHALL traduzir cada keycode virtual nativo do macOS capturado para o Usage Code HID canônico correspondente antes de transmitir o evento de tecla, e SHALL descartar (sem transmitir) qualquer tecla capturada cujo keycode não tenha mapeamento conhecido, registrando um aviso.

#### Scenario: Tecla mapeada é traduzida corretamente
- **WHEN** o usuário pressiona a tecla física correspondente a "A" no layout local
- **THEN** o evento transmitido carrega o Usage Code HID de "A", não o keycode nativo do macOS nem o caractere interpretado pelo layout

### Requirement: Tradução de HID Usage Code para keycode nativo do macOS na injeção
O sistema SHALL traduzir cada Usage Code HID recebido para o keycode virtual nativo do macOS correspondente ao layout de teclado configurado localmente antes de injetar o evento de tecla.

#### Scenario: Tecla recebida é injetada no layout local correto
- **WHEN** o nó recebe um evento de tecla com o Usage Code HID de "A", e o layout de teclado local é ABNT2
- **THEN** o evento é injetado usando o keycode nativo do macOS que produz "A" no layout ABNT2 local

### Requirement: Nó não reinjeta localmente eventos que está enviando como origem
O sistema SHALL, enquanto um nó macOS estiver atuando como origem de input para um nó remoto (posse de input transferida), suprimir a entrega local dos eventos que ele próprio está capturando e enviando, evitando duplicação de input na máquina de origem.

#### Scenario: Digitar enquanto controla remotamente não afeta a máquina local
- **WHEN** o nó local está em posse remota de input (controlando outro nó) e o usuário pressiona uma tecla
- **THEN** a tecla é capturada e enviada ao nó remoto, e nenhuma tecla é entregue a nenhuma aplicação na máquina local

### Requirement: Onboarding de permissões do sistema necessárias para captura/injeção
O sistema SHALL, ao detectar que a captura/injeção de input falhou por falta de permissão de Acessibilidade e/ou Monitoramento de Entrada do macOS, informar claramente ao usuário quais permissões faltam e apresentar um caminho direto para concedê-las nas Configurações do Sistema, em vez de falhar silenciosamente ou travar.

#### Scenario: Primeira execução sem as permissões concedidas
- **WHEN** o input-sharing tenta iniciar em um nó macOS que ainda não concedeu permissão de Acessibilidade e/ou Monitoramento de Entrada
- **THEN** o sistema exibe uma mensagem acionável explicando quais permissões faltam e como concedê-las, e o input-sharing permanece indisponível (sem travar o restante do produto) até que a permissão seja concedida

#### Scenario: Permissão concedida após o aviso
- **WHEN** o usuário concede as permissões solicitadas e reinicia o nó (ou o sistema detecta a concessão)
- **THEN** o input-sharing passa a funcionar normalmente nesse nó, sem exigir nenhuma outra reconfiguração
