package biz

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsDal "github.com/tokenlive/tokenlive-admin/internal/mods/ops/dal"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEndpointQueryByProviderIncludesDisabledActiveEndpoints(t *testing.T) {
	db := newEndpointQueryTestDB(t)
	biz := newEndpointQueryTestBiz(db)

	endpoints := []*schema.Endpoint{
		{
			ID:         "endpoint-enabled",
			Code:       "endpoint-enabled-code",
			ModelID:    "model-1",
			ProviderID: "provider-1",
			URL:        "https://enabled.example.test",
			Weight:     10,
			Enabled:    1,
			Deleted:    "0",
			CreatedAt:  time.Now(),
		},
		{
			ID:         "endpoint-disabled",
			Code:       "endpoint-disabled-code",
			ModelID:    "model-1",
			ProviderID: "provider-1",
			URL:        "https://disabled.example.test",
			Weight:     20,
			Enabled:    0,
			Deleted:    "0",
			CreatedAt:  time.Now(),
		},
		{
			ID:         "endpoint-deleted",
			Code:       "endpoint-deleted-code",
			ModelID:    "model-1",
			ProviderID: "provider-1",
			URL:        "https://deleted.example.test",
			Weight:     30,
			Enabled:    1,
			Deleted:    "1",
			CreatedAt:  time.Now(),
		},
	}
	require.NoError(t, db.Create(&endpoints).Error)

	result, err := biz.QueryEndpointsByProviderID(context.Background(), "provider-1")

	require.NoError(t, err)
	require.Len(t, result, 2)
	require.ElementsMatch(t, []string{"endpoint-enabled", "endpoint-disabled"}, []string{result[0].ID, result[1].ID})
}

// 供应商详情页的监控标签需要 model_code 才能按模型拉取端点流量趋势；
// 缺少 Preload 时 Model 为 nil，图表会静默空白而非报错。
func TestEndpointQueryByProviderPreloadsModel(t *testing.T) {
	db := newEndpointQueryTestDB(t)
	biz := newEndpointQueryTestBiz(db)

	require.NoError(t, db.Create(&schema.Model{
		ID:        "model-1",
		ModelCode: "gpt-4",
		ModelName: "GPT-4",
		Deleted:   "0",
		CreatedAt: time.Now(),
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "endpoint-1",
		Code:       "endpoint-1-code",
		ModelID:    "model-1",
		ProviderID: "provider-1",
		URL:        "https://example.test",
		Enabled:    1,
		Deleted:    "0",
		CreatedAt:  time.Now(),
	}).Error)

	result, err := biz.QueryEndpointsByProviderID(context.Background(), "provider-1")

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].Model, "Model 未预加载，监控标签的端点流量图会空白")
	require.Equal(t, "gpt-4", result[0].Model.ModelCode)
}

func newEndpointQueryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&schema.Endpoint{},
		&schema.Model{},
		&schema.DataPermission{},
		&opsSchema.AuditLog{},
	))
	return db
}

func newEndpointQueryTestBiz(db *gorm.DB) *Endpoint {
	trans := &util.Trans{DB: db}
	return &Endpoint{
		Trans:             trans,
		EndpointDAL:       &dal.Endpoint{DB: db},
		ConfigRedisSync:   &ConfigRedisSync{},
		DataPermissionBIZ: &DataPermission{Trans: trans, DataPermissionDAL: &dal.DataPermission{DB: db}},
		AuditLogBIZ:       &opsBiz.AuditLog{Trans: trans, AuditLogDAL: &opsDal.AuditLog{DB: db}},
	}
}
