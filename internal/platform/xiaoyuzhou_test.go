package platform

import (
	"context"
	"testing"
)

func TestXiaoyuzhouAdapter_Detect(t *testing.T) {
	adapter := NewXiaoyuzhouAdapter()

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://www.xiaoyuzhoufm.com/episode/abc123", true},
		{"https://xiaoyuzhoufm.com/episode/abc123", true},
		{"https://www.xiaoyuzhoufm.com/episode/abc-123_456", true},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", false},
		{"https://www.bilibili.com/video/BV1xx411c7XD", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := adapter.Detect(tt.url); got != tt.expected {
				t.Errorf("Detect(%q) = %v, want %v", tt.url, got, tt.expected)
			}
		})
	}
}

func TestXiaoyuzhouAdapter_GetID(t *testing.T) {
	adapter := NewXiaoyuzhouAdapter()

	tests := []struct {
		url      string
		expected string
		hasError bool
	}{
		{"https://www.xiaoyuzhoufm.com/episode/abc123", "abc123", false},
		{"https://xiaoyuzhoufm.com/episode/abc-123_456", "abc-123_456", false},
		{"https://invalid.url.com", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			id, err := adapter.GetID(tt.url)
			if (err != nil) != tt.hasError {
				t.Errorf("GetID(%q) error = %v, hasError %v", tt.url, err, tt.hasError)
				return
			}
			if id != tt.expected {
				t.Errorf("GetID(%q) = %q, want %q", tt.url, id, tt.expected)
			}
		})
	}
}

func TestXiaoyuzhouAdapter_Name(t *testing.T) {
	adapter := NewXiaoyuzhouAdapter()
	if got := adapter.Name(); got != "xiaoyuzhou" {
		t.Errorf("Name() = %v, want xiaoyuzhou", got)
	}
}

func TestXiaoyuzhouAdapter_RequiresCookie(t *testing.T) {
	adapter := NewXiaoyuzhouAdapter()
	if adapter.RequiresCookie() {
		t.Error("RequiresCookie() = true, want false")
	}
}

func TestXiaoyuzhouAdapter_Headers(t *testing.T) {
	adapter := NewXiaoyuzhouAdapter()
	if headers := adapter.Headers(); headers != nil {
		t.Errorf("Headers() = %v, want nil", headers)
	}
}

func TestXiaoyuzhouAdapter_BuildDownloadCommand(t *testing.T) {
	adapter := NewXiaoyuzhouAdapter()

	tests := []struct {
		name      string
		mediaID   string
		cookies   string
		outputDir string
		wantNil   bool
	}{
		{
			name:      "valid build",
			mediaID:   "https://media.xyzcdn.net/abc/test.m4a",
			cookies:   "",
			outputDir: "/tmp/downloads",
			wantNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := adapter.BuildDownloadCommand(context.Background(), tt.mediaID, tt.cookies, tt.outputDir)
			if tt.wantNil {
				if cmd != nil {
					t.Errorf("BuildDownloadCommand() = %v, want nil", cmd)
				}
				return
			}
			if cmd == nil {
				t.Error("BuildDownloadCommand() = nil, want non-nil")
				return
			}
			if cmd.Args[0] != "yt-dlp" {
				t.Errorf("cmd.Args[0] = %q, want %q", cmd.Args[0], "yt-dlp")
			}
		})
	}
}

func TestXiaoyuzhouAdapter_BuildMetadataURL(t *testing.T) {
	adapter := NewXiaoyuzhouAdapter()

	tests := []struct {
		mediaID string
		want    string
	}{
		{"https://media.xyzcdn.net/abc/test.m4a", "https://media.xyzcdn.net/abc/test.m4a"},
	}

	for _, tt := range tests {
		t.Run(tt.mediaID, func(t *testing.T) {
			if got := adapter.BuildMetadataURL(tt.mediaID); got != tt.want {
				t.Errorf("BuildMetadataURL(%q) = %q, want %q", tt.mediaID, got, tt.want)
			}
		})
	}
}
