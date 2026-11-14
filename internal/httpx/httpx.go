// Package httpx builds context-aware HTTP requests for search and updates.
package httpx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15"

// Client owns its HTTP client so callers and tests never change global HTTP state.
// Each operation supplies its own deadline through ctx.
type Client struct{ http *http.Client }

func New(client *http.Client) *Client { return &Client{http: client} }

func (c *Client) get(ctx context.Context, rawURL string, params, headers map[string]string) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("http %d for %s", resp.StatusCode, rawURL)
	}
	return resp, nil
}

// Get reads an API response. Redirects and gzip are handled by net/http.
func (c *Client) Get(ctx context.Context, rawURL string, params, headers map[string]string) ([]byte, error) {
	resp, err := c.get(ctx, rawURL, params, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// Download streams an asset rather than buffering the whole archive in memory.
func (c *Client) Download(ctx context.Context, rawURL string, dst io.Writer) error {
	resp, err := c.get(ctx, rawURL, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(dst, resp.Body)
	return err
}
