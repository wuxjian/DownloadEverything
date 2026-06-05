# Download Everything（全能下载）

自托管文件下载管理工具，集成 AI 智能搜索功能。通过 Web 界面管理下载任务，或利用 AI 自动搜索资源并提取下载链接。

## 功能特性

- **AI 智能搜索** — 四步流水线：搜索引擎搜索 → AI 筛选网页 → 抓取页面内容 → AI 提取下载链接，自动发现资源
- **多线程分片下载** — 支持 Range 请求的文件自动分片并发下载，大幅提升下载速度
- **断点续传** — 暂停后恢复下载，从已下载位置继续
- **自动重试** — 指数退避策略，可配置重试次数和间隔
- **并发控制** — 限制同时下载的任务数量
- **自定义请求头 / Cookie** — 为下载任务注入自定义 Header 和 Cookie，适用于需要登录态的资源
- **代理支持** — 支持 HTTP 和 SOCKS5 代理
- **实时进度推送** — 基于 Server-Sent Events（SSE）实时推送下载进度、速度和文件大小
- **Web 管理界面** — 响应式设计，三页面布局（下载管理、AI 搜索、系统设置）
- **单文件部署** — 前端资源通过 Go embed 嵌入，数据库使用纯 Go SQLite（无 CGO 依赖），一个二进制文件即可运行

## 快速开始

### 使用预编译二进制

从 [Releases](https://github.com/wu-xjian/download-everything/releases) 下载对应平台的可执行文件，直接运行：

```bash
./download-everything
```

### 从源码编译

```bash
git clone https://github.com/wu-xjian/download-everything.git
cd download-everything
go build -o download-everything .
./download-everything
```

启动后访问 **http://localhost:8080** 即可打开 Web 界面。

> 首次运行会自动创建 SQLite 数据库和默认配置，无需额外设置。

## 配置说明

所有配置通过 Web 界面「设置」页面管理，修改后即时生效，无需重启。

| 配置项 | 默认值 | 说明 |
|---|---|---|
| `port` | `8080` | HTTP 服务端口 |
| `down_dir` | `~/Downloads/DownloadEverything` | 文件下载保存目录 |
| `ai_endpoint` | `https://api.openai.com/v1` | OpenAI 兼容 API 端点 |
| `ai_model` | `gpt-4o-mini` | AI 模型名称 |
| `ai_key` | — | AI API Key |
| `tavily_key` | — | Tavily 搜索 API Key |
| `serper_key` | — | Serper (Google) 搜索 API Key |
| `max_concurrent` | `5` | 最大并发下载任务数 |
| `threads_per_file` | `4` | 每个文件的分片线程数 |
| `proxy_url` | — | 代理地址（如 `http://127.0.0.1:7890`） |
| `max_retries` | `3` | 下载失败最大重试次数 |
| `retry_interval` | `10` | 重试间隔（秒） |

### AI 功能配置说明

要使用 AI 搜索功能，需配置以下至少一项：

1. **AI 模型** — 填写 `ai_endpoint` 和 `ai_key`，支持任何 OpenAI 兼容 API（包括 GPT、DeepSeek、Ollama 等）
2. **搜索引擎** — 至少配置 `tavily_key` 或 `serper_key` 中的一个；两者都配置时会并发搜索并合并结果

## 使用指南

### 下载管理页面

- 查看所有下载任务的状态、进度、速度和文件大小
- 新建下载：输入 URL 即可创建任务
- 管理任务：暂停、恢复、重试、删除
- 统计卡片：总任务数、已完成数、下载中的任务数和总下载量

### AI 搜索页面

1. 输入关键词搜索，或粘贴网页 URL 手动解析
2. AI 自动搜索、筛选并提取下载链接
3. 选择合适的链接添加到下载队列
4. 搜索状态自动保存到本地，30 分钟内可恢复

### 设置页面

配置 AI 模型、搜索引擎 API Key、下载参数等。所有配置修改后即时生效。

## 技术栈

| 层级 | 技术 |
|---|---|
| 语言 | Go 1.25 |
| Web 框架 | Gin |
| 数据库 | SQLite（纯 Go 实现，无 CGO） |
| AI 接口 | OpenAI 兼容 API |
| 搜索引擎 | Tavily、Serper (Google) |
| 实时通信 | Server-Sent Events |
| 前端 | 原生 HTML / CSS / JavaScript（零框架依赖） |
| 资源嵌入 | Go embed |

## API 概览

### 页面路由

| 路由 | 说明 |
|---|---|
| `GET /` | 下载管理页面 |
| `GET /search` | AI 搜索页面 |
| `GET /settings` | 设置页面 |

### 下载任务 API

| 方法 | 路由 | 说明 |
|---|---|---|
| `POST` | `/api/tasks` | 创建下载任务 |
| `GET` | `/api/tasks` | 获取任务列表 |
| `GET` | `/api/tasks/:id` | 获取任务详情 |
| `POST` | `/api/tasks/:id/pause` | 暂停任务 |
| `POST` | `/api/tasks/:id/resume` | 恢复任务 |
| `POST` | `/api/tasks/:id/retry` | 重试失败任务 |
| `DELETE` | `/api/tasks/:id` | 删除任务 |
| `DELETE` | `/api/tasks` | 清空历史任务 |
| `GET` | `/api/tasks/events` | SSE 进度推送 |

### AI 搜索 API

| 方法 | 路由 | 说明 |
|---|---|---|
| `POST` | `/api/ai/search` | AI 搜索（SSE 流式返回） |
| `POST` | `/api/ai/parse-url` | 解析 URL 提取下载链接 |
| `POST` | `/api/ai/download` | 将 AI 搜索到的链接加入下载队列 |

### 配置 API

| 方法 | 路由 | 说明 |
|---|---|---|
| `GET` | `/api/settings` | 获取配置（API Key 自动脱敏） |
| `PUT` | `/api/settings` | 更新配置 |

## 从源码构建

```bash
# 构建当前平台可执行文件
go build -o download-everything .

# 交叉编译到其他平台
GOOS=linux GOARCH=amd64 go build -o download-everything-linux .
GOOS=darwin GOARCH=amd64 go build -o download-everything-macos .
```

构建产物为单文件，包含所有前端资源和 SQLite 数据库引擎，无外部依赖。

## 许可

[MIT](LICENSE)
