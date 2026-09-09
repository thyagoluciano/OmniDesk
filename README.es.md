# OmniDesk

<p align="center">
  <img src="assets/omnidesk.svg" alt="OmniDesk Logo" width="100" height="100">
</p>

<p align="center">
  <strong>Sincronización P2P de Portapapeles y Transferencia de Archivos en Red Local</strong><br>
  <em>Cross-Platform Local P2P Clipboard Sync & File Sharing for Linux, macOS & Windows</em>
</p>

<p align="center">
  <a href="https://github.com/thyagoluciano/OmniDesk/actions/workflows/ci.yml"><img src="https://github.com/thyagoluciano/OmniDesk/actions/workflows/ci.yml/badge.svg" alt="Estado CI"></a>
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go" alt="Versión Go"></a>
  <a href="#"><img src="https://img.shields.io/badge/Plataformas-Linux%20%7C%20macOS%20%7C%20Windows-blue" alt="Plataformas"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/licencia-MIT-green.svg" alt="Licencia"></a>
  <a href="ROADMAP.md"><img src="https://img.shields.io/badge/roadmap-activo-success.svg" alt="Roadmap"></a>
  <a href="CONTRIBUTING.md"><img src="https://img.shields.io/badge/PRs-bienvenidas-brightgreen.svg" alt="Contribuciones"></a>
</p>

<p align="center">
  <a href="README.md">🇧🇷 Português</a> &nbsp;•&nbsp;
  <a href="README.en.md">🇺🇸 English</a> &nbsp;•&nbsp;
  <strong>🇪🇸 Español</strong>
</p>

<p align="center">
  <a href="#-qué-es-omnidesk">Qué es</a> &nbsp;•&nbsp;
  <a href="#-instalación-rápida-instaladores">Instaladores</a> &nbsp;•&nbsp;
  <a href="#-cómo-utilizar">Cómo Usar</a> &nbsp;•&nbsp;
  <a href="#-modo-cli-línea-de-comandos">Modo CLI</a> &nbsp;•&nbsp;
  <a href="#-compilación-desde-el-código-fuente">Compilación</a> &nbsp;•&nbsp;
  <a href="#-comunidad-y-contribución">Comunidad</a>
</p>

---

## 💡 ¿Qué es OmniDesk?

**OmniDesk** es una herramienta de productividad multiplataforma (Linux, macOS y Windows) de código abierto diseñada para sincronizar el portapapeles (*clipboard*) y transferir archivos al instante entre tus propios ordenadores dentro de la misma red local (Wi-Fi o cable Ethernet), sin depender de la nube.

A diferencia de las soluciones basadas en servidores externos, OmniDesk opera **100% punto a punto (P2P)**:

- 🔒 **Privacidad Total y Cero Nube**: Ningún dato, texto o archivo sale de tu red local. Sin registro de cuentas, sin telemetría y sin rastreo de usuarios.
- ⚡ **Velocidad Máxima Local**: Transferencias directas aprovechando todo el ancho de banda de tu router Wi-Fi o red Ethernet.
- 🛡️ **Emparejamiento Sencillo y Seguro**: Conexión entre dispositivos protegida mediante un **código PIN de 6 dígitos** temporal y tokens criptográficos únicos.
- 🖥️ **Panel Web Moderno y Bandeja del Sistema**: Gestiona todo a través de una interfaz web intuitiva o desde el icono en la bandeja del sistema (*System Tray*).

---

## ✨ Características Principales

- 📋 **Sincronización Continua de Portapapeles**: Copia texto, enlaces o código en un ordenador y pégalo inmediatamente en el otro (con prevención de bucles infinitos de copia).
- 📁 **Transferencia Directa de Archivos**: Envía archivos de cualquier tamaño arrastrando y soltando (*drag & drop*) en el navegador o mediante la consola.
- 🔍 **Detección Automática en Red**: Descubrimiento instantáneo de ordenadores vecinos mediante mDNS (ZeroConf) y sondeo inteligente de subred.
- 💻 **Panel Web Intuitivo**: Interfaz gráfica accesible en `http://127.0.0.1:24850/ui/` o mediante el comando `omnidesk gui`.
- 🔔 **Notificaciones Nativas y System Tray**: Integración con la barra de tareas / barra de menú en Linux, macOS y Windows.
- ⚙️ **Inicio Automático con el Sistema**: Se ejecuta en segundo plano al iniciar sesión sin requerir permisos de administrador (`root`/`sudo`).

