# Design: OmniDesk Mobile Companion (Flutter)

## Visão Geral

O objetivo deste design é estender o OmniDesk para smartphones (iOS e Android) através de um aplicativo em Flutter/Dart puro localizado no diretório irmão `OmniDeskMobile/`.

O aplicativo se integra ao protocolo de rede existente do OmniDesk (mDNS + HTTP REST), eliminando a necessidade de servidores em nuvem ou contas de usuário.

---

## 1. Decisões Arquiteturais Fundamentais

### D1: Flutter com Dart Puro vs. Gomobile
* **Decisão:** Desenvolver o cliente inteiramente em Dart puro no Flutter.
* **Justificativa:** 
  - A API do OmniDesk é HTTP REST limpa e sem estado complexo.
  - Gomobile adiciona grande complexidade ao pipeline de build no iOS/Android, aumenta o tamanho do binário e tem gerenciamento de goroutines em background problemático no iOS.
  - Dart puro permite usar diretamente os plugins nativos do Flutter para câmera (QR Code), galeria de fotos, seletor de arquivos e armazenamento seguro (`Keychain` / `EncryptedSharedPreferences`).

### D2: Modelo de Conectividade P2P no Mobile
* **Decisão:** Modelo Híbrido Cliente/Servidor Local.
  - O celular atua primariamente como **cliente REST** ativo quando o usuário interage (disparando clipboards e arquivos para os computadores).
  - O celular executa um `HttpServer` leve em porta local (ex.: `24851`) enquanto estiver em primeiro plano ou com serviço ativo na LAN para receber arquivos e clipboards enviados pelos computadores.
  - Descoberta mDNS mútua: o celular busca computadores que anunciam `_omnidesk._tcp` e pode anunciar a si mesmo na rede local.

---

## 2. Protocolo de Pareamento por QR Code

```
  Desktop (Web UI /ui/)                           Mobile (Flutter App)
  ─────────────────────                           ────────────────────
   [Clica "Parear Celular"]
   Gera PIN (ex: 839102)
   Renderiza QR Code:
   {
     "v": 1,
     "id": "desk-mac-c1f9",
     "name": "MacBook Pro",
     "addrs": ["192.168.1.15:24850"],
     "pin": "839102"
   }
              │
              │  ◄── 1. Câmera escaneia QR Code
              ▼
                                                Decodifica JSON e lê PIN/IPs
                                                Gera ID e Nome do Celular
                                                POST /api/v1/pair/request
                                                Body: {
                                                  "requester_id": "mob-iphone-7a3b",
                                                  "requester_name": "iPhone 15 Pro",
                                                  "requester_addr": "192.168.1.42:24851"
                                                }
                                                       │
              ◄────────────────────────────────────────┘
   Cria sessão de pareamento
   Responde com session_id e PIN
              │
              └────────────────────────────────────────►
                                                POST /api/v1/pair/confirm
                                                Body: {
                                                  "session_id": "...",
                                                  "pin": "839102",
                                                  "requester_id": "mob-iphone-7a3b"
                                                }
                                                       │
              ◄────────────────────────────────────────┘
   Valida PIN e emite AuthToken
   Salva celular em TrustedDevice
   Responde { "success": true, "auth_token": "..." }
              │
              └────────────────────────────────────────►
                                                Salva Desktop + Token no
                                                flutter_secure_storage
                                                Feedback visual: "Pareado com Sucesso!"
```

### Formato do Payload do QR Code
```json
{
  "v": 1,
  "id": "node-unique-id",
  "name": "Nome Amigável do Computador",
  "addrs": ["192.168.1.15:24850"],
  "pin": "123456"
}
```

---

## 3. Sincronização de Clipboard

### Anti-Eco e Cache de Hashes
Assim como no motor Go (`internal/clipboard/clipboard.go`), o app mobile mantém um cache local em memória dos últimos hashes SHA-256 de textos recebidos e enviados com TTL de 10 segundos para evitar loops infinitos.

### Ciclo de Envio (Celular ➔ PC)
1. **Manual / 1-Toque:** Botão "Enviar Clipboard" no card do computador.
2. **Ao abrir o app (*On App Resume*):** O Flutter detecta retorno ao primeiro plano (`AppLifecycleState.resumed`), verifica se o clipboard do SO mudou e oferece envio ou sincroniza automaticamente (configurável).
3. **Chamada REST:**
   ```http
   POST http://<pc-addr>/api/v1/clipboard HTTP/1.1
   X-OmniDesk-Device-ID: mob-iphone-7a3b
   X-OmniDesk-Token: <token-de-pareamento>
   Content-Type: application/json

   {
     "sender_id": "mob-iphone-7a3b",
     "text": "Conteúdo copiado..."
   }
   ```

---

## 4. Transferência de Fotos e Arquivos

### 4.1 Celular ➔ Computador
* O app Flutter usa `image_picker` (para selecionar fotos/vídeos da Galeria) ou `file_picker` (para documentos e PDFs).
* O envio é realizado via streaming HTTP direto no body (sem multipart desnecessário, correspondendo exatamente ao leitor `io.Copy(out, r.Body)` do OmniDesk Go):
   ```http
   POST http://<pc-addr>/api/v1/files/upload?filename=foto.jpg HTTP/1.1
   X-OmniDesk-Device-ID: mob-iphone-7a3b
   X-OmniDesk-Token: <token-de-pareamento>
   Content-Type: application/octet-stream

   <bytes do arquivo>
   ```

### 4.2 Computador ➔ Celular
* O app mobile roda um `HttpServer` ouvindo na porta de escuta local (ex.: `24851`).
* Ao receber um arquivo em `/api/v1/files/upload`:
  - Se for extensão de imagem (`.jpg`, `.png`, `.heic`, etc.), o app salva no álbum de fotos local do smartphone via plugin de galeria.
  - Se for documento genérico (`.pdf`, `.zip`, `.doc`), salva na pasta `Documents` acessível pelo app e emite notificação local.

---

## 5. Estrutura de Telas do App Flutter

```
lib/
├── main.dart
├── core/
│   ├── network/          # HttpClient, HttpServer local, mDNS discovery
│   ├── storage/          # SecureStorage (tokens e dispositivos pareados)
│   └── anti_echo/        # SHA-256 hash deduplication
├── features/
│   ├── pairing/          # Leitor QR Code (mobile_scanner), handshake service
│   ├── devices/          # Lista e status dos computadores na LAN
│   ├── clipboard/        # Leitura e escrita no Clipboard do SO
│   └── transfers/        # Envio/recebimento de fotos e arquivos, feed histórico
└── ui/
    ├── home_page.dart    # BottomNavigationBar (Computadores / Transferências)
    ├── screens/
    │   ├── devices_tab.dart
    │   ├── transfers_tab.dart
    │   └── qr_scanner_screen.dart
    └── widgets/
        ├── computer_card.dart
        └── transfer_feed_item.dart
```

---

## 6. Alterações no Desktop (OmniDesk Go + Web)

* No `web/index.html` e `web/app.js`:
  - Inclusão do botão **"Parear Celular (QR Code)"** na barra superior ou na seção de dispositivos.
  - Modal com gerador de QR Code:
    - O dashboard chama o backend local para iniciar uma sessão de pareamento (`/api/v1/devices/pair`) que já gera um PIN.
    - O frontend web desenha o QR Code em tela com o IP da máquina, porta e o PIN gerado.
    - Quando o celular confirma na rede, o modal fecha automaticamente com indicação de sucesso.
