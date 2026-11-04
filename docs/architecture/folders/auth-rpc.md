# Folder: `auth/rpc/`

- Status: `active`
- Responsibility: Auth zRPC 服务入口、配置和逻辑层依赖注入
- Parent architecture: `../overview.md`
- Last synced Task: `TASK-002`
- Last verified: `2026-11-04`
- Source of truth: 真实代码、配置和测试

## Files

| File | Responsibility | Inputs/outputs or public boundary | Important constraints | Last changed Task |
|---|---|---|---|---|
| `auth.go` | 加载本地/etcd 配置并启动 RPC | `auth/rpc/etc/auth.yaml`、etcd | 先加载 ConfigCenter，再创建 ServiceContext | TASK-001 |
| `internal/config/config.go` | RPC 配置结构 | PostgreSQL、Redis、JWT、Mailer、ConfigCenter | 配置值由运行环境或 etcd 提供 | TASK-002 |
| `internal/svc/servicecontext.go` | 创建并注入 user model、Redis | `config.Config` | logic 不自行创建外部连接 | TASK-002 |

## Dependencies and effects

- Depends on: `auth/models`、`pkg/configcenter`、go-zero PostgreSQL/Redis clients
- Used by: Auth RPC server and logic
- Permissions/security: Token/Mailer/Event 适配器暂未在本 Task 构造
- Data/storage: PostgreSQL `public.user`、Redis
- External effects: 启动时建立 PostgreSQL 和 Redis 客户端
