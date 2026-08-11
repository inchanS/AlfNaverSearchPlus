// Package httpx is a tiny HTTP helper replacing alfred-pyworkflow's web.get:
// build a GET with query params and optional headers, enforce a success status,
// and return the response body. gzip is handled transparently by net/http.
package httpx

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DefaultUserAgent is a browser-like UA used unless a handler overrides it.
// Several Naver endpoints reject or alter responses for non-browser agents.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15"

var client = &http.Client{Timeout: 15 * time.Second}

// Get performs a GET request. params are URL query values (encoded like
// urllib), headers override or add request headers. It returns the response
// body on a 2xx/3xx status, or an error otherwise.
func Get(rawURL string, params, headers map[string]string) ([]byte, error) {
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

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	if _, ok := headers["user-agent"]; !ok {
		if _, ok := headers["User-Agent"]; !ok {
			req.Header.Set("User-Agent", DefaultUserAgent)
		}
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d for %s", resp.StatusCode, rawURL)
	}
	return body, nil
}
