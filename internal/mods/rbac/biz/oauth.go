package biz

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/cachex"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

const (
	oauthStateCacheNS  = "oauth_state"
	oauthTicketCacheNS = "oauth_ticket"
)

type OAuth struct {
	Cache               cachex.Cacher
	Trans               *util.Trans
	ExternalIdentityDAL *dal.ExternalIdentity
	UserDAL             *dal.User
	UserRoleDAL         *dal.UserRole
	TenantDAL           *dal.Tenant
	TenantModelDAL      *dal.TenantModel
	RoleDAL             *dal.Role
	LoginBIZ            *Login
	HTTPClient          *http.Client
}

type oauthState struct {
	Provider string `json:"provider"`
	Redirect string `json:"redirect"`
}

type oauthTicket struct {
	UserID string `json:"user_id"`
	Tenant string `json:"tenant"`
}

func (a *OAuth) GetEnabledProviders(ctx context.Context) []schema.OAuthProvider {
	var providers []schema.OAuthProvider
	if !config.C.OAuth.Enabled {
		return providers
	}
	if isOAuthProviderEnabled(schema.OAuthProviderGoogle, config.C.OAuth.Google) {
		providers = append(providers, schema.OAuthProvider{Provider: schema.OAuthProviderGoogle, LoginURL: "/api/v1/oauth/google/login"})
	}
	if isOAuthProviderEnabled(schema.OAuthProviderGitHub, config.C.OAuth.GitHub) {
		providers = append(providers, schema.OAuthProvider{Provider: schema.OAuthProviderGitHub, LoginURL: "/api/v1/oauth/github/login"})
	}
	return providers
}

func (a *OAuth) BuildLoginURL(ctx context.Context, provider, redirect string) (string, error) {
	cfg, err := oauthProviderConfig(provider)
	if err != nil {
		return "", err
	}
	state, err := randomHex(16)
	if err != nil {
		return "", err
	}
	stateBuf, err := json.Marshal(oauthState{Provider: provider, Redirect: redirect})
	if err != nil {
		return "", errors.WithStack(err)
	}
	if err := a.Cache.Set(ctx, oauthStateCacheNS, state, string(stateBuf), 10*time.Minute); err != nil {
		return "", err
	}

	authURL := oauthAuthURL(provider)
	params := url.Values{}
	params.Set("client_id", cfg.ClientID)
	params.Set("redirect_uri", cfg.RedirectURL)
	params.Set("response_type", "code")
	params.Set("state", state)
	params.Set("scope", strings.Join(oauthScopes(provider, cfg), " "))
	return authURL + "?" + params.Encode(), nil
}

func (a *OAuth) HandleCallback(ctx context.Context, provider, code, state string) (string, error) {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(state) == "" {
		return "", errors.BadRequest("", "OAuth code and state are required")
	}
	stateValue, ok, err := a.Cache.GetAndDelete(ctx, oauthStateCacheNS, state)
	if err != nil {
		return "", err
	} else if !ok {
		return "", errors.BadRequest("", "Invalid OAuth state")
	}
	var stored oauthState
	if err := json.Unmarshal([]byte(stateValue), &stored); err != nil {
		return "", errors.BadRequest("", "Invalid OAuth state")
	}
	if stored.Provider != provider {
		return "", errors.BadRequest("", "OAuth provider does not match state")
	}

	profile, err := a.fetchProfile(ctx, provider, code)
	if err != nil {
		return "", err
	}
	resolved, err := a.ResolveProfile(ctx, profile)
	if err != nil {
		return "", err
	}

	ticket, err := randomHex(24)
	if err != nil {
		return "", err
	}
	ticketBuf, err := json.Marshal(oauthTicket{UserID: resolved.User.ID, Tenant: resolved.User.Tenant})
	if err != nil {
		return "", errors.WithStack(err)
	}
	if err := a.Cache.Set(ctx, oauthTicketCacheNS, ticket, string(ticketBuf), 2*time.Minute); err != nil {
		return "", err
	}

	redirectURL := config.C.OAuth.TicketRedirectURL
	if redirectURL == "" {
		redirectURL = "/login"
	}
	redirectWithTicket, err := buildTicketRedirectURL(redirectURL, ticket, stored.Redirect)
	if err != nil {
		return "", err
	}
	return redirectWithTicket, nil
}

