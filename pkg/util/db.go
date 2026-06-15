package util

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Trans struct {
	DB *gorm.DB
}

type TransFunc func(context.Context) error

func (a *Trans) Exec(ctx context.Context, fn TransFunc) error {
	if _, ok := FromTrans(ctx); ok {
		return fn(ctx)
	}

	return a.DB.Transaction(func(db *gorm.DB) error {
		return fn(NewTrans(ctx, db))
	})
}

func GetDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	db := defDB
	if tdb, ok := FromTrans(ctx); ok {
		db = tdb
	}
	if FromRowLock(ctx) {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return db.WithContext(ctx)
}

func wrapQueryOptions(db *gorm.DB, opts QueryOptions) *gorm.DB {
	if len(opts.SelectFields) > 0 {
		db = db.Select(opts.SelectFields)
	}
	if len(opts.OmitFields) > 0 {
		db = db.Omit(opts.OmitFields...)
	}
	if len(opts.OrderFields) > 0 {
		db = db.Order(opts.OrderFields.ToSQL())
	}
	return db
}

func WrapPageQuery(ctx context.Context, db *gorm.DB, pp PaginationParam, opts QueryOptions, out interface{}) (*PaginationResult, error) {
	if pp.OnlyCount {
		var count int64
		err := db.Count(&count).Error
		if err != nil {
			return nil, err
		}
		return &PaginationResult{Total: count}, nil
	} else if !pp.Pagination {
		dbCtx := db.WithContext(ctx)
		// Use a cloned session for Count to avoid corrupting the Find query
		var count int64
		if err := dbCtx.Session(&gorm.Session{}).Count(&count).Error; err != nil {
			return nil, err
		}

		pageSize := pp.PageSize
		if pageSize > 0 {
			dbCtx = dbCtx.Limit(pageSize)
		}

		dbCtx = wrapQueryOptions(dbCtx, opts)
		err := dbCtx.Find(out).Error
		return &PaginationResult{Total: count}, err
	}

	total, err := FindPage(ctx, db, pp, opts, out)
	if err != nil {
		return nil, err
	}

	return &PaginationResult{
		Total:    total,
		Current:  pp.Current,
		PageSize: pp.PageSize,
	}, nil
}

func FindPage(ctx context.Context, db *gorm.DB, pp PaginationParam, opts QueryOptions, out interface{}) (int64, error) {
	db = db.WithContext(ctx)
	var count int64
	// Use a cloned session for Count because GORM's Count() strips the Select clause,
	// which would cause custom SELECT expressions (e.g. JOINed columns) to be lost
	// in the subsequent Find call.
	err := db.Session(&gorm.Session{}).Count(&count).Error
	if err != nil {
		return 0, err
	} else if count == 0 {
		return count, nil
	}

	current, pageSize := pp.Current, pp.PageSize
	if current > 0 && pageSize > 0 {
		db = db.Offset((current - 1) * pageSize).Limit(pageSize)
	} else if pageSize > 0 {
		db = db.Limit(pageSize)
	}

	db = wrapQueryOptions(db, opts)
	err = db.Find(out).Error
	return count, err
}

func FindOne(ctx context.Context, db *gorm.DB, opts QueryOptions, out interface{}) (bool, error) {
	db = db.WithContext(ctx)
	db = wrapQueryOptions(db, opts)
	result := db.First(out)
	if err := result.Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func Exists(ctx context.Context, db *gorm.DB) (bool, error) {
	db = db.WithContext(ctx)
	var count int64
	result := db.Count(&count)
	if err := result.Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// SoftDelete performs a logical delete by setting deleted to the record's id
// and deleted_at to the current time. The db parameter should already include
// the model scope (e.g., from GetXxxDB) with the deleted='0' filter applied.
// This ensures consistency across all tables that follow the deleted/deleted_at convention.
func SoftDelete(ctx context.Context, db *gorm.DB, id string) error {
	now := time.Now()
	result := db.Where("id=?", id).UpdateColumns(map[string]interface{}{
		"deleted":    id,
		"deleted_at": &now,
	})
	return result.Error
}
