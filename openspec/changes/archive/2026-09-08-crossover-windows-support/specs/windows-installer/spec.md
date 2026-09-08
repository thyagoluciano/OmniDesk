## Purpose

Provides Windows desktop integration, setup wizard installer packaging, desktop and Start Menu shortcuts, and user registry autostart configuration for Microsoft Windows.

## ADDED Requirements

### Requirement: User-Level Registry Autostart Management
The system SHALL provide sub-commands allowing users to enable or disable automatic startup upon Windows user logon via the current user registry.

#### Scenario: Enabling autostart on Windows
- **WHEN** the user executes `crossover install` or runs the setup installer
- **THEN** the system SHALL register the Crossover executable under `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` and verify the registration.

#### Scenario: Disabling autostart on Windows
- **WHEN** the user executes `crossover uninstall` or runs the uninstaller
- **THEN** the system SHALL remove the Crossover key from `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` and terminate the background process.

### Requirement: Windows Setup Wizard Packaging
The system SHALL provide an NSIS script and build script to compile a standalone graphical setup installer (`Crossover-Setup-<version>-x64.exe`).

#### Scenario: Executing the setup wizard
- **WHEN** the user runs the setup installer on Windows
- **THEN** the wizard SHALL guide the user, install `crossover.exe` into `%LOCALAPPDATA%\Programs\Crossover\`, create Start Menu and Desktop shortcuts, and configure autostart.

#### Scenario: Registering Windows uninstaller
- **WHEN** the setup wizard completes installation
- **THEN** the system SHALL create `uninstall.exe` and register Crossover in Windows "Installed Apps" / "Add or Remove Programs" for clean uninstallation.

### Requirement: Windows App-Mode Dashboard Launcher
The system SHALL launch the graphical dashboard in a dedicated desktop window on Windows using the pre-installed Microsoft Edge app-mode or the system default browser.

#### Scenario: Opening dashboard on Windows
- **WHEN** the user selects "Abrir Painel (Dashboard)" from the Windows system tray or executes `crossover gui`
- **THEN** the system SHALL launch the dashboard in a dedicated standalone application window without address or tab bars.

### Requirement: Windows Native Desktop Notifications
The system SHALL display native Windows toast notifications when files are received or pairing requests arrive.

#### Scenario: Receiving a file on Windows
- **WHEN** a remote peer finishes sending a file to the Windows node
- **THEN** the system SHALL dispatch a native Windows notification toast showing the filename and sender.
