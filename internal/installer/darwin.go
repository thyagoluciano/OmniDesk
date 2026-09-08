package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const launchAgentPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.omnidesk.app</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Applications/OmniDesk.app/Contents/MacOS/omnidesk</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>ProcessType</key>
    <string>Interactive</string>
</dict>
</plist>
`

// InstallDarwin configures the macOS LaunchAgent autostart service and unblocks Gatekeeper.
func InstallDarwin(autostart bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// 1. Check if OmniDesk.app is already in /Applications or in current dir
	appDst := "/Applications/OmniDesk.app"
	if _, err := os.Stat(appDst); err != nil {
		execPath, _ := os.Executable()
		if strings.Contains(execPath, "OmniDesk.app") {
			idx := strings.Index(execPath, "OmniDesk.app")
			bundleSrc := execPath[:idx+len("OmniDesk.app")]
			fmt.Printf("-> Copiando bundle para %s...\n", appDst)
			_ = exec.Command("cp", "-R", bundleSrc, appDst).Run()
		} else if _, err := os.Stat("OmniDesk.app"); err == nil {
			fmt.Printf("-> Copiando OmniDesk.app para %s...\n", appDst)
			_ = exec.Command("cp", "-R", "OmniDesk.app", appDst).Run()
		}
	}

	// 2. Remove quarantine and ad-hoc sign with local toolchain to fix "App está danificado"
	if _, err := os.Stat(appDst); err == nil {
		fmt.Println("-> Removendo quarentena do Gatekeeper (xattr -cr)...")
		_ = exec.Command("xattr", "-cr", appDst).Run()
		fmt.Println("-> Aplicando assinatura ad-hoc local (codesign)...")
		_ = exec.Command("codesign", "--force", "--deep", "-s", "-", appDst).Run()
		_ = exec.Command("chmod", "+x", filepath.Join(appDst, "Contents", "MacOS", "omnidesk")).Run()
	}

	agentDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	plistPath := filepath.Join(agentDir, "com.omnidesk.app.plist")
	fmt.Printf("-> Criando LaunchAgent em %s...\n", plistPath)
	if err := os.WriteFile(plistPath, []byte(launchAgentPlist), 0644); err != nil {
		return fmt.Errorf("failed to write LaunchAgent plist: %w", err)
	}

	if autostart {
		fmt.Println("-> Carregando serviço no launchctl...")
		_ = exec.Command("launchctl", "unload", plistPath).Run()
		if err := exec.Command("launchctl", "load", "-w", plistPath).Run(); err != nil {
			fmt.Printf("Aviso: Falha ao carregar no launchctl: %v\n", err)
		} else {
			fmt.Println("✓ OmniDesk configurado para inicialização automática no macOS!")
		}
	}

	return nil
}

// UninstallDarwin unloads and removes the LaunchAgent plist.
func UninstallDarwin() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.omnidesk.app.plist")
	_ = exec.Command("launchctl", "unload", plistPath).Run()
	_ = os.Remove(plistPath)
	fmt.Println("✓ Autostart do OmniDesk no macOS removido com sucesso.")
	return nil
}
