package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
)

type githubSource struct {
	client    *http.Client
	component string

	mu           sync.Mutex
	apiETag      string
	apiCandidate *Candidate
}

// NewGitHubSource uses only the admin or gateway repository's latest release.
// Unknown components fail locally when checked.
func NewGitHubSource(client *http.Client, component string) Source {
	return &githubSource{client: client, component: component}
}

// Latest resolves the latest stable release via GitHub's public /releases/latest
// redirect first (avoiding the 60 req/hr unauthenticated REST API quota) and
// falls back to the GitHub Releases API when the redirect channel is unavailable.
func (s *githubSource) Latest(ctx context.Context) (Candidate, error) {
	repository, err := componentRepository(s.component)
	if err != nil {
		return Candidate{}, err
	}
	candidate, redirectErr := s.latestFromRedirect(ctx, repository)
	if redirectErr == nil || errors.Is(redirectErr, ErrNoCandidate) || ctx.Err() != nil {
		return candidate, redirectErr
	}
	candidate, apiErr := s.latestFromAPI(ctx, repository)
	if apiErr == nil || errors.Is(apiErr, ErrNoCandidate) {
		return candidate, apiErr
	}
	return Candidate{}, apiErr
}

func (s *githubSource) latestFromRedirect(ctx context.Context, repository string) (Candidate, error) {
	endpoint := "https://github.com/" + repository + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Candidate{}, err
	}
	setSourceHeaders(req, "text/html", "")
	resp, err := noRedirectClient(s.client).Do(req)
	if err != nil {
		return Candidate{}, err
	}
	defer resp.Body.Close()
	if isRateLimited(resp) {
		return Candidate{}, fmt.Errorf("%w: source returned HTTP %d", ErrRateLimited, resp.StatusCode)
	}
	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return ParseGitHubReleaseRedirect(resp.Header.Get("Location"), s.component)
	default:
		return Candidate{}, fmt.Errorf("source redirect returned HTTP %d", resp.StatusCode)
	}
}

func (s *githubSource) latestFromAPI(ctx context.Context, repository string) (Candidate, error) {
	s.mu.Lock()
	etag := s.apiETag
	cached := s.apiCandidate
	s.mu.Unlock()

	endpoint := "https://api.github.com/repos/" + repository + "/releases/latest"
	body, nextETag, notModified, err := readSource(ctx, s.client, "api.github.com", endpoint, "application/vnd.github+json", etag)
	if err != nil {
		return Candidate{}, err
	}
	if notModified {
		if cached == nil {
			return Candidate{}, errors.New("unexpected 304 without cached release candidate")
		}
		return *cached, nil
	}
	candidate, err := ParseGitHubRelease(body, s.component)
	if err != nil {
		return Candidate{}, err
	}
	if nextETag != "" {
		copyCandidate := candidate
		s.mu.Lock()
		s.apiETag = nextETag
		s.apiCandidate = &copyCandidate
		s.mu.Unlock()
	}
	return candidate, nil
}

// ParseGitHubReleaseRedirect validates a GitHub /releases/latest Location header
// strictly against the controlled repository and extracts a stable release tag.
func ParseGitHubReleaseRedirect(location, component string) (Candidate, error) {
	repository, err := componentRepository(component)
	if err != nil {
		return Candidate{}, err
	}
	if location == "" {
		return Candidate{}, errors.New("missing redirect location")
	}
	base, _ := url.Parse("https://github.com/" + repository + "/releases/latest")
	target, err := base.Parse(location)
	if err != nil {
		return Candidate{}, errors.New("invalid redirect location")
	}
	if target.Scheme != "https" || target.Host != "github.com" || target.User != nil || target.RawQuery != "" || target.Fragment != "" {
		return Candidate{}, errors.New("untrusted redirect location")
	}
	if target.Path == "/"+repository+"/releases" {
		return Candidate{}, ErrNoCandidate
	}
	prefix := "/" + repository + "/releases/tag/"
	if !strings.HasPrefix(target.Path, prefix) {
		return Candidate{}, errors.New("unexpected redirect path")
	}
	rawEscapedTag := strings.TrimPrefix(target.Path, prefix)
	if rawEscapedTag == "" || strings.Contains(rawEscapedTag, "/") {
		return Candidate{}, ErrNoCandidate
	}
	rawTag, err := url.PathUnescape(rawEscapedTag)
	if err != nil || strings.Contains(rawTag, "/") {
		return Candidate{}, ErrNoCandidate
	}
	version, stable := productversion.StableVersion(rawTag)
	if !stable {
		return Candidate{}, ErrNoCandidate
	}
	return Candidate{
		Version:    version,
		ReleaseURL: "https://github.com/" + repository + "/releases/tag/" + url.PathEscape(rawTag),
	}, nil
}

// ParseGitHubRelease accepts stable tags only. The returned URL is built from
// the controlled repository and original validated tag, never from html_url.
func ParseGitHubRelease(body []byte, component string) (Candidate, error) {
	repository, err := componentRepository(component)
	if err != nil {
		return Candidate{}, err
	}
	var release struct {
		Tag        *string `json:"tag_name"`
		Draft      *bool   `json:"draft"`
		Prerelease *bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return Candidate{}, err
	}
	if release.Tag == nil || release.Draft == nil || release.Prerelease == nil {
		return Candidate{}, errors.New("incomplete release response")
	}
	version, stable := productversion.StableVersion(*release.Tag)
	if *release.Draft || *release.Prerelease || !stable {
		return Candidate{}, ErrNoCandidate
	}
	return Candidate{
		Version:    version,
		ReleaseURL: "https://github.com/" + repository + "/releases/tag/" + url.PathEscape(*release.Tag),
	}, nil
}

func componentRepository(component string) (string, error) {
	switch component {
	case "admin":
		return "tokenlive/tokenlive-admin", nil
	case "gateway":
		return "tokenlive/tokenlive-gateway", nil
	default:
		return "", errors.New("unsupported update component")
	}
}
