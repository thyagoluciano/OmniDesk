# OmniDesk

<p align="center">
  <img src="assets/omnidesk.svg" alt="OmniDesk Logo" width="96" height="96">
</p>

<p align="center">
  <strong>Sincronização P2P de Área de Transferência e Compartilhamento de Arquivos Local</strong><br>
  <em>Cross-Platform Local P2P Clipboard Sync & File Sharing for Linux, macOS & Windows</em><br>
  <em>Sincronización P2P de Portapapeles y Transferencia de Archivos Local</em>
</p>

<p align="center">
  <a href="#-português">🇧🇷 Português</a> &nbsp;•&nbsp;
  <a href="#-english">🇺🇸 English</a> &nbsp;•&nbsp;
  <a href="#-español">🇪🇸 Español</a>
</p>

---

## 🇧🇷 Português

### O que é o OmniDesk?

O **OmniDesk** é uma aplicação multiplataforma (Linux, macOS e Windows) de código aberto projetada para sincronizar a área de transferência (*clipboard*) e transferir arquivos em tempo real entre seus próprios computadores na rede local (LAN / Wi-Fi).

Diferente de soluções baseadas na nuvem, o OmniDesk opera **100% ponto a ponto (P2P)**:
- **Zero Servidores Externos / Zero Nuvem**: Seus dados e arquivos nunca saem da sua rede local.
- **Privacidade Total**: Sem rastreamento, sem contas, sem telemetria.
- **Velocidade Máxima**: Transferências diretas limitadas apenas pela velocidade do seu roteador ou cabo de rede.
- **Pareamento Seguro**: Conexão entre máquinas autenticada por **PIN de 6 dígitos** temporário e tokens criptográficos individuais.

---

### Principais Recursos

- 📋 **Sincronização Contínua de Clipboard**: Copie um texto, link ou código em um computador e cole instantaneamente no outro (com proteção contra loops infinitos de cópia).
- 📁 **Transferência Direta de Arquivos**: Envie arquivos de qualquer tamanho arrastando e soltando (*drag & drop*) no navegador ou através da linha de comando (`omnidesk send`).
- 🔍 **Descoberta Automática de Rede**: Detecção imediata de computadores vizinhos utilizando mDNS (ZeroConf) e varredura inteligente de sub-rede.
- 💻 **Dashboard Web Intuitivo**: Interface gráfica moderna acessível em `http://127.0.0.1:24850/ui/` ou pelo comando `omnidesk gui`.
- 🛡️ **Gerenciamento de Dispositivos Confiáveis**: Visualize dispositivos conectados e remova pares autorizados com um clique.
- 🔔 **Notificações Nativas e Ícone de Bandeja**: Suporte a bandeja do sistema (*System Tray*) no Linux, macOS e Windows, além de notificações na área de trabalho.
- ⚙️ **Instalação como Serviço do Sistema**: Inicia automaticamente em segundo plano no login do usuário sem necessidade de permissões de administrador (`root`/`sudo`).

---

### Instalação e Execução

#### Pré-requisitos
- **Go 1.20+** (caso deseje compilar a partir do código-fonte)
- Linux com X11 ou Wayland, macOS 11.0+ ou Windows 10/11.

---

#### 🐧 Linux (Ubuntu, Debian, Fedora, Arch)

1. **Compilar o binário:**
   ```bash
   go build -ldflags="-s -w" -o omnidesk ./cmd/omnidesk
   ```

2. **Instalar no sistema com inicialização automática:**
   ```bash
   ./omnidesk install
   ```
   *O instalador copia o binário para `~/.local/bin/omnidesk`, adiciona os ícones do sistema, cria o atalho `.desktop` e registra o serviço de usuário no systemd (`omnidesk.service`).*

3. **Gerenciar o serviço:**
   ```bash
   # Reiniciar o serviço
   systemctl --user restart omnidesk.service

   # Ver status e logs
   systemctl --user status omnidesk.service

   # Desinstalar
   omnidesk uninstall
   ```

