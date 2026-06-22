package gormx

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	sdmysql "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"
)

type ResolverConfig struct {
	DBType   string // mysql/postgres/sqlite3
	Sources  []string
	Replicas []string
	Tables   []string
}

type Config struct {
	Debug        bool
	PrepareStmt  bool
	DBType       string // mysql/postgres/sqlite3
	DSN          string
	MaxLifetime  int
	MaxIdleTime  int
	MaxOpenConns int
	MaxIdleConns int
	TablePrefix  string
	Resolver     []ResolverConfig
}

func New(cfg Config) (*gorm.DB, error) {
	zap.L().Info("Initializing database connection...", zap.String("dbType", cfg.DBType))
	var dialector gorm.Dialector

	switch strings.ToLower(cfg.DBType) {
	case "mysql":
		zap.L().Info("Checking/Creating database with MySQL helper...")
		if err := createDatabaseWithMySQL(cfg.DSN); err != nil {
			zap.L().Error("Database helper pre-checking failed", zap.Error(err))
			return nil, err
		}
		zap.L().Info("Database helper pre-checking completed.")
		// 延迟 150ms 以允许 TCP 连接彻底释放与协商，避免因为极速的连接 and 断开导致端口重用冲突（TIME_WAIT）或触发数据库的 RST 防刷机制
		time.Sleep(150 * time.Millisecond)
		dialector = mysql.Open(cfg.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	case "sqlite3":
		_ = os.MkdirAll(filepath.Dir(cfg.DSN), os.ModePerm)
		dialector = sqlite.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.DBType)
	}

	ormCfg := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   cfg.TablePrefix,
			SingularTable: true,
		},
		Logger:      &ZapGormLogger{LogLevel: logger.Warn},
		PrepareStmt: cfg.PrepareStmt,
	}

	if cfg.Debug {
		ormCfg.Logger = &ZapGormLogger{LogLevel: logger.Info}
	}

	zap.L().Info("Opening GORM connection...")
	db, err := gorm.Open(&safeDialector{Dialector: dialector}, ormCfg)
	if err != nil {
		zap.L().Error("Failed to open GORM connection", zap.Error(err))
		return nil, err
	}
	zap.L().Info("GORM connection opened.")

	if len(cfg.Resolver) > 0 {
		resolver := &dbresolver.DBResolver{}
		for _, r := range cfg.Resolver {
			resolverCfg := dbresolver.Config{}
			var open func(dsn string) gorm.Dialector
			dbType := strings.ToLower(r.DBType)
			switch dbType {
			case "mysql":
				open = mysql.Open
			case "postgres":
				open = postgres.Open
			case "sqlite3":
				open = sqlite.Open
			default:
				continue
			}

			for _, replica := range r.Replicas {
				if dbType == "sqlite3" {
					_ = os.MkdirAll(filepath.Dir(cfg.DSN), os.ModePerm)
				}
				resolverCfg.Replicas = append(resolverCfg.Replicas, open(replica))
			}
			for _, source := range r.Sources {
				if dbType == "sqlite3" {
					_ = os.MkdirAll(filepath.Dir(cfg.DSN), os.ModePerm)
				}
				resolverCfg.Sources = append(resolverCfg.Sources, open(source))
			}
			tables := stringSliceToInterfaceSlice(r.Tables)
			resolver.Register(resolverCfg, tables...)
			zap.L().Info(fmt.Sprintf("Use resolver, #tables: %v, #replicas: %v, #sources: %v \n",
				tables, r.Replicas, r.Sources))
		}

		resolver.SetMaxIdleConns(cfg.MaxIdleConns).
			SetMaxOpenConns(cfg.MaxOpenConns).
			SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second).
			SetConnMaxIdleTime(time.Duration(cfg.MaxIdleTime) * time.Second)
		if err := db.Use(resolver); err != nil {
			return nil, err
		}
	}

	if cfg.Debug {
		db = db.Debug()
	}

	zap.L().Info("Acquiring SQL DB handle...")
	sqlDB, err := db.DB()
	if err != nil {
		zap.L().Error("Failed to acquire SQL DB handle", zap.Error(err))
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.MaxIdleTime) * time.Second)

	zap.L().Info("Checking database connection availability (SELECT 1)...")
	// 强制执行带退避重试的可用性检查（执行 SELECT 1），保障连接 100% 真实可用，彻底解决瞬间网络握手被重置导致的启动失败，同时兼容不支持 Ping 的数据库网关
	var pingErr error
	for i := 0; i < 3; i++ {
		var dummy int
		pingErr = sqlDB.QueryRow("SELECT 1").Scan(&dummy)
		if pingErr == nil {
			break
		}
		zap.L().Warn(fmt.Sprintf("Database connection check failed (attempt %d/3): %v. Retrying in 1s...", i+1, pingErr))
		time.Sleep(1 * time.Second)
	}
	if pingErr != nil {
		zap.L().Error("Database connection check failed after 3 attempts", zap.Error(pingErr))
		return nil, fmt.Errorf("database connection check failed after 3 attempts: %w", pingErr)
	}
	zap.L().Info("Database connection is healthy and verified (SELECT 1 succeeded).")

	return db, nil
}

