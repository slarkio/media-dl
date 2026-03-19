package platform

import (
	"context"
	"testing"
)

func TestYouTubeAdapter_Detect(t *testing.T) {
	adapter := NewYouTubeAdapter()

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"https://youtu.be/dQw4w9WgXcQ", true},
		{"https://www.youtube.com/embed/dQw4w9WgXcQ", true},
		{"https://www.youtube.com/v/dQw4w9WgXcQ", true},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=xxx", true},
		{"https://www.bilibili.com/video/BV1xx411c7XD", false},
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

func TestYouTubeAdapter_GetID(t *testing.T) {
	adapter := NewYouTubeAdapter()

	tests := []struct {
		url      string
		expected string
		hasError bool
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"https://www.youtube.com/embed/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
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

func TestYouTubeAdapter_Name(t *testing.T) {
	adapter := NewYouTubeAdapter()
	if got := adapter.Name(); got != "youtube" {
		t.Errorf("Name() = %v, want youtube", got)
	}
}

func TestYouTubeAdapter_RequiresCookie(t *testing.T) {
	adapter := NewYouTubeAdapter()
	if adapter.RequiresCookie() {
		t.Error("RequiresCookie() = true, want false")
	}
}

func TestYouTubeAdapter_BuildDownloadCommand(t *testing.T) {
	adapter := NewYouTubeAdapter()

	tests := []struct {
		name      string
		mediaID   string
		cookies   string
		outputDir string
		wantNil   bool
	}{
		{
			name:      "valid build",
			mediaID:   "dQw4w9WgXcQ",
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

func TestYouTubeAdapter_BuildMetadataURL(t *testing.T) {
	adapter := NewYouTubeAdapter()

	tests := []struct {
		mediaID string
		want    string
	}{
		{"dQw4w9WgXcQ", "https://www.youtube.com/watch?v=dQw4w9WgXcQ"},
	}

	for _, tt := range tests {
		t.Run(tt.mediaID, func(t *testing.T) {
			if got := adapter.BuildMetadataURL(tt.mediaID); got != tt.want {
				t.Errorf("BuildMetadataURL(%q) = %q, want %q", tt.mediaID, got, tt.want)
			}
		})
	}
}
