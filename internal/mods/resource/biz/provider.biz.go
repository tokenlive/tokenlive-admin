package biz

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Provider business logic layer
type Provider struct {
	Trans             *util.Trans
	ProviderDAL       *dal.Provider
	DataPermissionBIZ *DataPermission
	ConfigRedisSync   *ConfigRedisSync
}

func (p *Provider) Query(ctx context.Context, params schema.ProviderQueryParam) (*schema.ProviderQueryResult, error) {
	params.Pagination = true

	result, err := p.ProviderDAL.Query(ctx, params, schema.ProviderQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Get the specified provider.
func (p *Provider) Get(ctx context.Context, id string) (*schema.Provider, error) {
	provider, err := p.ProviderDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if provider == nil {
		return nil, errors.NotFound("", "Provider not found")
	}

	if !util.FromIsRootUser(ctx) {
		ok, err := p.DataPermissionBIZ.HasReadPermission(ctx, schema.DataPermissionTypeProvider, id)
		if err != nil {
			return nil, err
		} else if !ok {
			return nil, errors.NotFound("", "Provider not found")
		}
	}

	return provider, nil
}

// Create a new provider.
func (p *Provider) Create(ctx context.Context, formItem *schema.ProviderForm) (*schema.Provider, error) {
	if exists, err := p.ProviderDAL.ExistsCode(ctx, formItem.Code); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Provider code already exists")
	}

	provider := &schema.Provider{
		ID:        util.NewXID(),
		Creator:   util.FromUsername(ctx),
		CreatedAt: time.Now(),
	}
	if err := formItem.FillTo(provider); err != nil {
		return nil, err
	}

	err := p.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := p.ProviderDAL.Create(ctx, provider); err != nil {
			return err
		}
		return p.DataPermissionBIZ.CreateByOwner(ctx, schema.DataPermissionTypeProvider, provider.ID, util.FromTenant(ctx))
	})
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// Update the specified provider.
func (p *Provider) Update(ctx context.Context, id string, formItem *schema.ProviderForm) error {
	provider, err := p.ProviderDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if provider == nil {
		return errors.NotFound("", "Provider not found")
	} else if provider.Code != formItem.Code {
		if exists, err := p.ProviderDAL.ExistsCode(ctx, formItem.Code); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Provider code already exists")
		}
	}

	if err := formItem.FillTo(provider); err != nil {
		return err
	}
	provider.Modifier = util.FromUsername(ctx)
	provider.UpdatedAt = time.Now()

	err = p.Trans.Exec(ctx, func(ctx context.Context) error {
		return p.ProviderDAL.Update(ctx, provider)
	})
	if err == nil {
		_ = p.ConfigRedisSync.SyncProviderID(ctx, provider.ID)
	}
	return err
}

// Delete the specified provider.
func (p *Provider) Delete(ctx context.Context, id string) error {
	provider, err := p.ProviderDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if provider == nil {
		return errors.NotFound("", "Provider not found")
	}

	// Check if provider has associated models
	affectedModelCodes, _ := p.ConfigRedisSync.GetModelCodesByProvider(ctx, id)
	if len(affectedModelCodes) > 0 {
		return errors.BadRequest("", "Cannot delete provider with associated models. Please remove all model associations first.")
	}

	err = p.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := p.ProviderDAL.Delete(ctx, id); err != nil {
			return err
		}
		return p.DataPermissionBIZ.DeleteByTypeAndDataId(ctx, schema.DataPermissionTypeProvider, id)
	})
	return err
}

// FetchModels calls the upstream provider's /v1/models API and returns the model list.
func (p *Provider) FetchModels(ctx context.Context, providerID string, formItem *schema.FetchModelsForm) (*schema.FetchModelsResult, error) {
	provider, err := p.ProviderDAL.Get(ctx, providerID)
	if err != nil {
		return nil, err
	} else if provider == nil {
		return nil, errors.NotFound("", "Provider not found")
	}

	apiKey := formItem.APIKey
	if apiKey == "" {
		keys := provider.GetApiKeys()
		if len(keys) > 0 {
			apiKey = keys[0]
		}
	}

	// Build the upstream URL
	baseURL := strings.TrimRight(formItem.BaseURL, "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(provider.URL, "/")
	}
	if baseURL == "" {
		return nil, errors.BadRequest("", "Base URL is required, please provide it or set provider URL")
	}
	reqURL := baseURL + "/models"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.BadRequest("", "Failed to create request: %s", err.Error())
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.BadRequest("", "Failed to call upstream: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.BadRequest("", "Upstream returned status %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp struct {
		Data []schema.UpstreamModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, errors.BadRequest("", "Failed to parse upstream response: %s", err.Error())
	}

	return &schema.FetchModelsResult{Models: modelsResp.Data}, nil
}
