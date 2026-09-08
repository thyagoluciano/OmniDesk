# macos-installer Specification

## Purpose

Provides macOS application bundle creation, high-resolution application icons, LaunchAgent background autostart configuration, and distributable disk image (`.dmg`) generation.

## Requirements

### Requirement: macOS Application Bundle Structure
The system SHALL support packaging into a standard macOS application bundle named `Crossover.app`.

#### Scenario: Building application bundle
- **WHEN** the macOS bundle build command is executed
- **THEN** the system SHALL create `Crossover.app` containing the macOS universal or arm64 binary, `Contents/Info.plist` with application metadata, and high-resolution `AppIcon.icns`.

#### Scenario: Launching bundle from Applications folder
- **WHEN** the user double-clicks `Crossover.app` or runs `open -a Crossover`
- **THEN** Crossover SHALL launch with tray menu, dashboard access, and background network listening active.

### Requirement: LaunchAgent User Autostart
The system SHALL provide configuration and commands to manage a user-level `LaunchAgent` for automatic startup upon login.

#### Scenario: Enabling macOS autostart
- **WHEN** the user enables autostart via CLI or installer script
- **THEN** a LaunchAgent plist file SHALL be created in `~/Library/LaunchAgents/com.crossover.app.plist` and loaded via `launchctl`.

#### Scenario: Disabling macOS autostart
- **WHEN** the user disables autostart
- **THEN** the LaunchAgent plist SHALL be unloaded via `launchctl` and removed from `~/Library/LaunchAgents/`.

### Requirement: DMG Disk Image Packaging
The system SHALL provide packaging automation to generate a distributable Apple Disk Image (`.dmg`) file.

#### Scenario: Generating disk image
- **WHEN** the macOS DMG packaging script is executed
- **THEN** the system SHALL create a `.dmg` file containing `Crossover.app` and a drag-and-drop symlink to `/Applications`.
