# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在此代码库中工作时提供指导。

## 项目概述

万悟（Wanwu）是一个企业级的一站式 AI Agent 开发平台。它为构建 AI 应用提供全面的解决方案，功能包括模型管理、知识库、RAG（检索增强生成）、MCP（模型上下文协议）集成、工作流和智能体。

**技术栈：**
- **后端：** Go 1.24+ 微服务架构
- **前端：** Vue.js 3（位于 `web/` 目录）
- **Python 服务：** 基于 Flask 的回调服务（位于 `callback/` 目录）
- **通信方式：** 服务间使用 gRPC，对外 API 使用 HTTP/REST
- **数据库：** MySQL（支持 TiDB 和 OceanBase 用于信创）、Redis、Elasticsearch
- **存储：** MinIO 对象存储
- **消息队列：** Kafka
- **协议：** Protocol Buffers 用于服务定义

## 常用开发命令

### 环境搭建

#### 首次运行前准备

```bash
# 1. 拷贝环境变量文件
cp .env.bak .env

# 2. 根据系统修改 .env 文件中的关键变量
# 编辑 .env 文件：

# 设置系统架构
WANWU_ARCH=amd64          # 或 arm64

# 设置外部访问 IP
# 注意：如果浏览器访问非 localhost 部署的万悟，需修改为实际 IP
WANWU_EXTERNAL_IP=localhost   # 或 192.168.xx.xx

# 配置 JWT 签名密钥（必须设置！）
# 使用一串自定义复杂随机字符串，用于生成 JWT token
WANWU_BFF_JWT_SIGNING_KEY=your-random-secret-key-here

# 3. 创建 Docker 运行网络
docker network create wanwu-net

# 4. 启动所有服务（首次运行会自动从 Docker Hub 拉取镜像）
# amd64 系统：
docker compose --env-file .env --env-file .env.image.amd64 up -d

# arm64 系统：
docker compose --env-file .env --env-file .env.image.arm64 up -d
```

#### 访问系统

```bash
# URL: http://localhost:8081
# 默认用户：admin
# 默认密码：Wanwu123456
```

#### 关闭服务

```bash
# amd64 系统：
docker compose --env-file .env --env-file .env.image.amd64 down

# arm64 系统：
docker compose --env-file .env --env-file .env.image.arm64 down
```

#### 镜像备份

如果拉取中间件等镜像遇到困难，可在网盘获取镜像备份（参考项目 README 中的"万悟镜像备份"）

### 构建 Go 服务

```bash
# 构建 amd64 架构的特定服务
make build-bff-amd64
make build-iam-amd64
make build-model-amd64
make build-mcp-amd64
make build-knowledge-amd64
make build-rag-amd64
make build-app-amd64
make build-operate-amd64
make build-assistant-amd64
make build-agent-amd64

# 构建 arm64 架构的特定服务
make build-bff-arm64
# ...（其他服务相同模式）
```

### 开发工作流（单独运行服务）

```bash
# 使用 Makefile.develop 进行开发
# 启动所有中间件
make -f Makefile.develop run-all-middleware

# 启动特定服务（示例：bff-service）
make -f Makefile.develop run-bff
make -f Makefile.develop stop-bff

# 可用服务：bff, iam, model, mcp, knowledge, rag, app, operate, assistant, agent, callback, workflow
```

### 源码开发模式（本地运行服务）

**重要：** 源码开发前需要先通过 Docker 启动完整的中间件环境。

#### 架构说明

源码开发模式采用**容器挂载本地二进制文件**的方式：
- Docker 容器持续运行，通过 volume 挂载本地编译的二进制文件
- 修改代码后只需重新编译并重启容器，无需删除重建
- 配置文件通过 volume 挂载，修改配置无需重启容器

```yaml
# docker-compose-develop.yaml 中的配置示例
volumes:
  - ./bin/${WANWU_ARCH}/agent-service:/app/bin/agent-service  # 挂载本地二进制文件
  - ./configs/microservice/agent-service/configs:/app/configs/...  # 挂载配置文件
```

