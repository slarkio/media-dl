package platform

import (
	"context"
	"errors"
	"testing"
)

func TestBilibiliAdapter_Detect(t *testing.T) {
	adapter := NewBilibiliAdapter()

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://www.bilibili.com/video/BV1xx411c7XD", true},
		{"https://bilibili.com/video/BV1xx411c7XD", true},
		{"https://b23.tv/abc123", true},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", false},
		{"https://xiaoyuzhoufm.com/episode/abc123", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := adapter.Detect(tt.url); got != tt.expected {
				t.Errorf("Detect(%q) = %v, want %v", tt.url, got, tt.expected)
			}
		})
	}
}

func TestBilibiliAdapter_GetID(t *testing.T) {
	adapter := NewBilibiliAdapter()

	tests := []struct {
		url      string
		expected string
		hasError bool
	}{
		{"https://www.bilibili.com/video/BV1xx411c7XD", "BV1xx411c7XD", false},
		{"https://bilibili.com/video/BV1xx411c7XD", "BV1xx411c7XD", false},
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

func TestBilibiliAdapter_Name(t *testing.T) {
	adapter := NewBilibiliAdapter()
	if got := adapter.Name(); got != "bilibili" {
		t.Errorf("Name() = %v, want bilibili", got)
	}
}

func TestBilibiliAdapter_RequiresCookie(t *testing.T) {
	adapter := NewBilibiliAdapter()
	if adapter.RequiresCookie() {
		t.Error("RequiresCookie() = true, want false")
	}
}

func TestBilibiliAdapter_Headers(t *testing.T) {
	adapter := NewBilibiliAdapter()
	headers := adapter.Headers()
	if headers == nil {
		t.Error("Headers() = nil, want map with Referer")
	}
	if headers["Referer"] != "https://www.bilibili.com" {
		t.Errorf("Headers()[Referer] = %q, want %q", headers["Referer"], "https://www.bilibili.com")
	}
}

func TestBilibiliAdapter_BuildDownloadCommand(t *testing.T) {
	adapter := NewBilibiliAdapter()

	tests := []struct {
		name      string
		mediaID   string
		cookies   string
		outputDir string
		wantNil   bool
	}{
		{
			name:      "valid build",
			mediaID:   "BV123456789",
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

func TestBilibiliAdapter_BuildMetadataURL(t *testing.T) {
	adapter := NewBilibiliAdapter()

	tests := []struct {
		mediaID string
		want    string
	}{
		{"BV123456789", "https://www.bilibili.com/video/BV123456789"},
	}

	for _, tt := range tests {
		t.Run(tt.mediaID, func(t *testing.T) {
			if got := adapter.BuildMetadataURL(tt.mediaID); got != tt.want {
				t.Errorf("BuildMetadataURL(%q) = %q, want %q", tt.mediaID, got, tt.want)
			}
		})
	}
}

func TestBilibiliAdapter_ShortURLResolver(t *testing.T) {
	adapter := NewBilibiliAdapter()

	var _ ShortURLResolver = adapter

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := adapter.ResolveShortURL(ctx, "https://b23.tv/abc123")
	if err == nil {
		t.Error("ResolveShortURL with cancelled context should return error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("ResolveShortURL error = %v, want context.Canceled", err)
	}
}
