package rbac

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/casbin/casbin/v3"
	casbinlog "github.com/casbin/casbin/v3/log"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/cachex"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Load rbac permissions to casbin
type Casbinx struct {
	enforcer        *atomic.Value `wire:"-"`
	ticker          *time.Ticker  `wire:"-"`
	DB              *gorm.DB
	Cache           cachex.Cacher
	MenuDAL         *dal.Menu
	MenuResourceDAL *dal.MenuResource
	RoleDAL         *dal.Role
}

func (a *Casbinx) GetEnforcer() *casbin.Enforcer {
	if v := a.enforcer.Load(); v != nil {
		return v.(*casbin.Enforcer)
	}
	return nil
}

type policyQueueItem struct {
	RoleID    string
	Resources schema.MenuResources
}

func (a *Casbinx) Load(ctx context.Context) error {
	if config.C.Middleware.Casbin.Disable {
		return nil
	}

	a.enforcer = new(atomic.Value)
	if err := a.load(ctx); err != nil {
		return err
	}

	go a.autoLoad(ctx)
	return nil
}

func (a *Casbinx) load(ctx context.Context) error {
	start := time.Now()
	roleResult, err := a.RoleDAL.Query(ctx, schema.RoleQueryParam{
		Status: schema.RoleStatusEnabled,
	}, schema.RoleQueryOptions{
		QueryOptions: util.QueryOptions{SelectFields: []string{"id"}},
	})
	if err != nil {
		return err
	} else if len(roleResult.Data) == 0 {
		return nil
	}

	var resCount int32
	queue := make(chan *policyQueueItem, len(roleResult.Data))
	threadNum := config.C.Middleware.Casbin.LoadThread
	lock := new(sync.Mutex)
	var policies [][]string

	wg := new(sync.WaitGroup)
	wg.Add(threadNum)
	for i := 0; i < threadNum; i++ {
		go func() {
			defer wg.Done()
			var localPolicies [][]string
			for item := range queue {
				for _, res := range item.Resources {
					localPolicies = append(localPolicies, []string{"p", item.RoleID, res.Path, res.Method})
				}
			}
			lock.Lock()
			policies = append(policies, localPolicies...)
			lock.Unlock()
		}()
	}

	for _, item := range roleResult.Data {
		resources, err := a.queryRoleResources(ctx, item.ID)
		if err != nil {
			logging.Context(ctx).Error("Failed to query role resources", zap.Error(err))
			continue
		}
		atomic.AddInt32(&resCount, int32(len(resources)))
		queue <- &policyQueueItem{
			RoleID:    item.ID,
			Resources: resources,
		}
	}
	close(queue)
	wg.Wait()

	adapter, err := gormadapter.NewAdapterByDBUseTableName(a.DB, "", "casbin_rule")
	if err != nil {
		logging.Context(ctx).Error("Failed to create gorm adapter", zap.Error(err))
		return err
	}

	modelFile := filepath.Join(config.C.General.WorkDir, config.C.Middleware.Casbin.ModelFile)
	e, err := casbin.NewEnforcer(modelFile, adapter)
	if err != nil {
		logging.Context(ctx).Error("Failed to create casbin enforcer", zap.Error(err))
		return err
	}
	if config.C.IsDebug() {
		casbinLogger := casbinlog.NewDefaultLogger()
		_ = casbinLogger.SetEventTypes([]casbinlog.EventType{casbinlog.EventEnforce})
		e.SetLogger(casbinLogger)
	}

	if len(policies) > 0 {
		err := a.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("DELETE FROM casbin_rule").Error; err != nil {
				return err
			}

			var rules []CasbinRule
			seen := make(map[string]bool)
			for _, p := range policies {
				if len(p) >= 4 {
					key := p[0] + "-" + p[1] + "-" + p[2] + "-" + p[3]
					if seen[key] {
						continue
					}
					seen[key] = true
					rules = append(rules, CasbinRule{
						Ptype: p[0],
						V0:    p[1],
						V1:    p[2],
						V2:    p[3],
					})
				}
			}

			if len(rules) > 0 {
				if err := tx.CreateInBatches(rules, 500).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			logging.Context(ctx).Error("Failed to save casbin policy to db", zap.Error(err))
			return err
		}

		if err := e.LoadPolicy(); err != nil {
			logging.Context(ctx).Error("Failed to reload casbin policy", zap.Error(err))
			return err
		}
	}

	a.enforcer.Store(e)

	logging.Context(ctx).Info("Casbin load policy",
		zap.Duration("cost", time.Since(start)),
		zap.Int("roles", len(roleResult.Data)),
		zap.Int32("resources", resCount),
		zap.Int("policies", len(policies)),
	)
	return nil
}

