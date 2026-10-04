# mobile-qr-pairing Specification

## Purpose
Permite que o aplicativo mobile OmniDesk (iOS / Android) realize pareamento instantâneo com computadores desktop na mesma rede local através do escaneamento de um QR Code na tela do computador, sem digitação manual de endereço IP ou PIN.

## Requirements

### Requirement: QR Code Generation on Desktop Web UI
The system SHALL / O painel Web do OmniDesk desktop DEVE fornecer uma ação para gerar e exibir um QR Code de pareamento contendo as informações de rede e credencial de sessão temporária da máquina.

#### Scenario: Abertura do modal de pareamento mobile no Desktop
- **WHEN** o usuário clica no botão "Parear Celular" no painel web (`/ui/`)
- **THEN** uma nova sessão de pareamento temporária é gerada com PIN de 6 dígitos e validade de 2 minutos
- **AND** um QR Code é renderizado em tela contendo o JSON codificado com `id`, `name`, `addrs` (IP:Porta) e `pin`

### Requirement: QR Code Scanning and Decoding on Mobile
The system SHALL / O aplicativo mobile OmniDesk DEVE acessar a câmera do smartphone para escanear e decodificar o QR Code exibido pelo computador.

#### Scenario: Leitura bem-sucedida de QR Code válido
- **WHEN** o usuário aponta a câmera do aplicativo para o QR Code de pareamento do OmniDesk
- **THEN** o payload JSON é decodificado e validado com sucesso
- **AND** o aplicativo inicia imediatamente o handshake HTTP com o IP e porta extraídos do QR Code

### Requirement: Automated Mutual Handshake
The system SHALL / O aplicativo mobile e o computador DEVEM concluir a autenticação mútua e troca de tokens criptográficos de forma automática a partir dos dados do QR Code.

#### Scenario: Handshake de pareamento sem intervenção manual
- **WHEN** o aplicativo mobile envia a requisição de pareamento contendo o PIN lido do QR Code para o computador
- **THEN** o computador valida o PIN com a sessão ativa correspondente
- **AND** ambos os nós persistem mutuamente o identificador, nome e token de autenticação em armazenamento seguro
- **AND** o modal no computador e a tela do aplicativo exibem confirmação de pareamento bem-sucedido
