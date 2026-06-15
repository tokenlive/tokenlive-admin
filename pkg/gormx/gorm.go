package gormx

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	sdmysql "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	var dialector gorm.Dialector

	switch strings.ToLower(cfg.DBType) {
	case "mysql":
		if err := createDatabaseWithMySQL(cfg.DSN); err != nil {
			return nil, err
		}
		// 延迟 150ms 以允许 TCP 连接彻底释放与协商，避免因为极速的连接和断开导致端口重用冲突（TIME_WAIT）或触发数据库的 RST 防刷机制
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
		Logger:      logger.Discard,
		PrepareStmt: cfg.PrepareStmt,
	}

	if cfg.Debug {
		ormCfg.Logger = logger.Default
	}

	db, err := gorm.Open(dialector, ormCfg)
	if err != nil {
		return nil, err
	}

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

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.MaxIdleTime) * time.Second)

	// 强制执行带退避重试的 Ping，保障连接 100% 真实可用，彻底解决瞬间网络握手被重置导致的启动失败
	var pingErr error
	for i := 0; i < 3; i++ {
		pingErr = sqlDB.Ping()
		if pingErr == nil {
			break
		}
		zap.L().Warn(fmt.Sprintf("Database connection handshake failed (attempt %d/3): %v. Retrying in 1s...", i+1, pingErr))
		time.Sleep(1 * time.Second)
	}
	if pingErr != nil {
		return nil, fmt.Errorf("database handshake failed after 3 attempts: %w", pingErr)
	}

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

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/", cfg.User, cfg.Passwd, cfg.Addr))
	if err != nil {
		return err
	}
	defer db.Close()

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET = `utf8mb4`;", cfg.DBName)
	
	// 针对数据库初始连接做最多 3 次退避重试，防止由于瞬时网络唤醒延迟导致创建数据库执行失败
	var execErr error
	for i := 0; i < 3; i++ {
		_, execErr = db.Exec(query)
		if execErr == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("failed to create database after 3 attempts: %w", execErr)
}
