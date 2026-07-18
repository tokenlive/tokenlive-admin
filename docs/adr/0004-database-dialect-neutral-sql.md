# 数据库方言中立的 SQL 写法

Status: accepted

本项目通过 `pkg/gormx` 抽象层支持多种数据库：生产环境默认使用 **MySQL**，开发/测试环境（含集成测试 `go test ./test/...`）使用 **SQLite**（内存模式）。同一份 DAL 代码必须在两种方言下都能正确运行。

GORM 会自动处理标准 CRUD、`Where`/`Order`/`Group`/分页等常规查询的方言差异。问题只出在 DAL 层手写的**原始 SQL 表达式**里调用了某一方言的专属函数——它在开发用的 SQLite 上直接抛错，但因为生产用 MySQL，不易在本地复现。

一个已发生的实例：`internal/mods/ops/dal/event.dal.go` 的 `TrendByHour` / `TrendByDay` 用了 MySQL 专属的 `DATE_FORMAT`，导致 `/api/v1/ops/events/statistics` 在 SQLite 下返回 500。

## Considered Options

1. **只支持 MySQL，开发环境也统一用 MySQL**：被否决，会增加本地开发和 CI 的依赖负担，削弱 `pkg/gormx` 多方言抽象的意义。
2. **（采用）保持多方言支持，DAL 里的方言专属 SQL 按 `db.Dialector.Name()` 分支适配**：在 DAL 内提供小的辅助函数，根据当前方言生成对应表达式。

## Consequences

- 手写原始 SQL（`Select`/`Group`/`Order`/`Raw`/`Exec` 中的表达式）时，若用到方言专属函数（日期格式化、JSON、字符串函数等），必须按 `db.Dialector.Name()` 分支处理。参考 `event.dal.go` 里的 `dateFormatHour` / `dateFormatDay`：SQLite 用 `strftime`，MySQL 用 `DATE_FORMAT`。
- 尽量把时间分桶、区间计算等逻辑放到 Go 层（如 `time.Now()` 计算边界），只把结果作为参数传入查询，从源头避免方言分歧。`internal/mods/dashboard` 即采用此模式。
- 常见需要留意的 MySQL 专属点：`DATE_FORMAT`、`DATE_ADD/SUB`、`UNIX_TIMESTAMP`、`FROM_UNIXTIME`、`STR_TO_DATE`、`INTERVAL`、`IFNULL`、`GROUP_CONCAT`、`JSON_EXTRACT`/`JSON_CONTAINS`、`ON DUPLICATE KEY`。
- 新增聚合/统计类查询后，务必用 SQLite 跑一遍集成测试（`go test ./test/...`）验证方言兼容性，不能只在 MySQL 上验证。
- GORM 标签里的列类型（`type:datetime`、`type:timestamp`、`type:bigint`、`default:CURRENT_TIMESTAMP` 等）为 MySQL 风格。SQLite 动态类型系统能宽松接受，建表不报错，`CURRENT_TIMESTAMP` 亦为两库通用关键字，因此无需为此分支——只是两库的物理列类型存在差异，一般无感知。
