# 扫码报工系统

- 机械加工行业二维码扫码报工系统，是一个传统的项目,包含前后端及APP端
- 完整的功能是Rails+Vue2项目,重构为Golang+Vue3(一些查询及界面美化未再实现),后续可自己实现或者美化APP
- APP集成AI对话查看进度(DeepSeek 模型)
- 相关需求人员可拿去自用,有问题可加微信: sanmanlechuntian 交流

| 子系统 | 技术栈 | 目录 | 用途 |
|---|---|---|---|
| 后端 API | Go · Gin · GORM · PostgreSQL · JWT | `backend/` | 统一数据接口 |
| 管理后台 | Vue 3 · TypeScript · Element Plus | `admin/` | 文员建单、权限管理(RBAC)、进度查看 |
| 报工 App | Flutter · Riverpod · Dio | `app/` | 扫码报工、查看进度 |

--- 

## 相关图片 
<img width="768" height="1396" alt="75ec9e04c46f7b0bb24fe9f00fddd75d" src="https://github.com/user-attachments/assets/6ea49e23-fde3-4cbf-841d-0507df150a39" />

<img width="3778" height="1213" alt="图片" src="https://github.com/user-attachments/assets/e3291f5f-d7be-4e15-b6de-4a5d8c206d5a" />
<img width="3616" height="1503" alt="图片" src="https://github.com/user-attachments/assets/26ac9129-6839-4ecc-9d5c-abfb1698d2f4" />

## app
<img width="768" height="1396" alt="78456ad5380d1fe1c83acf73995acfdd" src="https://github.com/user-attachments/assets/da7da28b-f4fc-4103-bf05-e879c3564756" />
<img width="768" height="1396" alt="05ef240b72a15c1130a700af4f1491b0" src="https://github.com/user-attachments/assets/c548deaa-12d4-4a91-9a52-23614c799042" />
<img width="768" height="1396" alt="d969eeb16de26a74255cbece43b4d7c5" src="https://github.com/user-attachments/assets/58164a66-ba3c-407d-9247-867b238aca7f" />


---

## 目录结构

```
work/
├── backend/          # Go 后端
│   ├── cmd/server/   # 入口 main.go
│   ├── config/       # config.yaml
│   └── internal/
│       ├── database/ # 连接 / migrate / seed
│       ├── handler/  # HTTP 处理器
│       ├── middleware/
│       ├── model/
│       ├── repository/
│       ├── router/
│       └── service/
├── admin/            # Vue3 管理后台
│   └── src/
│       ├── api/
│       ├── router/
│       ├── stores/
│       └── views/
└── app/              # Flutter 工人端
    └── lib/
        ├── pages/
        ├── providers/
        ├── services/
        └── widgets/
```

---

## 一、后端（backend）

### 环境要求

- Go ≥ 1.21
- PostgreSQL ≥ 14

### 配置

编辑 `backend/config/config.yaml`：

```yaml
server:
  port: "8080"
  mode: "debug"          # 生产改为 release

database:
  dsn: "host=localhost user=postgres password=postgres dbname=scan_report port=5432 sslmode=disable TimeZone=Asia/Shanghai"

jwt:
  secret: "your-strong-secret"   # 生产务必修改
  expireHours: 24
  refreshExpHours: 168

oss:
  endpoint: "oss-cn-hangzhou.aliyuncs.com"
  accessKeyID: "your-key-id"
  accessKeySecret: "your-key-secret"
  bucketName: "scan-report"
  domain: "https://scan-report.oss-cn-hangzhou.aliyuncs.com"

ai:
  provider: "deepseek"
  deepSeekAPIKey: "your-deepseek-api-key"
  deepSeekBaseURL: "https://api.deepseek.com"
  deepSeekModel: "deepseek-chat"
```

> OSS 仅用于图片上传，`accessKeyID` 留空则跳过初始化，其他功能不受影响。

> DeepSeek Key 只放后端，不要放 Flutter App。`deepSeekAPIKey` 留空时，AI 接口会流式返回 MCP 查询结果摘要，便于先联调。

### 创建数据库

```bash
psql -U postgres -c "CREATE DATABASE scan_report;"
```

### 开发启动

```bash
cd backend
go run ./cmd/server
```