#### 首次源码开发设置

1. 确保已按上述步骤启动所有 Docker 容器（中间件 + 服务）
2. 以 `agent-service` 为例，切换到本地开发模式：

```bash
# 1. 停止 Docker 中的 agent-service 容器
make -f Makefile.develop stop-agent

# 2. 编译本地可执行文件
# amd64 系统：
make build-agent-amd64
# arm64 系统：
make build-agent-arm64

# 3. 启动容器（会自动挂载本地编译的二进制文件）
make -f Makefile.develop run-agent
```

#### 开发循环流程（⚠️ 重要）

**修改代码后的正确操作流程：**

```bash
# 方式一：重新编译并重启容器（推荐）
make build-agent-amd64 && docker restart agent-service

# 方式二：使用 Makefile（会删除重建容器，较慢）
make -f Makefile.develop stop-agent && make build-agent-amd64 && make -f Makefile.develop run-agent
```

**可用服务名称：**
- `bff` - BFF 服务
- `iam` - IAM 服务
- `model` - 模型服务
- `mcp` - MCP 服务
- `knowledge` - 知识库服务
- `rag` - RAG 服务
- `app` - 应用服务
- `operate` - 运维服务
- `assistant` - 助手服务
- `agent` - 智能体服务
- `callback` - 回调服务
- `workflow` - 工作流服务

**示例：**
```bash
# 修改 agent-service 后
make build-agent-amd64 && docker restart agent-service

# 修改 bff-service 后
make build-bff-amd64 && docker restart bff-service

# 修改 model-service 后
make build-model-amd64 && docker restart model-service
```

#### ⚠️ 常见错误和注意事项

**错误做法：**
```bash
# ❌ 不要这样做：每次都删除重建容器
make -f Makefile.develop stop-agent && make build-agent-amd64 && make -f Makefile.develop run-agent
```
这样虽然也能工作，但效率低，不是最优的开发方式。

**正确做法：**
```bash
# ✅ 推荐：保持容器运行，只编译和重启
make build-agent-amd64 && docker restart agent-service
```

**为什么？**
- `docker compose down` 会删除容器，下次 `up` 需要重新创建
- `docker restart` 只是重启容器进程，速度更快
- 由于 volume 挂载，新编译的二进制文件会立即在容器内生效

#### 查看日志和调试

```bash
# 查看服务日志（最后 50 行）
docker logs agent-service --tail 50

# 实时跟踪日志
docker logs agent-service -f

# 查看所有服务状态
docker ps

# 进入容器调试
docker exec -it agent-service sh
```

#### 关闭所有服务

```bash
# amd64 系统：
docker compose --env-file .env --env-file .env.image.amd64 down

# arm64 系统：
docker compose --env-file .env --env-file .env.image.arm64 down
```

### 测试和代码检查

```bash
# 运行 Go 代码检查
make check

# 运行 Python 回调服务检查
make check-callback

# 生成 Swagger 文档
make doc

# 更新 Protocol Buffer 定义
make pb
```

### 前端开发

```bash
cd web/
npm install
npm run serve    # 开发服务器
npm run build    # 生产构建
```

### Python 回调服务

```bash
# 启动回调服务
make -f Makefile.develop run-callback

# 访问 Swagger 文档
# http://localhost:8669/apidocs
```

## 架构概览

### 微服务结构

项目采用微服务架构，包含以下服务：

**核心服务：**
- **bff-service** - Backend for Frontend，主 API 网关
- **iam-service** - 身份和访问管理（认证、授权、用户管理）
- **model-service** - LLM/Embedding 模型管理和提供商集成
- **mcp-service** - MCP（模型上下文协议）服务器管理
- **knowledge-service** - 知识库操作
- **rag-service** - RAG（检索增强生成）引擎
- **app-service** - 应用管理
- **operate-service** - 运维和指标
- **assistant-service** - AI 助手功能
- **agent-service** - AI Agent 框架

