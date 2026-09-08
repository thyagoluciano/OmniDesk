package transfer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileTransferStream(t *testing.T) {
	tempDir := t.TempDir()
	srcFile := filepath.Join(tempDir, "sample.txt")
	testContent := "Test file content streaming peer to peer!"
	if err := os.WriteFile(srcFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var receivedContent string
	var receivedToken string
	var receivedDeviceID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedDeviceID = r.Header.Get("X-OmniDesk-Device-ID")
		receivedToken = r.Header.Get("X-OmniDesk-Token")

		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		receivedContent = string(data)

		w.Header().Set("Content-Type", "application/json")
		respJSON := fmt.Sprintf(`{"status":"received","filename":"sample.txt","bytes_saved":%d}`, len(data))
		_, _ = w.Write([]byte(respJSON))
	}))
	defer server.Close()

	peerAddr := strings.TrimPrefix(server.URL, "http://")
	client := NewClient("local-sender-1")

	var progressCalled bool
	res, err := client.SendFile(context.Background(), srcFile, peerAddr, "dest-peer", "token-xyz", func(current, total int64) {
		progressCalled = true
	})

	if err != nil {
		t.Fatalf("SendFile failed: %v", err)
	}

	if res.Filename != "sample.txt" {
		t.Errorf("unexpected filename in result: %s", res.Filename)
	}
	if receivedContent != testContent {
		t.Errorf("received content mismatch: got %q, want %q", receivedContent, testContent)
	}
	if receivedDeviceID != "local-sender-1" || receivedToken != "token-xyz" {
		t.Errorf("auth headers mismatch: dev=%s, token=%s", receivedDeviceID, receivedToken)
	}
	if !progressCalled {
		t.Errorf("expected progress callback to be invoked")
	}
}