首次启动会自动执行：
1. **AutoMigrate** — 创建 / 更新所有表结构
2. **SQL Migrations** — 补充索引和约束
3. **Seed** — 写入初始数据（幂等，有数据则跳过）

Seed 默认数据：

| 类型 | 内容 |
|---|---|
| 默认账号 | 用户名 `admin`，密码 `admin123` |
| 角色 | 管理员、文员、操作工 |
| 权限 | 22 条，覆盖所有资源的 CRUD |
| 部门 | 管理部、生产部、品质部、仓储部 |
| 工序 | 下料、折弯、冲压、焊接、打磨、喷漆、组装、检测、包装 |

### 生产构建

```bash
cd backend
go build -o bin/server ./cmd/server
./bin/server
```

### Docker 部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/config/config.yaml ./config/
EXPOSE 8080
CMD ["./server"]
```

```bash
docker build -t scan-report-backend .
docker run -d -p 8080:8080 \
  -e DATABASE_DSN="host=db user=postgres ..." \
  --name scan-report-backend \
  scan-report-backend
```

### 主要 API 路由

```
POST   /api/auth/login
POST   /api/auth/refresh

# MCP / AI（需 JWT）
POST   /api/mcp
POST   /api/app/ai/chat/stream

# 管理端（需 JWT）
GET    /api/admin/orders
POST   /api/admin/orders
GET    /api/admin/orders/:id
POST   /api/admin/orders/:id/scrap
POST   /api/admin/orders/:id/processes
GET    /api/admin/orders/:id/report-progress
GET    /api/admin/users
POST   /api/admin/users
GET    /api/admin/roles
PUT    /api/admin/roles/:id/permissions
GET    /api/admin/processes
POST   /api/admin/process-aliases
POST   /api/admin/oss/presign

# 工人端（需 JWT）
GET    /api/app/scan?internal_no=xxx
POST   /api/app/report
GET    /api/app/report/history
GET    /api/app/orders/:id/progress
```

AI 查询进度采用：

```text
Flutter App AI 对话框
  -> /api/app/ai/chat/stream
  -> 后端调用 MCP 工具 query_order_progress
  -> DeepSeek 流式生成回答
```

MCP 工具：

| 工具 | 说明 |
|---|---|
| `query_order_progress` | 按内部单号、外部单号或零件名称查询工单和工序进度 |

---

## 二、管理后台（admin）

### 环境要求

- Node.js ≥ 18

### 安装依赖

```bash
cd admin
npm install
```

### 开发启动

```bash
npm run dev
```

默认访问 `http://localhost:5173`，使用 Seed 账号 `admin / admin123` 登录。

### 配置后端地址

编辑 `admin/src/api/http.ts`，修改 `baseURL`：

```ts
const http = axios.create({
  baseURL: 'http://your-server-ip:8080/api',
})
```

### 生产构建

```bash
npm run build
# 产出 dist/ 目录，部署到任意静态文件服务器（Nginx / CDN）
```

**Nginx 示例配置**（SPA history 路由）：

```nginx
server {
    listen 80;
    server_name admin.example.com;
    root /var/www/admin/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 功能模块

| 菜单 | 说明 |
|---|---|
| 工单管理 | 新建/编辑工单，按工序设置工时、截止日期，生成 QR 码 |
| 报废单 | 基于主工单创建报废子单（编号自动追加 `_BF01`） |
| 报工进度 | 按订单查看各工序完成率、超时预警 |
| 报工查询 | 按日期/人员搜索历史报工记录 |
| 工序模板 | 维护基础工序库 |
| 流程模板 | 将多道工序组合成模板，一键导入工单 |
| 用户管理 | 账号、部门、角色、权限矩阵 |

### 工单 Excel 批量导入

管理后台 `工单列表` 支持上传 `.xlsx` 文件批量创建工单。Excel 第一行为表头，后端会解析并逐行返回成功 / 失败结果。

| 表头 | 是否必填 | 说明 |
|---|---|---|
| 外部单号 | 否 | 客户或外部系统单号 |
| 零件名称 | 否 | 零件 / 产品名称 |
| 图纸编号 | 否 | 图号 |
| 订单数量 | 是 | 必须大于 0 |
| 单价 | 否 | 数字 |
| 总额 | 否 | 数字 |
| 订单日期 | 否 | 支持 `2026-06-29` 或 `2026-06-29 18:30` |

示例表头：

```text
外部单号 | 零件名称 | 图纸编号 | 订单数量 | 单价 | 总额 | 订单日期
```

导入时内部单号由后端自动生成。外部单号非空时不能与已有工单或同一 Excel 内其他行重复。工序流程和图纸链接不从 Excel 导入，可在工单详情页维护。

工单状态流转：

```text
草稿 draft（未设置工序）
  -> 已排工 ready（已设置工序，尚未报工）
  -> 进行中 active（已有报工记录）
  -> 已完成 completed（最终工序完成）
