package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// ProgressWriter wraps an io.Reader to report transfer progress.
type ProgressReader struct {
	reader     io.Reader
	total      int64
	current    int64
	onProgress func(current, total int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)
	if pr.onProgress != nil {
		pr.onProgress(pr.current, pr.total)
	}
	return n, err
}

// Client manages outbound file uploads to peer nodes.
type Client struct {
	myDeviceID string
	httpClient *http.Client
}

// NewClient creates a new file transfer client.
func NewClient(myDeviceID string) *Client {
	return &Client{
		myDeviceID: myDeviceID,
		httpClient: &http.Client{
			Timeout: 1 * time.Hour, // Allow large file streaming
		},
	}
}

// UploadResult contains information returned by the recipient node.
type UploadResult struct {
	Status     string `json:"status"`
	Filename   string `json:"filename"`
	BytesSaved int64  `json:"bytes_saved"`
}

// SendFile streams a local file to a remote peer.
func (c *Client) SendFile(ctx context.Context, filePath, peerAddr, peerID, token string, onProgress func(current, total int64)) (*UploadResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("cannot read file stats: %w", err)
	}

	fileName := filepath.Base(filePath)
	return c.SendStream(ctx, file, stat.Size(), fileName, peerAddr, peerID, token, onProgress)
}

// SendStream streams an io.Reader to a remote peer.
func (c *Client) SendStream(ctx context.Context, body io.Reader, size int64, fileName, peerAddr, peerID, token string, onProgress func(current, total int64)) (*UploadResult, error) {
	endpoint := fmt.Sprintf("http://%s/api/v1/files/upload?filename=%s", peerAddr, url.QueryEscape(fileName))

	var bodyReader io.Reader = body
	if onProgress != nil && size > 0 {
		bodyReader = &ProgressReader{
			reader:     body,
			total:      size,
			onProgress: onProgress,
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("cannot build upload request: %w", err)
	}

	if size > 0 {
		req.ContentLength = size
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-OmniDesk-Device-ID", c.myDeviceID)
	req.Header.Set("X-OmniDesk-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("peer rejected file (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var res UploadResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse peer response: %w", err)
	}

	return &res, nil
}
