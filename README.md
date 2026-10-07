# 无线频谱干扰三角定位

面向无线电频谱观测员、分析员与复核员的离线方位交汇分析系统。系统保存原始测向证据，执行确定性加权最小二乘定位，解释残差、几何退化和离群候选，并通过人工状态流形成可审计结论。

系统不接收实时监听流、不控制无线设备、不自动执法或派工。所有估计坐标与结论只用于决策支持，不能替代现场测量、法定程序或人工复核。

## Docker 快速启动

```bash
cp .env.example .env
# 修改 .env 中的数据库密码和 JWT_SECRET
docker compose up -d --build
docker compose ps
```

访问地址：

- 前端工作台：<http://127.0.0.1:18524>
- 后端健康检查：<http://127.0.0.1:19524/healthz>
- 后端就绪检查：<http://127.0.0.1:19524/readyz>

本地种子账号的密码均为 `Spectrum!2026`：

| 角色 | 邮箱 | 主要权限 |
| --- | --- | --- |
| 观测员 | `observer@spectrum.local` | 录入观测、创建案例、开始采集 |
| 分析员 | `analyst@spectrum.local` | 管理测向站、排除观测、运行定位、提交复核 |
| 复核员 | `reviewer@spectrum.local` | 确认或退回案例、查看审计 |
| 管理员 | `admin@spectrum.local` | 全部权限 |

停止本项目并删除专用数据卷：

```bash
docker compose down -v --remove-orphans
```

## 主要功能

- 维护 WGS84 测向站坐标、天线偏置、精度和校准状态；原始方位与偏置校正方位同时保留。
- 按案例录入频率、带宽、信号强度和质量，批量校验频率匹配与站点状态，排除操作保留原因和审计。
- 在本地笛卡尔坐标图中显示测向站、方位射线、估计点、不确定区域、逐站残差和离群证据。
- 二站几何交汇和三站以上加权最小二乘使用同一确定性求解器；近平行或近共线几何明确拒绝，不返回伪精确点。
- 当至少有四条有效观测时，可比较标准化残差并生成一次离群候选重算；原估计和候选结果都不可覆盖。
- 案例执行 `draft -> collecting -> analyzing -> pending_review -> confirmed -> closed`；退回从 `pending_review` 回到 `analyzing`，关闭后只读。
- JWT、RBAC、乐观锁、事务、内存令牌桶限流、request ID、结构化日志和不可变审计贯穿业务链。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 前端 | React 18、TypeScript、Vite、Material UI、Zustand、React Router、ECharts Core |
| 后端 | Go 1.22、Gin、GORM、validator/v10、JWT、bcrypt、slog |
| 正式数据库 | PostgreSQL 16 |
| 自包含验证 | GORM SQLite 内存数据库，仅供测试和 runtime smoke |
| 部署 | Docker Compose、Nginx 多阶段构建与 API 反向代理 |

## 项目结构

```text
.
├── backend/
│   ├── cmd/server/                  # 服务入口与优雅停机
│   ├── internal/config/             # 环境配置、双数据库驱动、迁移与种子
│   ├── internal/constants/          # 案例状态、观测质量、角色
│   ├── internal/model/              # GORM 实体和数据库约束
│   ├── internal/dto/                # HTTP 输入与批量校验证据
│   ├── internal/repository/         # 事务、条件更新、不可变审计
│   ├── internal/localization/       # 坐标、矩阵、残差、离群检测
│   ├── internal/service/            # 业务状态机与定位编排
│   ├── internal/handler/            # HTTP 参数与统一响应适配
│   ├── internal/middleware/         # request ID、日志、认证、RBAC、恢复、限流
│   ├── internal/router/             # API 路由与权限边界
│   └── pkg/api/                     # 统一响应与错误模型
├── frontend/src/
│   ├── api/                         # 按实体拆分的真实 API 客户端
│   ├── stores/                      # 认证和四个核心实体 Zustand store
│   ├── types/                       # 前后端一致的领域类型
│   ├── components/common/           # 质量、方位图、复核共享组件
│   ├── hooks/                       # useAuth、useLocalizationRun
│   ├── pages/                       # 站点、观测、定位、案例、审计
│   ├── router/                      # RBAC 路由守卫与工作台布局
│   └── utils/                       # 坐标绘图与格式化
├── docker-compose.yml
├── go.work
├── runtime_smoke.json
└── output/                          # 实际验收报告与内置 Browser 截图
```

