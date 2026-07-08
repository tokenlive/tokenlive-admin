package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePolicyScopeRejectsUnknownScopeType(t *testing.T) {
	err := validatePolicyScope("workspace", "", false)

	require.Error(t, err)
}

func TestValidatePolicyScopeRequiresScopeCodeForTenantAndUser(t *testing.T) {
	require.Error(t, validatePolicyScope("tenant", "", false))
	require.Error(t, validatePolicyScope("user", "", false))
}

func TestValidatePolicyScopeAllowsGlobalAndEmptyScope(t *testing.T) {
	require.NoError(t, validatePolicyScope("global", "", false))
	require.NoError(t, validatePolicyScope("", "", false))
}

func TestValidatePolicyScopeRejectsGlobalScopeCode(t *testing.T) {
	err := validatePolicyScope("global", "tenant-a", false)

	require.Error(t, err)
}
