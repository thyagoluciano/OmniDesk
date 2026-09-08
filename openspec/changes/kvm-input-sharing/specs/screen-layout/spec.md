## ADDED Requirements

### Requirement: Configuração do arranjo de telas por borda
O sistema SHALL permitir que o usuário configure, via dashboard web, quais bordas (topo, direita, baixo, esquerda) de cada nó pareado com permissão de input estão conectadas às bordas de quais outros nós, formando um grafo de adjacência de telas.

#### Scenario: Configurar adjacência entre dois nós
- **WHEN** o usuário arrasta o retângulo representando o nó "Laptop" para a direita do retângulo representando o nó "Desktop" no painel de arranjo e salva
- **THEN** o sistema registra que a borda direita do "Desktop" está conectada à borda esquerda do "Laptop"

### Requirement: Cada nó representado como um único retângulo lógico
O sistema SHALL representar cada nó pareado como um único retângulo no grafo de layout, usando a resolução total do desktop do nó (mesmo quando o nó tem múltiplos monitores físicos), sem modelar bordas internas entre monitores do mesmo nó nesta versão.

#### Scenario: Nó com múltiplos monitores tratado como um bloco
- **WHEN** um nó reporta uma área total de desktop combinando dois monitores físicos lado a lado
- **THEN** o sistema usa essa área total como o retângulo único desse nó no grafo de layout, sem expor os monitores individuais na configuração

### Requirement: Transição de borda dispara mudança de posse de input
O sistema SHALL, quando o cursor do nó atualmente em posse do input atinge uma borda com adjacência configurada, transferir a posse de input para o nó adjacente configurado naquela borda.

#### Scenario: Cursor cruza borda configurada
- **WHEN** o cursor no nó em posse do input atinge o pixel mais à direita da tela, e a borda direita desse nó está configurada como adjacente à borda esquerda de outro nó
- **THEN** a posse de input é transferida para o nó adjacente, e o cursor reaparece na borda esquerda do nó de destino

#### Scenario: Cursor atinge borda sem adjacência configurada
- **WHEN** o cursor no nó em posse do input atinge uma borda que não tem nenhuma adjacência configurada
- **THEN** a posse de input permanece no nó atual e o cursor é impedido de ultrapassar aquela borda (mesmo comportamento de uma borda física de tela)

### Requirement: Escalonamento proporcional de coordenada na travessia
O sistema SHALL, ao transferir a posse de input entre dois nós com dimensões de tela diferentes na borda de travessia, escalar proporcionalmente a coordenada paralela à borda entre a dimensão de origem e a dimensão de destino, em vez de apenas limitar (clamp) o valor.

#### Scenario: Travessia entre telas de alturas diferentes
- **WHEN** o cursor cruza da borda direita de um nó com 1440px de altura, na posição vertical relativa de 50%, para a borda esquerda de um nó com 1080px de altura
- **THEN** o cursor reaparece na posição vertical relativa de 50% da altura do nó de destino (540px), preservando a posição proporcional em vez de manter o valor absoluto em pixels

### Requirement: Offset de alinhamento entre bordas de tamanhos diferentes
O sistema SHALL permitir configurar um deslocamento (offset) de alinhamento entre duas bordas conectadas cujas dimensões não coincidem exatamente, para que o usuário controle onde a travessia começa a corresponder entre as duas telas.

#### Scenario: Alinhar telas de larguras diferentes pelo topo
- **WHEN** o usuário configura um offset de 0.0 entre a borda inferior de um nó mais largo e a borda superior de um nó mais estreito
- **THEN** a travessia alinha a extremidade inicial (topo/esquerda) das duas bordas, e a coordenada é escalada proporcionalmente a partir desse ponto
