## Context

O OmniDesk hoje é um conjunto de serviços request/response sobre HTTP puro (`internal/core/server.go`): cada operação (clipboard, upload de arquivo, pareamento) é uma chamada isolada, autenticada por `X-OmniDesk-Device-ID` + `X-OmniDesk-Token` contra `Config.TrustedDevices`. Não existe hoje nenhum canal persistente nem nenhuma dependência de captura/injeção de input do sistema operacional.

Controle de mouse/teclado é uma classe de problema diferente da já resolvida (clipboard/arquivo):
- **Latência perceptível a partir de ~15-20ms**, contra centenas de ms tolerados hoje.
- **Perda de evento tem semânticas opostas por tipo**: perder uma posição de mouse intermediária é inofensivo (a próxima corrige); perder um key-up é um bug visível (tecla "presa").
- **Superfície de risco muito maior**: um peer autorizado a mover o mouse e digitar pode operar a máquina inteira, não apenas ler/escrever no clipboard ou salvar um arquivo.
- **Captura/injeção de input é inerentemente específica de SO** — não existe API cross-platform no ecossistema Go hoje sem trazer cgo ou bibliotecas pesadas (ex.: robotgo). O projeto atualmente evita cgo onde possível (ver `internal/ui/systray_darwin_nocgo.go`, com build tag `darwin && !cgo`).

Esta proposta cobre a arquitetura completa (transporte, permissão, layout, máquina de estados), mas a implementação inicial (tasks.md) entrega apenas a plataforma Linux/X11, para validar o modelo ponta-a-ponta antes de multiplicar o esforço específico de SO em Windows, macOS e Wayland.

## Goals / Non-Goals

**Goals:**
- Canal de input com latência de rede desprezível em LAN e disciplina de fila que não deixa eventos de mouse atrasarem eventos de teclado.
- Modelo de permissão que não amplie silenciosamente o blast radius do pareamento existente.
- Modelo de layout de telas que suporte N nós, cada um podendo ter geometria (resolução) diferente, com transição de borda sem "salto" de cursor perceptível.
- Máquina de estados de posse de input robusta a queda de conexão (sem teclas/modificadores presos no destino).
- Base de protocolo e de dados agnóstica de SO, para que Windows/macOS/Wayland só precisem implementar a interface de captura/injeção, não redesenhar transporte ou layout.

**Non-Goals (nesta proposta):**
- Implementar captura/injeção para Windows, macOS ou Linux/Wayland (ficam como fases subsequentes — ver Migration Plan).
- Compartilhamento de área de trabalho remota (streaming de vídeo/tela) — este change é apenas input, não framebuffer.
- Arrastar-e-soltar arquivos entre PCs atravessando a borda de tela (fora de escopo; `omnidesk send`/dashboard continuam sendo o mecanismo de transferência).
- Layout por monitor individual dentro de um mesmo PC (v1 trata cada PC pareado como um único retângulo lógico, mesmo que tenha múltiplos monitores fisicamente — ver Decisões).
- Suporte a mais de um "operador" simultâneo (não há necessidade de dois mouses físicos ativos ao mesmo tempo neste modelo).

## Decisions

### 1. Transporte: WebSocket único sobre o servidor HTTP existente, sem canal UDP dedicado

Reaproveita `internal/core/server.go` (mesma porta, mesmo `authMiddleware` por token de dispositivo). Um único WebSocket por par de nós conectado, mas com duas filas lógicas dentro da mesma conexão:
- **Fila de movimento**: apenas o evento mais recente é mantido pendente (coalescing) — um novo `MouseMove` substitui o anterior ainda não enviado. Não há motivo para reproduzir posições intermediárias com atraso.
- **Fila de teclas/cliques**: FIFO estrita, nunca descarta, nunca reordena.

Alternativa considerada: canal duplo TCP (confiável) + UDP (perdível) para separar fisicamente as duas classes de evento, como fazem alguns KVMs de hardware/software. Rejeitada para v1: em LAN cabeada ou Wi-Fi doméstico a perda de pacote é rara o suficiente para não justificar operar dois protocolos, dois handshakes de segurança e duas superfícies de firewall/NAT. Pode ser revisitado se medições em campo mostrarem que a fila de movimento está competindo de forma mensurável com a fila de teclas dentro de um único WebSocket.

### 2. Framing binário, não JSON, para eventos de input

Diferente dos payloads JSON usados hoje em clipboard/pareamento (tolerantes a overhead, baixa frequência), eventos de mouse podem chegar a 100+/s. Formato binário fixo por tipo de evento (mouse move/click/scroll, key down/up), com um `uint8` de tipo + campos de tamanho fixo, evita overhead de parsing/alocação por evento e mantém o payload pequeno o bastante para caber em um único frame WebSocket sem fragmentação.

### 3. Teclado: tabela intermediária canônica baseada em USB HID Usage Codes

