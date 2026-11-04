# Personal Agent Entry

> 这是个人开发用的轻量工作流入口。项目事实写在 `.agent-workflow/docs/requirements/`，当前工作状态写在 `.agent-workflow/STATE.md`。

## 会话启动

每次进入项目后只读取：

1. 本文件 `AGENTS.md`
2. `.agent-workflow/MODULE.md`
3. `.agent-workflow/STATE.md`
4. `.agent-workflow/docs/requirements/project-profile.md`
5. 当前 Plan、Current Task 和上一个未归档 Task 报告（如果有）

不要批量读取历史 Plan、Review、源码或整个 `docs/`。只有用户明确要求架构扫描、需求涉及外部接口/权限/数据，或当前任务无法判断时，才追加相关文档。

## 自动路由

- `Mode=BOOTSTRAP`：和用户讨论项目目标、范围、技术栈和约束，填写需求基线；不生成 Plan。
- 用户明确要求“快速修改/直接改这一处”时：项目必须已完成 Bootstrap；如果 `Current task` 不为 `none`，优先归入该 Task 的 `DEVELOPMENT_CHECK`；如果没有 Current Task，当前也不是 `BOOTSTRAP`/`FROZEN`，且没有 `Status=in_progress`、`waiting_for_confirmation` 或 `blocked`，再进入 `Mode=QUICK_CHANGE`，只做小范围修改和定向检查，不创建 Plan、不跑完整 Test/Review。
- `Mode=QUICK_CHANGE`：继续当前 Quick Change；定向检查失败或用户要求继续修改时保持该模式，不重新发起流程。
- `Mode=PLAN` 且 `Status=waiting_for_plan`：继续讨论需求；只有用户明确要求某功能的 Plan 才创建当前 Plan。
- `Mode=PLAN` 且 `Status=waiting_for_confirmation`：讨论和修改当前 Plan，等待确认。
- `Mode=PLAN` 且 `Status=ready_for_task`：读取 Current Task，开始 Development。
- `Mode=TASK` 且 `Task session step=DEVELOPMENT`：完成当前 Task 的 Development；完成后进入用户检查，不自动 Test。
- `Mode=TASK` 且 `Task session step=DEVELOPMENT_CHECK`：汇报改动并等待用户；用户说“修改”就继续改当前 Task；如果用户要自己改，Agent 停止修改并保持该状态，在 `Blockers` 记录“等待用户改完”，用户确认后读取 diff、检查范围并重新汇报；用户说“继续”才进入 `Test → Review`。
- `Mode=TASK` 且 `Task session step=TEST` 或 `REVIEW`：严格完成当前阶段；Test 未完成不得进入 Review。
- 如果已有 Current Task，快速修改必须归入当前 Task 的 `DEVELOPMENT_CHECK`，不能另开独立 `QUICK_CHANGE`；如果已有 Test/Review 正在执行，先暂停并按当前 Task 处理。
- `Mode=ARCHITECTURE_AUDIT`：按用户指定范围扫描模块或整个项目，更新或报告架构文档；不改业务代码、不生成 Plan。
- `Mode=FROZEN`：项目已明确冻结；不生成 Plan、不改业务代码，只有用户明确解冻后才能回到 `PLAN`。
- `Mode=BLOCKED`：先解决阻塞，不得假装完成。

需求不清晰、当前要求和 Plan 不一致、或发现接口/权限/数据范围变化时，先使用 `clarification-gate` 反问。

## 文件职责

| 文件/目录 | 用途 |
|---|---|
| `AGENTS.md` | 入口和路由 |
| `.agent-workflow/MODULE.md` | 轻量 Plan、Task、验证、Review 和交接规则 |
| `.agent-workflow/STATE.md` | 当前状态、Task、验证摘要和阻塞 |
| `.agent-workflow/docs/requirements/project-profile.md` | 项目事实、技术栈、命令和限制 |
| `.agent-workflow/docs/requirements/project-vision.md` | 产品目标、范围和非目标 |
| `.agent-workflow/docs/plans/` | 当前功能 Plan |
| `.agent-workflow/docs/reviews/` | Task 交接报告 |
| `docs/architecture/` | 项目真实架构文档，按需创建；不属于工作流内部 |

不要修改本入口来记录项目事实。项目架构更新只写到项目根目录 `docs/architecture/`。
