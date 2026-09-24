package dal

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

func newProviderTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestProviderDAL_ScanTextJSON(t *testing.T) {
	db := newProviderTestDB(t)
	require.NoError(t, db.AutoMigrate(&schema.Provider{}))

	// Simulate SQLite storing api_keys and o_auth as TEXT (string) instead of BLOB
	err := db.Exec(`INSERT INTO provider (id, code, name, protocol, api_keys, o_auth, deleted) 
		VALUES ('p-text', 'test-code', 'Test Provider', 'openai', '[{"value":"sk-test-123","description":"demo"}]', '{"refresh_token":"rt-123"}', '0')`).Error
	require.NoError(t, err)

	providerDAL := &Provider{DB: db}
	ctx := util.NewIsRootUser(context.Background())

	// 1. Test Query (Find)
	result, err := providerDAL.Query(ctx, schema.ProviderQueryParam{})
	require.NoError(t, err, "Query should scan TEXT JSON columns without error")
	require.Len(t, result.Data, 1)

	p := result.Data[0]
	require.Equal(t, "p-text", p.ID)
	require.Equal(t, "test-code", p.Code)

	keys := p.GetApiKeys()
	require.Len(t, keys, 1)
	require.Equal(t, "sk-test-123", keys[0].Value)
	require.Equal(t, "demo", keys[0].Description)

	oauth := p.GetOAuth()
	require.NotNil(t, oauth)
	require.Equal(t, "rt-123", oauth.RefreshToken)

	// 2. Test Get
	single, err := providerDAL.Get(ctx, "p-text")
	require.NoError(t, err, "Get should scan TEXT JSON columns without error")
	require.NotNil(t, single)
	require.Equal(t, "p-text", single.ID)
	require.Equal(t, "sk-test-123", single.GetApiKeys()[0].Value)
}
