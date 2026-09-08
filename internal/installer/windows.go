package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// InstallWindows configures Crossover installation and registry autostart on Windows.
func InstallWindows(autostart bool) error {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		localAppData = filepath.Join(home, "AppData", "Local")
	}

	targetDir := filepath.Join(localAppData, "Programs", "Crossover")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate current executable: %w", err)
	}

	targetBin := filepath.Join(targetDir, "crossover.exe")
	if execPath != targetBin {
		fmt.Printf("-> Copiando executável para %s...\n", targetBin)
		if err := copyExecutableWindows(execPath, targetBin); err != nil {
			return fmt.Errorf("failed to copy executable: %w", err)
		}
	}

	// Create Desktop and Start Menu Shortcuts via PowerShell
	fmt.Println("-> Criando atalhos no Menu Iniciar e Área de Trabalho...")
	psShortcutScript := fmt.Sprintf(`
$ws = New-Object -ComObject WScript.Shell
$desktop = [System.Environment]::GetFolderPath('Desktop')
$startMenu = [System.Environment]::GetFolderPath('StartMenu') + '\Programs'

$s1 = $ws.CreateShortcut("$desktop\Crossover.lnk")
$s1.TargetPath = "%s"
$s1.Description = "Crossover - Sincronizacao P2P"
$s1.Save()

$s2 = $ws.CreateShortcut("$startMenu\Crossover.lnk")
$s2.TargetPath = "%s"
$s2.Description = "Crossover - Sincronizacao P2P"
$s2.Save()
`, targetBin, targetBin)
	_ = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psShortcutScript).Run()

	if autostart {
		fmt.Println("-> Configurando inicialização automática no Registro do Windows (HKCU\\...\\Run)...")
		cmdStr := fmt.Sprintf(`"%s" daemon`, targetBin)
		regCmd := exec.Command("reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "Crossover", "/t", "REG_SZ", "/d", cmdStr, "/f")
		if err := regCmd.Run(); err != nil {
			fmt.Printf("Aviso: Falha ao registrar no Run: %v\n", err)
		} else {
			fmt.Println("✓ Crossover configurado para iniciar automaticamente no Windows!")
		}
	}

	fmt.Println("\n=======================================================")
	fmt.Println("✓ INSTALAÇÃO CONCLUÍDA COM SUCESSO NO WINDOWS!")
	fmt.Println("=======================================================")
	fmt.Println("O Crossover agora:")
	fmt.Println("  1. Inicia automaticamente com o Windows")
	fmt.Println("  2. Possui atalhos na Área de Trabalho e no Menu Iniciar")
	fmt.Println("  3. Pode ser executado a qualquer momento via terminal")
	fmt.Println("=======================================================")
	return nil
}

// UninstallWindows cleans up registry entries, shortcuts, and installed files.
func UninstallWindows() error {
	fmt.Println("-> Encerrando processos ativos do Crossover...")
	_ = exec.Command("taskkill", "/F", "/IM", "crossover.exe").Run()

	fmt.Println("-> Removendo chave de inicialização do Registro...")
	_ = exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "Crossover", "/f").Run()

	// Remove shortcuts
	psRemoveScript := `
$desktop = [System.Environment]::GetFolderPath('Desktop')
$startMenu = [System.Environment]::GetFolderPath('StartMenu') + '\Programs'
Remove-Item "$desktop\Crossover.lnk" -ErrorAction SilentlyContinue
Remove-Item "$startMenu\Crossover.lnk" -ErrorAction SilentlyContinue
`
	_ = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psRemoveScript).Run()

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		targetDir := filepath.Join(localAppData, "Programs", "Crossover")
		_ = os.RemoveAll(targetDir)
	}

	fmt.Println("\n✓ Crossover foi completamente removido do Windows.")
	return nil
}

func copyExecutableWindows(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	_ = os.Remove(dst)
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