Nunca transmitir caractere interpretado (o layout do teclado de origem pode ser diferente do layout do teclado configurado no SO de destino — ex.: ABNT2 vs US). Cada implementação de captura por SO traduz o código nativo (X11 keycode, Windows VK, macOS keycode) para um HID Usage Code antes de enviar; cada implementação de injeção traduz de volta HID → código nativo do SO de destino. Essa tabela de tradução é meio da capability `input-capture-inject-<os>`, não do transporte.

### 4. Permissão de controle de input separada do pareamento, por direção

`TrustedDevice` ganha dois novos campos booleanos (ou um sub-struct), simétricos por natureza mas armazenados de forma independente em cada nó: "este dispositivo remoto pode me controlar" e "eu autorizei controlar este dispositivo remoto". A concessão é sempre confirmada nas duas pontas (o nó A pede, o nó B aprova explicitamente — reaproveitando o padrão de aprovação de um clique já usado em `handleApprovePIN`, mas sem gerar um PIN novo, já que o canal entre pares já é autenticado pelo token de pareamento existente). Revogar o pareamento (`RemoveTrustedDevice`) remove esse estado junto.

Alternativa considerada: herdar a permissão de input automaticamente do pareamento existente. Rejeitada — pareamento hoje é aprovado pensando em clipboard/arquivo; forçar o usuário a reavaliar o consentimento para uma capacidade de risco muito maior é mais seguro e é consistente com o padrão de opt-in explícito que already existe no pareamento.

### 5. Layout de telas: grafo de adjacência por borda, por PC (não por monitor)

Cada nó pareado com permissão de input é representado como um único retângulo lógico (usando a resolução total/bounding-box do desktop reportada pelo SO, mesmo que o PC tenha múltiplos monitores). O layout é um grafo onde cada aresta liga uma borda (`top`/`right`/`bottom`/`left`) de um nó a uma borda de outro, com um `offset` (0.0–1.0) alinhando os retângulos quando as dimensões não coincidem exatamente. Ao cruzar uma borda configurada, a coordenada perpendicular é preservada e a coordenada paralela à borda é escalada proporcionalmente entre a dimensão de origem e a de destino (não apenas clampada), para evitar sensação de salto do cursor.

Alternativa considerada: layout por monitor individual (grafo entre cada monitor físico de cada PC, replicando fielmente o painel de arranjo do macOS). Mais fiel, mas exige que cada implementação de captura/injeção exponha a topologia de multi-monitor do SO local de forma uniforme — complexidade desproporcional para v1. Fica registrado como evolução natural pós-v1 (ver Open Questions).

### 6. Máquina de estados de posse de input: simétrica e dinâmica, sem "servidor" fixo

Diferente do modelo clássico do Synergy (um PC com o mouse físico permanente = servidor, os demais = clientes burros), aqui qualquer nó pareado com permissão pode se tornar a origem física de input a qualquer momento (é o usuário fisicamente na frente daquele teclado). Cada nó roda tanto o lado de captura quanto o de injeção. O estado "quem possui o input agora" é local a cada par de nós conectados (não há necessidade de um coordenador global): o nó que está capturando localmente e ativo simplesmente passa a rotear os eventos capturados para o WebSocket em vez de injetá-los localmente, e para de fazer suas próprias chamadas de injeção local enquanto isso — mesma disciplina de "não reprocessar o que você mesmo gerou" que a prevenção de eco do clipboard já resolve em `internal/clipboard`, adaptada para o loop de captura/injeção.

### 7. Failsafe de desconexão: liberação sintética de teclas

O nó de origem mantém um conjunto local de teclas atualmente "pressionadas" (do ponto de vista do que foi enviado) por sessão de input ativa. Ao detectar queda do WebSocket (close, timeout de heartbeat, erro de escrita), dispara imediatamente eventos de key-up sintéticos para esse conjunto tanto localmente (se aplicável) quanto — quando a conexão permitir — sinalizando para o destino liberar tudo. Isso evita modificadores presos (Ctrl/Shift/Alt) que inutilizariam a máquina remota até uma tecla manual ser pressionada nela.

### 8. cgo: decisão adiada para a fase específica de macOS

Para a fase X11 (esta implementação) e para Windows (fase 2), não há necessidade de cgo — X11 tem bindings Go puros para XTest/XRecord, e Windows expõe `SendInput`/`SetWindowsHookEx` via `golang.org/x/sys/windows` ou syscalls diretas. A fase macOS precisará decidir entre habilitar cgo para essa plataforma especificamente (CGEventTap/CGEventPost via Cgo) ou usar `purego` (chamadas dlopen/dlsym sem cgo) — mantendo o padrão hoje usado em `internal/ui/systray_darwin_nocgo.go` de evitar cgo quando possível. Essa decisão fica registrada como tarefa de investigação na fase 3 (Windows) e decisão formal na fase 4 (macOS), não bloqueia a fase 1 (X11).

## Risks / Trade-offs