**支持服务：**
- **callback** - Python Flask 异步回调服务
- **tidb-setup** - 数据库初始化工具

### 目录结构

```
wanwu/
├── api/                    # 从 proto 定义生成的 gRPC Go 代码
├── callback/              # Python Flask 回调服务
├── cmd/                   # 各服务的主入口点
│   ├── bff-service/       # BFF 服务主程序
│   ├── iam-service/       # IAM 服务主程序
│   └── ...
├── configs/               # 配置文件
│   ├── microservice/      # 服务特定配置
│   └── middleware/        # 中间件配置（MySQL、Redis 等）
├── docs/                  # Swagger 文档
├── internal/              # 私有 Go 代码（内部包）
│   ├── bff-service/       # BFF 服务实现
│   ├── iam-service/       # IAM 服务实现
│   └── ...
├── pkg/                   # 共享/公共 Go 包
│   ├── model-provider/    # 模型提供商集成（OpenAI、Qwen 等）
│   ├── db/                # 数据库工具
│   ├── redis/             # Redis 客户端
│   ├── minio/             # MinIO 客户端
│   ├── es/                # Elasticsearch 客户端
│   ├── grpc-util/         # gRPC 工具
│   ├── http-client/       # HTTP 客户端工具
│   └── ...
├── proto/                 # Protocol Buffer 定义
│   ├── bff-service/       # BFF 服务 proto 定义
│   ├── iam-service/       # IAM 服务 proto 定义
│   └── ...
├── rag/                   # RAG 开源组件
├── web/                   # Vue.js 前端
└── project/               # 运行时项目文件
```

### 服务通信模式

1. **服务间通信：** 服务通过 `proto/` 中定义的 Protocol Buffers 使用 gRPC 通信
2. **外部 API：** 通过 bff-service 以 HTTP/REST 方式暴露
3. **异步操作：** 由回调服务（Python Flask）处理
4. **消息队列：** Kafka 用于异步事件处理

### `pkg/` 中的关键包

- **model-provider** - 各种 LLM 提供商的抽象层（OpenAI 兼容、Qwen、元景等）
- **db** - 数据库连接管理（基于 GORM）
- **redis** - Redis 客户端封装
- **minio** - 对象存储操作
- **es** - Elasticsearch 搜索操作
- **grpc-util** - gRPC 服务器/客户端工具
- **gin-util** - Gin 框架 HTTP 服务器工具

### 配置系统

每个服务都有自己的配置文件 `configs/microservice/<service-name>/configs/config.yaml`：

- 服务特定设置（主机、端口、端点）
- 数据库连接
- 中间件配置
- 功能开关
- i18n 设置（国际化）

环境变量定义在 `.env` 和 `.env.image.{arch}` 文件中。

### 数据库架构

- **主数据库：** MySQL（默认），支持 TiDB 和 OceanBase 用于信创适配
- **缓存：** Redis 用于会话管理和缓存
- **搜索：** Elasticsearch 用于全文搜索
- **存储：** MinIO 用于文件/对象存储（文档、模型等）

### 模型提供商系统

平台通过 `pkg/model-provider` 包支持多个 LLM 提供商：

- **OpenAI 兼容**提供商（包括联通元景）
- **Qwen**（阿里通义千问）
- **DeepSeek**
- **Ollama**（本地模型）
- 可通过实现提供商接口添加自定义提供商

### 前端架构

- **框架：** Vue.js 3 + Vue CLI
- **UI 组件：** 自定义组件库
- **构建工具：** Vue CLI + webpack
- **部署：** nginx 提供静态文件服务

## 开发工作流

### 添加新功能

1. 确定需要修改哪些服务
2. 如果更改服务接口，更新 `proto/` 中的 proto 定义
3. 重新生成 gRPC 代码：`make pb`
4. 在 `internal/<service>/` 中实现业务逻辑
5. 如需要，更新配置
6. 构建并测试特定服务
7. 如有 UI 更改，更新 `web/` 中的前端

