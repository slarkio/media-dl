package platform

import (
	"net/http"
	"sync"
	"time"
)

var (
	defaultClient *http.Client
	once          sync.Once
)

func NewHTTPClient() *http.Client {
	once.Do(func() {
		defaultClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	})
	return defaultClient
}