4. **Gerar pacote `.deb` para instalação:**
   ```bash
   ./scripts/build-deb.sh
   sudo dpkg -i dist/omnidesk_0.1.0_amd64.deb
   ```

---

#### 🍏 macOS (Apple Silicon M1/M2/M3/M4 & Intel)

1. **Gerar o aplicativo `OmniDesk.app`:**
   ```bash
   ./scripts/build-macos-app.sh
   ```

2. **Instalar no macOS:**
   - Execute o script automático:
     ```bash
     ./scripts/install-mac.command
     ```
   - Ou instale via linha de comando:
     ```bash
     ./omnidesk install
     ```
   *Copia para `/Applications/OmniDesk.app`, remove bloqueios do Gatekeeper (`xattr -cr`), aplica assinatura ad-hoc e configura o LaunchAgent para inicialização com o login.*

3. **Gerar imagem de disco `.dmg` instalável:**
   ```bash
   ./scripts/build-dmg.sh
   ```

---

#### 🪟 Windows (10 / 11)

1. **Compilar para Windows:**
   ```bash
   # No Linux/macOS (Cross-compilation):
   GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui -s -w" -o omnidesk.exe ./cmd/omnidesk

   # Ou execute o script de build:
   ./scripts/build-windows.sh
   ```

2. **Instalar no Windows:**
   - No terminal (PowerShell ou Prompt de Comando):
     ```cmd
     .\omnidesk.exe install
     ```
   - Ou utilize o assistente de instalação gerado pelo script NSIS (`dist\OmniDesk-Setup-0.1.0-x64.exe`).

3. **Desinstalar:**
   ```cmd
   .\omnidesk.exe uninstall
   ```

---

### Como Usar

#### 1. Pareamento entre dois computadores

Para que dois computadores conversem, eles precisam ser pareados uma única vez.

**Via Interface Gráfica (Web Dashboard):**
1. Abra o painel no computador digitando `omnidesk gui` ou acessando `http://127.0.0.1:24850/ui/`.
2. Na seção **Dispositivos Descobertos**, clique no botão **Parear** ao lado do outro computador (ou clique em **Aprovar PIN** no topo).
3. Um **PIN de 6 dígitos** será exibido. Confirme o mesmo PIN na tela do outro dispositivo para autorizar a conexão.

**Via Linha de Comando (Terminal):**
- No **Computador A** (solicitante):
  ```bash
  omnidesk pair 192.168.1.50
  # ou pelo nome:
  omnidesk pair MacBook-Pro
  ```
- No **Computador B** (autorizador):
  ```bash
  # Digite o PIN exibido no Computador A:
  omnidesk pair 481920
  
  # Ou simplesmente execute para ver solicitações pendentes:
  omnidesk pair
  ```

---

#### 2. Sincronização da Área de Transferência
- Uma vez pareados, qualquer texto copiado (`Ctrl+C` / `Cmd+C`) em uma máquina é enviado em frações de segundo para os outros computadores pareados na mesma rede.
- Você pode desativar temporariamente a sincronização a qualquer momento usando a chave seletora no Dashboard Web.

---

#### 3. Envio de Arquivos

- **Pelo Dashboard Web**:
  Basta arrastar arquivos e soltá-los sobre o card do dispositivo desejado ou clicar no botão **Enviar Arquivo** do dispositivo.
- **Pela Linha de Comando**:
  ```bash
  omnidesk send /caminho/do/documento.pdf "Nome-Do-Dispositivo"
  ```
- **Onde ficam os arquivos recebidos?**
  Todos os arquivos baixados são armazenados na pasta:
  - Linux / macOS: `~/Downloads/OmniDesk`
  - Windows: `%USERPROFILE%\Downloads\OmniDesk`

---

### Referência de Comandos da CLI

