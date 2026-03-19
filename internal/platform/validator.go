package platform

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func ValidateCookiePath(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("cookie path must be absolute")
	}
	if strings.Contains(path, "..") {
		return errors.New("cookie path traversal detected")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cookie file not accessible: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("cookie path must be a regular file")
	}
	return nil
}

func ValidateAudioURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.New("malformed audio URL")
	}
	if u.Scheme != "https" {
		return "", errors.New("audio URL must use HTTPS")
	}
	if u.Host != "media.xyzcdn.net" {
		return "", errors.New("audio URL has invalid host")
	}
	return rawURL, nil
}

func ValidateBilibiliRedirect(finalURL string) error {
	u, err := url.Parse(finalURL)
	if err != nil {
		return errors.New("invalid redirect URL")
	}
	host := u.Host
	validHosts := []string{
		"bilibili.com",
		"www.bilibili.com",
		"b23.tv",
	}
	for _, h := range validHosts {
		if host == h || strings.HasSuffix(host, "."+h) {
			return nil
		}
	}
	return errors.New("redirect URL outside Bilibili domain")
}

func ValidateOutputDir(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", errors.New("invalid output directory")
	}
	realDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		return "", errors.New("cannot resolve output directory symlinks")
	}
	return realDir, nil
}
