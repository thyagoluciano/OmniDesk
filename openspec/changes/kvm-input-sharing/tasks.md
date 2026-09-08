## 1. Protocolo e modelo de eventos de input

- [x] 1.1 Definir o formato binário de evento (tipo + payload de tamanho fixo) para: mouse move, mouse click (down/up), scroll, key down/up
- [x] 1.2 Implementar encode/decode desses eventos em Go (sem dependência externa de serialização)
- [x] 1.3 Definir e documentar a tabela de tradução canônica de USB HID Usage Codes usada por todas as implementações de captura/injeção

## 2. Transporte WebSocket

- [x] 2.1 Avaliar e escolher biblioteca WebSocket (ou implementação mínima sobre `net/http`) considerando dependências e tamanho de binário — escolhido `github.com/gorilla/websocket`
- [x] 2.2 Adicionar endpoint de upgrade WebSocket em `internal/core/server.go`, reaproveitando `authMiddleware`
- [x] 2.3 Implementar verificação de permissão de controle de input (capability `input-control-permission`) antes de aceitar o upgrade
- [x] 2.4 Implementar fila de coalescing para eventos de movimento (mantém só o mais recente pendente)
- [x] 2.5 Implementar fila FIFO confiável para eventos de tecla/clique, com prioridade sobre a fila de movimento
- [x] 2.6 Implementar heartbeat/detecção de queda de conexão com timeout configurável
- [x] 2.7 Implementar failsafe: ao detectar queda do canal, emitir key-up sintético para todas as teclas registradas como pressionadas na sessão ativa
- [x] 2.8 Testes de integração: coalescing sob carga, ordem preservada de teclas sob carga de movimento, failsafe de desconexão — `internal/inputshare/transport_test.go`, passam com `-race`

## 3. Permissão de controle de input

- [x] 3.1 Estender `TrustedDevice` em `internal/config/config.go` com os dois campos de permissão por direção — implementado como um único campo `InputControlGranted` por registro (cada nó só precisa saber "este peer pode me controlar"; a direção oposta é o mesmo campo no config do peer)
- [x] 3.2 Implementar fluxo de solicitação/aprovação de permissão — `PermissionManager` dedicado (`internal/inputshare/permission.go`), sessões pendentes em memória (mesmo padrão do `pairing.Manager`), aprovação persiste em `TrustedDevice`
- [x] 3.3 Implementar endpoints (peer-to-peer autenticado para solicitar/consultar status; loopback-only para aprovar/recusar/revogar), seguindo o padrão de `handleDevicesPair`
- [x] 3.4 Implementar revogação em cascata: automática, já que a permissão vive dentro do próprio `TrustedDevice` removido por `RemoveTrustedDevice`
- [x] 3.5 Testes unitários: concessão, revogação manual, revogação em cascata no despareamento — `internal/inputshare/permission_test.go`

## 4. Modelo e configuração de layout de telas

- [x] 4.1 Definir modelo de dados do grafo de adjacência (nó, borda, nó alvo, borda alvo, offset) e persistência em config
- [x] 4.2 Implementar cálculo de escalonamento proporcional de coordenada na travessia de borda
- [x] 4.3 Implementar validação de configuração (bordas/nós inválidos, auto-referência); ciclos não são um caso de erro no modelo atual (uma cadeia de bordas fechada é uma topologia válida — 4 telas em círculo, por exemplo)
- [x] 4.4 Expor endpoints (loopback-only) para ler e salvar o layout configurado
- [x] 4.5 Testes unitários: escalonamento de coordenada com resoluções diferentes, offset de alinhamento, borda sem adjacência configurada — `internal/inputshare/layout_test.go`

## 5. Máquina de estados de posse de input

- [x] 5.1 Implementar estado local de "quem possui o input agora" por par de nós conectados
- [x] 5.2 Implementar transição de posse ao detectar cursor cruzando borda com adjacência configurada
- [x] 5.3 Implementar supressão de injeção local no nó de origem enquanto ele estiver enviando input para o nó remoto
- [x] 5.4 Testes: cobertura via `internal/inputshare/manager_test.go` (detecção de borda, hot corner) e `transport_test.go` (transporte fim a fim simulado). **Não há teste de integração com dois processos `Manager` reais dialando um no outro** — validado apenas em nível de unidade; recomendado antes do merge se possível.

## 6. Mecanismos de escape

- [x] 6.1 Implementar hotkey global configurável de retorno de posse ao nó físico de origem
- [x] 6.2 Implementar hot corner fixo de retorno, independente das adjacências configuradas no grafo de layout
- [x] 6.3 Implementar toggle de pausa/retomada de compartilhamento de input por dispositivo na bandeja (`internal/ui/systray.go`) — implementado como "encerrar sessão ativa" (não há como pausar um dispositivo específico pela bandeja hoje, só ver/encerrar a sessão em andamento); pausa por dispositivo está no dashboard (8.x)
- [x] 6.4 Implementar toggle equivalente no dashboard web — pausa/retoma por dispositivo
- [x] 6.5 Testes: `TestAllHeld`, `TestHitsCorner`, `TestPausePeerBlocksAndClearsActiveSession` em `manager_test.go`. Cobre a lógica pura; não cobre a hotkey disparando de fato via um backend de captura real (depende de 7.6)