| Comando | Descrição |
| :--- | :--- |
| `omnidesk` | Inicia o agente com ícone na bandeja do sistema. |
| `omnidesk daemon` | Inicia o serviço em segundo plano (`--headless` para rodar sem interface/bandeja). |
| `omnidesk gui` | Abre o Painel de Controle Web no navegador padrão. |
| `omnidesk status` | Exibe o status do daemon, porta, nó local e lista de dispositivos pareados. |
| `omnidesk devices` | Descobre e lista nós OmniDesk ativos na rede local (LAN). |
| `omnidesk pair <alvo>` | Inicia a solicitação de pareamento com outro nó por IP ou nome. |
| `omnidesk pair <PIN>` | Autoriza uma solicitação de pareamento recebida utilizando o PIN de 6 dígitos. |
| `omnidesk send <arquivo> <alvo>` | Envia um arquivo diretamente para um dispositivo pareado. |
| `omnidesk install` | Configura o OmniDesk no sistema com inicialização automática no login. |
| `omnidesk uninstall` | Remove os binários, atalhos e serviços de inicialização. |
| `omnidesk version` | Exibe a versão instalada. |

---

## 🇺🇸 English

### What is OmniDesk?

**OmniDesk** is an open-source cross-platform application (Linux, macOS, and Windows) designed for real-time clipboard synchronization and secure file transfers between your devices on a local network (LAN / Wi-Fi).

Unlike cloud-dependent tools, OmniDesk operates **100% peer-to-peer (P2P)**:
- **Zero Cloud / Zero Relays**: Your files, links, and clipboard data never leave your local network.
- **Complete Privacy**: No accounts, no user tracking, no telemetry.
- **Blazing Fast**: Direct connections running at the maximum speed of your local Wi-Fi or gigabit Ethernet.
- **Secure Pairing**: Device authentication backed by a transient **6-digit PIN** and unique cryptographic tokens.

---

### Key Features

- 📋 **Seamless Clipboard Sync**: Copy text, links, or snippets on one machine and paste them instantly on your other machines (equipped with echo-loop prevention).
- 📁 **Direct File Transfer**: Drag-and-drop files of any size directly into the Web Dashboard or send them via CLI (`omnidesk send`).
- 🔍 **Instant Peer Discovery**: Automatic detection of neighboring devices using mDNS (ZeroConf) combined with smart subnet scanning.
- 💻 **Modern Web Dashboard**: Clean control interface served locally at `http://127.0.0.1:24850/ui/` or launched with `omnidesk gui`.
- 🛡️ **Trusted Devices Management**: View connected peers and unpair unauthorized devices with a single click.
- 🔔 **Native System Tray & Desktop Notifications**: Background integration with menu bar / system tray across Linux, macOS, and Windows.
- ⚙️ **User-Space Autostart**: Runs automatically on user login via systemd user services (Linux), LaunchAgent (macOS), or Registry Run (Windows) without requiring `root` or administrator privileges.

---

### Installation & Getting Started

#### Requirements
- **Go 1.20+** (if compiling from source)
- Linux (X11 or Wayland), macOS 11.0+, or Windows 10/11.

---

#### 🐧 Linux (Ubuntu, Debian, Fedora, Arch)

1. **Build the binary:**
   ```bash
   go build -ldflags="-s -w" -o omnidesk ./cmd/omnidesk
   ```

2. **Install with autostart:**
   ```bash
   ./omnidesk install
   ```
   *Copies the binary to `~/.local/bin/omnidesk`, installs icons, registers `.desktop` launcher, and configures the systemd user service (`omnidesk.service`).*

3. **Manage the service:**
   ```bash
   # Restart the service
   systemctl --user restart omnidesk.service

   # Check status and logs
   systemctl --user status omnidesk.service

   # Uninstall
   omnidesk uninstall
   ```

4. **Build `.deb` package:**
   ```bash
   ./scripts/build-deb.sh
   sudo dpkg -i dist/omnidesk_0.1.0_amd64.deb
   ```

---

#### 🍏 macOS (Apple Silicon M1/M2/M3/M4 & Intel)

1. **Build `OmniDesk.app`:**
   ```bash
   ./scripts/build-macos-app.sh
   ```

