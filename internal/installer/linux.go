package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"omnidesk/assets"
)

// InstallLinux installs the omnidesk binary, icons, desktop launcher, and user systemd service.
func InstallLinux(autostart bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	binDir := filepath.Join(home, ".local", "bin")
	appDir := filepath.Join(home, ".local", "share", "applications")
	iconDir := filepath.Join(home, ".local", "share", "icons", "hicolor", "256x256", "apps")
	svgDir := filepath.Join(home, ".local", "share", "icons", "hicolor", "scalable", "apps")
	serviceDir := filepath.Join(home, ".config", "systemd", "user")

	// Ensure directories exist
	for _, dir := range []string{binDir, appDir, iconDir, svgDir, serviceDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// 1. Install binary
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate current executable: %w", err)
	}

	targetBin := filepath.Join(binDir, "omnidesk")
	if execPath != targetBin {
		fmt.Printf("-> Instalando binário em %s...\n", targetBin)
		if err := copyExecutable(execPath, targetBin); err != nil {
			return fmt.Errorf("failed to copy binary: %w", err)
		}
	}

	// 2. Install desktop launcher
	targetDesktop := filepath.Join(appDir, "omnidesk.desktop")
	fmt.Printf("-> Registrando lançador de aplicativo em %s...\n", targetDesktop)
	if err := os.WriteFile(targetDesktop, assets.DesktopEntry, 0644); err != nil {
		return fmt.Errorf("failed to write .desktop file: %w", err)
	}

	// 3. Install icons
	targetPNG := filepath.Join(iconDir, "omnidesk.png")
	targetSVG := filepath.Join(svgDir, "omnidesk.svg")
	fmt.Printf("-> Instalando ícones do sistema...\n")
	_ = os.WriteFile(targetPNG, assets.IconPNG, 0644)
	_ = os.WriteFile(targetSVG, assets.IconSVG, 0644)

	// Update desktop & icon databases if tools exist
	_ = exec.Command("update-desktop-database", appDir).Run()
	_ = exec.Command("gtk-update-icon-cache", "-f", "-t", filepath.Join(home, ".local", "share", "icons", "hicolor")).Run()

	// 4. Install systemd user service
	targetService := filepath.Join(serviceDir, "omnidesk.service")
	fmt.Printf("-> Configurando serviço systemd de usuário em %s...\n", targetService)
	if err := os.WriteFile(targetService, assets.SystemdService, 0644); err != nil {
		return fmt.Errorf("failed to write systemd service unit: %w", err)
	}

	if autostart {
		fmt.Println("-> Ativando e iniciando serviço com 'systemctl --user'...")
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
		if err := exec.Command("systemctl", "--user", "enable", "--now", "omnidesk.service").Run(); err != nil {
			fmt.Printf("Aviso: Falha ao ativar systemd: %v. Você pode rodar manualmente: systemctl --user enable --now omnidesk.service\n", err)
		} else {
			fmt.Println("✓ Serviço OmniDesk ativado e iniciado em segundo plano!")
		}
	}

	fmt.Println("\n=======================================================")
	fmt.Println("✓ INSTALAÇÃO CONCLUÍDA COM SUCESSO!")
	fmt.Println("=======================================================")
	fmt.Println("O OmniDesk agora:")
	fmt.Println("  1. Inicia automaticamente junto com o seu login")
	fmt.Println("  2. Está disponível no menu de aplicativos do GNOME/Ubuntu")
	fmt.Println("  3. Pode ser acessado no terminal via comando 'omnidesk'")
	fmt.Println("=======================================================")
	return nil
}

// UninstallLinux removes all installed omnidesk files and stops the systemd user service.
func UninstallLinux() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	fmt.Println("-> Encerrando e desativando serviço systemd...")
	_ = exec.Command("systemctl", "--user", "stop", "omnidesk.service").Run()
	_ = exec.Command("systemctl", "--user", "disable", "omnidesk.service").Run()

	filesToRemove := []string{
		filepath.Join(home, ".config", "systemd", "user", "omnidesk.service"),
		filepath.Join(home, ".local", "share", "applications", "omnidesk.desktop"),
		filepath.Join(home, ".local", "share", "icons", "hicolor", "256x256", "apps", "omnidesk.png"),
		filepath.Join(home, ".local", "share", "icons", "hicolor", "scalable", "apps", "omnidesk.svg"),
		filepath.Join(home, ".local", "bin", "omnidesk"),
	}

	for _, f := range filesToRemove {
		if err := os.Remove(f); err == nil {
			fmt.Printf("-> Removido: %s\n", f)
		}
	}

	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	fmt.Println("\n✓ OmniDesk foi completamente desinstalado.")
	return nil
}

func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Remove existing target first (avoid ETXTBSY if running)
	_ = os.Remove(dst)

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
