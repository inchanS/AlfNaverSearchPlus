package httpx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequestEncodingAndHeaders(t *testing.T) {
	c := New(&http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("q") != "한 글 & /" || r.URL.Query().Get("existing") != "1" {
			t.Fatalf("query: %s", r.URL)
		}
		if r.Header.Get("User-Agent") != "custom" || r.Header.Get("Referer") != "https://example.invalid/" {
			t.Fatalf("headers: %v", r.Header)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("response"))}, nil
	})})
	data, err := c.Get(context.Background(), "https://example.invalid/?existing=1", map[string]string{"q": "한 글 & /"}, map[string]string{"USER-AGENT": "custom", "Referer": "https://example.invalid/"})
	if err != nil || string(data) != "response" {
		t.Fatalf("response: %q, %v", data, err)
	}
}

type trackedBody struct{ read, closed bool }

func (b *trackedBody) Read([]byte) (int, error) {
	b.read = true
	return 0, errors.New("must not read error body")
}
func (b *trackedBody) Close() error { b.closed = true; return nil }

func TestRejectsStatusBeforeReadingBody(t *testing.T) {
	for _, status := range []int{302, 404, 503} {
		body := &trackedBody{}
		c := New(&http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: body}, nil
		})})
		if _, err := c.Get(context.Background(), "https://example.invalid/", nil, nil); err == nil {
			t.Fatalf("accepted %d", status)
		}
		if body.read || !body.closed {
			t.Fatalf("error body read=%v closed=%v", body.read, body.closed)
		}
	}
}

func TestDeadlineCancelsRequest(t *testing.T) {
	c := New(&http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := c.Get(ctx, "https://example.invalid/", nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline error: %v", err)
	}
}

func TestDownloadAndReadFailures(t *testing.T) {
	c := New(&http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("User-Agent") != DefaultUserAgent {
			t.Fatal("missing default UA")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("archive"))}, nil
	})})
	var dst bytes.Buffer
	if err := c.Download(context.Background(), "https://example.invalid/", &dst); err != nil || dst.String() != "archive" {
		t.Fatalf("download: %q, %v", dst.String(), err)
	}
	body := &trackedBody{}
	c = New(&http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: body}, nil })})
	if _, err := c.Get(context.Background(), "https://example.invalid/", nil, nil); err == nil || !body.closed {
		t.Fatalf("read failure: %v, closed=%v", err, body.closed)
	}
}
