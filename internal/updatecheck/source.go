package updatecheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	maxResponseBytes = 1 << 20
	sourceUserAgent  = "tokenlive-admin-updatecheck"
)

// Candidate is a stable release advertised by a controlled publishing channel.
type Candidate struct {
	Version    string `json:"version"`
	ReleaseURL string `json:"release_url"`
}

// Source checks the current candidate without falling back to release history.
type Source interface {
	Latest(context.Context) (Candidate, error)
}

var (
	// ErrNoCandidate distinguishes a valid response without a stable candidate
	// from a failed check.
	ErrNoCandidate = errors.New("no stable candidate")
	// ErrRateLimited signals that the upstream source rejected the request due to rate limiting.
	ErrRateLimited = errors.New("update source rate limited")

	errResponseTooLarge = errors.New("source response too large")
)

// sourceClient borrows the transport and timeout, without changing the caller's
// client or allowing its cookie jar/redirect hook to attach business credentials.
func sourceClient(client *http.Client, allowedHost, accept, etag string) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}
	cloned := *client
	cloned.Jar = nil
	cloned.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many source redirects")
		}
		if req.URL.Scheme != "https" || req.URL.Host != allowedHost || req.URL.User != nil {
			return fmt.Errorf("source redirect must stay on HTTPS %s", allowedHost)
		}
		setSourceHeaders(req, accept, etag)
		return nil
	}
	return &cloned
}

// noRedirectClient borrows transport and timeout without following redirects.
func noRedirectClient(client *http.Client) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}
	cloned := *client
	cloned.Jar = nil
	cloned.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &cloned
}

func setSourceHeaders(req *http.Request, accept, etag string) {
	req.Header = make(http.Header)
	req.Header.Set("User-Agent", sourceUserAgent)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
}

func isRateLimited(resp *http.Response) bool {
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0"
}

func readSource(ctx context.Context, client *http.Client, allowedHost, endpoint, accept, etag string) ([]byte, string, bool, error) {
	return readSourceLimit(ctx, client, allowedHost, endpoint, accept, etag, maxResponseBytes)
}

func readSourceLimit(ctx context.Context, client *http.Client, allowedHost, endpoint, accept, etag string, maxBytes int64) ([]byte, string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", false, err
	}
	setSourceHeaders(req, accept, etag)
	resp, err := sourceClient(client, allowedHost, accept, etag).Do(req)
	if err != nil {
		return nil, "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return nil, resp.Header.Get("ETag"), true, nil
	}
	if isRateLimited(resp) {
		return nil, "", false, fmt.Errorf("%w: source returned HTTP %d", ErrRateLimited, resp.StatusCode)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", false, fmt.Errorf("source returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "", false, err
	}
	if int64(len(body)) > maxBytes {
		return nil, "", false, errResponseTooLarge
	}
	return body, resp.Header.Get("ETag"), false, nil
}