## 7. Captura e injeção via Linux/X11

- [x] 7.1 Implementar captura global de mouse/teclado via RECORD (`github.com/BurntSushi/xgb/record`) atrás de build tag `linux`
- [x] 7.2 Implementar injeção de eventos via XTest (`github.com/BurntSushi/xgb/xtest`) atrás da mesma build tag
- [x] 7.3 Implementar tradução de keycode X11 nativo → HID Usage Code na captura, com log de aviso para keycodes sem mapeamento (`keymap_linux.go`, offset evdev+8)
- [x] 7.4 Implementar tradução de HID Usage Code → keycode X11 nativo na injeção
- [x] 7.5 Detectar e tratar graciosamente a ausência de sessão X11 — `xgb.NewConn()` falhando (sem `DISPLAY`/Wayland puro) já cai no fallback "sem backend" tratado pelo `Manager.Start`
- [ ] 7.6 Testes manuais/documentados: captura e injeção end-to-end entre duas VMs/máquinas Linux X11 pareadas — **NÃO EXECUTADO nesta sessão** (sem ambiente Linux/X11 disponível). Compilação cruzada (`GOOS=linux go build`/`go vet`) passa limpa, mas o comportamento real do RECORD/XTest contra um X server de verdade não foi validado. **Risco conhecido e documentado em código** (ver comentário no topo de `capture_linux.go`): não está confirmado que `cook.Reply()` do xgb entrega corretamente todos os chunks subsequentes de um `EnableContext` (a extensão RECORD envia múltiplas respostas para a mesma requisição, um padrão fora do caso comum que o dispatch de cookies do xgb foi escrito para tratar). Precisa ser validado nesta tarefa antes de confiar na captura em produção; se o problema se confirmar, a mitigação é mover `EnableContext` para uma conexão dedicada que leia bytes brutos do socket, contornando o dispatch por cookie do xgb.

## 8. Dashboard web

- [x] 8.1 Implementar painel de arranjo visual de telas (canvas drag-and-drop) em `web/app.js` / `web/index.html` — arraste para posicionar, adjacência de borda inferida automaticamente por retângulos "encostados" (mesmo modelo mental do painel de Monitores do macOS)
- [x] 8.2 Implementar UI de solicitação/aprovação/revogação de permissão de controle de input por dispositivo
- [x] 8.3 Implementar indicador de qual nó está atualmente em posse do input
- [~] 8.4 Hot corner configurável pela UI (select); **hotkey de retorno NÃO tem UI de captura de combinação de teclas** — o campo é persistido e o backend já suporta, mas falta a tela de "pressione a combinação desejada" no dashboard. Fica como pendência explícita, não como "feito".

## 9. Validação end-to-end (Fase 1 completa)

- [ ] 9.1 Testar fluxo completo entre dois nós Linux/X11 pareados — **NÃO EXECUTADO**, requer hardware/VMs Linux reais
- [ ] 9.2 Testar failsafe de desconexão em condições reais de rede — a lógica de failsafe está testada em unidade (`transport_test.go`, simula a queda do socket), mas não em uma queda de rede real entre duas máquinas
- [ ] 9.3 Testar os três mecanismos de escape em sequência, em uso real — **NÃO EXECUTADO**
- [ ] 9.4 Medir latência percebida em LAN doméstica típica e documentar resultado — **NÃO EXECUTADO**

## 10. Roadmap subsequente (fora do escopo desta implementação)

As fases abaixo reaproveitam integralmente o protocolo, transporte, modelo de permissão, modelo de layout e máquina de estados entregues nas seções 1–5, implementando apenas a interface de captura/injeção específica de cada plataforma (equivalente à seção 7, mas para outro SO). Cada uma deve ser proposta como um change OpenSpec separado quando for iniciada:

- **Fase 2 — Windows**: captura via `SetWindowsHookEx`, injeção via `SendInput`, tradução VK ↔ HID Usage Code.
- **Fase 3 — macOS**: captura via `CGEventTap`, injeção via `CGEventPost`, decisão formal cgo vs `purego`, onboarding de permissões de Acessibilidade/Monitoramento de Entrada, tradução keycode macOS ↔ HID Usage Code.
- **Fase 4 — Linux/Wayland**: captura/injeção via `xdg-desktop-portal` (interface `RemoteDesktop`), detecção de suporte do compositor, fallback com mensagem clara quando indisponível.
