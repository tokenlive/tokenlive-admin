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

// ErrNoCandidate distinguishes a valid response without a stable candidate
// from a failed check.
var ErrNoCandidate = errors.New("no stable candidate")

// sourceClient borrows the transport and timeout, without changing the caller's
// client or allowing its cookie jar/redirect hook to attach business credentials.
func sourceClient(client *http.Client) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}
	cloned := *client
	cloned.Jar = nil
	cloned.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many source redirects")
		}
		if req.URL.Scheme != "https" || req.URL.Host != "api.github.com" || req.URL.User != nil {
			return errors.New("source redirect must stay on HTTPS api.github.com")
		}
		setSourceHeaders(req)
		return nil
	}
	return &cloned
}

func setSourceHeaders(req *http.Request) {
	req.Header = make(http.Header)
	req.Header.Set("User-Agent", sourceUserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
}

func readSource(ctx context.Context, client *http.Client, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	setSourceHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("source returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseBytes {
		return nil, errors.New("source response too large")
	}
	return body, nil
}
