## 项目概述
一个服务于joylive-agent的微服务治理界面操作控制台。

## 环境设定
- 项目的配置文件在目录`configs`下面。开发测试过程中运行项目可以使用`configs/dev`目录下的配置项。
- 项目包括了前端（frontend目录）与后端部分（internal与pkg目录）。
- 项目总表结构`scritps/init.sql`，围绕着该库表结构生成页面以及对应的管理API（internal/mods目录）。

## 禁止触碰

## 代码规范
- 请参照标准的Go编程规范生成代码。
- 创建者可以在auth.go逻辑中获取到，username, tenant已经放入上下文。有需要使用的地方皆可使用。
- 修改前端（`frontend` 目录）的 `.vue`、`.js` 等文件后，必须使用 `prettier` 进行代码格式化。可以在 `frontend` 目录下运行 `npx prettier --config .prettierrc --write <file_path>` 或是 `npm run prettier` 以保证代码风格合规。

## 验证方式
- 对库表中的数据操作要有对应的增加、删除方法，不要产生脏数据。
## Redis 连接与数据检查
- Redis 配置位于 `configs/dev/server.toml`，与 `.env` 文件中。
- 如需连接 Redis 进行数据检查或调试，可使用 `redis-cli` 命令行工具，示例：
  ```bash
  redis-cli -h 127.0.0.1 -p 30271 -a 123456
  ```
- Cache 默认使用 DB 0，Captcha 使用 DB 1，切换 DB 使用 `SELECT <db_number>`。
