# Agent 文件描述符监控功能

## 功能概述

Daemon 现在支持监控 Agent 进程的文件描述符(File Descriptors, FDs)使用情况,可以帮助及时发现文件描述符泄露问题。

## 实现说明

### 跨平台支持

文件描述符监控已实现跨平台支持:

- **macOS**: 使用 `lsof` 命令获取进程的文件描述符数量
- **Linux**: 读取 `/proc/<pid>/fd` 目录获取准确的文件描述符数量
- **Windows**: 暂不支持(Windows 使用句柄而非文件描述符)

### 采集机制

ResourceMonitor 会在定期采集资源时(默认60秒):

1. 尝试使用平台特定的实现获取文件描述符数量
2. 如果失败,回退到 gopsutil 库的 NumFDs() 方法
3. 将文件描述符数量保存到 ResourceDataPoint.OpenFiles 字段
4. 与 CPU、内存等指标一起记录和监控

### 阈值告警

可以为每个 Agent 配置文件描述符阈值,当超过阈值时会触发告警:

```go
threshold := &agent.ResourceThreshold{
    CPUThreshold:       80.0,                // CPU 使用率 80%
    MemoryThreshold:    1024 * 1024 * 500,   // 内存 500MB
    OpenFilesThreshold: 1000,                // 文件描述符 1000 个
    ThresholdDuration:  time.Minute * 5,     // 持续 5 分钟
}

resourceMonitor.SetThreshold(agentID, threshold)
```

### 告警日志

当文件描述符数量超过阈值时,会记录警告或错误日志:

```
WARN  agent open files over threshold
      agent_id: agent-001
      open_files: 1050
      threshold: 1000
      duration: 2m30s

ERROR agent open files over threshold for too long - possible fd leak
      agent_id: agent-001
      open_files: 1200
      threshold: 1000
      duration: 5m15s
      warning: file descriptor leak detected
```

## 使用示例

### 1. 基本监控

文件描述符监控默认启用,无需额外配置:

```bash
# 启动 Daemon
./daemon

# 查看日志中的文件描述符信息
tail -f /var/log/daemon.log | grep "open_files"
```

### 2. 配置阈值

在代码中设置 Agent 的资源阈值:

```go
// 为特定 Agent 设置阈值
resourceMonitor.SetThreshold("agent-001", &agent.ResourceThreshold{
    OpenFilesThreshold: 500,  // 当文件描述符超过 500 个时告警
    ThresholdDuration:  time.Minute * 3,
})
```

### 3. 查看当前文件描述符数量

通过 gRPC API 查询 Agent 的资源使用情况:

```go
// 获取实时资源数据
dataPoint, err := resourceMonitor.GetCurrentResources("agent-001")
if err == nil {
    fmt.Printf("Open Files: %d\n", dataPoint.OpenFiles)
}

// 获取历史数据
history, err := resourceMonitor.GetResourceHistory("agent-001", time.Hour)
for _, point := range history {
    fmt.Printf("%s: %d FDs\n", point.Timestamp, point.OpenFiles)
}
```

### 4. 监控多个 Agent

```go
// 为所有 Agent 设置统一阈值
agents := multiManager.ListAgents()
for _, instance := range agents {
    resourceMonitor.SetThreshold(instance.GetInfo().ID, &agent.ResourceThreshold{
        OpenFilesThreshold: 800,
        ThresholdDuration:  time.Minute * 5,
    })
}
```

## 文件描述符泄露检测

### 常见原因

1. **未关闭的文件**: 打开文件后忘记调用 `Close()`
2. **未关闭的网络连接**: TCP/HTTP 连接未正确关闭
3. **goroutine 泄露**: goroutine 中打开的文件随 goroutine 一起泄露
4. **第三方库问题**: 使用的库存在文件描述符泄露

### 诊断步骤

1. **查看告警日志**:
   ```bash
   grep "file descriptor leak" /var/log/daemon.log
   ```

2. **使用 lsof 详细查看**(macOS/Linux):
   ```bash
   # 查看特定进程打开的所有文件
   lsof -p <agent_pid>
   
   # 统计文件类型
   lsof -p <agent_pid> | awk '{print $5}' | sort | uniq -c
   ```

