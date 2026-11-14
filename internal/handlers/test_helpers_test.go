package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/inchanS/AlfNaverSearchPlus/internal/httpx"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func testRequest(t *testing.T, transport testTransport) *request {
	t.Helper()
	if transport == nil {
		transport = func(r *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected HTTP request: %s", r.URL)
			return nil, nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
	t.Cleanup(cancel)
	return &request{ctx: ctx, http: httpx.New(&http.Client{Transport: transport})}
}
