package downloader

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		errMsg   string
		expected bool
	}{
		{"connection refused", true},
		{"timeout occurred", true},
		{"429 Too Many Requests", true},
		{"500 Internal Server Error", true},
		{"502 Bad Gateway", true},
		{"503 Service Unavailable", true},
		{"file not found", false},
		{"permission denied", false},
		{"invalid argument", false},
	}

	for _, tt := range tests {
		t.Run(tt.errMsg, func(t *testing.T) {
			err := &testError{msg: tt.errMsg}
			got := isRetryableError(err)
			if got != tt.expected {
				t.Errorf("isRetryableError(%q) = %v, want %v", tt.errMsg, got, tt.expected)
			}
		})
	}
}

func TestMetadata_Structure(t *testing.T) {
	meta := Metadata{
		Title:       "Test Title",
		Duration:    120,
		Uploader:    "Test Uploader",
		UploadDate:  "20240101",
		Description: "Test Description",
		Thumbnail:   "https://example.com/thumb.jpg",
	}

	if meta.Title != "Test Title" {
		t.Errorf("Metadata.Title = %q, want %q", meta.Title, "Test Title")
	}
	if meta.Duration != 120 {
		t.Errorf("Metadata.Duration = %d, want %d", meta.Duration, 120)
	}
}

func TestNewYtDLP(t *testing.T) {
	y := New(true)
	if y == nil {
		t.Fatal("New returned nil")
	}
	if !y.verbose {
		t.Error("New(true) y.verbose = false, want true")
	}

	y2 := New(false)
	if y2.verbose {
		t.Error("New(false) y.verbose = true, want false")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

type mockAdapter struct {
	nameVal       string
	headersVal    map[string]string
	mediaID       string
	downloadCmd   *exec.Cmd
	metadataURL   string
	shouldFailCmd bool
}

func (m *mockAdapter) Name() string                       { return m.nameVal }
func (m *mockAdapter) Headers() map[string]string          { return m.headersVal }
func (m *mockAdapter) BuildDownloadCommand(ctx context.Context, mediaID, cookies, outputDir string) *exec.Cmd {
	if m.shouldFailCmd {
		return nil
	}
	return m.downloadCmd
}
func (m *mockAdapter) BuildMetadataURL(mediaID string) string {
	return m.metadataURL
}

func TestParseOutput(t *testing.T) {
	y := &YtDLP{verbose: false}

	tests := []struct {
		name      string
		output    string
		setupFile string
		want      string
		wantErr   bool
	}{
		{
			name:   "direct file path in output",
			output: "/path/to/video.m4a\n",
			want:   "/path/to/video.m4a",
		},
		{
			name:   "file with extension found via glob",
			output: "[download] path/to/file.m4a\n",
			setupFile: "test.m4a",
			want:     "test.m4a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if tt.setupFile != "" {
				fpath := filepath.Join(tmpDir, tt.setupFile)
				os.WriteFile(fpath, []byte("test"), 0644)
			}

			got, err := y.parseOutput(tt.output, tmpDir)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !strings.HasSuffix(got, tt.want) {
					t.Errorf("parseOutput() = %q, want suffix %q", got, tt.want)
				}
			}
		})
	}
}

func TestIsRetryableError_ExitCode(t *testing.T) {
	tests := []struct {
		name  string
		msg   string
		want  bool
	}{
		{"retryable message connection refused", "connection refused", true},
		{"retryable message timeout", "timeout occurred", true},
		{"retryable message 429", "429 Too Many Requests", true},
		{"retryable message 500", "500 Internal Server Error", true},
		{"non-retryable message", "file not found", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &testError{msg: tt.msg}
			if got := isRetryableError(err); got != tt.want {
				t.Errorf("isRetryableError(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

func TestDownload_CommandBuilding(t *testing.T) {
	tmpDir := t.TempDir()

	adapter := &mockAdapter{
		nameVal:     "test",
		mediaID:     "test123",
		metadataURL: "https://example.com/test",
		downloadCmd: exec.Command("echo", "test"),
	}

	cmd := adapter.BuildDownloadCommand(context.Background(), adapter.mediaID, "", tmpDir)
	if cmd == nil {
		t.Fatal("BuildDownloadCommand returned nil")
	}
	if cmd.Args[0] != "echo" {
		t.Errorf("cmd.Args[0] = %q, want %q", cmd.Args[0], "echo")
	}

	url := adapter.BuildMetadataURL(adapter.mediaID)
	if url != adapter.metadataURL {
		t.Errorf("BuildMetadataURL() = %q, want %q", url, adapter.metadataURL)
	}
}

func TestDownloadWithRetry_ContextCancellationDuringBackoff(t *testing.T) {
	y := &YtDLP{verbose: false}

	ctx, cancel := context.WithCancel(context.Background())

	adapter := &mockAdapter{
		nameVal:     "test",
		mediaID:     "test123",
		metadataURL: "https://example.com/test",
		downloadCmd: exec.Command("false"),
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, _, err := y.DownloadWithRetry(ctx, adapter, adapter.mediaID, "", t.TempDir())
	elapsed := time.Since(start)

	if err != context.Canceled {
		t.Errorf("DownloadWithRetry error = %v, want context.Canceled", err)
	}

	if elapsed >= 500*time.Millisecond {
		t.Errorf("DownloadWithRetry returned after %v, expected immediate return on context cancel", elapsed)
	}
}
