package policy

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// seedCreator marks rows injected by the policy seed mechanism, so they are
// never mistaken for an operation performed by a real user.
var seedCreator = "system"

// JSON columns are stored as *string in the schema structs, but written as
// plain JSON objects/arrays in the seed file. These wrappers capture the raw
// JSON at the outer depth and convert it on load.
type policyInvocationSeed struct {
	schema.PolicyInvocation
	RetryPolicy    json.RawMessage `json:"retry_policy,omitempty"`
	FallbackPolicy json.RawMessage `json:"fallback_policy,omitempty"`
}

type policyCircuitBreakSeed struct {
	schema.PolicyCircuitBreak
	CodePolicy    json.RawMessage `json:"code_policy,omitempty"`
	MessagePolicy json.RawMessage `json:"message_policy,omitempty"`
	DegradeConfig json.RawMessage `json:"degrade_config,omitempty"`
	ErrorCodes    json.RawMessage `json:"error_codes,omitempty"`
	ErrorMessages json.RawMessage `json:"error_messages,omitempty"`
}

var policySeedHandlers = map[string]func(context.Context, *gorm.DB, json.RawMessage) error{
	"policy_invocation":    seedPolicyInvocations,
	"policy_circuit_break": seedPolicyCircuitBreaks,
}

// initPolicySeedsFromFile loads policy templates from the seed file and
// creates each one only if no row with the same ID has ever existed
// (soft-deleted rows included). Existing rows are never overwritten, so
// operator modifications and deletions survive restarts.
func initPolicySeedsFromFile(ctx context.Context, db *gorm.DB, seedFile string) error {
	data, err := os.ReadFile(seedFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logging.Context(ctx).Warn("Policy seed file not found, skip init policy seeds from file", zap.String("file", seedFile))
			return nil
		}
		return err
	}

	var seeds map[string]json.RawMessage
	if err := json.Unmarshal(data, &seeds); err != nil {
		return errors.Wrapf(err, "Unmarshal JSON file '%s' failed", seedFile)
	}

	for table, raw := range seeds {
		handler, ok := policySeedHandlers[table]
		if !ok {
			return errors.Errorf("Unsupported policy seed table '%s' in file '%s'", table, seedFile)
		}
		if err := handler(ctx, db, raw); err != nil {
			return errors.Wrapf(err, "Init policy seeds for table '%s' failed", table)
		}
	}
	return nil
}

func seedPolicyInvocations(ctx context.Context, db *gorm.DB, raw json.RawMessage) error {
	var seeds []*policyInvocationSeed
	if err := json.Unmarshal(raw, &seeds); err != nil {
		return err
	}

	for _, item := range seeds {
		if item.ID == "" {
			return errors.Errorf("Policy invocation seed is missing 'id'")
		}
		exists, err := policySeedExists(ctx, db, new(schema.PolicyInvocation), item.ID)
		if err != nil {
			return err
		}
		if exists {
			logging.Context(ctx).Info("Policy invocation seed already exists, skip", zap.String("id", item.ID))
			continue
		}

		record := item.PolicyInvocation
		record.RetryPolicy = rawJSONToString(item.RetryPolicy)
		record.FallbackPolicy = rawJSONToString(item.FallbackPolicy)
		record.Creator = &seedCreator
		record.Modifier = &seedCreator
		if err := db.WithContext(ctx).Create(&record).Error; err != nil {
			return err
		}
		logging.Context(ctx).Info("Policy invocation seed created", zap.String("id", record.ID))
	}
	return nil
}

func seedPolicyCircuitBreaks(ctx context.Context, db *gorm.DB, raw json.RawMessage) error {
	var seeds []*policyCircuitBreakSeed
	if err := json.Unmarshal(raw, &seeds); err != nil {
		return err
	}

	for _, item := range seeds {
		if item.ID == "" {
			return errors.Errorf("Policy circuit break seed is missing 'id'")
		}
		exists, err := policySeedExists(ctx, db, new(schema.PolicyCircuitBreak), item.ID)
		if err != nil {
			return err
		}
		if exists {
			logging.Context(ctx).Info("Policy circuit break seed already exists, skip", zap.String("id", item.ID))
			continue
		}

		record := item.PolicyCircuitBreak
		record.CodePolicy = rawJSONToString(item.CodePolicy)
		record.MessagePolicy = rawJSONToString(item.MessagePolicy)
		record.DegradeConfig = rawJSONToString(item.DegradeConfig)
		record.ErrorCodes = rawJSONToString(item.ErrorCodes)
		record.ErrorMessages = rawJSONToString(item.ErrorMessages)
		record.Creator = &seedCreator
		record.Modifier = &seedCreator
		if err := db.WithContext(ctx).Create(&record).Error; err != nil {
			return err
		}
		logging.Context(ctx).Info("Policy circuit break seed created", zap.String("id", record.ID))
	}
	return nil
}

func policySeedExists(ctx context.Context, db *gorm.DB, model any, id string) (bool, error) {
	// Unscoped on purpose: a soft-deleted row still counts as existing, so a
	// seed deleted by an operator must not be revived on the next restart.
	var count int64
	if err := db.WithContext(ctx).Unscoped().Model(model).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func rawJSONToString(raw json.RawMessage) *string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	s := strings.TrimSpace(string(raw))
	return &s
}
