# LukatinCode / ClaudeCode

一个Claude Code翻版。集成持久化 Shell、代码检索与文件编辑、Todo 管理、子 Agent 搜索等能力，并可连接多家大模型（OpenAI/Anthropic/DeepSeek/Google/Qwen）。

模型的核心agent调用基于我之前写的[GoAgent](https://github.com/ccIisIaIcat/GoAgent)

## 功能概览
- 工具体系（与 Claude Code 行为保持一致）
  - Bash（持久化 Shell）：跨命令保留环境状态，适合构建/脚本执行
  - Grep/Glob/LS/Read/Edit/MultiEdit/Write：检索、浏览与批量精确编辑代码
  - TodoRead/TodoWrite：结构化待办清单，强约束校验（仅一个 in_progress）
  - WebFetch/WebSearch/Task：网页分析、联网搜索、子 Agent 扩展搜索
- TUI 界面
  - 旧版 tview 与新版 Bubble Tea TUI；显示工具调用、进度与 TodoList
- 日志与可观测性
  - 主进程日志：`log/lukatincode_*.log`
  - Todo 日志：`log/todowrite.txt`、`log/todoread.txt`
  - 其他工具的调试输出（如 grep）：位于 `log/` 目录
- 提示词与安全
  - 默认“少废话/≤4 行”约束；对有副作用命令给出一行说明
  - 动态注入运行环境信息到系统提示（工作目录/平台/日期等）

## 目录结构（节选）
```
ClaudeCode/
  coder/                 # App 主逻辑与 TUI
  function/              # 各工具的 Go 实现（Todo/Grep/Glob/...）
  SystemPromote/         # 系统提示词（systempromote.txt）
  agent/                 # 本地 GoAgent 源（供替换/调试用）
  docker-compose.yml     # 开箱即用的 Linux 容器开发环境
  Dockerfile             # 容器镜像定义（sleep infinity 常驻）
  LLMConfig.yaml         # 模型与密钥配置
  README.md
  log/                   # 运行日志（首次启动后自动创建）
```

## 环境要求
- Go 1.24+（已在 1.24 测试）
- Windows/macOS/Linux 任意平台
- 可选：Docker Desktop（用于在 Linux 容器内构建/运行）

## 快速开始
1) 配置模型与密钥（编辑 `LLMConfig.yaml`）
```yaml
AgentAPIKey:
  OpenAI:
    BaseUrl: https://api.openai.com/v1
    APIKey: sk-xxx
    Model: gpt-5-2025-08-07   # 你的目标模型，避免被默认模型覆盖
```
2) 运行（本机）
```bash
go run main.go
```
3) 构建可执行文件（本机）
```bash
go build -o dist/claudecode main.go
```

## 在 Docker 容器中运行（推荐用于 Linux 交叉编译）
1) 启动容器并常驻：
```bash
docker compose up -d
```
2) 进入容器：
```bash
docker exec -it go-linux-dev bash
```
3) 容器内执行：
```bash
cd /workspace
go run main.go
# 或构建 Linux 二进制
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/claudecode-linux-amd64 main.go
```

## 使用要点
- TUI：
  - Bubble Tea 版默认启动；输入消息回车发送；支持导出/清空/退出等快捷键
- Grep：
  - 语义与 Claude Code 一致，返回“包含匹配的文件路径”的 JSON，按修改时间降序
- Todo：
  - `TodoWrite` 支持三种输入：
    1. `{"todos":[{"id":"1","content":"…","status":"pending","priority":"medium"}]}`
    2. `[{"id":"1","content":"…","status":"pending","priority":"medium"}]`
    3. 纯文本每行一个任务（自动补全默认字段，容忍前缀 `1. / 1) / - / *` 等）

## 常见问题（FAQ）
- Q: OpenAI 模型为何总被替换为 4o？
  - A: 早期逻辑在 `ChatRequest.Model` 为空时会使用默认模型。当前版本会显式将 `LLMConfig.yaml` 内的 `OpenAI.Model` 传入，如仍异常请检查配置是否为空。
- Q: Shell 为什么在 `/tmp/test`？
  - A: 如看到该路径，多半源自历史示例文本。项目已在系统提示中动态注入真实运行目录；持久化 Shell 也会以当前工作目录启动。
- Q: 日志位置？
  - 主进程：`log/lukatincode_*.log`
  - Todo 与各工具：`log/` 目录下对应 txt 文件

## 许可证
- 本项目仅用于学习与内部开发演示。第三方 API 使用遵循其各自服务条款。

将 "This is a test file" 替换为 "This is an edited test file"