func stringSliceToInterfaceSlice(s []string) []interface{} {
	r := make([]interface{}, len(s))
	for i, v := range s {
		r[i] = v
	}
	return r
}

func createDatabaseWithMySQL(dsn string) error {
	cfg, err := sdmysql.ParseDSN(dsn)
	if err != nil {
		return err
	}

	zap.L().Info("Checking database status...", zap.String("addr", cfg.Addr), zap.String("dbName", cfg.DBName))

	// 1. 尝试使用完整 DSN 直接连接并执行 SELECT 1 探测数据库是否已存在且可用。
	// 如果已经存在，直接返回 nil，避免低权限应用账号执行 CREATE DATABASE 触发权限不足异常。
	testDB, err := sql.Open("mysql", dsn)
	if err == nil {
		var pingErr error
		for i := 0; i < 3; i++ {
			var dummy int
			pingErr = testDB.QueryRow("SELECT 1").Scan(&dummy)
			if pingErr == nil {
				testDB.Close()
				zap.L().Info("Database is online and reachable (SELECT 1)", zap.String("dbName", cfg.DBName))
				return nil
			}
			zap.L().Info("Database connection test query failed", zap.Error(pingErr), zap.Int("attempt", i+1))
			time.Sleep(100 * time.Millisecond)
		}
		testDB.Close()
	}

	// 2. 如果 Ping 失败（如数据库不存在），降级到连接无库名 DSN 尝试自动创建数据库。
	zap.L().Info("Database ping failed, attempting to connect to server to ensure database exists", zap.String("dbName", cfg.DBName))
	dbName := cfg.DBName
	cfg.DBName = ""
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		zap.L().Error("Failed to open connection to MySQL server for DB creation", zap.Error(err))
		return err
	}
	defer db.Close()

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET = `utf8mb4`;", dbName)
	
	zap.L().Info("Executing CREATE DATABASE query...", zap.String("query", query))
	// 针对数据库初始连接做最多 3 次退避重试，防止由于瞬时网络唤醒延迟导致创建数据库执行失败
	var execErr error
	for i := 0; i < 3; i++ {
		_, execErr = db.Exec(query)
		if execErr == nil {
			zap.L().Info("Helper successfully created database.")
			return nil
		}
		zap.L().Warn(fmt.Sprintf("Failed to execute CREATE DATABASE query (attempt %d/3): %v. Retrying in 200ms...", i+1, execErr))
		time.Sleep(200 * time.Millisecond)
	}
	zap.L().Error("Failed to create database after all attempts", zap.Error(execErr))
	return fmt.Errorf("failed to create database after 3 attempts: %w", execErr)
}

type ZapGormLogger struct {
	LogLevel logger.LogLevel
}

func (l *ZapGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &ZapGormLogger{LogLevel: level}
}

func (l *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		zap.L().Info(fmt.Sprintf(msg, data...))
	}
}

func (l *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		zap.L().Warn(fmt.Sprintf(msg, data...))
	}
}

func (l *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		zap.L().Error(fmt.Sprintf(msg, data...))
	}
}

func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	if err != nil && l.LogLevel >= logger.Error {
		zap.L().Error("GORM SQL Error", zap.Error(err), zap.String("sql", sql), zap.Duration("elapsed", elapsed), zap.Int64("rows", rows))
	} else if elapsed > 200*time.Millisecond && l.LogLevel >= logger.Warn {
		zap.L().Warn("GORM SQL Slow Query", zap.String("sql", sql), zap.Duration("elapsed", elapsed), zap.Int64("rows", rows))
	} else if l.LogLevel >= logger.Info {
		zap.L().Info("GORM SQL Trace", zap.String("sql", sql), zap.Duration("elapsed", elapsed), zap.Int64("rows", rows))
	}
}

