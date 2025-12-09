# 端口冲突问题修复

## 问题描述

Daemon 的 HTTP 心跳接收服务器硬编码在 **8081** 端口，与 Agent-001 的 HTTP API 端口（8081）冲突，导致 Agent-001 无法启动。

## 问题原因

在 `daemon/internal/daemon/daemon.go` 的 `startHTTPServer()` 方法中，HTTP 服务器端口被硬编码为 8081：

```go
// 创建HTTP服务器(默认监听8081端口)
// TODO: 可以从配置中读取端口
addr := ":8081"
```

## 修复方案

### 1. 添加配置字段

在 `daemon/internal/config/config.go` 的 `DaemonConfig` 结构体中添加 `HTTPPort` 字段：

```go
type DaemonConfig struct {
    // ... 其他字段
    HTTPPort int `mapstructure:"http_port"`  // HTTP服务器端口（用于接收Agent心跳），默认8084
}
```

### 2. 设置默认值

在 `setDefaults()` 函数中添加默认端口配置：

```go
if config.Daemon.HTTPPort == 0 {
    config.Daemon.HTTPPort = 8084 // 默认HTTP端口8084，避免与Agent端口冲突
}
```

### 3. 修改代码使用配置

修改 `daemon/internal/daemon/daemon.go` 的 `startHTTPServer()` 方法：

```go
// 创建HTTP服务器，从配置读取端口（默认8084）
httpPort := d.cfg.Daemon.HTTPPort
if httpPort == 0 {
    httpPort = 8084 // 默认端口8084，避免与Agent端口冲突
}
addr := fmt.Sprintf(":%d", httpPort)
```

### 4. 更新测试配置

在 `test/integration/config/daemon.test.yaml` 中添加 HTTP 端口配置：

```yaml
daemon:
  # ... 其他配置
  grpc_port: 9091  # gRPC 服务器端口
  http_port: 8084  # HTTP 服务器端口（用于接收 Agent 心跳），避免与 Agent-001 的 8081 冲突
```

## 端口分配

修复后的端口分配：

| 服务 | 端口 | 说明 |
|-----|------|------|
| Manager HTTP | 8080 | Manager HTTP API |
| Manager gRPC | 9090 | Manager gRPC 服务 |
| Daemon gRPC | 9091 | Daemon gRPC 服务 |
| **Daemon HTTP** | **8084** | **Daemon HTTP 心跳接收服务器** |
| Agent-001 HTTP | 8081 | Agent-001 HTTP API |
| Agent-002 HTTP | 8082 | Agent-002 HTTP API |
| Agent-003 HTTP | 8083 | Agent-003 HTTP API |

## 验证步骤

1. **重新构建 Daemon**:
   ```bash
   cd daemon
   make build
   ```

2. **停止旧进程并重新启动测试环境**:
   ```bash
   cd test/integration
   ./cleanup_test_env.sh
   ./start_test_env.sh
   ```

3. **验证端口占用**:
   ```bash
   # 检查 Daemon HTTP 端口（应该是 8084）
   lsof -ti:8084
   
   # 检查 Agent-001 端口（应该是 8081）
   lsof -ti:8081
   
   # 应该返回不同的 PID
   ```

4. **测试 Agent-001 健康检查**:
   ```bash
   curl http://127.0.0.1:8081/health
   # 应该返回 Agent-001 的健康检查响应
   ```

5. **测试 Daemon HTTP 心跳接收**:
   ```bash
   curl http://127.0.0.1:8084/heartbeat/stats
   # 应该返回 Daemon 的心跳统计信息
   ```

## 相关文件

- `daemon/internal/config/config.go` - 配置结构定义
- `daemon/internal/daemon/daemon.go` - Daemon 主逻辑
- `test/integration/config/daemon.test.yaml` - 测试环境配置

## 注意事项

- 默认 HTTP 端口已从 8081 改为 8084，避免与 Agent 端口冲突
- 如果需要在生产环境使用不同的端口，可以在配置文件中设置 `daemon.http_port`
- Agent 仍然可以通过 Unix Socket (`/tmp/daemon.sock`) 或 HTTP (`http://127.0.0.1:8084/heartbeat`) 发送心跳

---

**修复时间**: 2025-12-07  
**状态**: ✅ 已修复
