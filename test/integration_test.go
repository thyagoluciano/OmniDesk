package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"crossover/internal/config"
	"crossover/internal/core"
	"crossover/internal/discovery"
)

func TestFullE2EFlow(t *testing.T) {
	tempA := t.TempDir()
	tempB := t.TempDir()

	cfgA := &config.Config{
		DeviceID:       "node-macbook-1",
		DeviceName:     "MacBook Pro 1",
		ListenPort:     24891,
		DownloadDir:    filepath.Join(tempA, "downloads"),
		ClipboardSync:  true,
		TrustedDevices: make(map[string]config.TrustedDevice),
	}

	cfgB := &config.Config{
		DeviceID:       "node-linux-desktop",
		DeviceName:     "Ubuntu Desktop",
		ListenPort:     24892,
		DownloadDir:    filepath.Join(tempB, "downloads"),
		ClipboardSync:  true,
		TrustedDevices: make(map[string]config.TrustedDevice),
	}

	_ = os.MkdirAll(cfgA.DownloadDir, 0755)
	_ = os.MkdirAll(cfgB.DownloadDir, 0755)

	nodeA := core.NewNode(cfgA)
	nodeB := core.NewNode(cfgB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := nodeA.Start(ctx); err != nil {
		t.Fatalf("failed to start Node A: %v", err)
	}
	defer nodeA.Stop()

	if err := nodeB.Start(ctx); err != nil {
		t.Fatalf("failed to start Node B: %v", err)
	}
	defer nodeB.Stop()

	// 1. Handshake & PIN Pairing
	peerBAddr := fmt.Sprintf("127.0.0.1:%d", cfgB.ListenPort)
	peerAAddr := fmt.Sprintf("127.0.0.1:%d", cfgA.ListenPort)

	sess, err := nodeA.PairingMgr.InitiatePairing(peerBAddr, peerAAddr)
	if err != nil {
		t.Fatalf("pairing initiation failed: %v", err)
	}
	if len(sess.PIN) != 6 {
		t.Fatalf("expected 6 digit PIN, got %s", sess.PIN)
	}

	// Node B approves
	if err := nodeB.PairingMgr.ApproveSession(sess.ID); err != nil {
		t.Fatalf("node B failed to approve session: %v", err)
	}

	// Node A completes pairing
	if err := nodeA.PairingMgr.CompletePairing(peerBAddr, sess); err != nil {
		t.Fatalf("node A failed to complete pairing: %v", err)
	}

	// Verify mutual trust
	devB, okB := cfgA.GetTrustedDevice("node-linux-desktop")
	_, okA := cfgB.GetTrustedDevice("node-macbook-1")
	if !okB || !okA {
		t.Fatalf("devices should be mutually trusted: okA=%v, okB=%v", okA, okB)
	}

	// 2. Clipboard Synchronization (Node A -> Node B)
	// Simulate Node B discovery in Node A
	nodeA.OnPeerFound(discovery.DiscoveredPeer{
		ID:       cfgB.DeviceID,
		Name:     cfgB.DeviceName,
		Addr:     peerBAddr,
		LastSeen: time.Now(),
	})

	testText := "Texto compartilhado entre MacBook e Linux!"
	nodeA.ClipEngine.HandleLocalCopy(testText)

	// Wait briefly for network propagation
	time.Sleep(100 * time.Millisecond)

	// 3. File Transfer (Node A -> Node B)
	srcFile := filepath.Join(tempA, "projeto.zip")
	filePayload := []byte("PK\x03\x04conteudo do arquivo compactado de teste")
	if err := os.WriteFile(srcFile, filePayload, 0644); err != nil {
		t.Fatalf("failed to create sample file: %v", err)
	}

	uploadRes, err := nodeA.TransferCli.SendFile(ctx, srcFile, peerBAddr, devB.ID, devB.Token, nil)
	if err != nil {
		t.Fatalf("file upload failed: %v", err)
	}

	if uploadRes.Filename != "projeto.zip" {
		t.Errorf("unexpected uploaded filename: %s", uploadRes.Filename)
	}

	// Verify received file on Node B
	receivedFile := filepath.Join(cfgB.DownloadDir, "projeto.zip")
	receivedBytes, err := os.ReadFile(receivedFile)
	if err != nil {
		t.Fatalf("failed to read received file on Node B: %v", err)
	}
	if string(receivedBytes) != string(filePayload) {
		t.Errorf("file payload mismatch")
	}

	// 4. File collision test: send again, verify it becomes projeto (1).zip
	uploadRes2, err := nodeA.TransferCli.SendFile(ctx, srcFile, peerBAddr, devB.ID, devB.Token, nil)
	if err != nil {
		t.Fatalf("second file upload failed: %v", err)
	}
	if uploadRes2.Filename != "projeto (1).zip" {
		t.Errorf("expected duplicate name 'projeto (1).zip', got %s", uploadRes2.Filename)
	}

	t.Log("✓ E2E flow passed: pairing, clipboard propagation, and file streaming with deduplication.")
}
