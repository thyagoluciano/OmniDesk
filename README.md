# OmniDesk

<p align="center">
  <img src="assets/omnidesk.svg" alt="OmniDesk Logo" width="100" height="100">
</p>

<p align="center">
  <strong>Sincronização P2P de Área de Transferência e Transferência de Arquivos na Rede Local</strong><br>
  <em>Cross-Platform Local P2P Clipboard Sync & File Sharing for Linux, macOS & Windows</em>
</p>

<p align="center">
  <a href="https://github.com/thyagoluciano/OmniDesk/actions/workflows/ci.yml"><img src="https://github.com/thyagoluciano/OmniDesk/actions/workflows/ci.yml/badge.svg" alt="CI Status"></a>
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go" alt="Go Version"></a>
  <a href="#"><img src="https://img.shields.io/badge/Plataformas-Linux%20%7C%20macOS%20%7C%20Windows-blue" alt="Plataformas"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green.svg" alt="Licença"></a>
  <a href="ROADMAP.md"><img src="https://img.shields.io/badge/roadmap-ativo-success.svg" alt="Roadmap"></a>
  <a href="CONTRIBUTING.md"><img src="https://img.shields.io/badge/PRs-bem--vindos-brightgreen.svg" alt="Contribuições"></a>
</p>

<p align="center">
  <strong>🇧🇷 Português</strong> &nbsp;•&nbsp;
  <a href="README.en.md">🇺🇸 English</a> &nbsp;•&nbsp;
  <a href="README.es.md">🇪🇸 Español</a>
</p>

<p align="center">
  <a href="#-o-que-é-o-omnidesk">O que é</a> &nbsp;•&nbsp;
  <a href="#-instalação-rápida-instaladores">Instaladores</a> &nbsp;•&nbsp;
  <a href="#-como-usar">Como Usar</a> &nbsp;•&nbsp;
  <a href="#-modo-cli-linha-de-comando">Modo CLI</a> &nbsp;•&nbsp;
  <a href="#-compilando-do-código-fonte">Compilação</a> &nbsp;•&nbsp;
  <a href="#-comunidade-e-contribuição">Comunidade</a>
</p>

---

## 💡 O que é o OmniDesk?

O **OmniDesk** é uma ferramenta de produtividade multiplataforma (Linux, macOS e Windows) de código aberto projetada para sincronizar a área de transferência (*clipboard*) e transferir arquivos instantaneamente entre seus próprios computadores conectados na mesma rede local (Wi-Fi ou cabo Ethernet).

Diferente de soluções baseadas na nuvem, o OmniDesk opera **100% ponto a ponto (P2P)**:

- 🔒 **Privacidade Absoluta e Zero Nuvem**: Nenhum dado, texto ou arquivo sai da sua rede local. Não requer criação de conta, não realiza rastreamento e não possui telemetria.
- ⚡ **Velocidade Máxima da Rede Local**: Transferências diretas limitadas apenas pela largura de banda da sua rede Wi-Fi ou cabo de rede.
- 🛡️ **Pareamento Simples e Seguro**: Dispositivos conectam-se usando autenticação mútua protegida por um **PIN de 6 dígitos** temporário e tokens criptográficos exclusivos.
- 🖥️ **Interface Moderna e Bandeja do Sistema**: Gerencie conexões através de um painel web intuitivo ou pelo ícone na bandeja do sistema (*System Tray*).

---

## ✨ Recursos Principais

- 📋 **Sincronização Contínua de Clipboard**: Copie um texto, link ou trecho de código em um computador e cole instantaneamente no outro (com proteção integrada contra loops infinitos de cópia).
- 📁 **Transferência Direta de Arquivos**: Envie arquivos de qualquer tamanho arrastando e soltando (*drag & drop*) no navegador ou via terminal.
- 🔍 **Descoberta Automática de Rede**: Detecção imediata de computadores vizinhos utilizando mDNS (ZeroConf) e varredura de sub-rede.
- 💻 **Dashboard Web Intuitivo**: Interface gráfica moderna acessível em `http://127.0.0.1:24850/ui/` ou pelo comando `omnidesk gui`.
- 🔔 **Notificações Nativas e System Tray**: Ícone na barra de tarefas/menu bar no Linux, macOS e Windows com alertas de recebimento de arquivos e pareamento.
- ⚙️ **Inicialização com o Sistema**: Inicia automaticamente em segundo plano no login do usuário sem necessidade de privilégios de administrador (`root`/`sudo`).

