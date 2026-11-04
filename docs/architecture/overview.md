# Architecture Overview

- Status: `active`
- Last synced Task: `TASK-002`
- Last verified: `2026-11-04`
- Source of truth: 真实代码、配置和测试；本文档是导航摘要

## Project boundary

- Project root: `D:/Lime-server`
- Runtime/deployment: Go + go-zero；配置中心使用 etcd
- Main entry points: `auth/rpc/auth.go`、`auth/api/auth.go`
- External systems: etcd、PostgreSQL `public.user`、Redis

## Repository map

| Project folder | Responsibility | Main entry/document | Depends on / owned boundary | Status | Last changed Task |
|---|---|---|---|---|---|
| `auth/` | 认证 API/RPC 和用户模型 | `auth/rpc/auth.go` | PostgreSQL、Redis、etcd 配置 | active | TASK-002 |
| `pkg/` | 项目通用能力 | `pkg/configcenter/configcenter.go` | etcd 配置订阅 | active | TASK-001 |

## Key data flow

```text
auth/rpc/etc/auth.yaml
  -> pkg/configcenter 从 etcd 覆盖配置
  -> auth/rpc/internal/config.Config
  -> auth/rpc/internal/svc.ServiceContext
  -> PostgreSQL public.user / Redis
  -> Auth RPC logic
```

## Module and integration boundaries

| Boundary | Owner folder | Consumers | Contract / constraint |
|---|---|---|---|
| User persistence | `auth/models/` | Auth RPC | 以现有 PostgreSQL `public.user` 表为唯一结构来源；不由仓库迁移建表 |
| Auth RPC dependencies | `auth/rpc/internal/svc/` | Auth RPC logic | 通过 go-zero PostgreSQL/Redis 客户端注入，不在 logic 中创建连接 |
| Remote configuration | `pkg/configcenter/` | Auth RPC entrypoint | `ConfigCenter.Etcd` 负责启动期配置覆盖 |

## Cross-cutting rules

- Authentication/authorization: Auth RPC 业务逻辑负责，当前 Task 只建立依赖边界。
- Persistence/migrations: PostgreSQL 模型从真实表生成；本项目不修改表结构。
- Error handling/observability: 遵循 go-zero 默认服务和模型错误边界。
- External side effects: Redis 由 ServiceContext 建立连接；邮件、Token 和事件适配器留待后续 Task。

## Architecture change log

| Task or audit | Change | Affected folders/docs | Date |
|---|---|---|---|
| TASK-002 | Auth RPC 增加 PostgreSQL user model 和 Redis 的依赖注入边界 | `auth/models/`、`auth/rpc/internal/config/`、`auth/rpc/internal/svc/` | 2026-11-04 |