2. **Install on macOS:**
   - Run the helper script:
     ```bash
     ./scripts/install-mac.command
     ```
   - Or run from terminal:
     ```bash
     ./omnidesk install
     ```
   *Installs to `/Applications/OmniDesk.app`, strips Gatekeeper quarantine flags (`xattr -cr`), applies local ad-hoc code signature, and configures user LaunchAgent.*

3. **Build `.dmg` installer:**
   ```bash
   ./scripts/build-dmg.sh
   ```

---

#### 🪟 Windows (10 / 11)

1. **Compile for Windows:**
   ```bash
   # Cross-compile from Linux/macOS:
   GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui -s -w" -o omnidesk.exe ./cmd/omnidesk

   # Or execute the build script:
   ./scripts/build-windows.sh
   ```

2. **Install on Windows:**
   - In terminal (PowerShell or Command Prompt):
     ```cmd
     .\omnidesk.exe install
     ```
   - Or run the generated NSIS Setup Wizard (`dist\OmniDesk-Setup-0.1.0-x64.exe`).

3. **Uninstall:**
   ```cmd
   .\omnidesk.exe uninstall
   ```

---

### How to Use

#### 1. Pairing Two Computers

Devices only need to be paired once.

**Via Web Dashboard:**
1. Open the dashboard on your machine by running `omnidesk gui` or navigating to `http://127.0.0.1:24850/ui/`.
2. Under **Discovered Devices**, click **Pair** on the target computer (or use **Approve PIN** at the top).
3. A **6-digit PIN** will be displayed. Confirm the PIN on the other device to authorize pairing.

**Via Terminal (CLI):**
- On **Computer A** (requester):
  ```bash
  omnidesk pair 192.168.1.50
  # or by device name:
  omnidesk pair MacBook-Pro
  ```
- On **Computer B** (approver):
  ```bash
  # Enter the 6-digit PIN displayed on Computer A:
  omnidesk pair 481920

  # Or run interactively to view pending pairing requests:
  omnidesk pair
  ```

---

#### 2. Clipboard Synchronization
- Once paired, any text copied (`Ctrl+C` / `Cmd+C`) on one machine will be available to paste immediately on your other paired machines.
- Toggle clipboard synchronization on or off at any time via the switch in the Web Dashboard.

---

#### 3. Sending Files

- **Via Web Dashboard**:
  Drag and drop files directly onto the paired device card or click **Send File**.
- **Via Command Line**:
  ```bash
  omnidesk send /path/to/archive.zip "Target-Device-Name"
  ```
- **Inbound Download Folder**:
  Received files are automatically saved to:
  - Linux / macOS: `~/Downloads/OmniDesk`
  - Windows: `%USERPROFILE%\Downloads\OmniDesk`

---

### CLI Command Reference

| Command | Description |
| :--- | :--- |
| `omnidesk` | Launches the agent with system tray integration. |
| `omnidesk daemon` | Runs the agent in background (`--headless` to disable tray icon). |
| `omnidesk gui` | Opens the local Web Dashboard in your default browser. |
| `omnidesk status` | Displays daemon state, listening port, local ID, and trusted peers. |
| `omnidesk devices` | Scans and lists OmniDesk nodes active on your local network (LAN). |
| `omnidesk pair <target>` | Initiates a pairing request to another node by IP or hostname. |
| `omnidesk pair <PIN>` | Approves an incoming pairing request using the 6-digit PIN. |
| `omnidesk send <file> <target>` | Sends a file directly to a trusted device. |
| `omnidesk install` | Installs the app into user environment with autostart enabled. |
| `omnidesk uninstall` | Cleans up binaries, shortcuts, and autostart services. |
| `omnidesk version` | Displays current OmniDesk version. |

---

## 🇪🇸 Español

### ¿Qué es OmniDesk?

