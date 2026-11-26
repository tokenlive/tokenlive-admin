package policy

import (
	"strings"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	dialectMySQLCollation = "utf8mb4_bin"
	dialectTableComment   = "模型方言适配策略表"
)

func migratePolicyDialect(db *gorm.DB) error {
	dialect := new(schema.PolicyDialect)
	if db.Migrator().HasTable(dialect) {
		// Older SQL installs allowed NULL creation times.
		if err := db.Unscoped().Model(dialect).Where("created_at IS NULL").UpdateColumns(map[string]interface{}{
			"created_at": gorm.Expr("COALESCE(updated_at, deleted_at, CURRENT_TIMESTAMP)"),
			"updated_at": gorm.Expr("updated_at"),
		}).Error; err != nil {
			return err
		}
	}

	if err := db.AutoMigrate(dialect); err != nil {
		return err
	}
	if db.Dialector.Name() == "mysql" {
		if err := alignDialectMySQLTable(db, dialect.TableName()); err != nil {
			return err
		}
	}

	// Backfill names for existing rows, including soft-deleted audit history.
	return db.Unscoped().Model(dialect).Where("name = ''").UpdateColumns(map[string]interface{}{
		"name":       gorm.Expr("implementation"),
		"updated_at": gorm.Expr("updated_at"),
	}).Error
}

// GORM's portable tags handle ORM updates. Match the initialization SQL for
// MySQL table metadata and for timestamp updates made directly in SQL.
func alignDialectMySQLTable(db *gorm.DB, table string) error {
	var current struct {
		Collation      string
		TableComment   string
		TimestampExtra string
	}
	if err := db.Raw(`
		SELECT t.TABLE_COLLATION AS collation, t.TABLE_COMMENT AS table_comment, c.EXTRA AS timestamp_extra
		FROM information_schema.TABLES AS t
		JOIN information_schema.COLUMNS AS c
		  ON c.TABLE_SCHEMA = t.TABLE_SCHEMA AND c.TABLE_NAME = t.TABLE_NAME
		WHERE t.TABLE_SCHEMA = DATABASE() AND t.TABLE_NAME = ? AND c.COLUMN_NAME = 'updated_at'
	`, table).Scan(&current).Error; err != nil {
		return err
	}

	var changes []string
	if current.Collation != dialectMySQLCollation {
		changes = append(changes, "CONVERT TO CHARACTER SET utf8mb4 COLLATE "+dialectMySQLCollation)
	}
	if current.TableComment != dialectTableComment {
		changes = append(changes, "COMMENT = '"+dialectTableComment+"'")
	}
	if !strings.Contains(strings.ToLower(current.TimestampExtra), "on update current_timestamp") {
		changes = append(changes, "MODIFY COLUMN updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'")
	}
	if len(changes) == 0 {
		return nil
	}
	return db.Exec("ALTER TABLE ? "+strings.Join(changes, ", "), clause.Table{Name: table}).Error
}