### 修改服务接口

1. 更新 `proto/<service>/` 中的 `.proto` 文件
2. 运行 `make pb` 在 `api/` 中重新生成 Go 代码
3. 更新 `internal/<service>/` 中的服务实现
4. 更新调用服务中的客户端代码

### 数据库迁移

数据库初始化脚本位于 `configs/middleware/mysql/initdb.d/`。架构更改时：

1. 在 initdb.d 目录中创建迁移 SQL 文件
2. 如果使用 GORM，更新服务代码中的 ORM 模型
3. 先在开发环境中测试迁移

### 测试

- Go 测试：`go test ./...`（标准 Go 测试）
- 集成测试应与完整的中间件堆栈一起运行
- 前端测试：在 `web/` 目录中运行 `npm run test`

## 重要说明

### 多架构支持

项目支持 `amd64` 和 `arm64` 两种架构。始终指定架构：

- 构建：`make build-bff-amd64` 或 `make build-bff-arm64`
- 运行：使用相应的 `.env.image.{arch}` 文件配合 docker-compose

### 信创适配

平台支持国产数据库以满足中国合规要求：
- **TiDB**：使用 `docker-compose.tidb.yaml`
- **OceanBase**：使用 `docker-compose.oceanbase.yaml`

在 `.env` 中设置 `WANWU_DB_NAME` 来切换数据库。

### 环境变量

`.env` 中的关键环境变量：
- `WANWU_ARCH` - 目标架构（amd64/arm64）
- `WANWU_EXTERNAL_IP` - 外部访问 IP
- `WANWU_BFF_JWT_SIGNING_KEY` - JWT 签名密钥（必须设置）
- `WANWU_DB_NAME` - 数据库类型（mysql/tidb/oceanbase）

### 服务依赖

- 所有服务都依赖中间件（MySQL、Redis、MinIO、Kafka、Elasticsearch）
- bff-service 是主入口点
- 先启动中间件：`make -f Makefile.develop run-all-middleware`

### i18n（国际化）

平台支持多种语言。i18n 文件作为 Excel 文件管理并转换为 JSONL：
- 位置：`configs/microservice/bff-service/configs/wanwu-i18n.xlsx`
- 更新：使用 `make i18n-jsonl` 转换

### Swagger 文档

API 文档使用 Swagger 自动生成：
- 处理器注释位于 `internal/bff-service/server/http/handler/`
- 生成文档：`make doc`
- 在 `/swagger/*` 端点访问

### 工作流集成

平台使用以下项目作为工作流引擎：
- **v0.1.8 及以前：** [wanwu-agentscope](https://github.com/UnicomAI/wanwu-agentscope)
- **v0.2.0 开始：** [wanwu-workflow](https://github.com/UnicomAI/wanwu-workflow)

工作流模板可以是：
- 本地（基于文件）
- 远程（基于 HTTP）
- 通过 `WANWU_BFF_WORKFLOW_TEMPLATE_SERVER_MODE` 配置

### 推荐配置

- **CPU：** 8核或16核
- **内存：** 32G
- **硬盘：** 200G以上
- **GPU：** 不需要

---

## Swan仪表选型系统项目

**状态：** 进行中

**文档位置：** `docs/swan-workflow-design/`

**关键文件：**
- 项目进度：`docs/swan-workflow-design/TODO.md`
- 快速开始：`docs/swan-workflow-design/QUICKSTART.md`
- 设计文档：`docs/swan-workflow-design/*.md`

**知识库信息：**
- 知识库ID：`2029087979716218880`
- 文档数量：85个产品
- 向量化模型：qwen3-embedding-8B

**已完成：**
- ✅ 需求分析和设计
- ✅ 知识库部署（85个文档）
- 🔄 工作流创建（进行中）
- 📋 对话流创建（待开始）

**最后更新：** 2026-03-17
