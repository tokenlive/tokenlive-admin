package biz

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"go.uber.org/zap"
)

const (
	PolicySeedTableInvocation   = "policy_invocation"
	PolicySeedTableCircuitBreak = "policy_circuit_break"
)

type policySeedID struct {
	ID string `json:"id"`
}

// FirstPolicySeedID returns the first seed ID of the given table from
// policy_seed.json. Extra rows of the same table are ignored with a warning.
// A missing file or empty table returns ("", nil) so callers can skip.
func FirstPolicySeedID(ctx context.Context, table string) (string, error) {
	name := config.C.General.PolicySeedFile
	if name == "" {
		return "", nil
	}
	seedFile := filepath.Join(config.C.General.WorkDir, name)
	data, err := os.ReadFile(seedFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	var seeds map[string]json.RawMessage
	if err := json.Unmarshal(data, &seeds); err != nil {
		return "", errors.Wrapf(err, "Unmarshal JSON file '%s' failed", seedFile)
	}
	raw, ok := seeds[table]
	if !ok {
		return "", nil
	}

	var items []policySeedID
	if err := json.Unmarshal(raw, &items); err != nil {
		return "", errors.Wrapf(err, "Unmarshal policy seed table '%s' failed", table)
	}
	if len(items) == 0 || items[0].ID == "" {
		return "", nil
	}
	if len(items) > 1 {
		logging.Context(ctx).Warn("Policy seed table has multiple entries, only the first will be applied on model create",
			zap.String("table", table),
			zap.Int("count", len(items)),
			zap.String("id", items[0].ID),
		)
	}
	return items[0].ID, nil
}
