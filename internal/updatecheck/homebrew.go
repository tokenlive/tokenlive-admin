package updatecheck

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
)

const maxFormulaBytes = 64 << 10

var (
	formulaDeclaration = regexp.MustCompile(`(?m)^[ \t]*version\b[^\n]*$`)
	formulaVersion     = regexp.MustCompile(`^[ \t]*version[ \t]+"([^"\r\n]+)"[ \t]*(?:#[^\r\n]*)?\r?$`)
)

type homebrewSource struct {
	client *http.Client

	mu           sync.Mutex
	rawETag      string
	rawCandidate *Candidate
	apiETag      string
	apiCandidate *Candidate
}

// NewHomebrewSource uses the currently published Formula, not standalone's
// latest release, which may have been published before Homebrew was updated.
func NewHomebrewSource(client *http.Client) Source {
	return &homebrewSource{client: client}
}

// Latest reads Formula/tokenlive.rb from raw.githubusercontent.com first to avoid
// the 60 req/hr unauthenticated GitHub REST API quota, and falls back to the
// GitHub Contents API if the raw endpoint fails.
func (s *homebrewSource) Latest(ctx context.Context) (Candidate, error) {
	candidate, rawErr, fetched := s.latestFromRaw(ctx)
	if rawErr == nil || fetched || ctx.Err() != nil {
		return candidate, rawErr
	}
	return s.latestFromAPI(ctx)
}

func (s *homebrewSource) latestFromRaw(ctx context.Context) (Candidate, error, bool) {
	s.mu.Lock()
	etag := s.rawETag
	cached := s.rawCandidate
	s.mu.Unlock()

	endpoint := "https://raw.githubusercontent.com/tokenlive/homebrew-tokenlive/main/Formula/tokenlive.rb"
	body, nextETag, notModified, err := readSourceLimit(ctx, s.client, "raw.githubusercontent.com", endpoint, "text/plain", etag, maxFormulaBytes)
	if err != nil {
		return Candidate{}, err, errors.Is(err, errResponseTooLarge)
	}
	if notModified {
		if cached == nil {
			return Candidate{}, errors.New("unexpected 304 without cached formula candidate"), false
		}
		return *cached, nil, true
	}
	candidate, err := ParseFormula(body)
	if err != nil {
		return Candidate{}, err, true
	}
	if nextETag != "" {
		copyCandidate := candidate
		s.mu.Lock()
		s.rawETag = nextETag
		s.rawCandidate = &copyCandidate
		s.mu.Unlock()
	}
	return candidate, nil, true
}

func (s *homebrewSource) latestFromAPI(ctx context.Context) (Candidate, error) {
	s.mu.Lock()
	etag := s.apiETag
	cached := s.apiCandidate
	s.mu.Unlock()

	endpoint := "https://api.github.com/repos/tokenlive/homebrew-tokenlive/contents/Formula/tokenlive.rb"
	body, nextETag, notModified, err := readSource(ctx, s.client, "api.github.com", endpoint, "application/vnd.github+json", etag)
	if err != nil {
		return Candidate{}, err
	}
	if notModified {
		if cached == nil {
			return Candidate{}, errors.New("unexpected 304 without cached formula candidate")
		}
		return *cached, nil
	}
	var contents struct {
		Type     string  `json:"type"`
		Encoding string  `json:"encoding"`
		Content  *string `json:"content"`
	}
	if err := json.Unmarshal(body, &contents); err != nil {
		return Candidate{}, err
	}
	if contents.Type != "file" || contents.Encoding != "base64" || contents.Content == nil {
		return Candidate{}, errors.New("invalid Formula contents response")
	}
	decoded := base64.NewDecoder(base64.StdEncoding.Strict(), strings.NewReader(*contents.Content))
	formula, err := io.ReadAll(io.LimitReader(decoded, maxFormulaBytes+1))
	if err != nil {
		return Candidate{}, err
	}
	candidate, err := ParseFormula(formula)
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

// ParseFormula reads a single line-start version declaration without executing
// Ruby. Indentation and trailing comments are allowed; ambiguous or malformed
// declarations fail the check.
func ParseFormula(body []byte) (Candidate, error) {
	if len(body) > maxFormulaBytes {
		return Candidate{}, errors.New("Formula too large")
	}
	if !utf8.Valid(body) {
		return Candidate{}, errors.New("Formula is not valid UTF-8")
	}
	declarations := formulaDeclaration.FindAll(body, -1)
	if len(declarations) == 0 {
		return Candidate{}, ErrNoCandidate
	}
	if len(declarations) != 1 {
		return Candidate{}, errors.New("multiple Formula version declarations")
	}
	match := formulaVersion.FindSubmatch(declarations[0])
	if match == nil {
		return Candidate{}, errors.New("malformed Formula version declaration")
	}
	rawVersion := string(match[1])
	version, stable := productversion.StableVersion(rawVersion)
	if !stable {
		return Candidate{}, ErrNoCandidate
	}
	// The publisher prefixes its validated Formula version with one v. Keep
	// metadata in the URL even though comparison uses the canonical version.
	tag := "v" + strings.TrimPrefix(rawVersion, "v")
	return Candidate{
		Version:    version,
		ReleaseURL: "https://github.com/tokenlive/tokenlive-standalone/releases/tag/" + url.PathEscape(tag),
	}, nil
}
