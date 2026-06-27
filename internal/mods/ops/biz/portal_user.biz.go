package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
)

type PortalUser struct{}

type PortalUserResult struct {
	ID           string  `json:"id"`
	DisplayName  string  `json:"display_name"`
	PrimaryEmail *string `json:"primary_email"`
}

type PortalSearchResponse struct {
	Users []PortalUserResult `json:"users"`
}

func (a *PortalUser) Search(ctx context.Context, keyword string, limit int) ([]PortalUserResult, error) {
	portalCfg := config.C.Portal
	if portalCfg.BaseURL == "" {
		return nil, fmt.Errorf("portal base URL not configured")
	}

	searchURL := fmt.Sprintf("%s/internal/v1/users/search?keyword=%s&limit=%d",
		portalCfg.BaseURL, url.QueryEscape(keyword), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}

	if portalCfg.InternalAPIToken != "" {
		req.Header.Set("Authorization", "Bearer "+portalCfg.InternalAPIToken)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("portal returned unexpected status: %d", resp.StatusCode)
	}

	var res PortalSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return res.Users, nil
}