func (a *OAuth) ExchangeTicket(ctx context.Context, ticket string) (*schema.LoginToken, error) {
	ticketValue, ok, err := a.Cache.GetAndDelete(ctx, oauthTicketCacheNS, ticket)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.BadRequest("", "Invalid OAuth ticket")
	}
	var parsed oauthTicket
	if err := json.Unmarshal([]byte(ticketValue), &parsed); err != nil {
		return nil, errors.BadRequest("", "Invalid OAuth ticket")
	}
	user, err := a.UserDAL.Get(ctx, parsed.UserID, schema.UserQueryOptions{
		QueryOptions: util.QueryOptions{SelectFields: []string{"id", "status", "tenant"}},
	})
	if err != nil {
		return nil, err
	} else if user == nil || user.Status != schema.UserStatusActivated {
		return nil, errors.BadRequest("", "User status is not activated, please contact the administrator")
	}
	return a.LoginBIZ.genLoginResponse(ctx, parsed.UserID, parsed.Tenant, config.C.OAuth.RefreshToken)
}

func (a *OAuth) ResolveProfile(ctx context.Context, profile schema.OAuthProfile) (*schema.OAuthResolveResult, error) {
	profile.Email = strings.ToLower(strings.TrimSpace(profile.Email))
	profile.Provider = strings.ToLower(strings.TrimSpace(profile.Provider))
	profile.ProviderUserID = strings.TrimSpace(profile.ProviderUserID)
	profile.DisplayName = strings.TrimSpace(profile.DisplayName)

	if err := validateOAuthProfile(profile); err != nil {
		return nil, err
	}

	var result *schema.OAuthResolveResult
	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		identity, err := a.ExternalIdentityDAL.GetByProviderUserID(ctx, profile.Provider, profile.ProviderUserID)
		if err != nil {
			return err
		}
		if identity != nil {
			identity.Email = profile.Email
			identity.EmailVerified = profile.EmailVerified
			identity.DisplayName = profile.DisplayName
			identity.AvatarURL = profile.AvatarURL
			identity.UpdatedAt = time.Now()
			if err := a.ExternalIdentityDAL.UpdateSnapshot(ctx, identity); err != nil {
				return err
			}
			user, tenant, err := a.loadOAuthUserTenant(ctx, identity.UserID)
			if err != nil {
				return err
			}
			result = &schema.OAuthResolveResult{User: user, Tenant: tenant, Identity: identity}
			return nil
		}

		user, err := a.UserDAL.GetByEmail(ctx, profile.Email)
		if err != nil {
			return err
		}
		if user != nil {
			identity, err := a.createExternalIdentity(ctx, user.ID, profile)
			if err != nil {
				return err
			}
			tenant, err := a.loadTenantByCode(ctx, user.Tenant)
			if err != nil {
				return err
			}
			result = &schema.OAuthResolveResult{User: user, Tenant: tenant, Identity: identity}
			return nil
		}

		if !config.C.OAuth.Enabled || !config.C.OAuth.AllowSignup {
			return errors.Forbidden("", "OAuth signup is disabled")
		}
		if !isAllowedOAuthEmail(profile.Email) {
			return errors.Forbidden("", "OAuth signup is not allowed for this email")
		}

		created, err := a.createSelfServiceUser(ctx, profile)
		if err != nil {
			return err
		}
		result = created
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func isOAuthProviderEnabled(provider string, cfg config.OAuthProviderConfig) bool {
	return cfg.Enabled && cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.RedirectURL != "" && (provider == schema.OAuthProviderGoogle || provider == schema.OAuthProviderGitHub)
}

func oauthProviderConfig(provider string) (config.OAuthProviderConfig, error) {
	switch provider {
	case schema.OAuthProviderGoogle:
		cfg := config.C.OAuth.Google
		if !config.C.OAuth.Enabled || !isOAuthProviderEnabled(provider, cfg) {
			return cfg, errors.NotFound("", "OAuth provider not enabled")
		}
		return cfg, nil
	case schema.OAuthProviderGitHub:
		cfg := config.C.OAuth.GitHub
		if !config.C.OAuth.Enabled || !isOAuthProviderEnabled(provider, cfg) {
			return cfg, errors.NotFound("", "OAuth provider not enabled")
		}
		return cfg, nil
	default:
		return config.OAuthProviderConfig{}, errors.NotFound("", "OAuth provider not enabled")
	}
}

func oauthAuthURL(provider string) string {
	if provider == schema.OAuthProviderGoogle {
		return "https://accounts.google.com/o/oauth2/v2/auth"
	}
	return "https://github.com/login/oauth/authorize"
}

func oauthScopes(provider string, cfg config.OAuthProviderConfig) []string {
	if len(cfg.Scopes) > 0 {
		return cfg.Scopes
	}
	if provider == schema.OAuthProviderGoogle {
		return []string{"openid", "email", "profile"}
	}
	return []string{"read:user", "user:email"}
}

func buildTicketRedirectURL(rawURL, ticket, redirect string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.BadRequest("", "Invalid OAuth ticket redirect url")
	}
	if u.Fragment != "" {
		fragmentPath, fragmentQuery, _ := strings.Cut(u.Fragment, "?")
		q, err := url.ParseQuery(fragmentQuery)
		if err != nil {
			return "", errors.BadRequest("", "Invalid OAuth ticket redirect url")
		}
		q.Set("oauth_ticket", ticket)
		if redirect != "" {
			q.Set("redirect", redirect)
		}
		rawFragment := fragmentPath + "?" + q.Encode()
		u.RawFragment = rawFragment
		decodedFragment, err := url.PathUnescape(rawFragment)
		if err != nil {
			return "", errors.BadRequest("", "Invalid OAuth ticket redirect url")
		}
		u.Fragment = decodedFragment
		return u.String(), nil
	}
	q := u.Query()
	q.Set("oauth_ticket", ticket)
	if redirect != "" {
		q.Set("redirect", redirect)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func validateOAuthProfile(profile schema.OAuthProfile) error {
	if profile.Provider != schema.OAuthProviderGoogle && profile.Provider != schema.OAuthProviderGitHub {
		return errors.BadRequest("", "Unsupported OAuth provider")
	}
	if profile.ProviderUserID == "" {
		return errors.BadRequest("", "OAuth provider user id is required")
	}
	if !profile.EmailVerified {
		return errors.Forbidden("", "OAuth email must be verified")
	}
	if _, err := mail.ParseAddress(profile.Email); err != nil {
		return errors.BadRequest("", "Invalid OAuth email")
	}
	return nil
}

func isAllowedOAuthEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	for _, allowed := range config.C.OAuth.AllowedEmails {
		if strings.ToLower(strings.TrimSpace(allowed)) == email {
			return true
		}
	}
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	domain := email[at+1:]
	for _, allowed := range config.C.OAuth.AllowedDomains {
		if strings.ToLower(strings.TrimPrefix(strings.TrimSpace(allowed), "@")) == domain {
			return true
		}
	}
	return false
}

func (a *OAuth) createSelfServiceUser(ctx context.Context, profile schema.OAuthProfile) (*schema.OAuthResolveResult, error) {
	now := time.Now()
	userID := util.NewXID()
	tenantCode := "u-" + userID

	tenantAPIKey, err := generateTenantAPIKey()
	if err != nil {
		return nil, err
	}

	displayName := profile.DisplayName
	if displayName == "" {
		displayName = profile.Email
	}

	tenant := &schema.Tenant{
		ID:          util.NewXID(),
		Code:        tenantCode,
		Name:        displayName,
		Status:      schema.TenantStatusActivated,
		Description: "Self-service tenant",
		APIKey:      tenantAPIKey,
		Creator:     profile.Email,
		CreatedAt:   now,
	}
	if err := a.TenantDAL.Create(ctx, tenant); err != nil {
		return nil, err
	}

	user := &schema.User{
		ID:        userID,
		Username:  profile.Email,
		Name:      displayName,
		Email:     profile.Email,
		Tenant:    tenantCode,
		Status:    schema.UserStatusActivated,
		CreatedAt: now,
	}
	if err := a.UserDAL.Create(ctx, user); err != nil {
		return nil, err
	}

	if err := a.createDefaultUserRole(ctx, user.ID); err != nil {
		return nil, err
	}
	if err := a.createDefaultTenantModels(ctx, tenantCode, profile.Email); err != nil {
		return nil, err
	}

	identity, err := a.createExternalIdentity(ctx, user.ID, profile)
	if err != nil {
		return nil, err
	}
	return &schema.OAuthResolveResult{
		User:     user,
		Tenant:   tenant,
		Identity: identity,
		Created:  true,
	}, nil
}

func (a *OAuth) createDefaultUserRole(ctx context.Context, userID string) error {
	roleCode := strings.TrimSpace(config.C.OAuth.DefaultRoleCode)
	if roleCode == "" {
		return nil
	}
	role, err := a.RoleDAL.GetByCode(ctx, roleCode)
	if err != nil {
		return err
	} else if role == nil || role.Status != schema.RoleStatusEnabled {
		return errors.BadRequest("", "OAuth default role is not available")
	}
	return a.UserRoleDAL.Create(ctx, &schema.UserRole{
		ID:        util.NewXID(),
		UserID:    userID,
		RoleID:    role.ID,
		CreatedAt: time.Now(),
	})
}

func (a *OAuth) createDefaultTenantModels(ctx context.Context, tenantCode, creator string) error {
	modelIDs := config.C.OAuth.DefaultModelIDs
	if len(modelIDs) == 0 {
		return nil
	}
	items := make([]*schema.TenantModel, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		items = append(items, &schema.TenantModel{
			ID:         util.NewXID(),
			TenantCode: tenantCode,
			ModelID:    modelID,
			Creator:    creator,
		})
	}
	return a.TenantModelDAL.CreateInBatches(ctx, util.GetDB(ctx, a.TenantModelDAL.DB), items)
}

func (a *OAuth) createExternalIdentity(ctx context.Context, userID string, profile schema.OAuthProfile) (*schema.ExternalIdentity, error) {
	identity := &schema.ExternalIdentity{
		ID:             util.NewXID(),
		UserID:         userID,
		Provider:       profile.Provider,
		ProviderUserID: profile.ProviderUserID,
		Email:          profile.Email,
		EmailVerified:  profile.EmailVerified,
		DisplayName:    profile.DisplayName,
		AvatarURL:      profile.AvatarURL,
		CreatedAt:      time.Now(),
	}
	if err := a.ExternalIdentityDAL.Create(ctx, identity); err != nil {
		return nil, err
	}
	return identity, nil
}

func (a *OAuth) loadOAuthUserTenant(ctx context.Context, userID string) (*schema.User, *schema.Tenant, error) {
	user, err := a.UserDAL.Get(ctx, userID)
	if err != nil {
		return nil, nil, err
	} else if user == nil {
		return nil, nil, errors.NotFound("", "User not found")
	}
	tenant, err := a.loadTenantByCode(ctx, user.Tenant)
	return user, tenant, err
}

func (a *OAuth) loadTenantByCode(ctx context.Context, tenantCode string) (*schema.Tenant, error) {
	if tenantCode == "" {
		return nil, nil
	}
	var tenant schema.Tenant
	db := dal.GetTenantDB(ctx, a.TenantDAL.DB).Where("code = ? AND deleted = '0'", tenantCode)
	ok, err := util.FindOne(ctx, db, util.QueryOptions{}, &tenant)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return &tenant, nil
}

func generateTenantAPIKey() (string, error) {
	token, err := randomHex(16)
	if err != nil {
		return "", err
	}
	return "sk-t-" + token, nil
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.WithStack(err)
	}
	return hex.EncodeToString(buf), nil
}

