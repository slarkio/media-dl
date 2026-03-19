package platform

import (
	"context"
	"errors"
	"net/url"
	"os/exec"
	"strings"
)

var (
	NoPlatformFound = errors.New("no platform found for URL")
)

type PlatformAdapter interface {
	Detect(url string) bool
	GetID(url string) (string, error)
	Name() string
	RequiresCookie() bool
	Headers() map[string]string
	BuildDownloadCommand(ctx context.Context, mediaID, cookies, outputDir string) *exec.Cmd
	BuildMetadataURL(mediaID string) string
}

type AudioURLResolver interface {
	ResolveAudioURL(ctx context.Context, episodeID string) (string, error)
}

type ShortURLResolver interface {
	ResolveShortURL(ctx context.Context, shortURL string) (string, error)
}

type PlatformConfig struct {
	RequiresCookie bool
	Headers        map[string]string
}

var adapters []PlatformAdapter

func RegisterAdapter(a PlatformAdapter) {
	adapters = append(adapters, a)
}

func DetectPlatform(inputURL string) PlatformAdapter {
	for _, a := range adapters {
		if a.Detect(inputURL) {
			return a
		}
	}
	return nil
}

func CleanURL(inputURL string) string {
	u, err := url.Parse(inputURL)
	if err != nil {
		return inputURL
	}

	q := u.Query()
	trackingParams := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term",
		"fbclid", "gclid", "gclsrc",
		"mc_cid", "mc_eid",
		"ref", "source",
	}

	for _, param := range trackingParams {
		q.Del(param)
	}

	u.RawQuery = q.Encode()
	u.Fragment = ""
	return u.String()
}

func SanitizeFilename(name string) string {
	illegalChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, ch := range illegalChars {
		result = strings.ReplaceAll(result, ch, "_")
	}
	result = strings.TrimSpace(result)
	return result
}
