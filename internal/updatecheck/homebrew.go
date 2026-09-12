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
}

// NewHomebrewSource uses the currently published Formula, not standalone's
// latest release, which may have been published before Homebrew was updated.
func NewHomebrewSource(client *http.Client) Source {
	return &homebrewSource{client: sourceClient(client)}
}

func (s *homebrewSource) Latest(ctx context.Context) (Candidate, error) {
	body, err := readSource(ctx, s.client, "https://api.github.com/repos/tokenlive/homebrew-tokenlive/contents/Formula/tokenlive.rb")
	if err != nil {
		return Candidate{}, err
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
	return ParseFormula(formula)
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