**OmniDesk** es una aplicación multiplataforma (Linux, macOS y Windows) de código abierto diseñada para sincronizar el portapapeles (*clipboard*) y transferir archivos en tiempo real entre tus propios ordenadores dentro de la red local (LAN / Wi-Fi).

A diferencia de las soluciones basadas en la nube, OmniDesk opera **100% punto a punto (P2P)**:
- **Cero Servidores Externos / Cero Nube**: Tus datos y archivos nunca salen de tu red local.
- **Privacidad Absoluta**: Sin cuentas, sin rastreo de usuarios y sin telemetría.
- **Máxima Velocidad**: Transferencias directas limitadas únicamente por la velocidad de tu router Wi-Fi o cable Ethernet.
- **Emparejamiento Seguro**: Autenticación protegida mediante un **código PIN temporal de 6 dígitos** y tokens criptográficos individuales.

---

### Características Principales

- 📋 **Sincronización Continua de Portapapeles**: Copia texto, enlaces o código en un ordenador y pégalo instantáneamente en el otro (con protección integrada contra bucles de copia).
- 📁 **Transferencia Directa de Archivos**: Envía archivos de cualquier tamaño arrastrando y soltando (*drag & drop*) en el panel web o desde la terminal (`omnidesk send`).
- 🔍 **Detección Automática en Red**: Descubrimiento instantáneo de dispositivos vecinos mediante mDNS (ZeroConf) y sondeo inteligente de subred.
- 💻 **Panel Web Intuitivo**: Interfaz gráfica moderna disponible en `http://127.0.0.1:24850/ui/` o mediante el comando `omnidesk gui`.
- 🛡️ **Gestión de Dispositivos de Confianza**: Consulta dispositivos vinculados y desempareja equipos con un solo clic.
- 🔔 **Icono en la Bandeja del Sistema y Notificaciones**: Integración nativa con la barra de menú/bandeja (*System Tray*) en Linux, macOS y Windows.
- ⚙️ **Inicio Automático en Espacio de Usuario**: Se inicia automáticamente con tu sesión de usuario sin requerir permisos de administrador (`root`/`sudo`).

---

### Instalación y Puesta en Marcha

#### Requisitos
- **Go 1.20+** (en caso de compilar desde el código fuente)
- Linux (X11 o Wayland), macOS 11.0+ o Windows 10/11.

---

#### 🐧 Linux (Ubuntu, Debian, Fedora, Arch)

1. **Compilar el binario:**
   ```bash
   go build -ldflags="-s -w" -o omnidesk ./cmd/omnidesk
   ```

2. **Instalar en el sistema con inicio automático:**
   ```bash
   ./omnidesk install
   ```
   *Copia el ejecutable en `~/.local/bin/omnidesk`, instala los iconos, crea el acceso directo `.desktop` y registra el servicio de usuario en systemd (`omnidesk.service`).*

3. **Administrar el servicio:**
   ```bash
   # Reiniciar servicio
   systemctl --user restart omnidesk.service

   # Consultar estado y registros
   systemctl --user status omnidesk.service

   # Desinstalar
   omnidesk uninstall
   ```

4. **Generar paquete `.deb`:**
   ```bash
   ./scripts/build-deb.sh
   sudo dpkg -i dist/omnidesk_0.1.0_amd64.deb
   ```

---

#### 🍏 macOS (Apple Silicon M1/M2/M3/M4 e Intel)

1. **Generar la aplicación `OmniDesk.app`:**
   ```bash
   ./scripts/build-macos-app.sh
   ```

2. **Instalar en macOS:**
   - Ejecuta el asistente interactivo:
     ```bash
     ./scripts/install-mac.command
     ```
   - O instala por terminal:
     ```bash
     ./omnidesk install
     ```
   *Copia la aplicación a `/Applications/OmniDesk.app`, elimina las restricciones de Gatekeeper (`xattr -cr`), aplica la firma ad-hoc local y configura el LaunchAgent de inicio.*

3. **Crear instalador de imagen de disco `.dmg`:**
   ```bash
   ./scripts/build-dmg.sh
   ```

---

