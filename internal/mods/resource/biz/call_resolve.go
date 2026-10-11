package biz

import (
	"strings"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
)

// CallInput is the already-loaded material for one endpoint call.
// Callers own OAuth refresh, auth-type fallback, and protocol fallback;
// this function does not decide those.
type CallInput struct {
	EndpointRealModel string
	ModelCode         string
	AuthType          string
	EndpointAPIKey    string
	ProviderAPIKeys   []string
	Headers           map[string]string
	// OAuthAccountID is the provider OAuth account id. Empty means no injection.
	OAuthAccountID string
}

// CallSpec is the resolved identity of one upstream call.
// APIKeys keeps every selected key; callers decide whether to fan out or
// use the first one.
type CallSpec struct {
	RealModel string
	AuthType  string
	APIKeys   []string
	Headers   map[string]string
}

// ResolveCall applies the inheritance shared by the Redis publish path, the
// HTTP pull path, and endpoint probes.
//
// A non-oauth endpoint key wins. oauth_token always uses the provider keys,
// because that credential is a provider identity, not an endpoint override.
// An empty result is returned as-is; callers choose whether an empty key is
// publishable.
func ResolveCall(in CallInput) CallSpec {
	realModel := in.EndpointRealModel
	if realModel == "" {
		realModel = in.ModelCode
	}

	var apiKeys []string
	if in.AuthType != "oauth_token" && in.EndpointAPIKey != "" {
		apiKeys = []string{in.EndpointAPIKey}
	} else if len(in.ProviderAPIKeys) > 0 {
		apiKeys = append([]string(nil), in.ProviderAPIKeys...)
	}

	return CallSpec{
		RealModel: realModel,
		AuthType:  in.AuthType,
		APIKeys:   apiKeys,
		Headers:   injectOAuthAccountHeader(in.Headers, in.AuthType, in.OAuthAccountID),
	}
}

func providerAPIKeyValues(provider *schema.Provider) []string {
	if provider == nil {
		return nil
	}
	items := provider.GetApiKeys()
	if len(items) == 0 {
		return nil
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, item.Value)
	}
	return values
}

func providerOAuthAccountID(provider *schema.Provider) string {
	if provider == nil {
		return ""
	}
	cred := provider.GetOAuth()
	if cred == nil {
		return ""
	}
	return cred.AccountID
}

// injectOAuthAccountHeader copies headers and adds Chatgpt-Account-Id for an
// oauth_token call when the caller supplies an account id. An existing value
// wins, matched case-insensitively.
func injectOAuthAccountHeader(headers map[string]string, authType, accountID string) map[string]string {
	accountID = strings.TrimSpace(accountID)
	if authType != "oauth_token" || accountID == "" {
		return headers
	}
	for k := range headers {
		if strings.EqualFold(k, "Chatgpt-Account-Id") {
			return headers
		}
	}
	out := make(map[string]string, len(headers)+1)
	for k, v := range headers {
		out[k] = v
	}
	out["Chatgpt-Account-Id"] = accountID
	return out
}
