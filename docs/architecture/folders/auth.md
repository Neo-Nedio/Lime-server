# Folder: `auth/`

- Status: `active`
- Responsibility: 认证 HTTP/RPC 接口和 PostgreSQL 用户模型
- Parent architecture: `../overview.md`
- Last synced Task: `TASK-002`
- Last verified: `2026-11-04`
- Source of truth: 真实代码、配置和测试

## Child folders

| Folder | Responsibility | Document |
|---|---|---|
| `auth/models/` | `public.user` 的 go-zero 模型及认证扩展 | [auth/models](auth-models.md) |
| `auth/rpc/` | Auth RPC 入口、配置和依赖注入 | [auth/rpc](auth-rpc.md) |

## Dependencies and effects

- Depends on: `pkg/configcenter`、PostgreSQL、Redis
- Used by: Auth API/RPC logic
- Permissions/security: 认证规则由后续 RPC logic 实现
- Data/storage: PostgreSQL `public.user`、Redis
- External effects: 服务启动时创建数据库和 Redis 客户端