```

只有 `草稿 draft` 状态的工单可以删除。

---

## 三、工人 App（app）

### 环境要求

- Flutter ≥ 3.10
- Android SDK ≥ 33 / Xcode ≥ 14（iOS）

### 配置后端地址

编辑 `app/lib/services/api_service.dart`：

```dart
const _baseUrl = 'http://your-server-ip:8080/api';
```

### 安装依赖

```bash
cd app
flutter pub get
```

### 开发运行

```bash
# Android（真机/模拟器）
flutter run

# 指定设备
flutter devices
flutter run -d <device-id>
```

### 生产打包

```bash
# Android APK
flutter build apk --release

# Android AAB（上架 Google Play）
flutter build appbundle --release

# iOS（需要 Mac + Xcode）
flutter build ipa --release
```

产出路径：
- APK：`build/app/outputs/flutter-apk/app-release.apk`
- IPA：`build/ios/ipa/*.ipa`

### 功能说明

| 页面 | 说明 |
|---|---|
| 登录 | 用户名 + 密码，Token 存储在设备安全存储 |
| 扫码报工 | 扫描工单 QR → 选工序 → 填投入/完成/报废数量 → 上传照片 → 提交 |
| 报工记录 | 查看本人历史报工，支持按日期筛选 |
| 订单进度 | 输入工单号查询各工序完成进度 |
| 更多 | 查看账号信息、退出登录 |

---

## 四、整体部署架构

```
Internet
   │
   ├── [Nginx / CDN]  ──→  admin/dist  (管理后台静态资源)
   │
   ├── [Nginx]  /api/  ──→  backend:8080  (Go API)
   │                              │
   │                         PostgreSQL
   │                         阿里云 OSS（图片）
   │
   └── [Android/iOS App]  ──→  backend:8080  (工人 App)
```

### Docker Compose 快速启动（开发环境）

仓库已包含：

| 文件 | 说明 |
|---|---|
| `docker-compose.yml` | 启动 PostgreSQL、后端 API、管理后台 |
| `backend/Dockerfile` | 构建 Go 后端镜像 |
| `admin/Dockerfile` | 构建 Vue 管理后台并用 Nginx 托管 |
| `admin/nginx.conf` | SPA 路由与 `/api/` 反向代理 |

启动全部服务：

```bash
docker compose up -d --build
```

访问地址：

| 服务 | 地址 |
|---|---|
| 管理后台 | `http://localhost:5173` |
| 后端 API | `http://localhost:8080` |
| PostgreSQL | `localhost:5432` |

查看日志：

```bash
docker compose logs -f backend
docker compose logs -f admin
```

停止服务：

```bash
docker compose down
```

如需同时删除数据库数据卷：

```bash
docker compose down -v
```

---

## 五、常见问题

**Q: 首次登录用什么账号？**  
A: `admin / admin123`，登录后请立即在用户管理页修改密码。

**Q: 扫码报工扫什么码？**  
A: 工单二维码，内容为工单内部编号（格式 `SC_YYYYMM0001`），在管理后台工单详情页生成。

**Q: 图片上传失败怎么办？**  
A: 检查 `config.yaml` 中 OSS 配置是否填写，或在阿里云 OSS 控制台确认 Bucket 跨域规则已允许该域名。

**Q: 工人 App 连不上后端？**  
A: 确认手机与服务器在同一网络，且 `api_service.dart` 中的 `_baseUrl` 填写的是局域网 IP（非 `localhost`）。