3. **分析增长趋势**:
   - 通过历史数据查看文件描述符数量是否持续增长
   - 如果持续增长且不回落,很可能存在泄露

4. **重启恢复**:
   - 如果确认存在泄露,可以重启 Agent 暂时恢复
   - 然后排查代码找到泄露点

### 预防措施

1. **使用 defer 关闭资源**:
   ```go
   file, err := os.Open("file.txt")
   if err != nil {
       return err
   }
   defer file.Close()  // 确保关闭
   ```

2. **设置合理的超时**:
   ```go
   client := &http.Client{
       Timeout: 10 * time.Second,
   }
   ```

3. **定期监控**:
   - 设置适当的文件描述符阈值
   - 定期查看资源使用趋势

4. **代码审查**:
   - 检查所有文件和网络连接的打开/关闭是否成对
   - 使用静态分析工具检测潜在问题

## 技术细节

### macOS 实现

使用 `lsof -p <pid>` 命令:

```bash
$ lsof -p 12345
COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF    NODE NAME
agent   12345 user  0u   CHR   16,1  0t0     1234 /dev/null
agent   12345 user  1u   CHR   16,1  0t0     1234 /dev/null
agent   12345 user  2u   CHR   16,1  0t0     1234 /dev/null
agent   12345 user  3u  IPv4 0x1234  0t0      TCP *:8080 (LISTEN)
...
```

代码会解析输出,统计 FD 列中数字形式的文件描述符。

### Linux 实现

直接读取 `/proc/<pid>/fd` 目录:

```bash
$ ls -l /proc/12345/fd
total 0
lrwx------ 1 user user 64 Dec  9 11:00 0 -> /dev/null
lrwx------ 1 user user 64 Dec  9 11:00 1 -> /dev/null
lrwx------ 1 user user 64 Dec  9 11:00 2 -> /dev/null
lrwx------ 1 user user 64 Dec  9 11:00 3 -> socket:[1234567]
...
```

目录中的每个条目代表一个打开的文件描述符,统计条目数量即可。

### Windows 说明

Windows 使用句柄(Handles)而非文件描述符,当前实现返回错误,不影响其他资源监控功能。后续可考虑实现 Windows 句柄监控。

## 性能影响

- **macOS**: 每次采集需要执行一次 `lsof` 命令,耗时约 20-50ms
- **Linux**: 读取目录,耗时约 1-5ms
- **采集间隔**: 默认 60 秒,对系统性能影响可忽略不计

## 故障排查

### 问题: 日志中出现 "failed to get num fds"

**可能原因**:
- macOS: `lsof` 命令不存在或无执行权限
- Linux: 无权限读取 `/proc/<pid>/fd` 目录
- 进程已退出

**解决方法**:
1. 确认 `lsof` 已安装(macOS)
2. 确认 Daemon 有足够权限监控 Agent 进程
3. 检查 Agent 进程是否正常运行

### 问题: OpenFiles 始终为 0

**可能原因**:
- Windows 平台(不支持)
- 权限不足
- 平台特定实现失败且 gopsutil 也不支持

**解决方法**:
- 检查平台和权限
- 查看详细日志了解失败原因
- 如果是 Windows,这是正常行为

## 相关文件

- `daemon/internal/agent/resource_monitor.go`: 资源监控主逻辑
- `daemon/internal/agent/fds_darwin.go`: macOS 实现
- `daemon/internal/agent/fds_linux.go`: Linux 实现
- `daemon/internal/agent/fds_windows.go`: Windows 存根
- `daemon/internal/agent/fds_test.go`: 单元测试

## 更新日志

**v0.5.0** (2025-12-09):
- ✅ 新增文件描述符监控功能
- ✅ 支持 macOS (lsof) 和 Linux (/proc/fd)
- ✅ 新增 OpenFilesThreshold 阈值配置
- ✅ 新增文件描述符泄露检测和告警
- ✅ 添加跨平台测试用例
