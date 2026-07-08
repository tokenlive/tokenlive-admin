package biz

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

type routeDetailSyncCall struct {
	scopeType string
	scopeCode string
	modelID   string
}

type recordingPolicySyncer struct {
	calls []routeDetailSyncCall
}

func (r *recordingPolicySyncer) SyncPolicyChange(ctx context.Context, scopeType, scopeCode, modelID string) error {
	r.calls = append(r.calls, routeDetailSyncCall{
		scopeType: scopeType,
		scopeCode: scopeCode,
		modelID:   modelID,
	})
	return nil
}

func TestPolicyRouteDetailCreateSyncsParentRouteDimension(t *testing.T) {
	db := newPolicyRouteDetailSyncTestDB(t)
	syncer := &recordingPolicySyncer{}
	biz := newPolicyRouteDetailSyncTestBiz(db, syncer)
	seedPolicyRoute(t, db)

	_, err := biz.Create(context.Background(), &schema.PolicyRouteDetailForm{
		RouteId:      "route-1",
		RelationType: "AND",
		Enabled:      1,
	})

	require.NoError(t, err)
	require.Equal(t, []routeDetailSyncCall{{
		scopeType: "tenant",
		scopeCode: "tenant-a",
		modelID:   "model-1",
	}}, syncer.calls)
}

func TestPolicyRouteDetailUpdateSyncsParentRouteDimension(t *testing.T) {
	db := newPolicyRouteDetailSyncTestDB(t)
	syncer := &recordingPolicySyncer{}
	biz := newPolicyRouteDetailSyncTestBiz(db, syncer)
	seedPolicyRoute(t, db)
	seedPolicyRouteDetail(t, db)

	err := biz.Update(context.Background(), "detail-1", &schema.PolicyRouteDetailForm{
		RouteId:      "route-1",
		RelationType: "OR",
		Enabled:      1,
	})

	require.NoError(t, err)
	require.Equal(t, []routeDetailSyncCall{{
		scopeType: "tenant",
		scopeCode: "tenant-a",
		modelID:   "model-1",
	}}, syncer.calls)
}

func TestPolicyRouteDetailDeleteSyncsParentRouteDimension(t *testing.T) {
	db := newPolicyRouteDetailSyncTestDB(t)
	syncer := &recordingPolicySyncer{}
	biz := newPolicyRouteDetailSyncTestBiz(db, syncer)
	seedPolicyRoute(t, db)
	seedPolicyRouteDetail(t, db)

	err := biz.Delete(context.Background(), "detail-1")

	require.NoError(t, err)
	require.Equal(t, []routeDetailSyncCall{{
		scopeType: "tenant",
		scopeCode: "tenant-a",
		modelID:   "model-1",
	}}, syncer.calls)
}

func newPolicyRouteDetailSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&schema.PolicyRoute{},
		&schema.PolicyRouteDetail{},
	))
	return db
}

func newPolicyRouteDetailSyncTestBiz(db *gorm.DB, syncer PolicyChangeSyncer) *PolicyRouteDetail {
	return &PolicyRouteDetail{
		Trans:                &util.Trans{DB: db},
		PolicyRouteDetailDAL: &dal.PolicyRouteDetail{DB: db},
		PolicyRouteDAL:       &dal.PolicyRoute{DB: db},
		PolicyRedisSync:      syncer,
	}
}

func seedPolicyRoute(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Create(&schema.PolicyRoute{
		ID:        "route-1",
		ModelID:   "model-1",
		ScopeType: "tenant",
		ScopeCode: "tenant-a",
		Priority:  10,
		Name:      "tenant route",
		Enabled:   1,
		Deleted:   "0",
		CreatedAt: time.Now(),
	}).Error)
}

func seedPolicyRouteDetail(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Create(&schema.PolicyRouteDetail{
		ID:           "detail-1",
		RouteId:      "route-1",
		RelationType: "AND",
		Enabled:      1,
		Deleted:      "0",
		CreatedAt:    time.Now(),
	}).Error)
}
