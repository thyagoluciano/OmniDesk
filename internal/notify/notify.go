package notify

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
)

// Notifier triggers native desktop notifications across macOS and Linux.
type Notifier struct{}

// NewNotifier creates a new desktop notification dispatcher.
func NewNotifier() *Notifier {
	return &Notifier{}
}

// NotifyFileReceived sends a native desktop notification about a received file.
func (n *Notifier) NotifyFileReceived(fileName, senderName string, sizeBytes int64) {
	title := "OmniDesk"
	sizeStr := formatFileSize(sizeBytes)
	msg := fmt.Sprintf("Recebido: %s (%s) de %s", fileName, sizeStr, senderName)

	go func() {
		if err := n.SendNotification(title, msg); err != nil {
			log.Printf("[notify] %s: %s", title, msg)
		}
	}()
}

// SendNotification dispatches a title and message notification to the active OS.
func (n *Notifier) SendNotification(title, message string) error {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, message, title)
		cmd := exec.Command("osascript", "-e", script)
		return cmd.Run()

	case "linux":
		// Try notify-send first
		if path, err := exec.LookPath("notify-send"); err == nil {
			cmd := exec.Command(path, "-a", "OmniDesk", title, message)
			return cmd.Run()
		}
		return fmt.Errorf("no desktop notification tool found (notify-send)")

	case "windows":
		cleanTitle := strings.ReplaceAll(title, "'", "''")
		cleanMsg := strings.ReplaceAll(message, "'", "''")
		psScript := fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; $template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02); $template.GetElementsByTagName('text')[0].AppendChild($template.CreateTextNode('%s')) > $null; $template.GetElementsByTagName('text')[1].AppendChild($template.CreateTextNode('%s')) > $null; $toast = [Windows.UI.Notifications.ToastNotification]::new($template); [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('OmniDesk').Show($toast)`, cleanTitle, cleanMsg)
		cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", psScript)
		return cmd.Start()

	default:
		log.Printf("[notify] [%s] %s", title, message)
		return nil
	}
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
