# Y2K Pixel Blog — 操作手册

---

## 目录

1. [系统架构](#1-系统架构)
2. [安装与启动](#2-安装与启动)
3. [系统初始化](#3-系统初始化)
4. [内容管理](#4-内容管理)
5. [搜索系统](#5-搜索系统)
6. [SEO 与订阅](#6-seo-与订阅)
7. [AI 摘要](#7-ai-摘要)
8. [SSL 与域名配置](#8-ssl-与域名配置)
9. [安全机制](#9-安全机制)
10. [运维与排障](#10-运维与排障)

---

## 1. 系统架构

Y2K Pixel Blog 由三个独立服务组成，通过 Nginx 统一对外暴露：

- **前端 (`/frontend`)**：Next.js 16 + React 19。Canvas 粒子背景，Win98 风格可拖拽窗口，所有数据通过 API 动态加载。
- **管理后台 (`/admin`)**：Vite 5 + React SPA。独立于前端的管理界面，使用 98.css 复古样式，通过 JWT 认证访问。
- **API 服务 (`/backend`)**：Go 1.23 + Gin。34 个 REST 端点，包含内容管理、搜索、SSL 证书管理和 SEO 接口。
- **数据库**：PostgreSQL 16，使用 `pg_trgm` 扩展支持中日韩文全文检索。

---

## 2. 安装与启动

### 前置依赖

- Go 1.23+
- Node.js 20+
- Docker 及 Docker Compose

### 开发环境

```bash
# 启动数据库
docker compose up db -d

# 启动后端（新终端）
cd backend
cp .env.example .env
go run ./cmd/server/    # http://localhost:8080

# 启动前端（新终端）
cd frontend
npm install
npm run dev             # http://localhost:3000

# 启动管理后台（新终端）
cd admin
npm install
npm run dev             # http://localhost:5173
```

### 生产环境（Docker）

```bash
# 1. 复制并编辑环境变量
cp .env.example .env
# 设置 JWT_SECRET、DB_PASSWORD 等
# 生成密钥：openssl rand -base64 32

# 2. 修改 nginx.conf，将 example.com 替换为实际域名

# 3. 构建并启动所有服务
docker compose up -d --build
```

### Docker 服务清单

| 服务 | 镜像 | 职责 |
| :--- | :--- | :--- |
| `db` | postgres:16-alpine | 数据库，含 pg_trgm 扩展 |
| `api` | Go 1.23（自定义） | REST API、SSL 终端 |
| `frontend` | Node 20（自定义） | Next.js standalone SSR |
| `admin` | Nginx Alpine | Vite SPA，含路由回退 |
| `nginx` | nginx:alpine | 反向代理、限流、缓存 |

---

## 3. 系统初始化

首次访问管理后台时，系统会检测到尚未初始化并跳转到 `/setup` 向导页面。

**初始化步骤：**

1. **站点信息**：填写博客名称、描述和版权页脚
2. **管理员账户**：设置用户名和密码（最少 6 位）
3. **AI 配置**（可选）：填入兼容 OpenAI 的 API 地址、密钥和模型名称
4. **初始板块**（可选）：创建第一个内容分类

提交后系统完成初始化，使用刚创建的账户登录。

**注意**：初始化接口只能成功调用一次，之后被系统锁定。如需重置，需通过数据库手动操作。

---

## 4. 内容管理

### 文章状态

所有内容支持三种状态：

- **已发布**：对外可见，出现在搜索结果和 RSS 订阅中
- **草稿**：仅管理后台可见
- **回收站**：暂时隐藏，可还原，也可永久删除

### 内容类型

#### Article（文章）

标准 Markdown 博客文章。字段：标题、slug、标签、所属板块、摘要、正文（Markdown）。

#### Rating（评测）

用于影视游戏评测，除基础字段外还包含：

- **封面图片**：显示在评测卡片上的图片路径
- **综合评分**：0.0 ~ 10.0，精度 0.1
- **一句话简评**
- **雷达图数据**：可定义多组评分维度，前端自动绘制多边形雷达图

雷达图数据结构示例：
```json
[
  {
    "name": "画面表现",
    "axes": [
      { "label": "建模", "score": 8.5 },
      { "label": "光影", "score": 7.2 }
    ]
  }
]
```

#### Photo（图文相册）

多页左右分栏格式（左侧大图，右侧图注），支持翻页。每页独立配置图片路径和 Markdown 文字。

#### Page（独立页面）

用于"关于"、"留言板"等不随时间线归档的静态页面，可通过 `showInMenu` 字段控制是否显示在导航菜单中。

### 文章预览

草稿状态的文章可以生成一个有效期 15 分钟的预览链接（`POST /api/admin/preview/:slug`），供在发布前检查实际显示效果。

### 板块管理

板块（Board）是文章的分类容器，支持层级结构（父子板块）。每个板块可以设置：

- `slug`：英文唯一标识
- `name`：显示名称
- `color`：主题颜色（十六进制）
- `icon`：图标名称
- `parent`：父板块 slug（留空则为顶级）

---

## 5. 搜索系统

搜索基于 PostgreSQL 的 `pg_trgm` 扩展，使用 GIN 索引进行三元组匹配，原生支持中日韩文，无需额外分词器。

支持 6 种搜索语法：

| 语法 | 示例 | 搜索范围 |
| :--- | :--- | :--- |
| `关键词` | `Y2K` | 标题 + 标签 + 摘要 |
| `#标签` | `#科幻` | 仅标签（传递时需编码为 `%23`） |
| `~文本` | `~殖民网络` | 正文全文 |
| `@板块` | `@start` | 板块内所有内容 |
| `@板块 关键词` | `@start 星之彼方` | 板块内标题搜索 |
| `@板块 #标签` | `@start #科幻` | 板块内标签筛选 |

结果按 `similarity()` 相似度评分排序。详细用法见 [SEARCH_HELP.md](SEARCH_HELP.md)。

---

## 6. SEO 与订阅

所有 SEO 相关接口由后端动态生成，内容根据数据库中已发布的文章实时更新，链接域名取自管理后台配置的 `Frontend Domain`。

### XML Sitemap

`GET /sitemap.xml`

遵循 sitemaps.org 0.9 规范，包含：
- 首页（权重 1.0，每日更新）
- 所有板块页（权重 0.8，每周更新）
- 所有已发布文章（权重 0.6，取文章日期为 `lastmod`）

### RSS 订阅

`GET /feed.xml`

RSS 2.0 格式，包含最新 50 篇已发布文章，`pubDate` 使用 RFC1123Z 格式，语言标记为 `zh-CN`。

RSS 阅读器订阅地址即为 `https://你的域名/feed.xml`。

### Open Graph

`GET /api/og/:slug`

为每篇文章生成社交分享元数据（标题、摘要、封面图、标签），供 Telegram、微信、X 等平台生成预览卡片时使用。

### robots.txt

`GET /robots.txt`

允许所有爬虫访问，并自动附上 Sitemap 地址，便于搜索引擎发现。

---

## 7. AI 摘要

### 配置

在管理后台的站点设置中填入兼容 OpenAI 的 API 信息：

- **API 地址**：如 `https://api.deepseek.com/v1`
- **API 密钥**：仅存储在服务端，管理界面显示脱敏值
- **模型名称**：如 `deepseek-chat`

### 工作原理

`GET /api/ai/summary?title=xxx&tags=xxx`

首次请求时，后端向配置的 AI API 发送请求生成摘要，立即返回 `202 status: generating`；前端轮询直到结果生成完毕。生成结果缓存在数据库中，后续请求直接返回缓存内容。

### 清除缓存

若文章内容修改后希望重新生成摘要，可在管理后台手动清除对应文章的 AI 缓存（`DELETE /api/admin/ai-cache/:slug`）。

---

## 8. SSL 与域名配置

在管理后台的 **设置 > 域名 & SSL** 中配置。前端域名和后台域名**独立配置**，各自可选择不同的 SSL 模式。

| 模式 | 说明 |
| :--- | :--- |
| 关闭 | 不启用 SSL，仅 HTTP |
| 手动 | 上传 PEM 格式的证书和私钥 |
| 自动 | 通过 Let's Encrypt 自动申请，需服务器 80 端口公网可达 |

**技术实现**：后端使用 SNI（Server Name Indication）在 TLS 握手时识别请求域名，动态分发对应证书，支持前台和后台使用各自不同的证书文件。

启用 SSL 后，后端在 `:443` 提供 HTTPS 服务，在 `:80` 处理 ACME 验证和 HTTP 跳转。

---

## 9. 安全机制

| 机制 | 说明 |
| :--- | :--- |
| 密码存储 | bcrypt 哈希 |
| 身份验证 | JWT（HS256，24 小时过期） |
| 登录保护 | 失败 3 次后要求数学验证码，失败 10 次后封禁 IP 15 分钟 |
| 输入过滤 | 仅接受 Markdown，渲染时经 DOMPurify 净化 |
| SQL 注入 | 全部使用参数化查询 |
| SSL 密钥 | 存储在数据库，API 不返回原始 PEM |
| AI 密钥 | 仅存储在服务端 |

---

## 10. 运维与排障

### 数据库备份

```bash
docker exec y2k-blog-db pg_dump -U blog -d y2k_blog -f /tmp/backup.sql
docker cp y2k-blog-db:/tmp/backup.sql ./backup.sql
```

### 手动执行数据库迁移

新版本若包含迁移脚本，需手动应用到已有数据库：

```bash
docker cp backend/migrations/010_logo_license.sql y2k-blog-db:/tmp/
docker exec y2k-blog-db psql -U blog -d y2k_blog -f /tmp/010_logo_license.sql
```

### CORS 跨域问题排查

若遇到管理后台白屏或 API 请求被拦截，检查以下几点：

- 前端访问协议（http/https）与 API 地址协议是否一致
- 若管理后台通过 HTTPS 访问，API 地址也必须使用 HTTPS
- 检查 Nginx 配置中的代理头是否正确传递

### 忘记管理员密码

当前版本不提供前端找回密码功能。需要通过数据库直接修改：

```sql
-- 生成新的 bcrypt 哈希后替换
UPDATE settings SET value = '<新的bcrypt哈希>' WHERE key = 'admin_password_hash';
```
