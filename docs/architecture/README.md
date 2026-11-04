# Project Architecture Documentation

这里保存 Lime-server 的架构导航，不替代代码、配置、测试或当前 Plan。

## Documents

- [Overview](overview.md)：顶层目录、关键数据流和集成边界。
- [auth folder](folders/auth.md)：认证模块职责和子目录导航。
- [auth/models](folders/auth-models.md)：PostgreSQL `public.user` 模型边界。
- [auth/rpc](folders/auth-rpc.md)：Auth RPC 配置、依赖注入和外部连接边界。

## Maintenance

代码、配置和测试与本文档冲突时，以真实代码和已确认的用户/Plan 约束为准。
