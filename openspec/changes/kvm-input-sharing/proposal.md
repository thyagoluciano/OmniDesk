## Why

Hoje o OmniDesk sincroniza clipboard e transfere arquivos entre PCs pareados na LAN, mas o usuário ainda precisa de um mouse e teclado físicos por máquina. Quem trabalha com múltiplos computadores lado a lado (desktop + laptop + mac mini, por exemplo) quer controlar todos com um único mouse/teclado, movendo o cursor entre eles como se fossem uma tela estendida — cada PC mantendo seu próprio sistema, apenas recebendo os eventos de input. Isso é o espaço de produto do Synergy/Barrier/InputLeap, mas integrado ao modelo P2P, de confiança por pareamento e sem nuvem que o OmniDesk já oferece para clipboard e arquivos.

O requisito central é performance percebida: diferente de clipboard (tolera algumas centenas de ms), o controle de mouse/teclado precisa ser leve e de baixíssima latência para não parecer "com lag", além de permitir organizar visualmente as telas e escolher por quais bordas o cursor transita de um PC para o outro.

## What Changes

- Novo subsistema de **compartilhamento de input** (mouse + teclado) entre nós pareados, com um canal de transporte persistente (WebSocket) dedicado, separado do fluxo request/response usado hoje por clipboard e arquivos.
- Novo **modelo de permissão** de controle de input: opt-in, por par de dispositivos e por direção (autorizar ser controlado é independente de autorizar controlar), desacoplado da permissão de pareamento existente (que hoje libera clipboard+arquivo automaticamente). Revogar o pareamento revoga esta permissão também.
- Novo **modelo de layout de telas**: grafo de adjacência configurável por borda entre os PCs pareados (qual lado de qual tela conecta com qual lado de qual outra), com escalonamento proporcional de coordenada ao cruzar bordas com resoluções diferentes.
- Nova **máquina de estados de posse de input**: dinâmica e simétrica — qualquer nó pareado com permissão pode ser a origem física do mouse/teclado num dado momento; o nó de origem captura e envia, o nó de destino injeta; sem duplicação local (mesma classe de proteção que a prevenção de eco do clipboard já resolve hoje).
- **Failsafe de desconexão**: ao cair o canal de input, o nó de origem dispara automaticamente eventos de liberação (key-up) para todas as teclas que estavam pressionadas no destino, evitando modificadores presos.
- **Três mecanismos de escape** do controle remoto: hotkey global configurável, "hot corner" fixo de retorno (independente do grafo de layout configurado) e toggle por dispositivo na bandeja/dashboard.
- Captura e injeção nativas de input por sistema operacional, entregues em fases (esta proposta cobre a arquitetura completa; a primeira fatia de implementação cobre Linux X11 — ver design.md e tasks.md para o roadmap de Windows, macOS e Linux Wayland).
- **BREAKING**: nenhuma — funcionalidade inteiramente nova, aditiva sobre o pareamento e o dashboard existentes.

## Capabilities

### New Capabilities

- `input-sharing-transport`: canal WebSocket persistente e autenticado para transmitir eventos de mouse/teclado entre nós pareados, com disciplina de prioridade (coalescing de movimento, fila confiável para teclas/cliques) e failsafe de desconexão.
- `input-control-permission`: modelo de permissão opt-in, por par e por direção, para autorizar controle de mouse/teclado entre dois dispositivos já pareados, incluindo revogação em cascata com o despareamento.
- `screen-layout`: modelo de dados e configuração (via dashboard web) do arranjo lógico das telas dos PCs pareados — grafo de adjacência por borda, usado para decidir quando e para onde o cursor transita.
- `input-capture-inject-x11`: captura de eventos globais de mouse/teclado e injeção via X11 (XRecord/XTest) no Linux — primeira plataforma implementada, valida o transporte e a máquina de estados end-to-end.

### Modified Capabilities

(nenhuma — não há specs existentes no projeto; esta é a primeira leva de capabilities documentadas via OpenSpec)

## Impact

- **Código afetado**: `internal/core/server.go` e `node.go` (novo endpoint de upgrade WebSocket reaproveitando `authMiddleware`), `internal/pairing/pairing.go` e `internal/config/config.go` (novo estado de permissão de input por par/direção em `TrustedDevice`), `internal/discovery` (reaproveitado para resolver endereço do peer alvo, sem mudanças), `web/app.js` e `web/index.html` (novo painel de arranjo de telas e toggles de permissão de input), `internal/ui/systray.go` (novo item de toggle rápido).
- **Novo pacote**: um pacote `internal/inputshare` (ou similar) para a máquina de estados, protocolo de eventos e abstração de captura/injeção por SO, com implementações específicas atrás de build tags (seguindo o padrão já usado em `internal/ui` para diferenças por plataforma).
- **Dependências**: nenhuma dependência externa para o transporte (WebSocket pode ser implementado sobre `net/http` + biblioteca padrão ou uma lib leve tipo `nhooyr.io/websocket`/`gorilla/websocket` — decisão registrada em design.md); captura/injeção nativa por SO pode exigir novas dependências específicas de plataforma (ex.: bindings X11 no Linux), avaliadas por fase.
- **Segurança**: aumenta a superfície de risco do pareamento (controle total de mouse/teclado é mais sensível que clipboard/arquivo) — mitigado pelo modelo de permissão separada e opt-in descrito acima.
- **Compatibilidade**: aditivo; nós que não atualizarem ou não habilitarem a permissão continuam funcionando exatamente como hoje (clipboard e arquivos inalterados).
