package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrYtDLPNotFound = errors.New(`yt-dlp not found in PATH
Install: brew install yt-dlp  # macOS
        pip install yt-dlp    # Python`)
)

type YtDLP struct {
	verbose bool
}

type Metadata struct {
	Title       string `json:"title"`
	Duration    int    `json:"duration"`
	Uploader    string `json:"uploader"`
	UploadDate  string `json:"upload_date"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail_url"`
}

func New(verbose bool) *YtDLP {
	return &YtDLP{verbose: verbose}
}

func (y *YtDLP) CheckDependency() error {
	cmd := exec.Command("yt-dlp", "--version")
	if err := cmd.Run(); err != nil {
		return ErrYtDLPNotFound
	}
	return nil
}

type downloaderAdapter interface {
	Name() string
	Headers() map[string]string
	BuildDownloadCommand(ctx context.Context, mediaID, cookies, outputDir string) *exec.Cmd
	BuildMetadataURL(mediaID string) string
}

func (y *YtDLP) Download(ctx context.Context, p downloaderAdapter, mediaID, cookies, outputDir string) (string, *Metadata, error) {
	cmd := p.BuildDownloadCommand(ctx, mediaID, cookies, outputDir)
	if cmd == nil {
		return "", nil, fmt.Errorf("failed to build download command")
	}

	if y.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Running: %s\n", strings.Join(cmd.Args, " "))
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		if y.verbose {
			fmt.Fprintf(os.Stderr, "[DEBUG] stderr: %s\n", stderr.String())
		}
		return "", nil, fmt.Errorf("download failed: %w", err)
	}

	downloadedPath, err := y.parseOutput(string(output), outputDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to find downloaded file: %w", err)
	}

	metadata, err := y.getMetadata(ctx, p, mediaID)
	if err != nil {
		if y.verbose {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to get metadata: %v\n", err)
		}
	}

	return downloadedPath, metadata, nil
}

func (y *YtDLP) parseOutput(output string, outputDir string) (string, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "[") {
			if _, err := os.Stat(line); err == nil {
				return line, nil
			}
			if filepath.Ext(line) != "" {
				return line, nil
			}
		}
	}

	pattern := filepath.Join(outputDir, "*.m4a")
	files, _ := filepath.Glob(pattern)
	if len(files) > 0 {
		return files[0], nil
	}
	pattern = filepath.Join(outputDir, "*.mp3")
	files, _ = filepath.Glob(pattern)
	if len(files) > 0 {
		return files[0], nil
	}

	return "", errors.New("could not find downloaded file")
}

func (y *YtDLP) getMetadata(ctx context.Context, p downloaderAdapter, mediaID string) (*Metadata, error) {
	url := p.BuildMetadataURL(mediaID)

	args := []string{
		"--dump-json",
		"--no-playlist",
		"--print", "%(json)s",
		"--", url,
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		if y.verbose {
			fmt.Fprintf(os.Stderr, "[DEBUG] metadata error: %s\n", stderr.String())
		}
		return nil, err
	}

	var meta struct {
		Title       string `json:"title"`
		Duration    int    `json:"duration"`
		Uploader    string `json:"uploader"`
		UploadDate  string `json:"upload_date"`
		Description string `json:"description"`
		Thumbnail   string `json:"thumbnail"`
	}

	if err := json.Unmarshal(output, &meta); err != nil {
		return nil, err
	}

	return &Metadata{
		Title:       meta.Title,
		Duration:    meta.Duration,
		Uploader:    meta.Uploader,
		UploadDate:  meta.UploadDate,
		Description: meta.Description,
		Thumbnail:   meta.Thumbnail,
	}, nil
}

func (y *YtDLP) DownloadWithRetry(ctx context.Context, p downloaderAdapter, mediaID, cookies, outputDir string) (string, *Metadata, error) {
	maxRetries := 3
	backoff := 1 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			if y.verbose {
				fmt.Fprintf(os.Stderr, "[DEBUG] Retry %d/%d after %v\n", attempt, maxRetries, backoff)
			}
			select {
			case <-ctx.Done():
				return "", nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}

		if err := ctx.Err(); err != nil {
			return "", nil, err
		}

		path, meta, err := y.Download(ctx, p, mediaID, cookies, outputDir)
		if err == nil {
			return path, meta, nil
		}
		lastErr = err

		if !isRetryableError(err) {
			return "", nil, err
		}
	}

	return "", nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isRetryableError(err error) bool {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		switch code {
		case 1, 10, 30:
			return true
		default:
			return false
		}
	}

	errMsg := strings.ToLower(err.Error())
	retryable := []string{"connection refused", "timeout", "429", "500", "502", "503"}
	for _, s := range retryable {
		if strings.Contains(errMsg, s) {
			return true
		}
	}
	return false
}
