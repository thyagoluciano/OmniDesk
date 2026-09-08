//go:build !darwin || cgo

package ui

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"time"

	"omnidesk/internal/core"
	"fyne.io/systray"
)

// TrayApp manages the system tray icon and menu actions.
type TrayApp struct {
	node *core.Node
}

// NewTrayApp creates a new tray application.
func NewTrayApp(node *core.Node) *TrayApp {
	return &TrayApp{node: node}
}

// Start launches the system tray menu loop. Blocks the caller thread.
func (t *TrayApp) Start() {
	systray.Run(t.onReady, t.onExit)
}

func (t *TrayApp) onReady() {
	iconBytes := GenerateIconBytes()
	systray.SetIcon(iconBytes)
	systray.SetTooltip(fmt.Sprintf("OmniDesk: %s", t.node.Cfg.DeviceName))

	// Dashboard launcher item
	mDashboard := systray.AddMenuItem("Abrir Painel (Dashboard)", "Abrir a interface gráfica do OmniDesk")
	systray.AddSeparator()

	// Status item
	mStatus := systray.AddMenuItem(fmt.Sprintf("Nó: %s", t.node.Cfg.DeviceName), "Nome deste dispositivo")
	mStatus.Disable()

	mPeers := systray.AddMenuItem("Dispositivos conectados: 0", "Nós pareados online")
	mPeers.Disable()

	systray.AddSeparator()

	// Clipboard toggle
	clipTitle := "Pausar sincronização de clipboard"
	if !t.node.Cfg.IsClipboardSyncEnabled() {
		clipTitle = "Retomar sincronização de clipboard"
	}
	mClipToggle := systray.AddMenuItem(clipTitle, "Ativar/desativar cópia e cola entre máquinas")

	// Open downloads folder
	mOpenDownloads := systray.AddMenuItem("Abrir pasta de recebidos", "Abrir ~/Downloads/OmniDesk")

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Sair do OmniDesk", "Encerrar o serviço")

	// Goroutine to periodically update peer counter
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			online := len(t.node.GetOnlineTrustedPeers())
			mPeers.SetTitle(fmt.Sprintf("Dispositivos conectados: %d", online))
		}
	}()

	// Event handling loop
	go func() {
		for {
			select {
			case <-mDashboard.ClickedCh:
				OpenDashboard(t.node.Cfg.ListenPort)

			case <-mClipToggle.ClickedCh:
				enabled := !t.node.Cfg.IsClipboardSyncEnabled()
				_ = t.node.Cfg.SetClipboardSync(enabled)
				if enabled {
					mClipToggle.SetTitle("Pausar sincronização de clipboard")
				} else {
					mClipToggle.SetTitle("Retomar sincronização de clipboard")
				}

			case <-mOpenDownloads.ClickedCh:
				openFolder(t.node.Cfg.DownloadDir)

			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (t *TrayApp) onExit() {
	log.Println("[ui] exiting OmniDesk tray application")
	t.node.Stop()
}

func openFolder(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		return
	}
	_ = cmd.Start()
}
