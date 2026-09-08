# desktop-gui Specification

## Purpose

Provides a desktop graphical user interface displaying real-time connected peers, clipboard synchronization status, drag-and-drop file transfers, and interactive pairing management.

## Requirements

### Requirement: Peer Overview and Live Status
The system SHALL present a graphical dashboard displaying the local node information and all discovered and trusted peers on the local network in real-time.

#### Scenario: Displaying peer status
- **WHEN** the user opens the graphical dashboard
- **THEN** the interface SHALL render the local node name and ID, along with cards for each discovered and trusted peer showing their hostname, IP address, and connection status.

#### Scenario: Peer status changes
- **WHEN** a peer connects, disconnects, or updates its status on the local network
- **THEN** the graphical dashboard SHALL update the peer's visual indicator without requiring a full manual page refresh.

### Requirement: Drag-and-Drop File Sharing
The system SHALL allow the user to transfer files to any connected peer by dragging and dropping files onto the respective peer's interface card or dropzone.

#### Scenario: Initiating file transfer via dropzone
- **WHEN** the user drops one or more files onto a designated peer card or target dropzone
- **THEN** the system SHALL stream the selected files to the target peer and visually indicate upload progress and final transfer status.

#### Scenario: Handling invalid or cancelled file drop
- **WHEN** the user drops unsupported items or cancels an active transfer
- **THEN** the system SHALL abort the operation and display an informative error notification.

### Requirement: Interactive Pairing and PIN Management
The system SHALL present visual prompts and input dialogs for pairing with new devices via PIN verification.

#### Scenario: Initiating pairing to an untrusted peer
- **WHEN** the user clicks "Pair" on an untrusted discovered peer
- **THEN** the system SHALL send a pairing request and display the 6-digit PIN to the user to confirm on the remote machine.

#### Scenario: Authorizing an incoming pairing request
- **WHEN** an incoming pairing request is received from a remote peer
- **THEN** the system SHALL display a visual alert containing the requester's name, IP, and the 6-digit PIN, providing "Approve" and "Reject" actions.

### Requirement: System Tray Window Integration
The system SHALL integrate the graphical user interface with the operating system tray icon.

#### Scenario: Toggling window from system tray
- **WHEN** the user selects the "Open Dashboard" item from the system tray menu
- **THEN** the graphical window SHALL be brought to the foreground, or created if not already open.

#### Scenario: Closing the graphical window
- **WHEN** the user closes the dashboard window
- **THEN** the background synchronization and daemon service SHALL continue running in the system tray without terminating the application.