## API 清单

所有业务 API 使用 `/api/v1`，除登录外均要求 Bearer Token。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/auth/login` | 登录，使用独立限流桶 |
| `GET` | `/api/v1/auth/me` | 获取当前身份和角色 |
| `GET/POST` | `/api/v1/stations` | 测向站列表与登记 |
| `GET/PUT` | `/api/v1/stations/:id` | 站点详情与校准更新 |
| `GET` | `/api/v1/stations/:id/coverage` | 站点观测覆盖 |
| `GET/POST` | `/api/v1/observations` | 观测列表与录入 |
| `POST` | `/api/v1/observations/:id/exclude` | 保存原因并排除观测 |
| `GET` | `/api/v1/cases/:id/validate-observations` | 批量校验案例观测 |
| `GET/POST` | `/api/v1/cases` | 案例列表与草稿创建 |
| `POST` | `/api/v1/cases/:id/transition` | 带 version 的状态迁移 |
| `GET` | `/api/v1/localizations` | 查询不可覆盖的定位历史 |
| `POST` | `/api/v1/localizations/run` | 运行加权定位和离群候选，独立限流 |
| `GET` | `/api/v1/audits` | 复核员/管理员查询不可变审计 |

成功响应统一为 `{ data, request_id, meta? }`，错误响应为 `{ error: { code, message, details? }, request_id }`。分页使用 `page` 与 `page_size`，时间使用 RFC 3339 UTC。

## 共享枚举出现位置

`ObservationQuality = good | fair | poor | excluded`：

- 数据库约束与 model：`backend/internal/model/bearing_observation.go`
- 后端常量和权重：`backend/internal/constants/observation.go`
- DTO、repository、service、handler、router：`backend/internal/dto/bearing_observation.go`、`backend/internal/repository/bearing_observation.go`、`backend/internal/service/bearing_observation.go`、`backend/internal/handler/bearing_observation.go`、`backend/internal/router/router.go`
- 前端类型、API、store、共享组件、页面：`frontend/src/types/observation.ts`、`frontend/src/api/observations.ts`、`frontend/src/stores/observationStore.ts`、`frontend/src/components/common/QualityBadge.tsx`、`frontend/src/pages/ObservationsPage.tsx`、`frontend/src/pages/LocalizationPage.tsx`

`CaseStatus = draft | collecting | analyzing | pending_review | confirmed | closed`：

- 数据库约束与 model：`backend/internal/model/interference_case.go`
- 后端常量和状态机：`backend/internal/constants/case.go`
- DTO、repository、service、handler、router：`backend/internal/dto/interference_case.go`、`backend/internal/repository/interference_case.go`、`backend/internal/service/interference_case.go`、`backend/internal/handler/interference_case.go`、`backend/internal/router/router.go`
- 前端类型、API、store、复核组件、页面：`frontend/src/types/case.ts`、`frontend/src/api/cases.ts`、`frontend/src/stores/caseStore.ts`、`frontend/src/components/common/ReviewDecisionDialog.tsx`、`frontend/src/pages/CasesPage.tsx`、`frontend/src/pages/AuditPage.tsx`

## 定位算法与假设

1. 以参与观测站的平均经纬度为原点，在小范围内使用地球平均半径 `R=6371008.8m` 将 WGS84 差值转换为东、北方向局部坐标。
2. 方位角以正北为 0 度、顺时针增加。每条射线使用法向量构造 `A = Σ(w n nᵀ)`、`b = Σ(w n nᵀ s)`，求解 `A p = b`。
3. 权重为 `quality_weight / accuracy_deg²`；质量权重依次为 good `1.0`、fair `0.55`、poor `0.2`，excluded 不参与计算。
4. 通过 2×2 对称矩阵特征值计算条件数。最小特征值过小或条件数超过 `GEOMETRY_CONDITION_LIMIT` 时返回 `GEOMETRY_DEGENERATE`，不形成定位点。
5. 残差是观测方位与“测站指向估计点”的最小有符号角差；不确定半径综合站点距离、精度、加权 RMS 残差和几何因子，只表达模型不确定性。
6. 离群候选仅在原始有效观测不少于 4 条、剔除后仍不少于 3 条、最大标准化残差超过 2.5 且候选残差至少改善 20% 时生成。原估计仍永久保存。

以上是适用于小区域的软件演示模型，不包含电波传播、地形、多径、同步误差或法规判定，不能替代经校准的专业测向流程。

## 环境变量与端口

| 变量 | 示例 | 说明 |
| --- | --- | --- |
| `COMPOSE_PROJECT_NAME` | `spectrum-interference-triangulation` | 固定英文 Compose 名 |
| `FRONTEND_PORT` | `18524` | 前端宿主端口 |
| `BACKEND_PORT` | `19524` | 后端宿主端口 |
| `DB_PORT` | `57524` | PostgreSQL 宿主端口 |
| `DB_NAME/DB_USER/DB_PASSWORD` | 见 `.env.example` | PostgreSQL 连接配置 |
| `JWT_SECRET` | 至少 32 字符 | JWT HMAC 密钥，部署前必须替换 |
| `CORS_ORIGINS` | 两个本地前端地址 | 逗号分隔的允许来源 |
| `GEOMETRY_CONDITION_LIMIT` | `1000` | 几何退化条件数上限，最低 10 |
| `LOG_LEVEL` | `info` | `debug/info/warn/error` |

数据库使用命名卷 `postgres_data`，不绑定中文宿主路径。前端只请求 `/api`，Nginx 保留 `/api/v1` 路径并提供 SPA fallback。

## 本地开发与质量检查

后端可以使用独立 SQLite 文件快速开发：

```bash
export DB_DRIVER=sqlite
export DB_DSN='file:local-dev.db?_foreign_keys=on'
export JWT_SECRET='local-development-secret-at-least-32-bytes'
export PORT=19524
go run ./backend/cmd/server
```

另开终端启动前端：

```bash
npm --prefix frontend ci
npm --prefix frontend run dev
```

完整检查：

```bash
go work sync
go build ./backend/...
go vet ./backend/...
go test ./backend/...
npm --prefix frontend test
npm --prefix frontend run build
```

## 故障排查

- Compose 项目名为空：确认根目录 `.env` 存在且 `COMPOSE_PROJECT_NAME` 为英文；Compose 文件也有固定 `name` 兜底。
- 后端未 healthy：执行 `docker compose logs backend`，检查 JWT 长度、数据库密码和 PostgreSQL 健康状态。
- 定位返回 `FREQUENCY_MISMATCH`：确认每条观测与案例中心频率的偏差不超过该观测带宽的一半。
- 定位返回 `GEOMETRY_DEGENERATE`：增加不同方位几何的测向站，不能通过放宽显示精度规避退化证据。
- 状态迁移返回 `CASE_VERSION_CONFLICT`：其他请求已更新案例，刷新列表后使用新 version 重试。
- 登录后出现 401：清除当前标签页 `sessionStorage` 后重新登录；令牌不会持久化到其他浏览器会话。

## License

MIT License。该软件仅用于工程演示与离线决策支持，不构成无线电执法或现场处置建议。

