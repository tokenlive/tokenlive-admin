package systemversion

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/updatecheck"
	"github.com/tokenlive/tokenlive-admin/internal/versionstatus"
	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
	"github.com/tokenlive/tokenlive-admin/pkg/versionregistry"
)

// ProvideService constructs an unstarted service and registers its cleanup with
// Wire. The Redis client belongs to the application, not this module.
func ProvideService(ctx context.Context, rdb *redis.Client) (*versionstatus.Service, func(), error) {
	options, err := checkerOptions(config.C.UpdateCheck)
	if err != nil {
		return nil, nil, err
	}
	identity := config.C.RuntimeIdentity
	checker, err := updatecheck.NewChecker(ctx, options, sourcesForIdentity(identity, nil))
	if err != nil {
		return nil, nil, fmt.Errorf("configure update checks: %w", err)
	}
	namespace := config.C.Gateway.VersionNamespace
	if value, ok := os.LookupEnv("GATEWAY_VERSION_NAMESPACE"); ok {
		namespace = value
	}
	scope := "this_admin"
	store := versionregistry.NewMemoryStore(namespace, nil)
	if rdb != nil {
		scope = "shared"
		store = versionregistry.NewRedisStore(rdb, namespace)
	}
	service := versionstatus.New(identity, store, checker, scope)
	return service, service.Close, nil
}

// checkerOptions resolves the explicitly supported environment overrides.
// Config loading itself does not map environment variables to these fields.
func checkerOptions(cfg config.UpdateCheckConfig) (updatecheck.Options, error) {
	if value, ok := os.LookupEnv("UPDATE_CHECK_ENABLED"); ok {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return updatecheck.Options{}, fmt.Errorf("UPDATE_CHECK_ENABLED must be a boolean")
		}
		cfg.Enabled = enabled
	}
	if value, ok := os.LookupEnv("UPDATE_CHECK_INTERVAL_SECONDS"); ok {
		seconds, err := strconv.Atoi(value)
		if err != nil {
			return updatecheck.Options{}, fmt.Errorf("UPDATE_CHECK_INTERVAL_SECONDS must be an integer")
		}
		cfg.IntervalSeconds = seconds
	}
	options := updatecheck.Options{Enabled: cfg.Enabled}
	for _, item := range []struct {
		name    string
		seconds int
		target  *time.Duration
	}{
		{"interval", cfg.IntervalSeconds, &options.Interval},
		{"timeout", cfg.TimeoutSeconds, &options.Timeout},
		{"cooldown", cfg.CooldownSeconds, &options.Cooldown},
	} {
		// Reject overflow before converting so a huge configuration cannot
		// wrap into a short/negative timer. NewChecker validates negatives.
		if int64(item.seconds) > math.MaxInt64/int64(time.Second) ||
			int64(item.seconds) < math.MinInt64/int64(time.Second) {
			return updatecheck.Options{}, fmt.Errorf("update check %s seconds exceed duration range", item.name)
		}
		*item.target = time.Duration(item.seconds) * time.Second
	}
	return options, nil
}

func sourcesForIdentity(identity productversion.Identity, client *http.Client) map[string]updatecheck.Source {
	sources := make(map[string]updatecheck.Source)
	switch {
	case identity.Edition == "professional" && identity.InstallChannel == "release":
		sources["admin"] = updatecheck.NewGitHubSource(client, "admin")
		sources["gateway"] = updatecheck.NewGitHubSource(client, "gateway")
	case identity.Edition == "standalone" && identity.InstallChannel == "homebrew":
		sources["standalone"] = updatecheck.NewHomebrewSource(client)
	}
	return sources
}
