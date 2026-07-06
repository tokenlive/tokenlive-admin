package policy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"gorm.io/gorm"
)

func TestDropLegacyPolicyNameIndexes(t *testing.T) {
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&schema.PolicyTagging{}))
	require.NoError(t, db.Exec("CREATE UNIQUE INDEX uniq_policy_tagging_name ON policy_tagging (name, deleted)").Error)

	policy := &Policy{DB: db}
	require.NoError(t, policy.dropLegacyPolicyNameIndexes())

	require.False(t, db.Migrator().HasIndex(&schema.PolicyTagging{}, "uniq_policy_tagging_name"))
}
