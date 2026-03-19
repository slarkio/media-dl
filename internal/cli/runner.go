package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/slarkio/media-dl/internal/downloader"
	"github.com/slarkio/media-dl/internal/platform"
	"golang.org/x/net/html"
)

var b23tvPattern = regexp.MustCompile(`b23\.tv/[a-zA-Z0-9]+`)

type Runner struct {
	opts  DownloadOptions
	ytDLP *downloader.YtDLP
}

type JSONResult struct {
	Success       bool   `json:"success"`
	VideoID       string `json:"video_id,omitempty"`
	Platform      string `json:"platform,omitempty"`
	AudioPath     string `json:"audio_path,omitempty"`
	Metadata      any    `json:"metadata,omitempty"`
	ShownotesPath string `json:"shownotes_path,omitempty"`
	Error         string `json:"error,omitempty"`
}

func NewRunner(opts DownloadOptions) *Runner {
	return &Runner{
		opts:  opts,
		ytDLP: downloader.New(opts.Verbose),
	}
}

func (r *Runner) Run(ctx context.Context) error {
	// Flag validation
	if r.opts.AudioOnly && r.opts.ShownotesOnly {
		err := fmt.Errorf("--audio-only and --shownotes-only cannot be used together")
		if r.opts.JSON {
			printJSONError(err.Error())
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		return err
	}

	if r.opts.ShownotesOnly {
		cleanURL := platform.CleanURL(r.opts.URL)
		p := platform.DetectPlatform(cleanURL)
		if p == nil || p.Name() != "xiaoyuzhou" {
			err := fmt.Errorf("shownotes are only supported for xiaoyuzhou platform")
			if r.opts.JSON {
				printJSONError(err.Error())
			} else {
				fmt.Fprintln(os.Stderr, "Error:", err)
			}
			return err
		}
	}

	if err := r.ytDLP.CheckDependency(); err != nil {
		if r.opts.JSON {
			printJSONError(err.Error())
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		return err
	}

	cleanURL := platform.CleanURL(r.opts.URL)
	p := platform.DetectPlatform(cleanURL)
	if p == nil {
		err := fmt.Errorf("unsupported platform for URL: %s", r.opts.URL)
		if r.opts.JSON {
			printJSONError(err.Error())
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		return err
	}

	mediaID, err := p.GetID(cleanURL)
	if err != nil {
		if r.opts.JSON {
			printJSONError(err.Error())
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		return err
	}

	if resolver, ok := p.(platform.ShortURLResolver); ok {
		if b23tvPattern.MatchString(cleanURL) {
			resolvedURL, resolveErr := resolver.ResolveShortURL(ctx, cleanURL)
			if resolveErr != nil {
				if r.opts.JSON {
					printJSONError(resolveErr.Error())
				} else {
					fmt.Fprintln(os.Stderr, "Error:", resolveErr)
				}
				return resolveErr
			}
			mediaID, _ = p.GetID(resolvedURL)
			if r.opts.Verbose {
				fmt.Printf("[DEBUG] Resolved short URL: %s\n", resolvedURL)
			}
		}
	}

	if r.opts.Verbose {
		fmt.Printf("[DEBUG] Detected platform: %s, media ID: %s\n", p.Name(), mediaID)
	}

	outputDir := r.opts.OutputDir
	if outputDir == "" {
		outputDir = "."
	}
	validatedDir, err := platform.ValidateOutputDir(outputDir)
	if err != nil {
		err = fmt.Errorf("invalid output directory: %w", err)
		if r.opts.JSON {
			printJSONError(err.Error())
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		return err
	}
	outputDir = validatedDir

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create subdirectory with media ID for multi-file output
	mediaDir := filepath.Join(outputDir, mediaID)
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		return fmt.Errorf("failed to create media directory: %w", err)
	}
	outputDir = mediaDir

	var cookies string
	var downloadURL string
	var audioPath string
	var metadata any

	if !r.opts.ShownotesOnly {
		if r.opts.Cookie != "" {
			cookies = r.opts.Cookie
		} else if p.RequiresCookie() {
			err := fmt.Errorf("cookie required for %s but --cookie not provided", p.Name())
			if r.opts.JSON {
				printJSONError(err.Error())
			} else {
				fmt.Fprintln(os.Stderr, "Error:", err)
			}
			return err
		}

		if resolver, ok := p.(platform.AudioURLResolver); ok {
			downloadURL, err = resolver.ResolveAudioURL(ctx, mediaID)
			if err != nil {
				if r.opts.JSON {
					printJSONError(err.Error())
				} else {
					fmt.Fprintln(os.Stderr, "Error:", err)
				}
				return err
			}
			if r.opts.Verbose {
				fmt.Printf("[DEBUG] Resolved audio URL: %s\n", downloadURL)
			}
		} else {
			downloadURL = mediaID
		}

		if !r.opts.JSON {
			fmt.Println("[PROGRESS] Downloading...")
		}

		path, meta, dlErr := r.ytDLP.DownloadWithRetry(ctx, p, downloadURL, cookies, outputDir)
		if dlErr != nil {
			if r.opts.JSON {
				printJSONError(dlErr.Error())
			} else {
				fmt.Fprintln(os.Stderr, "Error:", dlErr)
			}
			return dlErr
		}
		audioPath = path
		metadata = meta
	}

	result := JSONResult{
		Success:   true,
		VideoID:   mediaID,
		Platform:  p.Name(),
		AudioPath: audioPath,
		Metadata:  metadata,
	}

	if p.Name() == "xiaoyuzhou" && !r.opts.AudioOnly {
		shownotesPath, shownotesErr := r.processXiaoyuzhouShownotes(ctx, mediaID, outputDir)
		result.ShownotesPath = shownotesPath
		if shownotesErr != nil && r.opts.Verbose {
			fmt.Fprintf(os.Stderr, "[DEBUG] Shownotes processing failed: %v\n", shownotesErr)
		}
	} else if metadata != nil {
		metaPath := filepath.Join(outputDir, "metadata.json")
		if data, err := json.Marshal(metadata); err == nil {
			os.WriteFile(metaPath, data, 0644)
		}
	}

	if r.opts.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	} else {
		if audioPath != "" {
			fmt.Printf("Downloaded: %s\n", audioPath)
		}
		if result.ShownotesPath != "" {
			fmt.Printf("Shownotes: %s\n", result.ShownotesPath)
		}
	}

	return nil
}

func (r *Runner) processXiaoyuzhouShownotes(ctx context.Context, episodeID, outputDir string) (string, error) {
	shownotes, err := platform.FetchXiaoyuzhouShownotes(ctx, episodeID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch shownotes: %w", err)
	}

	if shownotes == "" {
		return "", nil
	}

	// Fetch metadata for title and description
	meta, _ := platform.FetchXiaoyuzhouMetadata(ctx, episodeID)

	// Prepend title and description
	header := ""
	if meta != nil {
		if meta.Title != "" {
			header += "# " + meta.Title + "\n\n"
		}
		if meta.Description != "" {
			header += meta.Description + "\n\n"
		}
	}

	shownotes = convertHTMLImagesToMarkdown(shownotes)
	shownotes = stripHTMLTags(shownotes)

	// Prepend header to shownotes
	fullContent := header + "---\n\n" + shownotes

	shownotesPath := filepath.Join(outputDir, "shownotes.md")
	if err := os.WriteFile(shownotesPath, []byte(fullContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write shownotes: %w", err)
	}

	return shownotesPath, nil
}

func convertHTMLImagesToMarkdown(htmlStr string) string {
	imgRe := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`)
	return imgRe.ReplaceAllString(htmlStr, "![]($1)")
}

func stripHTMLTags(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr
	}

	var sb strings.Builder

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "script" || n.Data == "style" || n.Data == "head" {
				return
			}
			if n.Data == "p" || n.Data == "div" || n.Data == "li" || n.Data == "tr" {
				sb.WriteString("\n")
			}
			if n.Data == "br" {
				sb.WriteString("\n")
			}
			if n.Data == "a" {
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						sb.WriteString(attr.Val)
						sb.WriteString(" ")
						break
					}
				}
			}
		}
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return sb.String()
}

func printJSONError(msg string) {
	result := JSONResult{Success: false, Error: msg}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.Encode(result)
}