#### 🪟 Windows (10 / 11)

1. **Compilar para Windows:**
   ```bash
   # Compilación cruzada desde Linux/macOS:
   GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui -s -w" -o omnidesk.exe ./cmd/omnidesk

   # O utiliza el script de construcción:
   ./scripts/build-windows.sh
   ```

2. **Instalar en Windows:**
   - En la consola (PowerShell o Símbolo del Sistema):
     ```cmd
     .\omnidesk.exe install
     ```
   - O ejecuta el instalador generado por NSIS (`dist\OmniDesk-Setup-0.1.0-x64.exe`).

3. **Desinstalar:**
   ```cmd
   .\omnidesk.exe uninstall
   ```

---

### Cómo Utilizar

#### 1. Emparejamiento entre dos dispositivos

Los dispositivos solo necesitan emparejarse una única vez.

**Desde el Panel Web:**
1. Abre el panel ejecutando `omnidesk gui` o accediendo a `http://127.0.0.1:24850/ui/`.
2. En la sección **Dispositivos Descubiertos**, haz clic en **Emparejar** junto al dispositivo deseado (o haz clic en **Aprobar PIN**).
3. Se generará un **código PIN de 6 dígitos**. Introduce y confirma dicho PIN en el otro ordenador para completar la vinculación.

**Desde la Línea de Comandos (CLI):**
- En el **Ordenador A** (solicitante):
  ```bash
  omnidesk pair 192.168.1.50
  # o mediante el nombre:
  omnidesk pair MacBook-Pro
  ```
- En el **Ordenador B** (autorizador):
  ```bash
  # Introduce el PIN mostrado en el Ordenador A:
  omnidesk pair 481920

  # O ejecuta sin parámetros para ver solicitudes entrantes:
  omnidesk pair
  ```

---

#### 2. Sincronización del Portapapeles
- Tras el emparejamiento, cualquier texto copiado (`Ctrl+C` / `Cmd+C`) se transmitirá de forma instantánea al resto de tus dispositivos vinculados.
- Puedes activar o desactivar la sincronización en cualquier momento desde el interruptor del Panel Web.

---

#### 3. Envío de Archivos

- **Desde el Panel Web**:
  Arrastra los archivos y suéltalos directamente sobre la tarjeta del dispositivo conectado, o pulsa en **Enviar Archivo**.
- **Desde la Terminal**:
  ```bash
  omnidesk send /ruta/al/archivo.zip "Nombre-Del-Dispositivo"
  ```
- **Carpeta de Descargas**:
  Los archivos recibidos se guardan automáticamente en:
  - Linux / macOS: `~/Downloads/OmniDesk`
  - Windows: `%USERPROFILE%\Downloads\OmniDesk`

---

### Referencia de Comandos CLI

| Comando | Descripción |
| :--- | :--- |
| `omnidesk` | Inicia el agente con icono en la bandeja del sistema. |
| `omnidesk daemon` | Inicia el proceso en segundo plano (`--headless` para omitir la bandeja). |
| `omnidesk gui` | Abre el Panel de Control Web en el navegador predeterminado. |
| `omnidesk status` | Muestra el estado del daemon, puerto, identificador local y equipos vinculados. |
| `omnidesk devices` | Explora y lista los nodos OmniDesk visibles en la red local (LAN). |
| `omnidesk pair <objetivo>` | Envía una solicitud de emparejamiento por IP o nombre. |
| `omnidesk pair <PIN>` | Autoriza una solicitud entrante mediante el PIN de 6 dígitos. |
| `omnidesk send <archivo> <objetivo>` | Envía un archivo directamente a un dispositivo emparejado. |
| `omnidesk install` | Configura OmniDesk en el sistema con inicio automático en el login. |
| `omnidesk uninstall` | Elimina ejecutables, accesos directos y servicios de autoinicio. |
| `omnidesk version` | Muestra la versión actual de OmniDesk. |

---

## Licença / License / Licencia

MIT License &copy; 2026 Thyago Luciano
