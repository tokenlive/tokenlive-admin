package biz

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
)

const (
	geminiDefaultBaseURL         = "https://generativelanguage.googleapis.com/v1beta"
	geminiModelsMaxPages         = 50
	geminiModelsMaxResponseBytes = 10 << 20 // 10 MiB
)

var geminiAPIVersionPattern = regexp.MustCompile(`(?i)^v\d+(?:beta\d*|alpha\d*)?$`)

type geminiCatalogModel struct {
	Name                       string   `json:"name"`
	DisplayName                string   `json:"displayName"`
	Description                string   `json:"description"`
	InputTokenLimit            *int     `json:"inputTokenLimit"`
	OutputTokenLimit           *int     `json:"outputTokenLimit"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
}

func validateFetchModelsBaseURLOverride(provider *schema.Provider, formItem *schema.FetchModelsForm) error {
	if provider == nil || formItem == nil {
		return nil
	}
	overrideURL := strings.TrimSpace(formItem.BaseURL)
	if overrideURL == "" || strings.TrimSpace(formItem.APIKey) != "" {
		return nil
	}
	// Only guard when stored credentials would be forwarded to a different host.
	hasStoredCredential := len(provider.GetApiKeys()) > 0
	if !hasStoredCredential {
		return nil
	}
	trustedBaseURL := strings.TrimSpace(provider.URL)
	if trustedBaseURL == "" {
		switch {
		case provider.Protocol == "gemini":
			trustedBaseURL = geminiDefaultBaseURL
		case isCodexOAuthProvider(provider):
			trustedBaseURL = codexDefaultBaseURL
		case provider.Protocol == "joycode":
			trustedBaseURL = "https://api-ai.jd.com"
		default:
			return nil
		}
	}
	if !sameURLOrigin(trustedBaseURL, overrideURL) {
		return errors.BadRequest("", "Overriding base_url to a different host requires an explicit api_key")
	}
	return nil
}

func sameURLOrigin(a, b string) bool {
	ua, errA := url.Parse(strings.TrimSpace(a))
	ub, errB := url.Parse(strings.TrimSpace(b))
	if errA != nil || errB != nil || ua.Host == "" || ub.Host == "" {
		return false
	}
	if !strings.EqualFold(ua.Scheme, ub.Scheme) {
		return false
	}
	return strings.EqualFold(ua.Hostname(), ub.Hostname()) && effectiveURLPort(ua) == effectiveURLPort(ub)
}

func effectiveURLPort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

func (p *Provider) fetchGeminiModels(ctx context.Context, baseURL, apiKey string) (*schema.FetchModelsResult, error) {
	baseURL, err := normalizeGeminiModelsBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	catalogURL, _ := url.Parse(baseURL)
	catalogURL.Path += "/models"

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	client := &http.Client{
		Timeout: 30 * time.Second,
		// Go forwards custom auth headers on redirects, including x-goog-api-key.
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	models := make([]schema.UpstreamModel, 0)
	seenModelIDs := make(map[string]struct{})
	seenPageTokens := make(map[string]bool)
	pageToken := ""
	for pageCount := 0; ; pageCount++ {
		if pageCount >= geminiModelsMaxPages {
			return nil, errors.BadRequest("", "Gemini models API exceeded maximum pagination limit")
		}
		query := catalogURL.Query()
		query.Set("pageSize", "60")
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		catalogURL.RawQuery = query.Encode()
		pageModels, nextPageToken, err := fetchGeminiModelsPage(ctx, client, catalogURL.String(), apiKey)
		if err != nil {
			return nil, err
		}
		for _, model := range pageModels {
			if _, dup := seenModelIDs[model.ID]; dup {
				continue
			}
			seenModelIDs[model.ID] = struct{}{}
			models = append(models, model)
		}
		if nextPageToken == "" {
			return &schema.FetchModelsResult{Models: models, BaseURL: baseURL}, nil
		}
		if seenPageTokens[nextPageToken] {
			return nil, errors.BadRequest("", "Gemini models API returned a repeated pagination token")
		}
		seenPageTokens[nextPageToken] = true
		pageToken = nextPageToken
	}
}

func normalizeGeminiModelsBaseURL(baseURL string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = geminiDefaultBaseURL
	}
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return "", errors.BadRequest("", "Invalid Gemini base URL: an absolute HTTP(S) URL is required")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", errors.BadRequest("", "Gemini base URL must not contain query parameters or a fragment")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	for {
		lowerPath := strings.ToLower(u.Path)
		switch {
		case strings.HasSuffix(lowerPath, "/models"):
			u.Path = strings.TrimRight(u.Path[:len(u.Path)-len("/models")], "/")
		case strings.HasSuffix(lowerPath, "/openai"):
			u.Path = strings.TrimRight(u.Path[:len(u.Path)-len("/openai")], "/")
		default:
			goto doneTrim
		}
	}
doneTrim:
	if base := path.Base(u.Path); geminiAPIVersionPattern.MatchString(base) {
		u.Path = u.Path[:len(u.Path)-len(base)] + strings.ToLower(base)
	} else {
		u.Path += "/v1beta"
	}
	u.RawPath = ""
	return u.String(), nil
}

func fetchGeminiModelsPage(ctx context.Context, client *http.Client, reqURL, apiKey string) ([]schema.UpstreamModel, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, "", errors.BadRequest("", "Failed to create Gemini models request: %s", err.Error())
	}
	if apiKey != "" {
		req.Header.Set("x-goog-api-key", apiKey)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", errors.BadRequest("", "Failed to call Gemini models API: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, "", errors.BadRequest("", "Gemini models API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, geminiModelsMaxResponseBytes+1))
	if err != nil {
		return nil, "", errors.BadRequest("", "Failed to read Gemini models response: %s", err.Error())
	}
	if len(body) > geminiModelsMaxResponseBytes {
		return nil, "", errors.BadRequest("", "Gemini models API response exceeded size limit")
	}

	var response map[string]json.RawMessage
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, "", errors.BadRequest("", "Failed to parse Gemini models response: %s", err.Error())
	}
	if response == nil {
		return nil, "", errors.BadRequest("", "Gemini models response must be an object")
	}
	var catalogModels []geminiCatalogModel
	if raw, exists := response["models"]; exists {
		if err := json.Unmarshal(raw, &catalogModels); err != nil || string(raw) == "null" {
			return nil, "", errors.BadRequest("", "Failed to parse Gemini models list")
		}
	} else {
		// Protobuf JSON can omit an empty repeated field, but not replace it with OpenAI data.
		for key := range response {
			if key != "nextPageToken" {
				return nil, "", errors.BadRequest("", "Gemini models response is missing the models list")
			}
		}
	}
	var nextPageToken string
	if raw, exists := response["nextPageToken"]; exists {
		if err := json.Unmarshal(raw, &nextPageToken); err != nil {
			return nil, "", errors.BadRequest("", "Failed to parse Gemini pagination token: %s", err.Error())
		}
	}
	models := make([]schema.UpstreamModel, 0, len(catalogModels))
	for _, model := range catalogModels {
		name := strings.TrimSpace(model.Name)
		id := strings.TrimPrefix(name, "models/")
		if id == "" {
			return nil, "", errors.BadRequest("", "Gemini models response contains a model without a name")
		}
		inputLimit := sanitizeNonNegativeLimit(model.InputTokenLimit)
		outputLimit := sanitizeNonNegativeLimit(model.OutputTokenLimit)
		models = append(models, schema.UpstreamModel{
			ID:                         id,
			Object:                     "model",
			OwnedBy:                    "google",
			Name:                       name,
			DisplayName:                model.DisplayName,
			Description:                model.Description,
			InputTokenLimit:            inputLimit,
			OutputTokenLimit:           outputLimit,
			SupportedGenerationMethods: model.SupportedGenerationMethods,
		})
	}
	return models, nextPageToken, nil
}

func sanitizeNonNegativeLimit(limit *int) *int {
	if limit == nil || *limit < 0 {
		return nil
	}
	v := *limit
	return &v
}
