//go:build !darwin || cgo

package ui

import (
	"fmt"
	"log"
	_ "image/png"
	"os/exec"
	"runtime"
	"time"

	"fyne.io/systray"
	"omnidesk/internal/core"
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
	iconBytes := GetTrayIcon()
	systray.SetIcon(iconBytes)
	systray.SetTooltip(fmt.Sprintf("OmniDesk: %s", t.node.Cfg.DeviceName))

	// On Linux, fyne.io/systray starts onReady() concurrently before DBus connection
	// and property exports are complete. Calls to SetIcon/SetTooltip during that window
	// are silently dropped (if props == nil { return }). Re-applying with retries
	// ensures the icon and tooltip are exported to DBus and the NewIcon signal is emitted.
	go func() {
		delays := []time.Duration{
			50 * time.Millisecond,
			150 * time.Millisecond,
			300 * time.Millisecond,
			600 * time.Millisecond,
			1200 * time.Millisecond,
			2500 * time.Millisecond,
		}
		for _, d := range delays {
			time.Sleep(d)
			systray.SetIcon(iconBytes)
			systray.SetTooltip(fmt.Sprintf("OmniDesk: %s", t.node.Cfg.DeviceName))
		}
	}()

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

	// Input-sharing (KVM) status/escape item — a quick way to reclaim
	// mouse/keyboard control without touching the dashboard, per
	// specs/input-sharing-transport "Pausa de compartilhamento de input
	// via bandeja/dashboard".
	mInputStatus := systray.AddMenuItem("Controle remoto: inativo", "Nenhuma sessão de controle de mouse/teclado ativa")
	mInputStatus.Disable()

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Sair do OmniDesk", "Encerrar o serviço")

	// Goroutine to periodically update peer counter and input-sharing status
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			online := len(t.node.GetOnlineTrustedPeers())
			mPeers.SetTitle(fmt.Sprintf("Dispositivos conectados: %d", online))

			if t.node.InputMgr == nil {
				continue
			}
			if peerID, sending, active := t.node.InputMgr.ActiveSession(); active {
				name := peerID
				if dev, ok := t.node.Cfg.GetTrustedDevice(peerID); ok && dev.Name != "" {
					name = dev.Name
				}
				if sending {
					mInputStatus.SetTitle(fmt.Sprintf("Controlando %s (clique para encerrar)", name))
				} else {
					mInputStatus.SetTitle(fmt.Sprintf("Sendo controlado por %s (clique para encerrar)", name))
				}
				mInputStatus.Enable()
			} else {
				mInputStatus.SetTitle("Controle remoto: inativo")
				mInputStatus.Disable()
			}
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

			case <-mInputStatus.ClickedCh:
				if t.node.InputMgr != nil {
					t.node.InputMgr.StopSession()
				}

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
