package biz

import "testing"

func TestResolveCall(t *testing.T) {
	tests := []struct {
		name string
		in   CallInput
		want CallSpec
	}{
		{
			name: "endpoint key wins for api key auth",
			in: CallInput{
				EndpointRealModel: "alias-model",
				ModelCode:         "gpt-5",
				AuthType:          "api_key",
				EndpointAPIKey:    "ep-key",
				ProviderAPIKeys:   []string{"provider-key"},
			},
			want: CallSpec{RealModel: "alias-model", AuthType: "api_key", APIKeys: []string{"ep-key"}},
		},
		{
			name: "oauth ignores endpoint key and keeps every provider key",
			in: CallInput{
				ModelCode:       "gpt-5",
				AuthType:        "oauth_token",
				EndpointAPIKey:  "stale-snapshot",
				ProviderAPIKeys: []string{"token-a", "token-b"},
			},
			want: CallSpec{RealModel: "gpt-5", AuthType: "oauth_token", APIKeys: []string{"token-a", "token-b"}},
		},
		{
			name: "empty endpoint key falls back to provider keys",
			in: CallInput{
				ModelCode:       "gpt-5",
				AuthType:        "api_key",
				ProviderAPIKeys: []string{"provider-key"},
			},
			want: CallSpec{RealModel: "gpt-5", AuthType: "api_key", APIKeys: []string{"provider-key"}},
		},
		{
			name: "no key stays empty",
			in:   CallInput{ModelCode: "gpt-5", AuthType: "api_key"},
			want: CallSpec{RealModel: "gpt-5", AuthType: "api_key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveCall(tt.in)
			if got.RealModel != tt.want.RealModel || got.AuthType != tt.want.AuthType || !sameStrings(got.APIKeys, tt.want.APIKeys) {
				t.Fatalf("ResolveCall() = %+v, want %+v", got, tt.want)
			}
			if got.Headers != nil {
				t.Fatalf("headers = %#v, want nil", got.Headers)
			}
		})
	}
}

func TestResolveCallOAuthAccountHeader(t *testing.T) {
	existing := map[string]string{"chatgpt-account-id": "already"}
	got := ResolveCall(CallInput{
		AuthType:       "oauth_token",
		Headers:        existing,
		OAuthAccountID: "injected",
	})
	if got.Headers["chatgpt-account-id"] != "already" || len(got.Headers) != 1 {
		t.Fatalf("existing header changed: %#v", got.Headers)
	}
	if _, ok := existing["Chatgpt-Account-Id"]; ok {
		t.Fatal("resolver mutated the caller header map")
	}

	injected := ResolveCall(CallInput{
		AuthType:       "oauth_token",
		Headers:        map[string]string{"X-Trace": "1"},
		OAuthAccountID: " acct ",
	})
	if injected.Headers["Chatgpt-Account-Id"] != "acct" || injected.Headers["X-Trace"] != "1" {
		t.Fatalf("header injection = %#v", injected.Headers)
	}

	plain := ResolveCall(CallInput{
		AuthType:       "api_key",
		Headers:        map[string]string{"X-Trace": "1"},
		OAuthAccountID: "acct",
	})
	if _, ok := plain.Headers["Chatgpt-Account-Id"]; ok {
		t.Fatal("non-oauth call received an account header")
	}
}

func TestResolveCallDoesNotAliasProviderKeys(t *testing.T) {
	keys := []string{"token-a"}
	got := ResolveCall(CallInput{AuthType: "oauth_token", ProviderAPIKeys: keys})
	keys[0] = "mutated"
	if len(got.APIKeys) != 1 || got.APIKeys[0] != "token-a" {
		t.Fatalf("resolver aliased the caller key slice: %#v", got.APIKeys)
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
