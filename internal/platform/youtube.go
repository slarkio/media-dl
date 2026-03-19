package platform

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
)

var (
	youtubePatterns = []*regexp.Regexp{
		regexp.MustCompile(`youtube\.com/watch\?v=([a-zA-Z0-9_-]{11})`),
		regexp.MustCompile(`youtu\.be/([a-zA-Z0-9_-]{11})`),
		regexp.MustCompile(`youtube\.com/embed/([a-zA-Z0-9_-]{11})`),
		regexp.MustCompile(`youtube\.com/v/([a-zA-Z0-9_-]{11})`),
	}
)

type YouTubeAdapter struct{}

func NewYouTubeAdapter() *YouTubeAdapter {
	return &YouTubeAdapter{}
}

func (a *YouTubeAdapter) Detect(urlStr string) bool {
	for _, pattern := range youtubePatterns {
		if pattern.MatchString(urlStr) {
			return true
		}
	}
	return false
}

func (a *YouTubeAdapter) GetID(urlStr string) (string, error) {
	for _, pattern := range youtubePatterns {
		if pattern.MatchString(urlStr) {
			return extractVideoID(pattern, urlStr)
		}
	}
	return "", errors.New("invalid YouTube URL")
}

func (a *YouTubeAdapter) Name() string {
	return "youtube"
}

func (a *YouTubeAdapter) RequiresCookie() bool {
	return false
}

func (a *YouTubeAdapter) Headers() map[string]string {
	return nil
}

func (a *YouTubeAdapter) GetConfig() PlatformConfig {
	return PlatformConfig{
		RequiresCookie: false,
		Headers:        nil,
	}
}

func extractVideoID(pattern *regexp.Regexp, urlStr string) (string, error) {
	matches := pattern.FindStringSubmatch(urlStr)
	if len(matches) < 2 {
		return "", errors.New("video ID not found")
	}
	return matches[1], nil
}

func (a *YouTubeAdapter) BuildDownloadCommand(ctx context.Context, mediaID, cookies, outputDir string) *exec.Cmd {
	args := []string{
		"--no-playlist",
		"-f", "bestaudio[ext=m4a]/bestaudio",
		"--audio-format", "m4a",
		"--audio-quality", "0",
		"--embed-metadata",
		"--print", "after_move:filepath",
		"-o", filepath.Join(outputDir, "%(title)s.%(ext)s"),
	}
	if cookies != "" {
		if err := ValidateCookiePath(cookies); err != nil {
			return nil
		}
		args = append(args, "--cookies", cookies)
	}
	args = append(args, fmt.Sprintf("https://www.youtube.com/watch?v=%s", mediaID))
	return exec.CommandContext(ctx, "yt-dlp", args...)
}

func (a *YouTubeAdapter) BuildMetadataURL(mediaID string) string {
	return fmt.Sprintf("https://www.youtube.com/watch?v=%s", mediaID)
}