- **[Risco] Wayland não tem API de captura global padrão** → Mitigação: fase dedicada (última do roadmap) usando `xdg-desktop-portal` (interface `RemoteDesktop`), aceitando que só funcione em compositores que implementam o portal (GNOME/KDE recentes) e que a experiência de permissão seja mais fricção (diálogo do compositor por sessão). Documentar explicitamente as limitações ao usuário quando detectado `XDG_SESSION_TYPE=wayland` sem portal disponível.
- **[Risco] Permissão de controle de input comprometida = controle total da máquina** → Mitigação: permissão separada e opt-in por direção (Decisão 4), failsafe de liberação de teclas (Decisão 7), e os três mecanismos de escape (hotkey global, hot corner fixo, toggle na bandeja/dashboard) descritos no proposal.
- **[Risco] Diferença de layout de teclado entre origem e destino gera tecla errada** → Mitigação: tabela HID Usage Code intermediária (Decisão 3), nunca transmitir caractere interpretado.
- **[Risco] Resoluções muito diferentes entre PCs geram cursor "saltando" ou ilegível na travessia de borda** → Mitigação: escalonamento proporcional de coordenada (Decisão 5), não apenas clamp.
- **[Risco] Fila de movimento competindo com fila de teclas no mesmo WebSocket introduz jitter perceptível sob carga alta** → Mitigação: coalescing agressivo de movimento (Decisão 1); se medições mostrarem problema real, revisitar canal duplo TCP/UDP como evolução, não como retrabalho de zero.
- **[Trade-off] Layout por PC (retângulo único), não por monitor** → Simplifica v1 significativamente, mas usuários com múltiplos monitores por PC terão transições de borda menos precisas nas bordas internas entre os próprios monitores do mesmo PC. Aceito conscientemente para v1 (ver Open Questions).
- **[Risco] macOS exige permissão de Acessibilidade + Monitoramento de Entrada, concedida manualmente pelo usuário no System Settings** → Mitigação (fase 4): fluxo de onboarding claro no instalador/dashboard detectando a ausência da permissão e guiando o usuário, seguindo o padrão já usado pelo `internal/installer/darwin.go` para outras permissões do sistema.

## Migration Plan

Este change **não substitui nem quebra** nenhuma funcionalidade existente — é inteiramente aditivo. O roadmap de implementação é fatiado em changes/fases sequenciais, cada uma entregável e testável isoladamente:

1. **Fase 1 (esta implementação — ver tasks.md)**: fundação de protocolo, transporte WebSocket, modelo de permissão, modelo de layout, máquina de estados, failsafe, e captura/injeção via X11 no Linux. Entrega o KVM funcional entre dois nós Linux/X11.
2. **Fase 2 (change subsequente)**: captura/injeção via Windows (`SendInput`/`SetWindowsHookEx`), reaproveitando 100% do transporte/permissão/layout/máquina de estados da Fase 1.
3. **Fase 3 (change subsequente)**: captura/injeção via macOS (CGEventTap/CGEventPost), incluindo a decisão formal cgo vs purego e o onboarding de permissões de Acessibilidade/Monitoramento de Entrada.
4. **Fase 4 (change subsequente)**: captura/injeção via Linux/Wayland usando `xdg-desktop-portal` RemoteDesktop, com fallback explícito e mensagem clara quando o compositor não suportar.

Rollback: cada fase é isolada por build tags de plataforma (seguindo o padrão de `internal/ui`); desabilitar uma fase específica não afeta as demais nem o restante do OmniDesk. A feature inteira pode ser desligada via a permissão opt-in (Decisão 4) permanecendo não concedida por padrão.

## Open Questions

- Layout por monitor individual (em vez de por PC) deve entrar como evolução pós-v1, ou já vale a pena desenhar o modelo de dados hoje para ser extensível sem migração de config depois? (Recomendação: deixar o schema do grafo aberto para futuramente referenciar `monitor_id` além de `node_id`, sem implementar a UI/captura de topologia multi-monitor ainda.)
- O fluxo de aprovação da permissão de input (Decisão 4) deve reaproveitar literalmente o mecanismo de sessão de pareamento (`pairing.Manager`), ou merece um mecanismo mais simples e dedicado (ex.: endpoint autenticado de "solicitar/aprovar" sem sessão com expiração)? Fica para detalhamento durante a implementação da Fase 1.
- Qual biblioteca WebSocket usar (`nhooyr.io/websocket`, `gorilla/websocket`, ou implementação mínima sobre `net/http` sem dependência externa)? Avaliar durante a Fase 1 considerando manutenção, tamanho de binário e compatibilidade com o padrão "poucas dependências" do projeto.
- Heartbeat/timeout exato para detectar queda de conexão e disparar o failsafe de liberação de teclas (Decisão 7) — valor precisa balancear "detectar rápido" vs "não disparar falso positivo em uma rede Wi-Fi instável".