type safeDialector struct {
	gorm.Dialector
}

func (d *safeDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return &safeMigrator{
		Migrator: d.Dialector.Migrator(db),
		dbType:   db.Dialector.Name(),
	}
}

type safeMigrator struct {
	gorm.Migrator
	dbType string
}

func (m *safeMigrator) CreateTable(values ...interface{}) error {
	if m.dbType == "sqlite" {
		return m.Migrator.CreateTable(values...)
	}
	for _, value := range values {
		zap.L().Warn("SafeMigrator: table creation detected! CreateTable skipped.",
			zap.String("table", getTableName(value)),
		)
	}
	return nil
}

func (m *safeMigrator) AddColumn(value interface{}, name string) error {
	if m.dbType == "sqlite" {
		return m.Migrator.AddColumn(value, name)
	}
	zap.L().Warn("SafeMigrator: table column addition detected! AddColumn skipped.",
		zap.String("table", getTableName(value)),
		zap.String("column", name),
	)
	return nil
}

func (m *safeMigrator) AlterColumn(value interface{}, field string) error {
	if m.dbType == "sqlite" {
		return m.Migrator.AlterColumn(value, field)
	}
	zap.L().Warn("SafeMigrator: table column structure discrepancy detected! AlterColumn skipped.",
		zap.String("table", getTableName(value)),
		zap.String("field", field),
	)
	return nil
}

func (m *safeMigrator) DropColumn(value interface{}, name string) error {
	if m.dbType == "sqlite" {
		return m.Migrator.DropColumn(value, name)
	}
	zap.L().Warn("SafeMigrator: table column extra discrepancy detected (column in DB but not in struct)! DropColumn skipped.",
		zap.String("table", getTableName(value)),
		zap.String("column", name),
	)
	return nil
}

func (m *safeMigrator) DropIndex(value interface{}, name string) error {
	if m.dbType == "sqlite" {
		return m.Migrator.DropIndex(value, name)
	}
	zap.L().Warn("SafeMigrator: table index discrepancy detected! DropIndex skipped.",
		zap.String("table", getTableName(value)),
		zap.String("index", name),
	)
	return nil
}

func (m *safeMigrator) DropConstraint(value interface{}, name string) error {
	if m.dbType == "sqlite" {
		return m.Migrator.DropConstraint(value, name)
	}
	zap.L().Warn("SafeMigrator: table constraint discrepancy detected! DropConstraint skipped.",
		zap.String("table", getTableName(value)),
		zap.String("constraint", name),
	)
	return nil
}

func (m *safeMigrator) CreateConstraint(value interface{}, name string) error {
	if m.dbType == "sqlite" {
		return m.Migrator.CreateConstraint(value, name)
	}
	zap.L().Warn("SafeMigrator: table constraint creation detected! CreateConstraint skipped.",
		zap.String("table", getTableName(value)),
		zap.String("constraint", name),
	)
	return nil
}

func (m *safeMigrator) CreateIndex(value interface{}, name string) error {
	if m.dbType == "sqlite" {
		return m.Migrator.CreateIndex(value, name)
	}
	zap.L().Warn("SafeMigrator: table index creation detected! CreateIndex skipped.",
		zap.String("table", getTableName(value)),
		zap.String("index", name),
	)
	return nil
}

func getTableName(value interface{}) string {
	if value == nil {
		return "unknown"
	}
	if str, ok := value.(string); ok {
		return str
	}
	if val, ok := value.(interface{ TableName() string }); ok {
		return val.TableName()
	}

	t := reflect.TypeOf(value)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
	}

	return t.Name()
}

func (m *safeMigrator) BuildIndexOptions(opts []schema.IndexOption, stmt *gorm.Statement) []interface{} {
	if bio, ok := m.Migrator.(migrator.BuildIndexOptionsInterface); ok {
		return bio.BuildIndexOptions(opts, stmt)
	}
	return nil
}

func (m *safeMigrator) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	if gdt, ok := m.Migrator.(migrator.GormDataTypeInterface); ok {
		return gdt.GormDBDataType(db, field)
	}
	return ""
}
