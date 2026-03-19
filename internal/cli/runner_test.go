package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestConvertHTMLImagesToMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`<img src="http://example.com/img.jpg">`, `![](http://example.com/img.jpg)`},
		{`<img src="https://example.com/abc.png" alt="test"/>`, `![](https://example.com/abc.png)`},
		{`<img alt="no src"/><img src="http://test.com/1.jpg"/>`, `<img alt="no src"/>![](http://test.com/1.jpg)`},
	}

	for _, tt := range tests {
		t.Run(tt.input[:min(40, len(tt.input))], func(t *testing.T) {
			got := convertHTMLImagesToMarkdown(tt.input)
			if got != tt.expected {
				t.Errorf("convertHTMLImagesToMarkdown(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"paragraph", "<p>Hello World</p>", "Hello World"},
		{"link", "<a href=\"http://example.com\">Link Text</a>", "Link Text"},
		{"nested", "<div><p>Nested <strong>text</strong></p></div>", "Nested"},
		{"script removed", "<script>alert('xss')</script>content", "content"},
		{"style removed", "<style>body{color:red}</style>text", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTMLTags(tt.input)
			if tt.contains != "" && !contains(got, tt.contains) {
				t.Errorf("stripHTMLTags(%q) = %q, should contain %q", tt.input, got, tt.contains)
			}
		})
	}
}

func TestStripHTMLTags_Br(t *testing.T) {
	input := "Line1<br>Line2"
	got := stripHTMLTags(input)
	if !contains(got, "Line1") || !contains(got, "Line2") {
		t.Errorf("stripHTMLTags(%q) = %q, should contain Line1 and Line2", input, got)
	}
}

func TestPrintJSONError(t *testing.T) {
	// Just verify it doesn't panic
	printJSONError("test error message")
}

func TestJSONResult_Structure(t *testing.T) {
	result := JSONResult{
		Success:   true,
		VideoID:   "abc123",
		Platform:  "xiaoyuzhou",
		AudioPath: "/path/to/audio.m4a",
		Metadata:  nil,
	}

	if !result.Success {
		t.Error("JSONResult.Success = false, want true")
	}
	if result.VideoID != "abc123" {
		t.Errorf("JSONResult.VideoID = %q, want %q", result.VideoID, "abc123")
	}
}

func TestJSONResult_Error(t *testing.T) {
	result := JSONResult{
		Success: false,
		Error:   "something went wrong",
	}

	if result.Success {
		t.Error("JSONResult.Success = true, want false")
	}
	if result.Error != "something went wrong" {
		t.Errorf("JSONResult.Error = %q, want %q", result.Error, "something went wrong")
	}
}

func TestRunner_NewRunner(t *testing.T) {
	opts := DownloadOptions{
		URL:      "https://example.com",
		OutputDir: "/tmp",
		Verbose:  true,
	}

	runner := NewRunner(opts)
	if runner == nil {
		t.Fatal("NewRunner returned nil")
	}
	if runner.opts.URL != opts.URL {
		t.Errorf("runner.opts.URL = %q, want %q", runner.opts.URL, opts.URL)
	}
	if runner.opts.Verbose != opts.Verbose {
		t.Errorf("runner.opts.Verbose = %v, want %v", runner.opts.Verbose, opts.Verbose)
	}
}

func TestRunner_OutputDirCreation(t *testing.T) {
	// Create a temp directory that doesn't exist
	tmpDir := filepath.Join(os.TempDir(), "test-runner-output-dir", "nested")
	defer os.RemoveAll(filepath.Dir(tmpDir))

	// Verify directory doesn't exist
	_, err := os.Stat(tmpDir)
	if !os.IsNotExist(err) {
		t.Skip("tmp dir already exists, skipping")
	}

	// MkdirAll should succeed
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestAudioURLResolverInterface(t *testing.T) {
	// Test that the interface check works correctly
	// XiaoyuzhouAdapter implements AudioURLResolver
	xiaoyuzhouAdapter := &xiaoyuzhouMockAdapter{}

	// This should satisfy the AudioURLResolver interface
	var resolver interface {
		ResolveAudioURL(ctx context.Context, episodeID string) (string, error)
	}
	resolver = xiaoyuzhouAdapter

	if resolver == nil {
		t.Error("interface assignment failed")
	}
}

type xiaoyuzhouMockAdapter struct{}

func (a *xiaoyuzhouMockAdapter) ResolveAudioURL(ctx context.Context, episodeID string) (string, error) {
	return "https://media.xyzcdn.net/test.m4a", nil
}

func TestDownloadOptions_Validation(t *testing.T) {
	opts := DownloadOptions{
		URL:       "https://xiaoyuzhoufm.com/episode/abc123",
		OutputDir: "",
		JSON:      false,
		Verbose:   false,
	}

	runner := NewRunner(opts)
	if runner == nil {
		t.Fatal("NewRunner returned nil")
	}

	// Verify default output dir handling
	if runner.opts.OutputDir != "" {
		t.Errorf("runner.opts.OutputDir = %q, want empty string", runner.opts.OutputDir)
	}
}
