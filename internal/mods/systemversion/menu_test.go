package systemversion

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUpdateMenuCreatesDelegatableResourcesWithoutGrantingExistingRoles(t *testing.T) {
	for _, name := range []string{"menu.json", "menu_cn.json"} {
		t.Run(name, func(t *testing.T) {
			isolatedConfig(t)
			db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "menu.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = sqlDB.Close() })
			if err := db.AutoMigrate(&schema.Menu{}, &schema.MenuResource{}, &schema.RoleMenu{}, &schema.Role{}); err != nil {
				t.Fatal(err)
			}
			// An existing system-menu grant must not inherit this new capability.
			parent := &schema.Menu{ID: "existing-system", Code: "system", Type: "page", Status: "enabled"}
			for _, value := range []any{
				parent,
				&schema.Role{ID: "existing-role", Name: "admin", Status: "enabled"},
				&schema.RoleMenu{ID: "existing-grant", RoleID: "existing-role", MenuID: parent.ID},
			} {
				if err := db.Create(value).Error; err != nil {
					t.Fatal(err)
				}
			}
			menuDAL := &dal.Menu{DB: db}
			resourceDAL := &dal.MenuResource{DB: db}
			menu := &biz.Menu{
				Trans: &util.Trans{DB: db}, MenuDAL: menuDAL,
				MenuResourceDAL: resourceDAL, RoleMenuDAL: &dal.RoleMenu{DB: db},
			}
			for range 2 {
				if err := menu.InitFromFile(context.Background(), filepath.Join("../../../configs", name)); err != nil {
					t.Fatal(err)
				}
			}
			var nodes []schema.Menu
			if err := db.Where("code = ?", "versionUpdates").Find(&nodes).Error; err != nil {
				t.Fatal(err)
			}
			if len(nodes) != 1 || nodes[0].ParentID != parent.ID || nodes[0].Type != "button" {
				t.Fatalf("missing or duplicated dedicated update authorization node: %+v", nodes)
			}
			var resources []schema.MenuResource
			if err := db.Where("menu_id = ?", nodes[0].ID).Order("method").Find(&resources).Error; err != nil {
				t.Fatal(err)
			}
			if len(resources) != 2 || resources[0].Method != "GET" || resources[0].Path != "/api/v1/system/updates" ||
				resources[1].Method != "POST" || resources[1].Path != "/api/v1/system/updates/check" {
				t.Fatalf("wrong update management resources: %+v", resources)
			}
			var grants []schema.RoleMenu
			if err := db.Find(&grants).Error; err != nil {
				t.Fatal(err)
			}
			if len(grants) != 1 || grants[0].MenuID != parent.ID {
				t.Fatalf("menu import automatically granted new permission: %+v", grants)
			}
			// Exercise the normal role/menu query after explicit delegation.
			grant := &schema.RoleMenu{ID: "explicit-grant", RoleID: "existing-role", MenuID: nodes[0].ID}
			if err := db.Create(grant).Error; err != nil {
				t.Fatal(err)
			}
			result, err := menuDAL.Query(context.Background(), schema.MenuQueryParam{RoleID: "existing-role", Status: "enabled"})
			if err != nil || len(result.Data) != 2 {
				t.Fatalf("explicit delegation unavailable to RBAC: %+v %v", result, err)
			}
			if err := db.Delete(grant).Error; err != nil {
				t.Fatal(err)
			}
			result, err = menuDAL.Query(context.Background(), schema.MenuQueryParam{RoleID: "existing-role", Status: "enabled"})
			if err != nil || len(result.Data) != 1 {
				t.Fatalf("delegation could not be revoked: %+v %v", result, err)
			}
		})
	}
}
