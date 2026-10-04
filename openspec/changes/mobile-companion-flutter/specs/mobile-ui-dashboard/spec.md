# mobile-ui-dashboard Specification

## Purpose
Define a estrutura visual, layout de navegação e fluxos de interação do aplicativo mobile OmniDesk desenvolvido em Flutter, composto pela tela de Computadores Pareados e pela tela de Central de Transferências.

## Requirements

### Requirement: Bottom Navigation Structure
The system SHALL / O aplicativo mobile DEVE disponibilizar uma barra de navegação inferior (*Bottom Navigation Bar*) permitindo alternar de forma persistente entre as abas "Computadores" e "Transferências".

#### Scenario: Alternância entre abas
- **WHEN** o usuário toca no ícone de "Transferências" ou "Computadores" na barra inferior
- **THEN** a interface transita suavemente para a respectiva tela mantendo o estado de rolagem e dados carregados

### Requirement: Paired Computers Screen
The system SHALL / A aba de Computadores DEVE listar todos os computadores pareados, indicando status de conectividade em tempo real e atalhos de envio rápido.

#### Scenario: Visualização de computadores
- **WHEN** a aba de Computadores é exibida
- **THEN** cada computador é representado por um card exibindo: nome do dispositivo, endereço de rede, indicador de status (badge verde Online / cinza Offline) e data/hora do último contato
- **AND** cada card possui botões de 1-toque para "Enviar Clipboard" e "Enviar Arquivo"

#### Scenario: Acionamento do leitor QR Code
- **WHEN** o usuário toca no botão de adição (+) na barra de título
- **THEN** a tela de scanner de QR Code é aberta em tela cheia com visor de câmera ativo e instruções de uso

### Requirement: Transfers History Screen
The system SHALL / A aba de Transferências DEVE exibir um feed cronológico de todos os itens trafegados (textos copiados, fotos e arquivos enviados ou recebidos).

#### Scenario: Exibição do feed de transferências
- **WHEN** a aba de Transferências é aberta
- **THEN** os eventos mais recentes aparecem no topo com ícone do tipo (texto, foto, documento), direção (enviado ou recebido), dispositivo envolvido, tamanho e timestamp relativo

#### Scenario: Interação com item do histórico
- **WHEN** o usuário toca em um item de texto
- **THEN** o texto pode ser copiado novamente para a área de transferência local
- **WHEN** o usuário toca em um item de arquivo recebido
- **THEN** é acionada a visualização do arquivo ou a folha nativa de compartilhamento do sistema