---

## 📦 Instalación Rápida (Instaladores)

Recomendamos utilizar los paquetes precompilados e instaladores listos para tu sistema operativo:

### 🍏 macOS (Apple Silicon M1/M2/M3/M4 e Intel)

- **Instalador de Imagen de Disco (`.dmg`)**:
  1. Descarga el archivo `OmniDesk.dmg` desde la página de [Releases](https://github.com/thyagoluciano/OmniDesk/releases).
  2. Abre la imagen de disco y arrastra **OmniDesk** a tu carpeta de **Aplicaciones** (`/Applications`).
- **Asistente Automático por Script**:
  - Si clonaste o descargaste el repositorio:
    ```bash
    ./scripts/install-mac.command
    ```
  *Copia la app a `/Applications`, elimina las restricciones de Gatekeeper (`xattr -cr`) y configura el LaunchAgent de autoinicio.*

---

### 🐧 Linux (Ubuntu, Debian, Fedora, Arch)

- **Paquete Debian / Ubuntu (`.deb`)**:
  ```bash
  # Descarga el archivo .deb desde Releases e instálalo:
  sudo dpkg -i omnidesk_0.1.0_amd64.deb
  ```
  *Instala el binario en `/usr/bin/omnidesk`, los iconos, el acceso directo `.desktop` y el servicio de usuario systemd.*

- **Instalación Directa en Espacio de Usuario**:
  Si ya descargaste el binario:
  ```bash
  ./omnidesk install
  ```
  *Instala en `~/.local/bin`, registra el acceso directo y activa el servicio systemd (`omnidesk.service`).*

---

### 🪟 Windows (10 / 11)

- **Asistente de Instalación (`.exe`)**:
  1. Descarga `OmniDesk-Setup-0.1.0-x64.exe` desde la página de [Releases](https://github.com/thyagoluciano/OmniDesk/releases).
  2. Ejecuta el asistente de instalación y sigue los pasos en pantalla.
- **Instalación por Terminal (PowerShell / CMD)**:
  ```powershell
  .\omnidesk.exe install
  ```
  *Registra la entrada de inicio automático en el registro de usuario de Windows.*

---

## 🚀 Cómo Utilizar

### 1. Acceder a OmniDesk
Una vez iniciado, OmniDesk permanece activo en la **bandeja del sistema** (*System Tray* / Barra de Menú).

- Haz clic derecho sobre el icono de la bandeja y selecciona **Abrir Panel**.
- O accede en tu navegador a: **`http://127.0.0.1:24850/ui/`**
- O ejecuta en la terminal: `omnidesk gui`

---

### 2. Emparejamiento en 1 Clic

Los equipos solo deben emparejarse una única vez para establecer confianza mutua:

1. Abre el panel en el **Ordenador A** y localiza la sección **Dispositivos Descubiertos**.
2. Haz clic en **Emparejar** junto al dispositivo correspondiente.
3. Se mostrará un **PIN de 6 dígitos** temporal en la pantalla del Ordenador A.
4. En el **Ordenador B**, haz clic en **Aprobar PIN** en la parte superior, introduce los 6 dígitos y confirma.
5. ¡Listo! Ambos dispositivos quedan conectados permanentemente con tokens de seguridad criptográficos.

---

### 3. Sincronización del Portapapeles
- Tras el emparejamiento, cualquier texto, enlace o código copiado (`Ctrl+C` / `Cmd+C`) se replica al instante en el otro ordenador.
- Solo debes pegar (`Ctrl+V` / `Cmd+V`) en el segundo dispositivo.
- Puedes pausar o reactivar la sincronización en cualquier momento desde el interruptor en el Panel Web.

---

### 4. Envío Directo de Archivos
- **En el Panel Web**: Arrastra y suelta (*drag & drop*) cualquier archivo sobre la tarjeta del dispositivo vinculado, o pulsa en **Enviar Archivo**.
- **¿Dónde se guardan las descargas?**
  - **Linux / macOS**: `~/Downloads/OmniDesk`
  - **Windows**: `%USERPROFILE%\Downloads\OmniDesk`

---

## 💻 Modo CLI (Línea de Comandos)

Para usuarios avanzados, automatización de scripts o administración de servidores sin interfaz gráfica (*headless*), OmniDesk ofrece una completa interfaz CLI.

### Referencia de Comandos

| Comando | Descripción |
| :--- | :--- |
| `omnidesk` | Inicia OmniDesk con icono en la bandeja del sistema (*System Tray*). |
| `omnidesk daemon` | Inicia el proceso en segundo plano (`--headless` para omitir la bandeja gráfica). |
| `omnidesk gui` | Abre el Panel de Control Web en el navegador predeterminado. |
| `omnidesk status` | Muestra el estado del daemon, puerto HTTP, ID local y equipos vinculados. |
| `omnidesk devices` | Explora y lista los nodos OmniDesk activos en la red local (LAN). |
| `omnidesk pair <objetivo>` | Envía una solicitud de emparejamiento por IP o nombre de equipo. |
| `omnidesk pair <PIN>` | Autoriza una solicitud entrante mediante el PIN de 6 dígitos. |
| `omnidesk send <archivo> <objetivo>` | Envía un archivo directamente a un dispositivo emparejado. |
| `omnidesk install` | Configura OmniDesk en el sistema con inicio automático en el login. |
| `omnidesk uninstall` | Elimina ejecutables, accesos directos y servicios de autoinicio. |
| `omnidesk version` | Muestra la versión instalada de OmniDesk. |

### Ejemplos Prácticos en la Terminal

```bash
# Comprobar el estado y dispositivos vinculados
omnidesk status

# Descubrir dispositivos cercanos en la red local
omnidesk devices

# Iniciar emparejamiento por nombre o IP del equipo
omnidesk pair MacBook-Pro
# o: omnidesk pair 192.168.1.50

# En el segundo ordenador, autorizar con el PIN mostrado:
omnidesk pair 481920

# Enviar un archivo a un equipo vinculado
omnidesk send ~/Documentos/informe.pdf "MacBook-Pro"
```

### Gestión del Servicio en Linux (systemd)

```bash
# Ver estado del servicio
systemctl --user status omnidesk.service

# Reiniciar el servicio
systemctl --user restart omnidesk.service

# Detener el servicio
systemctl --user stop omnidesk.service
```

---

## 🛠️ Compilación desde el Código Fuente

Si deseas contribuir o compilar OmniDesk por tu cuenta:

### Requisitos Previos
- **Go 1.22+** instalado ([golang.org](https://golang.org/dl/))
- Compilador C/C++ estándar del sistema (para bibliotecas nativas de portapapeles y bandeja)

### Compilación Rápida
```bash
# Clonar el repositorio
git clone https://github.com/thyagoluciano/OmniDesk.git
cd OmniDesk

# Compilar el binario
go build -ldflags="-s -w" -o omnidesk ./cmd/omnidesk
```

### Generación de Paquetes e Instaladores
Disponemos de scripts preparados en el directorio `scripts/`:

```bash
# Generar paquete Debian (.deb):
./scripts/build-deb.sh

# Generar paquete de macOS (.app y .dmg):
./scripts/build-macos-app.sh
./scripts/build-dmg.sh

# Compilar para Windows (Nativo o compilación cruzada):
./scripts/build-windows.sh
```

---

## 🤝 Comunidad y Contribución

¡Agradecemos enormemente cualquier contribución! Para participar en el desarrollo:

- 📖 **[Guía de Contribución](CONTRIBUTING.md)**: Pasos para configurar el entorno y enviar Pull Requests.
- 🗺️ **[Roadmap del Producto](ROADMAP.md)**: Conoce las funcionalidades actuales, en progreso y futuras.
- 🤝 **[Código de Conducta](CODE_OF_CONDUCT.md)**: Nuestras normas de convivencia comunitaria.
- 🛡️ **[Política de Seguridad](SECURITY.md)**: Instrucciones para reportar vulnerabilidades de forma responsable.
- 💬 **[Discusiones en GitHub](https://github.com/thyagoluciano/OmniDesk/discussions)**: Foro para dudas, ideas y sugerencias.
- 🐛 **[Abrir una Issue](https://github.com/thyagoluciano/OmniDesk/issues)**: Informar sobre errores o solicitar funciones.

---

## 📄 Licencia

Este proyecto se distribuye bajo los términos de la licencia [MIT](LICENSE) &copy; 2026 Thyago Luciano.
