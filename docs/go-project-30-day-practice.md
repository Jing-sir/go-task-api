# Go 项目 30 天实践

## 进度

- Day 1-16: 已完成
- Day 17: WebSocket 实时通信

## Day 17 记录

### 已做

- 写完 `Hub`
- 写完 `Client`
- 写完 `HandleWebSocket`
- `main.go` 已注册 `hub`
- `router.go` 已注册 `/api/v1/ws`
- `CreateTask` 成功后会把任务 JSON 广播给所有在线用户
- 前端已接入 WebSocket，支持自动重连

### 核心思路

- `Hub` 负责管连接和广播
- `Client` 负责读写循环
- 创建任务后，把 `task` 转成 JSON，再丢进 `hub.Broadcast(...)`
- 前端收到消息后，直接把任务插进列表

### 验证

- WebSocket 握手成功
- 创建任务后能收到广播
- `ping / pong` 心跳可用
- `go test ./...` 通过
- 前端 `pnpm typecheck` 和 `pnpm build` 通过

### 过程中踩到的坑

- CORS 只放行了 `http://localhost:5173`，但前端实际跑在 `http://127.0.0.1:15173`
- `task.title` 数据库是 `varchar(20)`，但后端校验一开始写成了 `max=32`

### 现在的结果

- 本地前后端能连通
- 任务创建后会实时出现在在线页面

## Day 18

- WebSocket 心跳
- 断线重连
- 连接状态提示
