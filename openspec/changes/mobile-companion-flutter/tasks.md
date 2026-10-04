# Tasks: mobile-companion-flutter

## 1. Suporte a QR Code no Desktop (OmniDesk Web Dashboard)
- [x] 1.1 Adicionar botão "Parear Celular (QR Code)" na barra de navegação/dispositivos do dashboard (`web/index.html`).
- [x] 1.2 Integrar renderizador leve de QR Code (SVG/Canvas) em `web/app.js` e desenhar o payload `{ v, id, name, addrs, pin }`.
- [x] 1.3 Implementar polling de confirmação no modal para fechar automaticamente com aviso de sucesso assim que o celular concluir o pareamento.

## 2. Scaffold do Projeto Flutter (`OmniDeskMobile/`)
- [x] 2.1 Criar a estrutura inicial do projeto Flutter no diretório `OmniDeskMobile/` com suporte configurado para iOS e Android.
- [x] 2.2 Configurar dependências no `pubspec.yaml` (`mobile_scanner`, `dio`, `flutter_secure_storage`, `image_picker`, `file_picker`, `path_provider`).
- [x] 2.3 Configurar permissões de Câmera (`NSCameraUsageDescription`), Rede Local (`NSLocalNetworkUsageDescription`) e Galeria no `ios/Runner/Info.plist` e `android/app/src/main/AndroidManifest.xml`.

## 3. Core de Rede, Segurança e Anti-Eco no Mobile
- [x] 3.1 Implementar serviço de armazenamento seguro (`flutter_secure_storage`) para persistir dados do próprio celular (`DeviceID`, `DeviceName`) e a lista de computadores pareados (`TrustedDevice` com `Token`).
- [x] 3.2 Implementar cliente HTTP REST (`Dio`) configurado com timeouts e interceptor para injetar cabeçalhos `X-OmniDesk-Device-ID` e `X-OmniDesk-Token`.
- [x] 3.3 Implementar utilitário de Anti-Eco em memória utilizando SHA-256 e janela de expiração (TTL de 10 segundos).

## 4. Fluxo de Pareamento via QR Code (Mobile)
- [x] 4.1 Criar tela de leitura de QR Code utilizando `mobile_scanner` com mira visual e validação de formato JSON.
- [x] 4.2 Implementar serviço de handshake: disparar requisição para `/api/v1/pair/request` e em seguida confirmar via `/api/v1/pair/confirm` com o PIN extraído do QR Code.
- [x] 4.3 Persistir o computador pareado com sucesso e navegar de volta à tela principal com notificação de confirmação.

## 5. Sincronização de Área de Transferência
- [x] 5.1 Implementar envio de clipboard celular ➔ PC através de ação manual de 1-toque ("Enviar Clipboard") no card do computador.
- [x] 5.2 Implementar sincronização automática opcional acionada pelo evento de ciclo de vida `AppLifecycleState.resumed` (ao focar no app).
- [x] 5.3 Implementar servidor HTTP local no app Flutter para escutar `POST /api/v1/clipboard`, validar anti-eco e atualizar a área de transferência do sistema operacional.

## 6. Transferência de Fotos e Arquivos
- [x] 6.1 Implementar envio de fotos e vídeos da galeria (`image_picker`) em streaming para `/api/v1/files/upload?filename=...` do PC.
- [x] 6.2 Implementar seletor e envio de documentos genéricos (`file_picker`) com monitoramento de progresso de upload.
- [x] 6.3 Implementar recebimento de arquivos no servidor HTTP local do celular: salvar imagens na galeria do aparelho e documentos na pasta de arquivos do app.

## 7. Interface do Usuário (Flutter UI)
- [x] 7.1 Criar a navegação principal (`HomePage`) com `BottomNavigationBar` alternando entre as abas "Computadores" e "Transferências".
- [x] 7.2 Implementar a tela de Computadores: card do status local do smartphone, lista de PCs pareados com badge de status (Online/Offline) e botões de ação rápida.
- [x] 7.3 Implementar a Central de Transferências: feed cronológico com filtros (Todos / Textos / Arquivos), prévia de texto com botão "Copiar Novamente" e cartões de arquivos transferidos.

## 8. Validação e Testes End-to-End
- [x] 8.1 Testar geração de QR Code no desktop e leitura na câmera do smartphone.
- [x] 8.2 Testar envio e recepção de texto/URLs entre iPhone/Android e macOS/Linux/Windows, checando ausência de loops de eco.
- [x] 8.3 Testar transferência de fotos de alta resolução e arquivos PDF em streaming direto na rede local.