func (a *Casbinx) queryRoleResources(ctx context.Context, roleID string) (schema.MenuResources, error) {
	menuResult, err := a.MenuDAL.Query(ctx, schema.MenuQueryParam{
		RoleID: roleID,
		Status: schema.MenuStatusEnabled,
	}, schema.MenuQueryOptions{
		QueryOptions: util.QueryOptions{
			SelectFields: []string{"id", "parent_id", "parent_path"},
		},
	})
	if err != nil {
		return nil, err
	} else if len(menuResult.Data) == 0 {
		return nil, nil
	}

	menuIDs := make([]string, 0, len(menuResult.Data))
	menuIDMapper := make(map[string]struct{})
	for _, item := range menuResult.Data {
		if _, ok := menuIDMapper[item.ID]; ok {
			continue
		}
		menuIDs = append(menuIDs, item.ID)
		menuIDMapper[item.ID] = struct{}{}
		if pp := item.ParentPath; pp != "" {
			for _, pid := range strings.Split(pp, util.TreePathDelimiter) {
				if pid == "" {
					continue
				}
				if _, ok := menuIDMapper[pid]; ok {
					continue
				}
				menuIDs = append(menuIDs, pid)
				menuIDMapper[pid] = struct{}{}
			}
		}
	}

	menuResourceResult, err := a.MenuResourceDAL.Query(ctx, schema.MenuResourceQueryParam{
		MenuIDs: menuIDs,
	})
	if err != nil {
		return nil, err
	}

	return menuResourceResult.Data, nil
}

func (a *Casbinx) autoLoad(ctx context.Context) {
	var lastUpdated int64
	a.ticker = time.NewTicker(time.Duration(config.C.Middleware.Casbin.AutoLoadInterval) * time.Second)
	for range a.ticker.C {
		val, ok, err := a.Cache.Get(ctx, config.CacheNSForRole, config.CacheKeyForSyncToCasbin)
		if err != nil {
			logging.Context(ctx).Error("Failed to get cache", zap.Error(err), zap.String("key", config.CacheKeyForSyncToCasbin))
			continue
		} else if !ok {
			continue
		}

		updated, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			logging.Context(ctx).Error("Failed to parse cache value", zap.Error(err), zap.String("val", val))
			continue
		}

		if lastUpdated < updated {
			if err := a.load(ctx); err != nil {
				logging.Context(ctx).Error("Failed to load casbin policy", zap.Error(err))
			} else {
				lastUpdated = updated
			}
		}
	}
}

func (a *Casbinx) Release(ctx context.Context) error {
	if a.ticker != nil {
		a.ticker.Stop()
	}
	return nil
}

type CasbinRule struct {
	ID    int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Ptype string `gorm:"column:ptype"`
	V0    string `gorm:"column:v0"`
	V1    string `gorm:"column:v1"`
	V2    string `gorm:"column:v2"`
	V3    string `gorm:"column:v3"`
	V4    string `gorm:"column:v4"`
	V5    string `gorm:"column:v5"`
}

func (CasbinRule) TableName() string {
	return "casbin_rule"
}