---

## 📦 Instalação Rápida (Instaladores)

Recomendamos utilizar os pacotes de instalação prontos do OmniDesk para o seu sistema operacional:

### 🍏 macOS (Apple Silicon M1/M2/M3/M4 & Intel)

- **Instalador `.dmg`**:
  1. Baixe o arquivo `OmniDesk.dmg` na página de [Releases](https://github.com/thyagoluciano/OmniDesk/releases).
  2. Abra a imagem de disco e arraste o **OmniDesk** para a pasta **Aplicativos** (`/Applications`).
- **Script Assistente Automático**:
  - Se baixou o repositório, basta dar duplo clique ou executar:
    ```bash
    ./scripts/install-mac.command
    ```
  *O script copia o app para `/Applications`, remove bloqueios do Gatekeeper (`xattr -cr`) e configura a inicialização automática via LaunchAgent.*

---

### 🐧 Linux (Ubuntu, Debian, Fedora, Arch)

- **Pacote Debian / Ubuntu (`.deb`)**:
  ```bash
  # Baixe o arquivo .deb das Releases e instale:
  sudo dpkg -i omnidesk_0.1.0_amd64.deb
  ```
  *O pacote adiciona o binário em `/usr/bin/omnidesk`, integra os ícones do sistema, o lançador `.desktop` no menu de aplicativos e o serviço systemd de usuário.*

- **Instalação Direta no Ambiente do Usuário**:
  Se você já possui o binário baixado:
  ```bash
  ./omnidesk install
  ```
  *Configura o OmniDesk em `~/.local/bin`, registra o atalho no menu e ativa o serviço do systemd (`omnidesk.service`).*

---

### 🪟 Windows (10 / 11)

- **Instalador com Assistente (`.exe`)**:
  1. Baixe o instalador `OmniDesk-Setup-0.1.0-x64.exe` na página de [Releases](https://github.com/thyagoluciano/OmniDesk/releases).
  2. Execute o assistente de instalação e siga os passos na tela.
- **Instalação via Terminal (PowerShell / CMD)**:
  ```powershell
  .\omnidesk.exe install
  ```
  *Configura o atalho de inicialização no registro do usuário para iniciar junto com o Windows.*

---

## 🚀 Como Usar

### 1. Acessando o OmniDesk
Ao ser instalado ou executado, o OmniDesk permanece ativo na **bandeja do sistema** (*System Tray* / Barra de Menus).

- Clique com o botão direito no ícone da bandeja e selecione **Abrir Painel**.
- Ou abra no seu navegador: **`http://127.0.0.1:24850/ui/`**
- Ou execute no terminal: `omnidesk gui`

---

### 2. Pareando Dispositivos em 1 Clique

Os computadores precisam ser pareados uma única vez para estabelecer uma conexão de confiança:

1. Abra o painel no **Computador A** e vá na seção **Dispositivos Descobertos**.
2. Clique em **Parear** ao lado do outro computador.
3. Um **PIN de 6 dígitos** temporário será exibido na tela do Computador A.
4. No **Computador B**, clique em **Aprovar PIN** no topo do painel, digite o código de 6 dígitos e confirme.
5. Pronto! Ambos os computadores agora estão conectados com certificados criptográficos seguros.

---

### 3. Sincronização de Área de Transferência
- Uma vez pareados, qualquer texto, link ou código copiado (`Ctrl+C` / `Cmd+C`) em uma máquina é replicado automaticamente na outra.
- Basta colar (`Ctrl+V` / `Cmd+V`) no segundo computador.
- Você pode ligar ou desligar a sincronização a qualquer momento usando a chave seletora no Dashboard Web.

---

### 4. Transferência de Arquivos
- **No Dashboard Web**: Arraste e solte (*drag & drop*) qualquer arquivo sobre o cartão do dispositivo de destino ou clique no botão **Enviar Arquivo**.
- **Onde ficam os arquivos recebidos?**
  - **Linux / macOS**: `~/Downloads/OmniDesk`
  - **Windows**: `%USERPROFILE%\Downloads\OmniDesk`

---

## 💻 Modo CLI (Linha de Comando)

Para quem prefere utilizar o terminal, automatizar fluxos ou gerenciar máquinas sem interface gráfica (*headless*), o OmniDesk inclui uma interface de linha de comando completa.

### Referência de Comandos

| Comando | Descrição |
| :--- | :--- |
| `omnidesk` | Inicia o OmniDesk com suporte à bandeja do sistema (*System Tray*). |
| `omnidesk daemon` | Inicia o serviço em segundo plano (`--headless` para omitir a bandeja gráfica). |
| `omnidesk gui` | Abre o Painel de Controle Web no navegador padrão. |
| `omnidesk status` | Exibe o status do daemon, porta HTTP, identificador local e pares conectados. |
| `omnidesk devices` | Varre e lista todos os nós OmniDesk ativos na rede local (LAN). |
| `omnidesk pair <alvo>` | Inicia solicitação de pareamento com outro nó por IP ou nome de host. |
| `omnidesk pair <PIN>` | Autoriza uma solicitação de pareamento recebida utilizando o PIN de 6 dígitos. |
| `omnidesk send <arquivo> <alvo>` | Envia um arquivo diretamente para um dispositivo pareado. |
| `omnidesk install` | Configura o OmniDesk no sistema com inicialização automática no login. |
| `omnidesk uninstall` | Remove os binários, atalhos e serviços de inicialização automática. |
| `omnidesk version` | Exibe a versão atual instalada do OmniDesk. |

### Exemplos Práticos no Terminal

```bash
# Verificar o status e dispositivos pareados
omnidesk status

# Descobrir computadores vizinhos na rede
omnidesk devices

# Solicitar pareamento pelo nome ou IP do computador
omnidesk pair MacBook-Pro
# ou: omnidesk pair 192.168.1.50

# No outro computador, autorizar informando o PIN exibido:
omnidesk pair 481920

# Enviar um arquivo para um dispositivo pareado
omnidesk send ~/Documentos/relatorio.pdf "MacBook-Pro"
```

### Gerenciamento de Serviços no Linux (systemd)

```bash
# Ver status do serviço do OmniDesk
systemctl --user status omnidesk.service

# Reiniciar o serviço
systemctl --user restart omnidesk.service

# Parar o serviço
systemctl --user stop omnidesk.service
```

---

## 🛠️ Compilando do Código-Fonte

Caso deseje contribuir ou compilar o OmniDesk por conta própria:

### Pré-requisitos
- **Go 1.22+** instalado ([golang.org](https://golang.org/dl/))
- Compilador C/C++ padrão do sistema (para bibliotecas nativas de área de transferência e atalhos)

### Compilação Rápida
```bash
# Clonar o repositório
git clone https://github.com/thyagoluciano/OmniDesk.git
cd OmniDesk

# Compilar o binário
go build -ldflags="-s -w" -o omnidesk ./cmd/omnidesk
```

### Gerar os Pacotes e Instaladores
Disponibilizamos scripts prontos no diretório `scripts/`:

```bash
# Gerar pacote .deb (Linux):
./scripts/build-deb.sh

# Gerar pacote .app e .dmg (macOS):
./scripts/build-macos-app.sh
./scripts/build-dmg.sh

# Compilar para Windows (Cross-compilation ou nativo):
./scripts/build-windows.sh
```

---

## 🤝 Comunidade e Contribuição

Contribuições são muito bem-vindas! Se você tem ideias para novos recursos, melhorias ou correções de bugs:

- 📖 **[Guia de Contribuição](CONTRIBUTING.md)**: Passos detalhados para configurar o ambiente e enviar Pull Requests.
- 🗺️ **[Roadmap do Produto](ROADMAP.md)**: Acompanhe o que já foi feito, o que está em desenvolvimento e os próximos passos.
- 🤝 **[Código de Conduta](CODE_OF_CONDUCT.md)**: Nossas diretrizes de convivência e respeito mútuo.
- 🛡️ **[Política de Segurança](SECURITY.md)**: Procedimentos para relatar vulnerabilidades de forma responsável.
- 💬 **[Discussões no GitHub](https://github.com/thyagoluciano/OmniDesk/discussions)**: Espaço para dúvidas, sugestões e ideias.
- 🐛 **[Abrir uma Issue](https://github.com/thyagoluciano/OmniDesk/issues)**: Reportar falhas ou propor melhorias.

---

## 📄 Licença

Este projeto é licenciado sob os termos da licença [MIT](LICENSE) &copy; 2026 Thyago Luciano.