func (a *OAuth) httpClient() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
}

func (a *OAuth) fetchProfile(ctx context.Context, provider, code string) (schema.OAuthProfile, error) {
	cfg, err := oauthProviderConfig(provider)
	if err != nil {
		return schema.OAuthProfile{}, err
	}
	token, err := a.exchangeProviderToken(ctx, provider, cfg, code)
	if err != nil {
		return schema.OAuthProfile{}, err
	}
	if provider == schema.OAuthProviderGoogle {
		return a.fetchGoogleProfile(ctx, token)
	}
	return a.fetchGitHubProfile(ctx, token)
}

func (a *OAuth) exchangeProviderToken(ctx context.Context, provider string, cfg config.OAuthProviderConfig, code string) (string, error) {
	tokenURL := "https://oauth2.googleapis.com/token"
	if provider == schema.OAuthProviderGitHub {
		tokenURL = "https://github.com/login/oauth/access_token"
	}
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", cfg.RedirectURL)
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", errors.WithStack(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := a.httpClient().Do(req)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.BadRequest("", "OAuth token exchange failed")
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", errors.BadRequest("", "Invalid OAuth token response")
	}
	if tokenResp.AccessToken == "" {
		return "", errors.BadRequest("", "OAuth token response missing access token")
	}
	return tokenResp.AccessToken, nil
}

