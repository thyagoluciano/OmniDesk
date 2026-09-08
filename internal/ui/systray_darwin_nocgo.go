//go:build darwin && !cgo

package ui

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"crossover/internal/core"
)

type TrayApp struct {
	node *core.Node
}

func NewTrayApp(node *core.Node) *TrayApp {
	return &TrayApp{node: node}
}

func (t *TrayApp) Start() {
	log.Println("[ui] Menu de bandeja (Systray) indisponível em build cruzada sem CGO. Executando em modo daemon.")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	t.node.Stop()
}
