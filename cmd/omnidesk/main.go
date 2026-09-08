package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"omnidesk/internal/config"
	"omnidesk/internal/core"
	"omnidesk/internal/discovery"
	"omnidesk/internal/installer"
	"omnidesk/internal/pairing"
	"omnidesk/internal/transfer"
	"omnidesk/internal/ui"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		runDaemon([]string{})
		return
	}

	command := os.Args[1]

	// If flags like --headless or --port are passed directly
	if strings.HasPrefix(command, "-") && command != "--version" && command != "-v" && command != "--help" && command != "-h" {
		runDaemon(os.Args[1:])
		return
	}

	switch command {
	case "daemon":
		runDaemon(os.Args[2:])
	case "gui", "dashboard":
		runGUI(os.Args[2:])
	case "install":
		runInstall(os.Args[2:])
	case "uninstall":
		runUninstall(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "devices":
		runDevices(os.Args[2:])
	case "pair":
		runPair(os.Args[2:])
	case "send":
		runSend(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("OmniDesk version %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Comando desconhecido: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`OmniDesk - Sincronização P2P de Clipboard e Arquivos para macOS e Linux

Uso:
  omnidesk                   Inicia o agente OmniDesk com ícone no tray
  omnidesk <comando> [argumentos]

Comandos disponíveis:
  daemon               Inicia o agente OmniDesk em segundo plano
                       Opções: --headless (roda sem interface gráfica)
  install              Instala o OmniDesk no sistema com ícone e autostart
                       Opções: --autostart=false (não iniciar no login)
  uninstall            Remove o OmniDesk e o serviço de inicialização
  status               Exibe o estado atual do nó local e configurações
  devices              Descobre e lista os dispositivos na rede local (LAN)
  gui                  Abre o painel gráfico de controle no navegador/janela
  pair <dispositivo>   Inicia o pareamento com PIN de 6 dígitos com outro nó
                       Opções: --approve <session_id> (aprova solicitação recebida)
  send <arquivo> <alvo> Envia um arquivo diretamente para um dispositivo pareado
  version              Exibe a versão do OmniDesk
  help                 Exibe esta ajuda`)
}

func runInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	autostart := fs.Bool("autostart", true, "Habilitar inicialização automática no login")
	_ = fs.Parse(args)

	var err error
	switch runtime.GOOS {
	case "darwin":
		err = installer.InstallDarwin(*autostart)
	case "windows":
		err = installer.InstallWindows(*autostart)
	default:
		err = installer.InstallLinux(*autostart)
	}

	if err != nil {
		log.Fatalf("Erro na instalação: %v", err)
	}
}

func runUninstall(args []string) {
	var err error
	switch runtime.GOOS {
	case "darwin":
		err = installer.UninstallDarwin()
	case "windows":
		err = installer.UninstallWindows()
	default:
		err = installer.UninstallLinux()
	}

	if err != nil {
		log.Fatalf("Erro na desinstalação: %v", err)
	}
}

func runGUI(args []string) {
	cfg, err := config.Load()
	port := 24850
	if err == nil && cfg.ListenPort > 0 {
		port = cfg.ListenPort
	}
	fmt.Printf("Abrindo painel do OmniDesk (porta %d)...\n", port)
	ui.OpenDashboard(port)
}

func runDaemon(args []string) {
	fs := flag.NewFlagSet("daemon", flag.ExitOnError)
	headless := fs.Bool("headless", false, "Executar em modo headless (sem menu de bandeja)")
	port := fs.Int("port", 0, "Porta local para escuta (padrão do config)")
	_ = fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Erro ao carregar configurações: %v", err)
	}

	if *port > 0 {
		cfg.ListenPort = *port
	}

	node := core.NewNode(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := node.Start(ctx); err != nil {
		log.Fatalf("Erro ao iniciar nó OmniDesk: %v", err)
	}

	fmt.Println("=====================================================")
	fmt.Printf("  OmniDesk v%s ativo e monitorando\n", version)
	fmt.Printf("  Nó:    %s (ID: %s)\n", cfg.DeviceName, cfg.DeviceID)
	fmt.Printf("  Porta: %d\n", cfg.ListenPort)
	fmt.Printf("  Pasta: %s\n", cfg.DownloadDir)
	fmt.Println("=====================================================")

	// Send desktop notification confirming startup
	_ = node.Notifier.SendNotification("OmniDesk Conectado", fmt.Sprintf("Nó '%s' ativo na porta %d", cfg.DeviceName, cfg.ListenPort))

	// Graceful shutdown on SIGINT / SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Encerrando OmniDesk...")
		node.Stop()
		cancel()
		os.Exit(0)
	}()

	// If headless or no display, wait on signals
	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" || runtime.GOOS == "darwin"
	if *headless || !hasDisplay {
		fmt.Println("[omnidesk] Executando em modo headless (sem bandeja). Pressione Ctrl+C para sair.")
		select {}
	} else {
		fmt.Println("Ícone adicionado na barra superior do sistema (tray).")
		fmt.Println("Pressione Ctrl+C no terminal para encerrar.")
		// Start Systray (blocks main thread)
		tray := ui.NewTrayApp(node)
		tray.Start()
	}
}

func runStatus(args []string) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Erro ao carregar configurações: %v", err)
	}

	// Try querying local daemon if running
	daemonURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/status", cfg.ListenPort)
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(daemonURL)

	daemonRunning := err == nil && resp.StatusCode == http.StatusOK
	if resp != nil {
		_ = resp.Body.Close()
	}

	statusStr := "Parado (offline)"
	if daemonRunning {
		statusStr = "Ativo (online)"
	}

	fmt.Printf("=== Estado do OmniDesk ===\n")
	fmt.Printf("Status do Daemon:   %s\n", statusStr)
	fmt.Printf("Nome do Nó:         %s\n", cfg.DeviceName)
	fmt.Printf("Identificador (ID): %s\n", cfg.DeviceID)
	fmt.Printf("Porta de Escuta:    %d\n", cfg.ListenPort)
	fmt.Printf("Pasta de Recebidos: %s\n", cfg.DownloadDir)
	fmt.Printf("Sincronizar Texto:  %t\n", cfg.IsClipboardSyncEnabled())

	trusted := cfg.ListTrustedDevices()
	fmt.Printf("\nDispositivos Confiáveis Cadastrados (%d):\n", len(trusted))
	if len(trusted) == 0 {
		fmt.Println("  (Nenhum dispositivo pareado ainda. Use 'omnidesk pair' para conectar).")
	} else {
		for _, dev := range trusted {
			lastSeen := "Nunca"
			if !dev.LastSeen.IsZero() {
				lastSeen = dev.LastSeen.Format("02/01/2006 15:04:05")
			}
			fmt.Printf("  * %s (ID: %s) | Último endereço: %s | Visto em: %s\n",
				dev.Name, dev.ID, dev.LastAddr, lastSeen)
		}
	}
}

func runDevices(args []string) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Erro ao carregar configurações: %v", err)
	}

	fmt.Println("Buscando dispositivos OmniDesk na rede local (mDNS + Varredura de sub-rede)...")

	peerMap := make(map[string]discovery.DiscoveredPeer)

	// 1. Fast Subnet Probe (works over all Wi-Fi routers)
	probeCtx, cancelProbe := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelProbe()
	for _, p := range discovery.ProbeSubnet(probeCtx, cfg.ListenPort, cfg.DeviceID) {
		peerMap[p.ID] = p
	}

	// 2. mDNS Discovery
	mdnsCtx, cancelMdns := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelMdns()
	disc := discovery.NewService(cfg.DeviceID, cfg.DeviceName, cfg.ListenPort, nil)
	_ = disc.Start(mdnsCtx)
	<-mdnsCtx.Done()
	disc.Stop()

	for _, p := range disc.GetPeers() {
		peerMap[p.ID] = p
	}

	fmt.Printf("\nDispositivos encontrados na LAN (%d):\n", len(peerMap))
	if len(peerMap) == 0 {
		fmt.Println("  Nenhum outro nó OmniDesk detectado.")
		fmt.Println("  Dica: você pode parear diretamente usando o IP do outro computador:")
		fmt.Println("  Exemplo: omnidesk pair <ip-do-outro-computador>")
		return
	}

	for _, p := range peerMap {
		status := "NÃO PAREADO"
		if _, ok := cfg.GetTrustedDevice(p.ID); ok {
			status = "PAREADO / CONFIÁVEL"
		}
		fmt.Printf("  * %-20s [ID: %s] Endereço: %-21s Status: %s\n", p.Name, p.ID, p.Addr, status)
	}
}

func runPair(args []string) {
	fs := flag.NewFlagSet("pair", flag.ExitOnError)
	approveID := fs.String("approve", "", "ID da sessão para aprovar no dispositivo de destino")
	pinFlag := fs.String("pin", "", "PIN de 6 dígitos para autorizar o pareamento")
	_ = fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Erro ao carregar configurações: %v", err)
	}

	// Case 1: Approve using --pin <PIN>
	if *pinFlag != "" {
		approveViaDaemon(cfg.ListenPort, *pinFlag)
		return
	}

	remaining := fs.Args()

	// Case 2: No arguments -> Check daemon for pending pairing requests interactively
	if len(remaining) == 0 && *approveID == "" {
		checkPendingViaDaemon(cfg.ListenPort)
		return
	}

	// Case 3: First argument is a 6-digit numeric PIN (e.g. "omnidesk pair 582914")
	if len(remaining) == 1 && isNumericPin(remaining[0]) {
		approveViaDaemon(cfg.ListenPort, remaining[0])
		return
	}

	// Case 4: Legacy --approve <session-id>
	if *approveID != "" {
		approveViaDaemon(cfg.ListenPort, *approveID)
		return
	}

	pMgr := core.NewNode(cfg).PairingMgr
	target := remaining[0]
	var targetAddr string

	// If target looks like IP:Port, use directly
	if strings.Contains(target, ":") {
		targetAddr = target
	} else if net.ParseIP(target) != nil {
		// Plain IP passed, append default port
		targetAddr = fmt.Sprintf("%s:%d", target, cfg.ListenPort)
	} else {
		// Discover device by name: Try subnet probe first
		fmt.Printf("Procurando nó '%s' na rede...\n", target)
		probeCtx, cancelProbe := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancelProbe()
		for _, p := range discovery.ProbeSubnet(probeCtx, cfg.ListenPort, cfg.DeviceID) {
			if strings.EqualFold(p.Name, target) || strings.EqualFold(p.ID, target) {
				targetAddr = p.Addr
				break
			}
		}

		if targetAddr == "" {
			disc := discovery.NewService(cfg.DeviceID, cfg.DeviceName, cfg.ListenPort, nil)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = disc.Start(ctx)
			<-ctx.Done()
			disc.Stop()

			for _, p := range disc.GetPeers() {
				if strings.EqualFold(p.Name, target) || strings.EqualFold(p.ID, target) {
					targetAddr = p.Addr
					break
				}
			}
		}
	}

	if targetAddr == "" {
		fmt.Fprintf(os.Stderr, "Dispositivo '%s' não encontrado na rede local.\n", target)
		fmt.Fprintf(os.Stderr, "Dica: Tente conectar diretamente pelo IP: omnidesk pair <ip-do-dispositivo>\n")
		os.Exit(1)
	}

	localIP := getOutboundIP(targetAddr)
	myAddr := fmt.Sprintf("%s:%d", localIP, cfg.ListenPort)
	fmt.Printf("Conectando ao nó %s para solicitar pareamento...\n", targetAddr)

	session, err := pMgr.InitiatePairing(targetAddr, myAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Falha na solicitação de pareamento: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=======================================================")
	fmt.Printf("   PIN DE PAREAMENTO GERADO: [ %s ]\n", session.PIN)
	fmt.Println("=======================================================")
	fmt.Printf("Confirme no dispositivo de destino (%s) para autorizar.\n", session.ResponderName)
	fmt.Println("Aguardando confirmação...")

	// Poll until approved or timeout (max 90 seconds)
	for i := 0; i < 45; i++ {
		time.Sleep(2 * time.Second)
		err := pMgr.CompletePairing(targetAddr, session)
		if err == nil {
			fmt.Println("\n=======================================================")
			fmt.Printf("✓ PAREAMENTO COM '%s' CONCLUÍDO COM SUCESSO!\n", session.ResponderName)
			fmt.Println("=======================================================")
			fmt.Println("Para ativar a sincronização contínua de clipboard e arquivos,")
			fmt.Println("inicie o OmniDesk agora executando:")
			fmt.Println("  ./omnidesk-mac")
			fmt.Println("=======================================================")
			return
		}
	}

	fmt.Println("\nTempo limite de pareamento esgotado sem confirmação.")
}

func runSend(args []string) {
	if len(args) < 2 {
		fmt.Println("Uso: omnidesk send <caminho-do-arquivo> <dispositivo-alvo>")
		return
	}

	filePath := args[0]
	target := args[1]

	if _, err := os.Stat(filePath); err != nil {
		fmt.Fprintf(os.Stderr, "Arquivo não encontrado: %s\n", filePath)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Erro ao carregar configurações: %v", err)
	}

	// Find target in trusted devices
	var targetDev *config.TrustedDevice
	for _, dev := range cfg.ListTrustedDevices() {
		if strings.EqualFold(dev.Name, target) || strings.EqualFold(dev.ID, target) {
			copyDev := dev
			targetDev = &copyDev
			break
		}
	}

	if targetDev == nil {
		fmt.Fprintf(os.Stderr, "Dispositivo '%s' não encontrado na lista de dispositivos confiáveis.\nExecute 'omnidesk status' para ver os nós pareados.\n", target)
		os.Exit(1)
	}

	// Discover target address on LAN
	targetAddr := targetDev.LastAddr
	fmt.Printf("Localizando endereço de '%s' na rede...\n", targetDev.Name)

	disc := discovery.NewService(cfg.DeviceID, cfg.DeviceName, cfg.ListenPort, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = disc.Start(ctx)
	<-ctx.Done()
	disc.Stop()

	for _, p := range disc.GetPeers() {
		if p.ID == targetDev.ID {
			targetAddr = p.Addr
			break
		}
	}

	if targetAddr == "" {
		fmt.Fprintf(os.Stderr, "Dispositivo '%s' está offline ou inacessível no momento.\n", targetDev.Name)
		os.Exit(1)
	}

	client := transfer.NewClient(cfg.DeviceID)
	fmt.Printf("Enviando '%s' para %s (%s)...\n", filepath.Base(filePath), targetDev.Name, targetAddr)

	res, err := client.SendFile(context.Background(), filePath, targetAddr, targetDev.ID, targetDev.Token, func(current, total int64) {
		pct := float64(current) / float64(total) * 100
		fmt.Printf("\rProgresso: %.1f%% (%d / %d bytes)", pct, current, total)
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nFalha no envio: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Arquivo '%s' enviado com sucesso para %s! (%d bytes salvos)\n", res.Filename, targetDev.Name, res.BytesSaved)
}

func isNumericPin(s string) bool {
	clean := strings.ReplaceAll(strings.TrimSpace(s), " ", "")
	if len(clean) != 6 {
		return false
	}
	for _, r := range clean {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func getOutboundIP(target string) string {
	conn, err := net.Dial("udp", target)
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func approveViaDaemon(port int, pin string) {
	cleanPIN := strings.ReplaceAll(strings.TrimSpace(pin), " ", "")
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/pair/approve-pin", port)
	reqBody, _ := json.Marshal(map[string]string{"pin": cleanPIN})

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao contatar daemon local: %v\nO OmniDesk precisa estar em execução nesta máquina ('omnidesk daemon').\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Falha na aprovação: %s\n", strings.TrimSpace(string(b)))
		return
	}

	var res struct {
		Success    bool   `json:"success"`
		DeviceName string `json:"device_name"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)

	fmt.Println("=======================================================")
	fmt.Printf("✓ PAREAMENTO APROVADO COM SUCESSO!\n")
	fmt.Printf("O dispositivo '%s' agora é confiável e conectado.\n", res.DeviceName)
	fmt.Println("=======================================================")
}

func checkPendingViaDaemon(port int) {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/pair/pending", port)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("Uso: omnidesk pair <ip-ou-nome-do-dispositivo> (para solicitar pareamento)")
		fmt.Println("     omnidesk pair <PIN-de-6-digitos>          (para autorizar pareamento)")
		return
	}
	defer resp.Body.Close()

	var pending []*pairing.Session
	if err := json.NewDecoder(resp.Body).Decode(&pending); err != nil || len(pending) == 0 {
		fmt.Println("Nenhuma solicitação de pareamento pendente encontrada no momento.")
		fmt.Println("\nUso:")
		fmt.Println("  omnidesk pair <ip-ou-nome>        (conectar a outro nó)")
		fmt.Println("  omnidesk pair <PIN-de-6-digitos>  (autorizar pareamento)")
		return
	}

	for _, s := range pending {
		fmt.Println("=======================================================")
		fmt.Printf("SOLICITAÇÃO DE PAREAMENTO PENDENTE:\n")
		fmt.Printf("  * Dispositivo: %s (ID: %s)\n", s.RequesterName, s.RequesterID)
		fmt.Printf("  * PIN:         [ %s ]\n", s.PIN)
		fmt.Println("=======================================================")
		fmt.Print("Deseja autorizar esta máquina? (S/n): ")

		reader := bufio.NewReader(os.Stdin)
		ans, _ := reader.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))

		if ans == "" || ans == "s" || ans == "sim" || ans == "y" || ans == "yes" {
			approveViaDaemon(port, s.PIN)
		} else {
			fmt.Println("Pareamento rejeitado.")
		}
	}
}
