# Contributing to OmniDesk

Thank you for your interest in contributing to **OmniDesk**! :tada: :sparkles:

OmniDesk is a community-driven, 100% local-first, peer-to-peer (P2P) productivity suite built with Go, modern web technologies, and cross-platform system integrations. We welcome contributions of all kinds: code, bug reports, feature proposals, documentation improvements, and translations.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Our Core Philosophy](#our-core-philosophy)
- [How Can I Contribute?](#how-can-i-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Features & Enhancements](#suggesting-features--enhancements)
  - [Documentation & Translations](#documentation--translations)
  - [Code Contributions](#code-contributions)
- [Development Setup](#development-setup)
  - [Prerequisites](#prerequisites)
  - [Cloning & Building](#cloning--building)
  - [Running Tests](#running-tests)
- [Project Architecture](#project-architecture)
- [Development Workflow & Standards](#development-workflow--standards)
  - [Branch Naming](#branch-naming)
  - [Conventional Commits](#conventional-commits)
  - [Code Style & Formatting](#code-style--formatting)
  - [Cross-Platform Guidelines](#cross-platform-guidelines)
- [Pull Request Process](#pull-request-process)
- [Community & Getting Help](#community--getting-help)

---

## Code of Conduct

All contributors and maintainers are expected to adhere to our [Code of Conduct](CODE_OF_CONDUCT.md). Please read it before participating in discussions or submitting contributions.

---

## Our Core Philosophy

When proposing changes or writing code, keep OmniDesk's core tenets in mind:

1. **100% Local-First & Zero Cloud**: Data, clipboard contents, and files must never leave the local area network. No external telemetries, cloud analytics, or third-party servers.
2. **Privacy & Security by Default**: Devices only interact after mutual pairing with PIN verification and cryptographically signed tokens.
3. **Cross-Platform Parity**: Features should work seamlessly across Linux (X11 & Wayland), macOS (Intel & Apple Silicon), and Windows.
4. **Lightweight & Efficient**: Minimal idle CPU and memory footprints. OmniDesk runs in the background 24/7 without hindering user workflow.

---

## How Can I Contribute?

### Reporting Bugs

Before creating a bug report, please check existing [GitHub Issues](https://github.com/thyagoluciano/OmniDesk/issues) to avoid duplicates.

If you encounter a new bug, please open an issue using the **[Bug Report Template](https://github.com/thyagoluciano/OmniDesk/issues/new?template=bug_report.yml)**. Please include:
- A clear and concise title.
- Exact steps to reproduce the issue.
- Expected behavior vs. actual behavior.
- Operating system, architecture, and desktop environment (e.g., Ubuntu 22.04 / GNOME X11, macOS 14.5 Sonoma Apple Silicon, Windows 11).
- OmniDesk version (`omnidesk version`).
- Daemon logs or terminal output.

### Suggesting Features & Enhancements

We are eager to hear ideas for improving OmniDesk! To propose a new feature:
1. Review our [Roadmap](ROADMAP.md) to see if the feature is already planned.
2. Open a feature request via the **[Feature Request Template](https://github.com/thyagoluciano/OmniDesk/issues/new?template=feature_request.yml)**.
3. Describe the problem your feature solves, why it aligns with our local-first philosophy, and any alternative solutions you considered.

### Documentation & Translations

OmniDesk is used globally! You can help by:
- Improving or fixing typos in our guides, docstrings, or README.
- Enhancing translations in Portuguese (PT-BR), English (EN), or Spanish (ES).
- Writing setup guides for specific Linux window managers or enterprise LANs.

### Code Contributions

Whether fixing an open issue, adding a missing platform feature, or optimizing performance, follow the steps below to set up your environment.

---

## Development Setup

### Prerequisites

- **Go 1.20+** (Go 1.22+ recommended): [Install Go](https://go.dev/doc/install)
- **Git**
- **C Compiler / System Headers** (depending on platform):
  - **Linux**: Standard build tools and X11 development libraries (if working on Linux input sharing or clipboard):
    ```bash
    # Debian/Ubuntu
    sudo apt-get install build-essential libx11-dev libxtst-dev

    # Fedora
    sudo dnf install gcc libX11-devel libXtst-devel
    ```
  - **macOS**: Xcode Command Line Tools (`xcode-select --install`)
  - **Windows**: [Git Bash](https://gitforwindows.org/) or PowerShell; MinGW/GCC (optional).

### Cloning & Building

1. **Fork** the repository on GitHub: `https://github.com/thyagoluciano/OmniDesk`
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/OmniDesk.git
   cd OmniDesk
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/thyagoluciano/OmniDesk.git
   ```
4. **Compile the binary**:
   ```bash
   go build -o omnidesk ./cmd/omnidesk
   ```

### Running Tests

Ensure all unit and integration tests pass before opening a PR:

```bash
# Run all tests
go test -v ./...

# Run tests with race detection (Linux & macOS)
go test -race ./...

# Run vetting
go vet ./...
```

---

## Project Architecture

A high-level overview of the repository structure:

```text
OmniDesk/
├── cmd/
│   └── omnidesk/             # Main application entry point & CLI subcommands
├── internal/
│   ├── clipboard/            # Clipboard watching and cross-platform synchronization
│   ├── config/               # Configuration files, paths, and persistent storage
│   ├── core/                 # Core daemon node, HTTP/WebSocket API server & routing
│   ├── discovery/            # Peer discovery via mDNS (ZeroConf) and subnet scanning
│   ├── inputshare/           # Cross-device KVM input sharing (mouse & keyboard)
│   ├── installer/            # System services, .desktop integration & background daemons
│   ├── notify/               # Native desktop notifications
│   ├── pairing/              # PIN generation, pairing handshake and cryptographic tokens
│   ├── transfer/             # Direct P2P streaming file upload/download engine
│   └── ui/                   # System tray integration and browser opener
├── web/                      # Embedded HTML5/CSS3/Vanilla JS control dashboard
├── scripts/                  # Packaging and build automation scripts (.deb, .app, .dmg)
└── assets/                   # App icons, SVG logos, systemd unit files, desktop entries
```

---

## Development Workflow & Standards

### Branch Naming

Create a feature branch with a descriptive name prefixed by the change type:

- `feat/feature-name` (e.g., `feat/wayland-input-capture`)
- `fix/issue-description` (e.g., `fix/clipboard-loop-detection`)
- `docs/doc-update` (e.g., `docs/troubleshooting-guide`)
- `refactor/subsystem-name` (e.g., `refactor/discovery-engine`)
- `test/test-name` (e.g., `test/transfer-resume-cases`)

### Conventional Commits

We follow the **[Conventional Commits specification](https://www.conventionalcommits.org/)**. Each commit message must be structured as follows:

```text
<type>(<optional scope>): <description>

[optional body]

[optional footer(s)]
```

#### Allowed Types:
- **`feat`**: A new user-facing feature or enhancement.
- **`fix`**: A bug fix.
- **`docs`**: Documentation changes only.
- **`style`**: Changes that do not affect code meaning (white-space, formatting, missing semicolons).
- **`refactor`**: Code changes that neither fix a bug nor add a feature.
- **`perf`**: A code change that improves performance.
- **`test`**: Adding missing tests or correcting existing tests.
- **`chore`**: Build scripts, CI workflow changes, dependency upgrades.

#### Examples:
```text
feat(inputshare): add boundary edge detection for dual monitor setup
fix(transfer): handle partial chunk timeout gracefully on slow Wi-Fi
docs: add troubleshooting steps for Ubuntu Wayland users
test(clipboard): add test case for rapid multiline buffer changes
```

### Code Style & Formatting

- All Go code must be formatted using `gofmt`:
  ```bash
  gofmt -s -w .
  ```
- Keep functions modular and testable.
- Document exported packages, structs, and functions according to [Go Doc comments conventions](https://go.dev/doc/comment).
- Avoid unnecessary external dependencies. Prefer the Go standard library where possible.

### Cross-Platform Guidelines

OmniDesk runs on Linux, macOS, and Windows. When adding platform-specific functionality:
- Use Go build constraints (`//go:build <os>`) at the top of OS-specific files (e.g., `capture_linux.go`, `capture_darwin.go`, `capture_windows.go`).
- Provide stub or fallback implementations for unsupported platforms so that builds compile cleanly across all target architectures.

---

## Pull Request Process

1. **Keep Pull Requests Focused**: A PR should address a single concern, feature, or bug fix. Smaller, well-scoped PRs are reviewed and merged much faster.
2. **Update Documentation**: If your PR changes behavior, adds CLI flags, or adds new endpoints, update the README, Roadmap, or relevant docs.
3. **Add Tests**: Include unit or integration tests demonstrating that your fix or feature works as intended.
4. **Fill Out the PR Template**: When submitting your PR, complete all sections of our [Pull Request Template](.github/PULL_REQUEST_TEMPLATE.md).
5. **Pass Automated CI**: All GitHub Actions CI checks (linting, multi-platform tests) must pass green.
6. **Code Review**: A maintainer will review your code. Be open to feedback and suggestions.

---

## Community & Getting Help

- **Discussions & Questions**: Use [GitHub Discussions](https://github.com/thyagoluciano/OmniDesk/discussions) to ask questions or discuss architectural ideas.
- **Security Vulnerabilities**: See [SECURITY.md](SECURITY.md).
- **Maintainer**: Thyago Luciano ([@thyagoluciano](https://github.com/thyagoluciano)) - `thyagoluciano@gmail.com`.

Thank you for helping make OmniDesk better for everyone! :rocket:
