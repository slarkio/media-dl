package platform

import (
	"context"
	"encoding/json"
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
	xiaoyuzhouPattern = regexp.MustCompile(`xiaoyuzhoufm\.com/episode/([a-zA-Z0-9_-]+)`)
	audioURLRe        = regexp.MustCompile(`https://media\.xyzcdn\.net/[a-zA-Z0-9_/]+\.m4a`)
	nextDataRe        = regexp.MustCompile(`__NEXT_DATA__[^>]*>([^<]+)</script>`)
	ogTitleRe         = regexp.MustCompile(`<meta[^>]+property=["']og:title["'][^>]+content=["']([^"']+)["']`)
	ogDescRe          = regexp.MustCompile(`<meta[^>]+property=["']og:description["'][^>]+content=["']([^"']+)["']`)
)

type XiaoyuzhouAdapter struct {
	client *http.Client
}

func NewXiaoyuzhouAdapter() *XiaoyuzhouAdapter {
	return &XiaoyuzhouAdapter{
		client: NewHTTPClient(),
	}
}

func (a *XiaoyuzhouAdapter) Detect(urlStr string) bool {
	return xiaoyuzhouPattern.MatchString(urlStr)
}

func (a *XiaoyuzhouAdapter) GetID(urlStr string) (string, error) {
	matches := xiaoyuzhouPattern.FindStringSubmatch(urlStr)
	if len(matches) < 2 {
		return "", errors.New("invalid Xiaoyuzhou URL")
	}
	return matches[1], nil
}

func (a *XiaoyuzhouAdapter) Name() string {
	return "xiaoyuzhou"
}

func (a *XiaoyuzhouAdapter) RequiresCookie() bool {
	return false
}

func (a *XiaoyuzhouAdapter) Headers() map[string]string {
	return nil
}

func (a *XiaoyuzhouAdapter) GetConfig() PlatformConfig {
	return PlatformConfig{
		RequiresCookie: false,
		Headers:        nil,
	}
}

func (a *XiaoyuzhouAdapter) ResolveAudioURL(ctx context.Context, episodeID string) (string, error) {
	pageURL := fmt.Sprintf("https://www.xiaoyuzhoufm.com/episode/%s", episodeID)

	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Go-cli)")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch episode page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	audioURL := extractAudioURL(string(body))
	if audioURL == "" {
		return "", errors.New("could not find audio URL in episode page")
	}

	if _, err := ValidateAudioURL(audioURL); err != nil {
		return "", fmt.Errorf("audio URL validation failed: %w", err)
	}

	return audioURL, nil
}

func extractAudioURL(htmlContent string) string {
	matches := audioURLRe.FindStringSubmatch(htmlContent)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

type NextData struct {
	Props struct {
		PageProps struct {
			Episode struct {
				Title       string `json:"title"`
				Shownotes   string `json:"shownotes"`
				Duration    int    `json:"duration"`
				PublishTime string `json:"publishTime"`
			} `json:"episode"`
		} `json:"pageProps"`
	} `json:"props"`
}

func FetchXiaoyuzhouShownotes(ctx context.Context, episodeID string) (string, error) {
	adapter := &XiaoyuzhouAdapter{client: NewHTTPClient()}
	pageURL := fmt.Sprintf("https://www.xiaoyuzhoufm.com/episode/%s", episodeID)

	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Go-cli)")

	resp, err := adapter.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch shownotes: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	shownotes, err := extractShownotesFromHTML(string(body))
	if err != nil {
		return "", fmt.Errorf("failed to extract shownotes: %w", err)
	}

	return shownotes, nil
}

func extractShownotesFromHTML(htmlContent string) (string, error) {
	matches := nextDataRe.FindStringSubmatch(htmlContent)
	if len(matches) < 2 {
		return "", errors.New("could not find __NEXT_DATA__")
	}

	var nextData NextData
	if err := json.Unmarshal([]byte(matches[1]), &nextData); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	if nextData.Props.PageProps.Episode.Shownotes == "" {
		return "", errors.New("shownotes is empty")
	}

	return nextData.Props.PageProps.Episode.Shownotes, nil
}

type XiaoyuzhouMetadata struct {
	Title       string `json:"title"`
	Duration    int    `json:"duration"`
	Description string `json:"description"`
	PodcastName string `json:"podcast_name"`
}

func FetchXiaoyuzhouMetadata(ctx context.Context, episodeID string) (*XiaoyuzhouMetadata, error) {
	adapter := &XiaoyuzhouAdapter{client: NewHTTPClient()}
	pageURL := fmt.Sprintf("https://www.xiaoyuzhoufm.com/episode/%s", episodeID)

	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Go-cli)")

	resp, err := adapter.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	htmlContent := string(body)

	matches := nextDataRe.FindStringSubmatch(htmlContent)
	if len(matches) < 2 {
		return nil, errors.New("could not find __NEXT_DATA__")
	}

	var nextData NextData
	if err := json.Unmarshal([]byte(matches[1]), &nextData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	ep := nextData.Props.PageProps.Episode
	meta := &XiaoyuzhouMetadata{
		Title:    ep.Title,
		Duration:  ep.Duration,
	}

	// Extract og:title and og:description from HTML meta tags
	if matches := ogTitleRe.FindStringSubmatch(htmlContent); len(matches) > 1 {
		meta.Title = matches[1]
	}
	if matches := ogDescRe.FindStringSubmatch(htmlContent); len(matches) > 1 {
		meta.Description = matches[1]
	}

	// Extract podcast name from page title (format: "Episode Title - Podcast Name | 小宇宙")
	if titleMatches := regexp.MustCompile(` - ([^|]+) \| `).FindStringSubmatch(meta.Title); len(titleMatches) > 1 {
		meta.PodcastName = strings.TrimSpace(titleMatches[1])
	}

	return meta, nil
}

func (a *XiaoyuzhouAdapter) BuildDownloadCommand(ctx context.Context, mediaID, cookies, outputDir string) *exec.Cmd {
	args := []string{
		"--no-playlist",
		"-f", "bestaudio[ext=m4a]/bestaudio",
		"--audio-format", "m4a",
		"--audio-quality", "0",
		"--embed-metadata",
		"--print", "after_move:filepath",
		"-o", filepath.Join(outputDir, "audio.%(ext)s"),
		"--add-header", "Referer:https://www.xiaoyuzhoufm.com",
	}
	if cookies != "" {
		if err := ValidateCookiePath(cookies); err != nil {
			return nil
		}
		args = append(args, "--cookies", cookies)
	}
	args = append(args, mediaID)
	return exec.CommandContext(ctx, "yt-dlp", args...)
}

func (a *XiaoyuzhouAdapter) BuildMetadataURL(mediaID string) string {
	return mediaID
}