func (a *OAuth) fetchGoogleProfile(ctx context.Context, accessToken string) (schema.OAuthProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return schema.OAuthProfile{}, errors.WithStack(err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := a.httpClient().Do(req)
	if err != nil {
		return schema.OAuthProfile{}, errors.WithStack(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return schema.OAuthProfile{}, errors.WithStack(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return schema.OAuthProfile{}, errors.BadRequest("", "OAuth profile request failed")
	}
	var userInfo struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return schema.OAuthProfile{}, errors.BadRequest("", "Invalid OAuth profile response")
	}
	return schema.OAuthProfile{
		Provider:       schema.OAuthProviderGoogle,
		ProviderUserID: userInfo.Sub,
		Email:          userInfo.Email,
		EmailVerified:  userInfo.EmailVerified,
		DisplayName:    userInfo.Name,
		AvatarURL:      userInfo.Picture,
	}, nil
}

func (a *OAuth) fetchGitHubProfile(ctx context.Context, accessToken string) (schema.OAuthProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return schema.OAuthProfile{}, errors.WithStack(err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := a.httpClient().Do(req)
	if err != nil {
		return schema.OAuthProfile{}, errors.WithStack(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return schema.OAuthProfile{}, errors.WithStack(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return schema.OAuthProfile{}, errors.BadRequest("", "OAuth profile request failed")
	}
	var userInfo struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return schema.OAuthProfile{}, errors.BadRequest("", "Invalid OAuth profile response")
	}
	email, verified, err := a.fetchGitHubPrimaryEmail(ctx, accessToken, userInfo.Email)
	if err != nil {
		return schema.OAuthProfile{}, err
	}
	displayName := userInfo.Name
	if displayName == "" {
		displayName = userInfo.Login
	}
	return schema.OAuthProfile{
		Provider:       schema.OAuthProviderGitHub,
		ProviderUserID: strconv.FormatInt(userInfo.ID, 10),
		Email:          email,
		EmailVerified:  verified,
		DisplayName:    displayName,
		AvatarURL:      userInfo.AvatarURL,
	}, nil
}

func (a *OAuth) fetchGitHubPrimaryEmail(ctx context.Context, accessToken, fallbackEmail string) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", false, errors.WithStack(err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := a.httpClient().Do(req)
	if err != nil {
		return "", false, errors.WithStack(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, errors.WithStack(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := json.Unmarshal(body, &emails); err == nil {
			for _, item := range emails {
				if item.Primary {
					return item.Email, item.Verified, nil
				}
			}
			for _, item := range emails {
				if item.Verified {
					return item.Email, true, nil
				}
			}
		}
	}
	if fallbackEmail != "" {
		return fallbackEmail, false, nil
	}
	return "", false, errors.BadRequest("", "GitHub verified email not found")
}
