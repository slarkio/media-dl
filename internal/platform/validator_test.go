package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCookiePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		setup     func(string) error
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "absolute path to existing file",
			path:      "/tmp/cookies.txt",
			setup:     func(p string) error { return os.WriteFile(p, []byte("test"), 0644) },
			shouldErr: false,
		},
		{
			name:      "relative path rejected",
			path:      "./cookies.txt",
			shouldErr: true,
			errMsg:    "cookie path must be absolute",
		},
		{
			name:      "path traversal detected",
			path:      "/tmp/../etc/passwd",
			shouldErr: true,
			errMsg:    "cookie path traversal detected",
		},
		{
			name:      "non-existent file",
			path:      "/nonexistent/cookies.txt",
			shouldErr: true,
			errMsg:    "cookie file not accessible",
		},
		{
			name:      "directory instead of file",
			path:      "/tmp",
			shouldErr: true,
			errMsg:    "cookie path must be a regular file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				if err := tt.setup(tt.path); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				defer os.Remove(tt.path)
			}

			err := ValidateCookiePath(tt.path)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateAudioURL(t *testing.T) {
	tests := []struct {
		name      string
		rawURL    string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid HTTPS URL with correct host",
			rawURL:    "https://media.xyzcdn.net/abc/test.m4a",
			shouldErr: false,
		},
		{
			name:      "HTTP scheme rejected",
			rawURL:    "http://media.xyzcdn.net/abc/test.m4a",
			shouldErr: true,
			errMsg:    "audio URL must use HTTPS",
		},
		{
			name:      "wrong host rejected",
			rawURL:    "https://other.host.com/abc/test.m4a",
			shouldErr: true,
			errMsg:    "audio URL has invalid host",
		},
		{
			name:      "no scheme rejected",
			rawURL:    "not-a-url",
			shouldErr: true,
			errMsg:    "audio URL must use HTTPS",
		},
		{
			name:      "empty URL",
			rawURL:    "",
			shouldErr: true,
			errMsg:    "audio URL must use HTTPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateAudioURL(tt.rawURL)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateBilibiliRedirect(t *testing.T) {
	tests := []struct {
		name      string
		finalURL  string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "bilibili.com valid",
			finalURL:  "https://www.bilibili.com/video/BV123456789",
			shouldErr: false,
		},
		{
			name:      "www.bilibili.com valid",
			finalURL:  "https://www.bilibili.com/video/BV123456789",
			shouldErr: false,
		},
		{
			name:      "video.bilibili.com subdomain valid",
			finalURL:  "https://video.bilibili.com/video/BV123456789",
			shouldErr: false,
		},
		{
			name:      "b23.tv short URL valid",
			finalURL:  "https://b23.tv/abc123",
			shouldErr: false,
		},
		{
			name:      "b23.tv subdomain valid",
			finalURL:  "https://test.b23.tv/abc123",
			shouldErr: false,
		},
		{
			name:      "external domain rejected",
			finalURL:  "https://example.com/video/BV123",
			shouldErr: true,
			errMsg:    "redirect URL outside Bilibili domain",
		},
		{
			name:      "invalid host rejected",
			finalURL:  "not-a-url",
			shouldErr: true,
			errMsg:    "redirect URL outside Bilibili domain",
		},
		{
			name:      "domain suffix bypass blocked - bilibili.com.example.com",
			finalURL:  "https://bilibili.com.example.com/video/BV123",
			shouldErr: true,
			errMsg:    "redirect URL outside Bilibili domain",
		},
		{
			name:      "domain suffix bypass blocked - www.bilibili.com.example.com",
			finalURL:  "https://www.bilibili.com.example.com/video/BV123",
			shouldErr: true,
			errMsg:    "redirect URL outside Bilibili domain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBilibiliRedirect(tt.finalURL)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateOutputDir(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		dir       string
		setup     func(string) (string, error)
		shouldErr bool
	}{
		{
			name:      "existing directory",
			dir:       tmpDir,
			shouldErr: false,
		},
		{
			name:      "relative path resolved",
			dir:       ".",
			shouldErr: false,
		},
		{
			name:      "nonexistent directory",
			dir:       "/nonexistent/path/to/dir",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.dir
			if tt.setup != nil {
				resolved, err := tt.setup(dir)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				dir = resolved
			}

			_, err := ValidateOutputDir(dir)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateOutputDir_SymlinkResolution(t *testing.T) {
	tmpDir := t.TempDir()
	realDir := filepath.Join(tmpDir, "realdir")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatalf("failed to create real dir: %v", err)
	}

	symlinkPath := filepath.Join(tmpDir, "link")
	if err := os.Symlink(realDir, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	resolved, err := ValidateOutputDir(symlinkPath)
	if err != nil {
		t.Errorf("unexpected error resolving symlink: %v", err)
	}

	realResolved, _ := filepath.EvalSymlinks(symlinkPath)
	if resolved != realResolved {
		t.Errorf("resolved path = %q, want %q", resolved, realResolved)
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
