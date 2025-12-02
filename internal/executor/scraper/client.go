package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/multi-worker/internal/config"
)

// HTTPClient is a configured HTTP client for scraping
type HTTPClient struct {
	client    *http.Client
	userAgent string
	rateLimit time.Duration
	lastReq   time.Time
}

// NewHTTPClient creates a new HTTP client for scraping
func NewHTTPClient(cfg config.ScraperConfig) *HTTPClient {
	transport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
	}

	if cfg.ProxyURL != "" {
		proxyURL, err := url.Parse(cfg.ProxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return &HTTPClient{
		client: &http.Client{
			Timeout:   cfg.RequestTimeout,
			Transport: transport,
		},
		userAgent: cfg.UserAgent,
		rateLimit: time.Duration(cfg.RateLimitMs) * time.Millisecond,
	}
}

// Get performs an HTTP GET request with rate limiting
func (c *HTTPClient) Get(ctx context.Context, url string) ([]byte, error) {
	// Apply rate limiting
	if elapsed := time.Since(c.lastReq); elapsed < c.rateLimit {
		time.Sleep(c.rateLimit - elapsed)
	}
	c.lastReq = time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// GetJSON performs an HTTP GET request expecting JSON response
func (c *HTTPClient) GetJSON(ctx context.Context, url string) ([]byte, error) {
	if elapsed := time.Since(c.lastReq); elapsed < c.rateLimit {
		time.Sleep(c.rateLimit - elapsed)
	}
	c.lastReq = time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// PostJSON performs an HTTP POST request with JSON body
func (c *HTTPClient) PostJSON(ctx context.Context, url string, body io.Reader) ([]byte, error) {
	if elapsed := time.Since(c.lastReq); elapsed < c.rateLimit {
		time.Sleep(c.rateLimit - elapsed)
	}
	c.lastReq = time.Now()

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
