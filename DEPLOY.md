# 部署指南

本项目采用前后端一体化架构，通过Docker多阶段构建将前端和后端打包成单一镜像。

## 构建方式

### 1. 本地开发构建

```bash
# 构建前端和后端
make build-all

# 启动服务（开发模式）
make serve

# 后台启动服务
make serve-d
```

### 2. Docker镜像构建

```bash
# 构建Docker镜像
make docker-build

# 构建并推送镜像到仓库
make docker-push
```

### 3. 手动Docker构建

```bash
# 构建镜像
docker build -t tokenlive-admin:v1.0.0 .

# 运行容器
docker run -d -p 8040:8040 --name joylive tokenlive-admin:v1.0.0
```

## 镜像结构

Docker镜像采用多阶段构建：

1. **前端构建阶段**：使用Node.js 18构建Vue前端
2. **后端构建阶段**：使用Golang构建后端服务
3. **生产镜像阶段**：基于Alpine Linux，仅包含必要的运行时文件

### 镜像内容

- `/usr/bin/tokenlive-admin`：后端二进制文件
- `/app/dist`：前端静态文件
- `/app/configs`：配置文件目录

## 环境变量

运行容器时可以通过环境变量配置：

```bash
docker run -d \
  -p 8040:8040 \
  -e GIN_MODE=release \
  -v /path/to/configs:/app/configs \
  -v /path/to/data:/app/data \
  tokenlive-admin:v1.0.0
```

## 配置文件

配置文件位于 `configs/` 目录：

- `dev/`：开发环境配置
- `prod/`：生产环境配置（Docker默认使用）

## 端口说明

- **8040**：HTTP服务端口

## 健康检查

服务启动后可以通过以下方式检查：

```bash
curl http://localhost:8040/health
```

## 常见问题

### 1. 如何修改前端API地址？

修改 `frontend/.env.prod` 中的 `VITE_API_HTTP` 配置，然后重新构建。

### 2. 如何持久化数据？

挂载 `/app/data` 目录到宿主机：

```bash
docker run -d -v /host/data:/app/data tokenlive-admin:v1.0.0
```

### 3. 如何查看日志？

```bash
docker logs -f joylive
```
