package biz

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
)

func TestFirstPolicySeedIDUsesFirstEntryOnly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "policy_seed.json"), []byte(`{
  "policy_invocation": [
    {"id": "seed-invocation-1"},
    {"id": "seed-invocation-2"}
  ]
}`), 0o644))
	setPolicySeedConfig(t, dir, "policy_seed.json")

	id, err := FirstPolicySeedID(context.Background(), PolicySeedTableInvocation)
	require.NoError(t, err)
	require.Equal(t, "seed-invocation-1", id)
}

func TestFirstPolicySeedIDMissingFileReturnsEmpty(t *testing.T) {
	setPolicySeedConfig(t, t.TempDir(), "missing.json")

	id, err := FirstPolicySeedID(context.Background(), PolicySeedTableInvocation)
	require.NoError(t, err)
	require.Empty(t, id)
}

func setPolicySeedConfig(t *testing.T, workDir, seedFile string) {
	t.Helper()
	prevWorkDir := config.C.General.WorkDir
	prevSeedFile := config.C.General.PolicySeedFile
	config.C.General.WorkDir = workDir
	config.C.General.PolicySeedFile = seedFile
	t.Cleanup(func() {
		config.C.General.WorkDir = prevWorkDir
		config.C.General.PolicySeedFile = prevSeedFile
	})
}
