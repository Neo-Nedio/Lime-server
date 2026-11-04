# Folder: `auth/models/`

- Status: `active`
- Responsibility: 从现有 PostgreSQL `public.user` 表生成的 go-zero model，以及认证所需的手写查询/状态扩展
- Parent architecture: `../overview.md`
- Last synced Task: `TASK-002`
- Last verified: `2026-11-04`
- Source of truth: 真实代码、配置和测试

## Files

| File | Responsibility | Inputs/outputs or public boundary | Important constraints | Last changed Task |
|---|---|---|---|---|
| `usermodel_gen.go` | goctl 生成的表模型 | `User`、基础 CRUD、按 email 查询 | 标记为 generated，不手工修改 | TASK-002 |
| `usermodel.go` | 认证查询、注册写入和状态更新扩展 | `UserModel` | 手写扩展与生成文件分离 | TASK-002 |
| `vars.go` | 模型通用错误 | `ErrNotFound` | 遵循 go-zero sqlx 错误 | TASK-002 |

## Dependencies and effects

- Depends on: go-zero `sqlx`
- Used by: `auth/rpc/internal/svc`
- Permissions/security: 不处理 Token 或权限判断
- Data/storage: PostgreSQL `public.user`
- External effects: 读写现有用户表，不创建或迁移表
