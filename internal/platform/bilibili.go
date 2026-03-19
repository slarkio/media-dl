package platform

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	bilibiliPattern   = regexp.MustCompile(`bilibili\.com/video/(BV[a-zA-Z0-9]+)`)
	b23tvPattern      = regexp.MustCompile(`b23\.tv/([a-zA-Z0-9]+)`)
	bilibiliBVPattern = regexp.MustCompile(`(BV[a-zA-Z0-9]{10})`)
)

type BilibiliAdapter struct {
	client *http.Client
}

func NewBilibiliAdapter() *BilibiliAdapter {
	return &BilibiliAdapter{
		client: NewHTTPClient(),
	}
}

func (a *BilibiliAdapter) Detect(urlStr string) bool {
	return bilibiliPattern.MatchString(urlStr) || b23tvPattern.MatchString(urlStr)
}

func (a *BilibiliAdapter) GetID(urlStr string) (string, error) {
	return a.GetIDWithContext(context.Background(), urlStr)
}

func (a *BilibiliAdapter) GetIDWithContext(ctx context.Context, urlStr string) (string, error) {
	if b23tvPattern.MatchString(urlStr) {
		resolved, err := a.ResolveShortURL(ctx, urlStr)
		if err != nil {
			return "", fmt.Errorf("failed to resolve b23.tv short URL: %w", err)
		}
		urlStr = resolved
	}

	matches := bilibiliPattern.FindStringSubmatch(urlStr)
	if len(matches) < 2 {
		return "", errors.New("invalid Bilibili URL")
	}
	return matches[1], nil
}

func (a *BilibiliAdapter) ResolveShortURL(ctx context.Context, shortURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", shortURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; bot)")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.Request != nil && resp.Request.URL != nil {
		finalURL := resp.Request.URL.String()
		if err := ValidateBilibiliRedirect(finalURL); err != nil {
			return "", fmt.Errorf("SSRF detected: %w", err)
		}
		return finalURL, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	hrefMatches := b23tvPattern.FindSubmatch(body)
	if len(hrefMatches) < 1 {
		return "", errors.New("could not find redirect URL")
	}

	return string(hrefMatches[0]), nil
}

func (a *BilibiliAdapter) Name() string {
	return "bilibili"
}

func (a *BilibiliAdapter) RequiresCookie() bool {
	return false
}

func (a *BilibiliAdapter) Headers() map[string]string {
	return map[string]string{
		"Referer": "https://www.bilibili.com",
	}
}

func (a *BilibiliAdapter) GetConfig() PlatformConfig {
	return PlatformConfig{
		RequiresCookie: false,
		Headers:        a.Headers(),
	}
}

func extractBVFromURL(urlStr string) (string, error) {
	matches := bilibiliBVPattern.FindStringSubmatch(urlStr)
	if len(matches) < 2 {
		return "", errors.New("BV ID not found in URL")
	}
	return matches[1], nil
}

func normalizeBilibiliURL(bvid string) string {
	return strings.TrimPrefix(bvid, "https://www.bilibili.com/video/")
}

func (a *BilibiliAdapter) BuildDownloadCommand(ctx context.Context, mediaID, cookies, outputDir string) *exec.Cmd {
	args := []string{
		"--no-playlist",
		"-f", "bestaudio[ext=m4a]/bestaudio",
		"--audio-format", "m4a",
		"--audio-quality", "0",
		"--embed-metadata",
		"--print", "after_move:filepath",
		"-o", filepath.Join(outputDir, "%(title)s.%(ext)s"),
		"--add-header", "Referer:https://www.bilibili.com",
	}
	if cookies != "" {
		if err := ValidateCookiePath(cookies); err != nil {
			return nil
		}
		args = append(args, "--cookies", cookies)
	}
	args = append(args, fmt.Sprintf("https://www.bilibili.com/video/%s", mediaID))
	return exec.CommandContext(ctx, "yt-dlp", args...)
}

func (a *BilibiliAdapter) BuildMetadataURL(mediaID string) string {
	return fmt.Sprintf("https://www.bilibili.com/video/%s", mediaID)
}
