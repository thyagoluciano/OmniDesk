## Why

Hoje o OmniDesk atende exclusivamente computadores desktop (Linux, macOS e Windows), sincronizando clipboard, arquivos e periféricos via rede local sem dependência de nuvem. No entanto, uma das maiores fontes de atrito diário no fluxo de trabalho é a ponte entre **smartphones (iOS / Android)** e os **computadores na mesma mesa**:
- O usuário frequentemente tira fotos, recebe comprovantes ou copia códigos de autenticação / links no celular e precisa recorrer a artifícios incômodos para mandar para o PC (como mandar mensagem para si mesmo no WhatsApp, salvar em rascunhos de e-mail ou sincronizar via nuvens de terceiros).
- A sincronização deve respeitar a premissa fundadora do OmniDesk: **100% P2P na rede local (Wi-Fi), com zero telemetria e zero servidores em nuvem**.

Além disso, sistemas operacionais mobile modernos (Android 10+ e iOS 14+) bloqueiam leitura contínua de clipboard em background. Por isso, a solução mobile deve priorizar a conveniência do **pareamento instantâneo via QR Code** e uma UX orientada a ações intencionais fluidas (Share Sheet nativo do sistema e auto-sync ao abrir o app), sem atrito de digitação.

## What Changes

- Criação de um projeto Flutter independente na pasta `OmniDeskMobile/` (Dart puro), mantendo o código mobile isolado do backend em Go.
- **Pareamento por QR Code**:
  - O painel Web do OmniDesk Desktop (`web/`) ganha uma modal "Parear Celular" que exibe um QR Code contendo as informações da máquina (`id`, `name`, `addrs`, `pin`).
  - O app Flutter escaneia o QR Code usando a câmera e executa o handshake de pareamento via REST de forma automática e instantânea, sem digitação de PIN ou IP.
- **Sincronização de Área de Transferência (Texto & URLs)**:
  - Celular ➔ PC: Envio de 1-toque através do botão "Enviar Clipboard", sincronização automática ao trazer o app para primeiro plano (*on-resume*) e integração com Share Sheet.
  - PC ➔ Celular: Broadcast automático para dispositivos mobile pareados quando o app estiver conectado.
- **Transferência de Mídia e Arquivos**:
  - Celular ➔ PC: Seleção de fotos da galeria ou documentos e streaming direto para o endpoint `/api/v1/files/upload` do desktop.
  - PC ➔ Celular: Recepção de arquivos e salvamento automático de fotos na Galeria e documentos na pasta de arquivos do celular.
- **Interface Mobile (Flutter UI)**:
  - Navegação em abas inferiores (*Bottom Navigation Bar*).
  - **Aba 1 (Computadores)**: Lista de PCs da rede com status online/offline em tempo real, botão de escanear QR Code e botões de ação rápida.
  - **Aba 2 (Transferências)**: Feed histórico com filtros de textos copiados, fotos e arquivos transferidos, com status e ações (copiar, abrir, salvar).

## Capabilities

### New Capabilities

- `mobile-qr-pairing`: Protocolo de pareamento com codificação JSON no QR Code do Desktop e leitor no Flutter para handshake mútuo via REST sem digitação manual.
- `mobile-clipboard-sync`: Comunicação de clipboard de texto/URLs entre o aplicativo Flutter e os nós desktop do OmniDesk via endpoints HTTP REST com prevenção anti-eco.
- `mobile-media-file-transfer`: Envio e recebimento em streaming de fotos e arquivos entre o smartphone e os computadores da LAN.
- `mobile-ui-dashboard`: Arquitetura de telas do aplicativo Flutter em `OmniDeskMobile/` (Aba de Computadores e Aba de Central de Transferências).

### Modified Capabilities

- `desktop-gui`: Inclusão do botão "Parear Celular" e modal com renderização de QR Code no dashboard web existente (`web/`).

## Impact

- **Código do Desktop (OmniDesk Go)**: Adição de biblioteca JS de QR Code leve (ex: `qrcode.js` ou geração SVG inline) no painel web (`web/index.html` e `web/app.js`). Nenhuma quebra de API no backend Go — todos os endpoints existentes (`/api/v1/pair/*`, `/api/v1/clipboard`, `/api/v1/files/upload`) são mantidos e reutilizados.
- **Novo Projeto (`OmniDeskMobile/`)**: Projeto Flutter completo (Android + iOS), gerenciado por `pubspec.yaml`, com arquitetura modular em Dart puro.
- **Dependências Mobile**: `mobile_scanner` (leitura QR), `nsd` ou `bonsoir` (mDNS), `dio` ou `http` (cliente REST), `shelf` ou `HttpServer` (servidor local), `flutter_secure_storage` (chaves e tokens criptográficos), `image_picker` / `file_picker`.
- **Compatibilidade**: Totalmente retrocompatível; os desktops existentes continuam se comunicando normalmente.
