package gatewaycontract

import "strconv"

// Keys is the single place that assembles aigw:* Redis keys.
//
// A constructor that Gateway reads but Admin never writes (Policies.Global,
// User.Models) is still here, so the next caller does not re-spell the string.
var Keys = struct {
	Config   configKeys
	Policies policyKeys
	Tenant   tenantKeys
	User     userKeys
	Status   statusKeys
	Channel  channelKeys
	Events   eventsKeys
	Circuit  circuitKeys
	APIKey   apiKeyKeys
}{
	Policies: policyKeys{Global: "aigw:policies:global"},
	Events:   eventsKeys{Policy: "aigw:events:policy"},
	Circuit: circuitKeys{
		OpenEndpoints: "aigw:cb:open_endpoints",
		OpenServices:  "aigw:cb:open_services",
	},
}

type configKeys struct{}

// ModelVersions is the HASH of modelCode → routing generation.
func (configKeys) ModelVersions() string { return "aigw:config:model_versions" }

// Endpoints is the STRING holding a JSON []ResolvedEndpoint for a normal model.
func (configKeys) Endpoints(modelCode string) string {
	return "aigw:config:endpoints:" + modelCode
}

// SmartRouting is the STRING holding a JSON RuntimeSmartRouting. It is mutually
// exclusive with Endpoints for the same model.
func (configKeys) SmartRouting(modelCode string) string {
	return "aigw:config:smart_routing:" + modelCode
}

// Alias is the STRING mapping an alias to a model code.
func (configKeys) Alias(alias string) string { return "aigw:config:alias:" + alias }

// ModelAliases is the SET of aliases that point at a model.
func (configKeys) ModelAliases(modelCode string) string {
	return "aigw:config:model_aliases:" + modelCode
}

type policyKeys struct {
	// Global is read by Gateway and never written by Admin.
	Global string
}

// FieldBilling is the hash field carrying a model's price map. It is distinct
// from FieldAll, which carries governance JSON.
const FieldBilling = "*:billing"

// FieldAll is the hash field carrying the dimension's default governance policy.
const FieldAll = "*"

// BillingField is the per-model price field Gateway HMGets alongside a
// governance field (modelCode + ":billing").
func BillingField(modelCode string) string { return modelCode + ":billing" }

func (policyKeys) User(userID string) string { return "aigw:policies:user:" + userID }
func (policyKeys) Tenant(tenantCode string) string {
	return "aigw:policies:tenant:" + tenantCode
}
func (policyKeys) Model(modelCode string) string { return "aigw:policies:model:" + modelCode }

type tenantKeys struct{}

const tenantPrefix = "aigw:tenant:"

// Prefix is the SCAN pattern root for tenant grant keys.
func (tenantKeys) Prefix() string { return tenantPrefix }

// ModelsSuffix is the SCAN suffix of a tenant's allowed-model set.
func (tenantKeys) ModelsSuffix() string { return ":models" }

// EndpointsSuffix is the SCAN suffix of one model's endpoint whitelist.
func (tenantKeys) EndpointsSuffix(modelCode string) string {
	return ":model:" + modelCode + ":endpoints"
}

// Models is the SET of model codes a tenant may call.
func (tenantKeys) Models(tenantCode string) string {
	return tenantPrefix + tenantCode + ":models"
}

// Endpoints is the SET of endpoint IDs a tenant may call for one model.
// An empty or missing set means allow all.
func (tenantKeys) Endpoints(tenantCode, modelCode string) string {
	return tenantPrefix + tenantCode + ":model:" + modelCode + ":endpoints"
}

// Providers is the legacy provider whitelist. Admin only deletes it.
func (tenantKeys) Providers(tenantCode, modelCode string) string {
	return tenantPrefix + tenantCode + ":model:" + modelCode + ":providers"
}

// Temp is the staging key used before an atomic RENAME onto key.
func (tenantKeys) Temp(key string) string { return key + ":tmp" }

type userKeys struct{}

// Models is read by Gateway and never written by Admin.
func (userKeys) Models(userID string) string { return "aigw:user:" + userID + ":models" }

type statusKeys struct{}

func (statusKeys) Global(minute int64, metric string) string {
	return "aigw:status:global:" + strconv.FormatInt(minute, 10) + ":" + metric
}

func (statusKeys) Model(modelCode string, minute int64, metric string) string {
	return "aigw:status:model:" + modelCode + ":" + strconv.FormatInt(minute, 10) + ":" + metric
}

func (statusKeys) Endpoint(endpointID string, minute int64, metric string) string {
	return "aigw:status:endpoint:" + endpointID + ":" + strconv.FormatInt(minute, 10) + ":" + metric
}

func (statusKeys) Provider(provider string, minute int64, metric string) string {
	return "aigw:status:provider:" + provider + ":" + strconv.FormatInt(minute, 10) + ":" + metric
}

func (statusKeys) Daily(metric, date string) string {
	return "aigw:status:daily:" + metric + ":" + date
}

// Status counter suffixes written by Gateway.
const (
	MetricSuccess      = "s"
	MetricFailure      = "f"
	MetricTTFTSum      = "ttft_sum"
	MetricTTFTCount    = "ttft_cnt"
	MetricOutputTokens = "out"
	MetricDurationMs   = "dur_ms"
)

// Daily counter names written by Gateway.
const (
	DailyRequests            = "req"
	DailyInputTokens         = "input_tokens"
	DailyOutputTokens        = "output_tokens"
	DailyCachedTokens        = "cached_tokens"
	DailyCacheCreationTokens = "cache_creation_tokens"
	DailyCost                = "cost"
)

type channelKeys struct{}

func (channelKeys) PolicyUpdate() string { return "aigw:channel:policy_update" }
func (channelKeys) APIKeyUpdate() string { return "aigw:channel:apikey_update" }

// PurgePayload is the only payload Admin publishes on either channel.
const PurgePayload = "purge"

type eventsKeys struct {
	// Policy is the Redis stream of gateway policy-execution events.
	Policy string
}

type circuitKeys struct {
	// OpenEndpoints and OpenServices are read by Admin and written by neither
	// side. Live circuit-breaker state arrives over HTTP metrics.
	OpenEndpoints string
	OpenServices  string
}

type apiKeyKeys struct{}

// Hash is the HASH identifying a runtime API key by its HMAC.
func (apiKeyKeys) Hash(keyHash string) string { return "aigw:apikey_hash:" + keyHash }
