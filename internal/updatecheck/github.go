package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
)

type githubSource struct {
	client    *http.Client
	component string
}

// NewGitHubSource uses only the admin or gateway repository's latest release.
// Unknown components fail locally when checked.
func NewGitHubSource(client *http.Client, component string) Source {
	return &githubSource{client: sourceClient(client), component: component}
}

func (s *githubSource) Latest(ctx context.Context) (Candidate, error) {
	repository, err := componentRepository(s.component)
	if err != nil {
		return Candidate{}, err
	}
	body, err := readSource(ctx, s.client, "https://api.github.com/repos/"+repository+"/releases/latest")
	if err != nil {
		return Candidate{}, err
	}
	return ParseGitHubRelease(body, s.component)
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
