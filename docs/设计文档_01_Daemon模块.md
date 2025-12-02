# Daemon模块设计文档

**版本：** 1.0  
**日期：** 2025-12-01  
**模块编号：** MOD-DAEMON

---

## 目录

1. [模块概述](#1-模块概述)
2. [架构设计](#2-架构设计)
3. [详细设计](#3-详细设计)
4. [数据结构](#4-数据结构)
5. [配置管理](#5-配置管理)
6. [安全设计](#6-安全设计)
7. [错误处理](#7-错误处理)
8. [测试设计](#8-测试设计)

---

## 1. 模块概述

### 1.1 模块职责

Daemon是运行在每台被管主机上的守护进程，主要职责包括：

- 系统资源采集与上报
- Agent进程生命周期管理
- 与Manager通信（注册、心跳、指标上报）
- 版本更新执行（Agent更新和自更新）
- 本地日志管理

### 1.2 设计目标

| 目标 | 指标 |
|------|------|
| 轻量级 | CPU < 1%（空闲），内存 < 30MB |
| 高可用 | 自动恢复，异常率 < 0.1% |
| 安全性 | TLS通信，签名验证 |
| 可扩展 | 插件化指标采集 |

### 1.3 依赖关系

```
┌─────────────────────────────────────────────┐
│                  Manager                     │
└──────────────────────┬──────────────────────┘
                       │ gRPC/TLS
┌──────────────────────▼──────────────────────┐
│                  Daemon                      │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐        │
│  │Collector│ │ AgentMgr│ │ Updater │        │
│  └─────────┘ └────┬────┘ └─────────┘        │
└───────────────────┼─────────────────────────┘
                    │ Socket/HTTP
┌───────────────────▼─────────────────────────┐
│                  Agent                       │
└─────────────────────────────────────────────┘
```

---

## 2. 架构设计

### 2.1 整体架构

```
┌────────────────────────────────────────────────────────────────┐
│                         Daemon Process                          │
├────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                     Core Engine                           │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐     │  │
│  │  │  Config │  │  Logger │  │  Signal │  │ Scheduler│     │  │
│  │  │ Manager │  │         │  │ Handler │  │          │     │  │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘     │  │
│  └──────────────────────────────────────────────────────────┘  │
├────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐              │
│  │  Collector  │ │  AgentMgr   │ │   Updater   │              │
│  │   Module    │ │   Module    │ │   Module    │              │
│  ├─────────────┤ ├─────────────┤ ├─────────────┤              │
│  │ • CPU       │ │ • Lifecycle │ │ • Download  │              │
│  │ • Memory    │ │ • Health    │ │ • Verify    │              │
│  │ • Disk      │ │ • Restart   │ │ • Backup    │              │
│  │ • Network   │ │ • Heartbeat │ │ • Rollback  │              │
│  │ • [Plugin]  │ │             │ │ • SelfUpdate│              │
│  └─────────────┘ └─────────────┘ └─────────────┘              │
├────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   Communication Layer                     │  │
│  │  ┌───────────────┐  ┌───────────────┐                    │  │
│  │  │  gRPC Client  │  │ Local Socket  │                    │  │
│  │  │  (to Manager) │  │ (to Agent)    │                    │  │
│  │  └───────────────┘  └───────────────┘                    │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
```

### 2.2 模块划分

| 模块 | 职责 | 关键组件 |
|------|------|----------|
| Core Engine | 核心引擎，负责启动、配置、调度 | ConfigManager, Logger, SignalHandler, Scheduler |
| Collector | 系统资源采集 | CPUCollector, MemoryCollector, DiskCollector, NetworkCollector |
| AgentMgr | Agent进程管理 | ProcessManager, HealthChecker, HeartbeatReceiver |
| Updater | 版本更新管理 | Downloader, Verifier, Backuper, Rollbacker |
| Communication | 通信层 | gRPCClient, LocalSocket |

### 2.3 线程模型

```
Main Goroutine
    │
    ├── Config Loader (初始化)
    │
    ├── gRPC Connection Manager (长连接维护)
    │
    ├── Collector Goroutine (定时采集)
    │       └── 每60秒执行一次资源采集
    │
    ├── Reporter Goroutine (定时上报)
    │       └── 每60秒向Manager上报数据
    │
    ├── Agent Monitor Goroutine (进程监控)
    │       └── 每30秒检查Agent状态
    │
    ├── Heartbeat Receiver Goroutine (心跳接收)
    │       └── 监听Agent心跳
    │
    ├── Update Handler Goroutine (更新处理)
    │       └── 监听更新指令
    │
    └── Signal Handler Goroutine (信号处理)
            └── 处理SIGTERM, SIGINT等
```

---

## 3. 详细设计

### 3.1 核心引擎 (Core Engine)

#### 3.1.1 启动流程

```go
func main() {
    // 1. 解析命令行参数
    // 2. 加载配置文件
    // 3. 初始化日志
    // 4. 初始化各模块
    // 5. 启动Agent（如果未运行）
    // 6. 连接Manager
    // 7. 注册节点
    // 8. 启动定时任务
    // 9. 等待退出信号
}
```

```
┌─────────────────────────────────────────────────────────────┐
│                      启动流程图                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐ │
│  │解析参数 │───▶│加载配置 │───▶│初始化   │───▶│初始化   │ │
│  │         │    │         │    │日志     │    │各模块   │ │
│  └─────────┘    └─────────┘    └─────────┘    └────┬────┘ │
│                                                     │      │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌────▼────┐ │
│  │等待退出 │◀───│启动定时 │◀───│注册节点 │◀───│连接     │ │
│  │信号     │    │任务     │    │         │    │Manager  │ │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘ │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### 3.1.2 优雅退出流程

```go
func gracefulShutdown(ctx context.Context) {
    // 1. 停止接收新任务
    // 2. 等待当前任务完成（超时30秒）
    // 3. 断开Manager连接
    // 4. 保存状态到本地
    // 5. 关闭日志
    // 注意：不停止Agent进程
}
```

### 3.2 资源采集模块 (Collector)

#### 3.2.1 采集器接口

```go
// Collector 采集器接口
type Collector interface {
    // Name 返回采集器名称
    Name() string
    // Collect 执行采集，返回指标数据
    Collect(ctx context.Context) (*Metrics, error)
    // Interval 返回采集间隔
    Interval() time.Duration
}

// Metrics 指标数据结构
type Metrics struct {
    Name      string                 `json:"name"`
    Timestamp time.Time              `json:"timestamp"`
    Values    map[string]interface{} `json:"values"`
}
```

#### 3.2.2 CPU采集器

```go
type CPUCollector struct {
    lastStats *cpu.TimesStat
}

func (c *CPUCollector) Collect(ctx context.Context) (*Metrics, error) {
    // 使用 gopsutil 采集CPU信息
    percent, _ := cpu.Percent(time.Second, false)
    info, _ := cpu.Info()
    times, _ := cpu.Times(false)
    
    return &Metrics{
        Name:      "cpu",
        Timestamp: time.Now(),
        Values: map[string]interface{}{
            "usage_percent": percent[0],
            "cores":         len(info),
            "model":         info[0].ModelName,
            "user":          times[0].User,
            "system":        times[0].System,
            "idle":          times[0].Idle,
        },
    }, nil
}
```

#### 3.2.3 内存采集器

```go
type MemoryCollector struct{}

func (c *MemoryCollector) Collect(ctx context.Context) (*Metrics, error) {
    vm, _ := mem.VirtualMemory()
    swap, _ := mem.SwapMemory()
    
    return &Metrics{
        Name:      "memory",
        Timestamp: time.Now(),
        Values: map[string]interface{}{
            "total":         vm.Total,
            "available":     vm.Available,
            "used":          vm.Used,
            "used_percent":  vm.UsedPercent,
            "cached":        vm.Cached,
            "buffers":       vm.Buffers,
            "swap_total":    swap.Total,
            "swap_used":     swap.Used,
        },
    }, nil
}
```

#### 3.2.4 磁盘采集器

```go
type DiskCollector struct {
    mountPoints []string // 监控的挂载点，空则监控所有
}

func (c *DiskCollector) Collect(ctx context.Context) (*Metrics, error) {
    partitions, _ := disk.Partitions(false)
    diskMetrics := make([]map[string]interface{}, 0)
    
    for _, p := range partitions {
        usage, _ := disk.Usage(p.Mountpoint)
        io, _ := disk.IOCounters(p.Device)
        
        diskMetrics = append(diskMetrics, map[string]interface{}{
            "mountpoint":   p.Mountpoint,
            "device":       p.Device,
            "fstype":       p.Fstype,
            "total":        usage.Total,
            "used":         usage.Used,
            "free":         usage.Free,
            "used_percent": usage.UsedPercent,
            "read_bytes":   io[p.Device].ReadBytes,
            "write_bytes":  io[p.Device].WriteBytes,
        })
    }
    
    return &Metrics{
        Name:      "disk",
        Timestamp: time.Now(),
        Values:    map[string]interface{}{"disks": diskMetrics},
    }, nil
}
```

#### 3.2.5 网络采集器

```go
type NetworkCollector struct {
    interfaces []string // 监控的网卡，空则监控所有
    lastStats  map[string]net.IOCountersStat
}

func (c *NetworkCollector) Collect(ctx context.Context) (*Metrics, error) {
    stats, _ := net.IOCounters(true)
    netMetrics := make([]map[string]interface{}, 0)
    
    for _, s := range stats {
        netMetrics = append(netMetrics, map[string]interface{}{
            "interface":    s.Name,
            "bytes_sent":   s.BytesSent,
            "bytes_recv":   s.BytesRecv,
            "packets_sent": s.PacketsSent,
            "packets_recv": s.PacketsRecv,
            "errin":        s.Errin,
            "errout":       s.Errout,
            "dropin":       s.Dropin,
            "dropout":      s.Dropout,
        })
    }
    
    return &Metrics{
        Name:      "network",
        Timestamp: time.Now(),
        Values:    map[string]interface{}{"interfaces": netMetrics},
    }, nil
}
```

#### 3.2.6 采集器管理器

```go
type CollectorManager struct {
    collectors []Collector
    results    chan *Metrics
    mu         sync.RWMutex
    latest     map[string]*Metrics
}

func (cm *CollectorManager) Start(ctx context.Context) {
    for _, c := range cm.collectors {
        go cm.runCollector(ctx, c)
    }
}

func (cm *CollectorManager) runCollector(ctx context.Context, c Collector) {
    ticker := time.NewTicker(c.Interval())
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            metrics, err := c.Collect(ctx)
            if err != nil {
                log.Error("collect failed", zap.String("collector", c.Name()), zap.Error(err))
                continue
            }
            cm.mu.Lock()
            cm.latest[c.Name()] = metrics
            cm.mu.Unlock()
        }
    }
}

func (cm *CollectorManager) GetLatest() map[string]*Metrics {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    result := make(map[string]*Metrics)
    for k, v := range cm.latest {
        result[k] = v
    }
    return result
}
```

### 3.3 Agent管理模块 (AgentMgr)

#### 3.3.1 进程管理器

```go
type ProcessManager struct {
    config       *AgentConfig
    process      *os.Process
    pid          int
    restartCount int
    lastRestart  time.Time
    mu           sync.Mutex
}

// Start 启动Agent进程
func (pm *ProcessManager) Start(ctx context.Context) error {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    // 检查是否已运行
    if pm.isRunning() {
        return nil
    }
    
    // 构建启动命令
    cmd := exec.Command(pm.config.BinaryPath, pm.config.Args...)
    cmd.Dir = pm.config.WorkDir
    cmd.Env = pm.config.Env
    
    // 设置进程组，确保Agent独立运行
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,
        Pgid:    0,
    }
    
    // 重定向输出
    cmd.Stdout = pm.config.Stdout
    cmd.Stderr = pm.config.Stderr
    
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("start agent failed: %w", err)
    }
    
    pm.process = cmd.Process
    pm.pid = cmd.Process.Pid
    
    // 分离进程，Daemon退出不影响Agent
    go cmd.Wait()
    
    log.Info("agent started", zap.Int("pid", pm.pid))
    return nil
}

// Stop 停止Agent进程
func (pm *ProcessManager) Stop(ctx context.Context, graceful bool) error {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    if !pm.isRunning() {
        return nil
    }
    
    if graceful {
        // 发送SIGTERM，等待优雅退出
        pm.process.Signal(syscall.SIGTERM)
        
        select {
        case <-time.After(30 * time.Second):
            // 超时强制杀死
            pm.process.Kill()
        case <-ctx.Done():
            pm.process.Kill()
        }
    } else {
        pm.process.Kill()
    }
    
    pm.process = nil
    pm.pid = 0
    return nil
}

// Restart 重启Agent
func (pm *ProcessManager) Restart(ctx context.Context) error {
    // 计算退避时间
    backoff := pm.calculateBackoff()
    if backoff > 0 {
        log.Info("waiting before restart", zap.Duration("backoff", backoff))
        time.Sleep(backoff)
    }
    
    if err := pm.Stop(ctx, true); err != nil {
        return err
    }
    
    if err := pm.Start(ctx); err != nil {
        return err
    }
    
    pm.restartCount++
    pm.lastRestart = time.Now()
    return nil
}

// calculateBackoff 计算退避时间
func (pm *ProcessManager) calculateBackoff() time.Duration {
    // 如果距离上次重启超过5分钟，重置计数
    if time.Since(pm.lastRestart) > 5*time.Minute {
        pm.restartCount = 0
        return 0
    }
    
    switch {
    case pm.restartCount < 1:
        return 0
    case pm.restartCount < 3:
        return 10 * time.Second
    case pm.restartCount < 5:
        return 30 * time.Second
    default:
        return 60 * time.Second
    }
}

func (pm *ProcessManager) isRunning() bool {
    if pm.process == nil {
        return false
    }
    // 检查进程是否存在
    err := pm.process.Signal(syscall.Signal(0))
    return err == nil
}
```

#### 3.3.2 健康检查器

```go
type HealthChecker struct {
    processManager *ProcessManager
    config         *HealthCheckConfig
    heartbeatCh    chan *Heartbeat
    lastHeartbeat  time.Time
    mu             sync.RWMutex
}

type HealthCheckConfig struct {
    CheckInterval     time.Duration // 检查间隔，默认30秒
    HeartbeatTimeout  time.Duration // 心跳超时，默认90秒（3次）
    CPUThreshold      float64       // CPU阈值，默认50%
    MemoryThreshold   uint64        // 内存阈值，默认500MB
    ThresholdDuration time.Duration // 超限持续时间，默认60秒
}

func (hc *HealthChecker) Start(ctx context.Context) {
    ticker := time.NewTicker(hc.config.CheckInterval)
    defer ticker.Stop()
    
    var overThresholdSince time.Time
    
    for {
        select {
        case <-ctx.Done():
            return
            
        case hb := <-hc.heartbeatCh:
            hc.mu.Lock()
            hc.lastHeartbeat = time.Now()
            hc.mu.Unlock()
            hc.processHeartbeat(hb)
            
        case <-ticker.C:
            status := hc.checkHealth(ctx)
            
            switch status {
            case HealthStatusDead:
                log.Warn("agent process not running, restarting")
                hc.processManager.Restart(ctx)
                
            case HealthStatusNoHeartbeat:
                log.Warn("agent heartbeat timeout, restarting")
                hc.processManager.Restart(ctx)
                
            case HealthStatusOverThreshold:
                if overThresholdSince.IsZero() {
                    overThresholdSince = time.Now()
                } else if time.Since(overThresholdSince) > hc.config.ThresholdDuration {
                    log.Warn("agent resource over threshold, restarting")
                    hc.processManager.Restart(ctx)
                    overThresholdSince = time.Time{}
                }
                
            case HealthStatusHealthy:
                overThresholdSince = time.Time{}
            }
        }
    }
}

type HealthStatus int

const (
    HealthStatusHealthy HealthStatus = iota
    HealthStatusDead
    HealthStatusNoHeartbeat
    HealthStatusOverThreshold
)

func (hc *HealthChecker) checkHealth(ctx context.Context) HealthStatus {
    // 1. 检查进程是否存在
    if !hc.processManager.isRunning() {
        return HealthStatusDead
    }
    
    // 2. 检查心跳
    hc.mu.RLock()
    lastHB := hc.lastHeartbeat
    hc.mu.RUnlock()
    
    if time.Since(lastHB) > hc.config.HeartbeatTimeout {
        return HealthStatusNoHeartbeat
    }
    
    // 3. 检查资源占用
    proc, err := process.NewProcess(int32(hc.processManager.pid))
    if err != nil {
        return HealthStatusDead
    }
    
    cpuPercent, _ := proc.CPUPercent()
    memInfo, _ := proc.MemoryInfo()
    
    if cpuPercent > hc.config.CPUThreshold || memInfo.RSS > hc.config.MemoryThreshold {
        return HealthStatusOverThreshold
    }
    
    return HealthStatusHealthy
}
```

#### 3.3.3 心跳接收器

```go
type HeartbeatReceiver struct {
    socketPath string
    listener   net.Listener
    healthCh   chan *Heartbeat
}

type Heartbeat struct {
    PID       int       `json:"pid"`
    Timestamp time.Time `json:"timestamp"`
    Version   string    `json:"version"`
    Status    string    `json:"status"`
    CPU       float64   `json:"cpu"`
    Memory    uint64    `json:"memory"`
}

func (hr *HeartbeatReceiver) Start(ctx context.Context) error {
    // 清理旧的socket文件
    os.Remove(hr.socketPath)
    
    var err error
    hr.listener, err = net.Listen("unix", hr.socketPath)
    if err != nil {
        return fmt.Errorf("listen failed: %w", err)
    }
    
    go hr.acceptLoop(ctx)
    return nil
}

func (hr *HeartbeatReceiver) acceptLoop(ctx context.Context) {
    for {
        conn, err := hr.listener.Accept()
        if err != nil {
            select {
            case <-ctx.Done():
                return
            default:
                log.Error("accept failed", zap.Error(err))
                continue
            }
        }
        go hr.handleConnection(ctx, conn)
    }
}

func (hr *HeartbeatReceiver) handleConnection(ctx context.Context, conn net.Conn) {
    defer conn.Close()
    
    decoder := json.NewDecoder(conn)
    for {
        var hb Heartbeat
        if err := decoder.Decode(&hb); err != nil {
            return
        }
        
        select {
        case hr.healthCh <- &hb:
        case <-ctx.Done():
            return
        }
    }
}
```

### 3.4 版本更新模块 (Updater)

#### 3.4.1 更新管理器

```go
type UpdateManager struct {
    config     *UpdateConfig
    downloader *Downloader
    verifier   *Verifier
    backuper   *Backuper
    processMgr *ProcessManager
}

type UpdateConfig struct {
    DownloadDir   string
    BackupDir     string
    MaxBackups    int
    VerifyTimeout time.Duration
}

type UpdateRequest struct {
    Component   string `json:"component"`   // "agent" or "daemon"
    Version     string `json:"version"`
    DownloadURL string `json:"download_url"`
    Hash        string `json:"hash"`        // SHA-256
    Signature   string `json:"signature"`   // Base64 encoded
}

type UpdateResult struct {
    Success     bool   `json:"success"`
    OldVersion  string `json:"old_version"`
    NewVersion  string `json:"new_version"`
    Error       string `json:"error,omitempty"`
    RolledBack  bool   `json:"rolled_back"`
}
```

#### 3.4.2 更新流程

```go
func (um *UpdateManager) Update(ctx context.Context, req *UpdateRequest) *UpdateResult {
    result := &UpdateResult{
        OldVersion: um.getCurrentVersion(req.Component),
        NewVersion: req.Version,
    }
    
    // 1. 下载更新包
    pkgPath, err := um.downloader.Download(ctx, req.DownloadURL)
    if err != nil {
        result.Error = fmt.Sprintf("download failed: %v", err)
        return result
    }
    defer os.Remove(pkgPath)
    
    // 2. 验证签名
    if err := um.verifier.VerifySignature(pkgPath, req.Signature); err != nil {
        result.Error = fmt.Sprintf("signature verification failed: %v", err)
        return result
    }
    
    // 3. 验证哈希
    if err := um.verifier.VerifyHash(pkgPath, req.Hash); err != nil {
        result.Error = fmt.Sprintf("hash verification failed: %v", err)
        return result
    }
    
    // 4. 备份当前版本
    backupPath, err := um.backuper.Backup(req.Component)
    if err != nil {
        result.Error = fmt.Sprintf("backup failed: %v", err)
        return result
    }
    
    // 5. 执行更新
    if err := um.doUpdate(ctx, req, pkgPath); err != nil {
        // 6. 更新失败，回滚
        if rollbackErr := um.backuper.Restore(backupPath, req.Component); rollbackErr != nil {
            result.Error = fmt.Sprintf("update failed: %v, rollback failed: %v", err, rollbackErr)
        } else {
            result.Error = fmt.Sprintf("update failed: %v, rolled back", err)
            result.RolledBack = true
        }
        return result
    }
    
    // 7. 验证新版本
    if err := um.verifyNewVersion(ctx, req); err != nil {
        // 验证失败，回滚
        if rollbackErr := um.backuper.Restore(backupPath, req.Component); rollbackErr != nil {
            result.Error = fmt.Sprintf("verification failed: %v, rollback failed: %v", err, rollbackErr)
        } else {
            result.Error = fmt.Sprintf("verification failed: %v, rolled back", err)
            result.RolledBack = true
        }
        return result
    }
    
    // 8. 清理旧备份
    um.backuper.CleanOldBackups(req.Component, um.config.MaxBackups)
    
    result.Success = true
    return result
}

func (um *UpdateManager) doUpdate(ctx context.Context, req *UpdateRequest, pkgPath string) error {
    if req.Component == "agent" {
        return um.updateAgent(ctx, pkgPath)
    }
    return um.updateDaemon(ctx, pkgPath)
}

func (um *UpdateManager) updateAgent(ctx context.Context, pkgPath string) error {
    // 1. 停止Agent
    if err := um.processMgr.Stop(ctx, true); err != nil {
        return fmt.Errorf("stop agent failed: %w", err)
    }
    
    // 2. 替换文件
    if err := um.replaceFile(pkgPath, um.processMgr.config.BinaryPath); err != nil {
        return fmt.Errorf("replace file failed: %w", err)
    }
    
    // 3. 启动新版本
    if err := um.processMgr.Start(ctx); err != nil {
        return fmt.Errorf("start agent failed: %w", err)
    }
    
    return nil
}
```

#### 3.4.3 下载器

```go
type Downloader struct {
    httpClient *http.Client
    downloadDir string
}

func (d *Downloader) Download(ctx context.Context, url string) (string, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return "", err
    }
    
    resp, err := d.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("download failed: status %d", resp.StatusCode)
    }
    
    // 创建临时文件
    tmpFile, err := os.CreateTemp(d.downloadDir, "update-*.tmp")
    if err != nil {
        return "", err
    }
    defer tmpFile.Close()
    
    // 写入文件
    if _, err := io.Copy(tmpFile, resp.Body); err != nil {
        os.Remove(tmpFile.Name())
        return "", err
    }
    
    return tmpFile.Name(), nil
}
```

#### 3.4.4 验证器

```go
type Verifier struct {
    publicKey *rsa.PublicKey
}

func (v *Verifier) VerifyHash(filePath, expectedHash string) error {
    f, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer f.Close()
    
    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil {
        return err
    }
    
    actualHash := hex.EncodeToString(h.Sum(nil))
    if actualHash != expectedHash {
        return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actualHash)
    }
    
    return nil
}

func (v *Verifier) VerifySignature(filePath, signatureBase64 string) error {
    // 读取文件内容
    content, err := os.ReadFile(filePath)
    if err != nil {
        return err
    }
    
    // 解码签名
    signature, err := base64.StdEncoding.DecodeString(signatureBase64)
    if err != nil {
        return fmt.Errorf("decode signature failed: %w", err)
    }
    
    // 计算哈希
    hashed := sha256.Sum256(content)
    
    // 验证签名
    if err := rsa.VerifyPKCS1v15(v.publicKey, crypto.SHA256, hashed[:], signature); err != nil {
        return fmt.Errorf("signature verification failed: %w", err)
    }
    
    return nil
}
```

#### 3.4.5 备份器

```go
type Backuper struct {
    backupDir string
}

func (b *Backuper) Backup(component string) (string, error) {
    srcPath := b.getComponentPath(component)
    
    // 生成备份文件名
    timestamp := time.Now().Format("20060102150405")
    backupName := fmt.Sprintf("%s_%s.bak", component, timestamp)
    backupPath := filepath.Join(b.backupDir, backupName)
    
    // 复制文件
    if err := copyFile(srcPath, backupPath); err != nil {
        return "", err
    }
    
    return backupPath, nil
}

func (b *Backuper) Restore(backupPath, component string) error {
    dstPath := b.getComponentPath(component)
    return copyFile(backupPath, dstPath)
}

func (b *Backuper) CleanOldBackups(component string, keepCount int) {
    pattern := filepath.Join(b.backupDir, component+"_*.bak")
    files, _ := filepath.Glob(pattern)
    
    if len(files) <= keepCount {
        return
    }
    
    // 按时间排序，删除旧的
    sort.Strings(files)
    for _, f := range files[:len(files)-keepCount] {
        os.Remove(f)
    }
}
```

#### 3.4.6 Daemon自更新

```go
func (um *UpdateManager) updateDaemon(ctx context.Context, pkgPath string) error {
    // Daemon自更新需要特殊处理
    
    // 1. 准备更新脚本
    script := um.generateUpdateScript(pkgPath)
    scriptPath := filepath.Join(um.config.DownloadDir, "update.sh")
    if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
        return err
    }
    
    // 2. 启动更新进程
    cmd := exec.Command("/bin/bash", scriptPath)
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,
    }
    
    if err := cmd.Start(); err != nil {
        return err
    }
    
    // 3. 退出当前进程，让更新脚本完成工作
    log.Info("daemon self-update initiated, exiting...")
    os.Exit(0)
    
    return nil
}

func (um *UpdateManager) generateUpdateScript(pkgPath string) string {
    daemonPath := os.Args[0]
    return fmt.Sprintf(`#!/bin/bash
# 等待旧Daemon退出
sleep 2

# 备份旧版本
cp %s %s.bak

# 替换新版本
cp %s %s
chmod +x %s

# 启动新版本
systemctl restart daemon

# 清理
rm -f %s
rm -f $0
`, daemonPath, daemonPath, pkgPath, daemonPath, daemonPath, pkgPath)
}
```

### 3.5 通信模块 (Communication)

#### 3.5.1 gRPC客户端

```go
type GRPCClient struct {
    conn      *grpc.ClientConn
    client    pb.DaemonServiceClient
    config    *GRPCConfig
    nodeID    string
    reconnectCh chan struct{}
}

type GRPCConfig struct {
    ServerAddr    string
    TLSCertFile   string
    TLSKeyFile    string
    CACertFile    string
    Timeout       time.Duration
    RetryInterval time.Duration
}

func (gc *GRPCClient) Connect(ctx context.Context) error {
    // 加载TLS证书
    cert, err := tls.LoadX509KeyPair(gc.config.TLSCertFile, gc.config.TLSKeyFile)
    if err != nil {
        return err
    }
    
    caCert, err := os.ReadFile(gc.config.CACertFile)
    if err != nil {
        return err
    }
    
    certPool := x509.NewCertPool()
    certPool.AppendCertsFromPEM(caCert)
    
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      certPool,
    }
    
    // 建立连接
    gc.conn, err = grpc.DialContext(ctx, gc.config.ServerAddr,
        grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            Time:                10 * time.Second,
            Timeout:             3 * time.Second,
            PermitWithoutStream: true,
        }),
    )
    if err != nil {
        return err
    }
    
    gc.client = pb.NewDaemonServiceClient(gc.conn)
    return nil
}

func (gc *GRPCClient) Register(ctx context.Context, info *NodeInfo) error {
    resp, err := gc.client.Register(ctx, &pb.RegisterRequest{
        Hostname: info.Hostname,
        IP:       info.IP,
        OS:       info.OS,
        Arch:     info.Arch,
        Labels:   info.Labels,
    })
    if err != nil {
        return err
    }
    
    gc.nodeID = resp.NodeId
    return nil
}

func (gc *GRPCClient) ReportMetrics(ctx context.Context, metrics map[string]*Metrics) error {
    data, _ := json.Marshal(metrics)
    
    _, err := gc.client.ReportMetrics(ctx, &pb.MetricsRequest{
        NodeId:    gc.nodeID,
        Timestamp: time.Now().Unix(),
        Data:      data,
    })
    return err
}

func (gc *GRPCClient) Heartbeat(ctx context.Context) error {
    _, err := gc.client.Heartbeat(ctx, &pb.HeartbeatRequest{
        NodeId:    gc.nodeID,
        Timestamp: time.Now().Unix(),
    })
    return err
}
```

---

## 4. 数据结构

### 4.1 节点信息

```go
type NodeInfo struct {
    NodeID     string            `json:"node_id"`
    Hostname   string            `json:"hostname"`
    IP         string            `json:"ip"`
    OS         string            `json:"os"`
    Arch       string            `json:"arch"`
    Labels     map[string]string `json:"labels"`
    DaemonVer  string            `json:"daemon_version"`
    AgentVer   string            `json:"agent_version"`
    RegisterAt time.Time         `json:"register_at"`
}
```

### 4.2 Agent状态

```go
type AgentStatus struct {
    PID         int       `json:"pid"`
    Status      string    `json:"status"` // running, stopped, unknown
    Version     string    `json:"version"`
    StartTime   time.Time `json:"start_time"`
    CPU         float64   `json:"cpu_percent"`
    Memory      uint64    `json:"memory_bytes"`
    Restarts    int       `json:"restart_count"`
    LastRestart time.Time `json:"last_restart"`
}
```

### 4.3 上报数据结构

```go
type ReportData struct {
    NodeID      string                 `json:"node_id"`
    Timestamp   time.Time              `json:"timestamp"`
    Metrics     map[string]*Metrics    `json:"metrics"`
    AgentStatus *AgentStatus           `json:"agent_status"`
    Logs        []LogEntry             `json:"logs,omitempty"`
}

type LogEntry struct {
    Level     string    `json:"level"`
    Message   string    `json:"message"`
    Timestamp time.Time `json:"timestamp"`
    Fields    map[string]interface{} `json:"fields,omitempty"`
}
```

---

## 5. 配置管理

### 5.1 配置文件结构

```yaml
# daemon.yaml

# 基础配置
daemon:
  id: ""  # 留空则自动生成
  log_level: info
  log_file: /var/log/daemon/daemon.log
  pid_file: /var/run/daemon.pid
  work_dir: /var/lib/daemon

# Manager连接配置
manager:
  address: "manager.example.com:8443"
  tls:
    cert_file: /etc/daemon/certs/client.crt
    key_file: /etc/daemon/certs/client.key
    ca_file: /etc/daemon/certs/ca.crt
  heartbeat_interval: 60s
  reconnect_interval: 10s
  timeout: 30s

# Agent管理配置
agent:
  binary_path: /usr/local/bin/agent
  work_dir: /var/lib/agent
  config_file: /etc/agent/agent.yaml
  socket_path: /var/run/agent.sock
  health_check:
    interval: 30s
    heartbeat_timeout: 90s
    cpu_threshold: 50
    memory_threshold: 524288000  # 500MB
    threshold_duration: 60s
  restart:
    max_retries: 10
    backoff_base: 10s
    backoff_max: 60s

# 采集器配置
collectors:
  cpu:
    enabled: true
    interval: 60s
  memory:
    enabled: true
    interval: 60s
  disk:
    enabled: true
    interval: 60s
    mount_points: []  # 空则采集所有
  network:
    enabled: true
    interval: 60s
    interfaces: []  # 空则采集所有

# 更新配置
update:
  download_dir: /var/lib/daemon/downloads
  backup_dir: /var/lib/daemon/backups
  max_backups: 5
  verify_timeout: 300s
  public_key_file: /etc/daemon/keys/update.pub
```

### 5.2 配置加载

```go
type Config struct {
    Daemon     DaemonConfig     `yaml:"daemon"`
    Manager    ManagerConfig    `yaml:"manager"`
    Agent      AgentConfig      `yaml:"agent"`
    Collectors CollectorConfigs `yaml:"collectors"`
    Update     UpdateConfig     `yaml:"update"`
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    config := &Config{}
    if err := yaml.Unmarshal(data, config); err != nil {
        return nil, err
    }
    
    // 设置默认值
    setDefaults(config)
    
    // 验证配置
    if err := validateConfig(config); err != nil {
        return nil, err
    }
    
    return config, nil
}
```

---

## 6. 安全设计

### 6.1 通信安全

```
┌─────────────────────────────────────────────────────────────┐
│                      安全通信架构                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Daemon ◄──── mTLS (双向认证) ────► Manager                │
│    │                                                        │
│    │         ┌─────────────────────────────────────┐       │
│    │         │  • TLS 1.3                          │       │
│    │         │  • 客户端证书认证                    │       │
│    │         │  • 服务端证书验证                    │       │
│    │         │  • 证书轮换支持                      │       │
│    │         └─────────────────────────────────────┘       │
│    │                                                        │
│  Daemon ◄──── Unix Socket ────► Agent                      │
│              (本地进程间通信，无需加密)                      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 6.2 更新包安全

```
┌─────────────────────────────────────────────────────────────┐
│                    更新包安全验证流程                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. 来源验证                                                │
│     └── 验证下载URL是否来自可信Manager                      │
│                                                             │
│  2. 签名验证                                                │
│     ├── 使用预置公钥验证RSA/ECDSA签名                       │
│     └── 签名覆盖完整文件内容                                │
│                                                             │
│  3. 完整性校验                                              │
│     ├── 计算SHA-256哈希                                     │
│     └── 与预期哈希值比对                                    │
│                                                             │
│  4. 版本检查                                                │
│     └── 确保新版本号大于当前版本                            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 6.3 敏感信息保护

- 配置文件中的敏感信息（如密钥路径）需要适当权限保护
- 内存中的密钥材料使用后及时清除
- 日志中不输出敏感信息

---

## 7. 错误处理

### 7.1 错误码定义

| 错误码 | 名称 | 说明 |
|--------|------|------|
| D1001 | ConfigLoadError | 配置加载失败 |
| D1002 | ConfigValidateError | 配置验证失败 |
| D2001 | ManagerConnectError | 连接Manager失败 |
| D2002 | ManagerRegisterError | 节点注册失败 |
| D2003 | ManagerReportError | 数据上报失败 |
| D3001 | AgentStartError | Agent启动失败 |
| D3002 | AgentStopError | Agent停止失败 |
| D3003 | AgentHealthError | Agent健康检查失败 |
| D4001 | UpdateDownloadError | 更新包下载失败 |
| D4002 | UpdateVerifyError | 更新包验证失败 |
| D4003 | UpdateApplyError | 更新应用失败 |
| D4004 | UpdateRollbackError | 回滚失败 |
| D5001 | CollectError | 资源采集失败 |

### 7.2 重试策略

```go
type RetryConfig struct {
    MaxRetries int
    BaseDelay  time.Duration
    MaxDelay   time.Duration
    Multiplier float64
}

func RetryWithBackoff(ctx context.Context, config *RetryConfig, fn func() error) error {
    delay := config.BaseDelay
    
    for i := 0; i < config.MaxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        
        if i == config.MaxRetries-1 {
            return err
        }
        
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(delay):
        }
        
        delay = time.Duration(float64(delay) * config.Multiplier)
        if delay > config.MaxDelay {
            delay = config.MaxDelay
        }
    }
    
    return nil
}
```

---

## 8. 测试设计

### 8.1 单元测试

| 测试模块 | 测试点 | 覆盖率要求 |
|----------|--------|------------|
| Collector | 各采集器正确性 | > 80% |
| AgentMgr | 进程管理、健康检查 | > 80% |
| Updater | 下载、验证、备份、回滚 | > 90% |
| Communication | 连接、重连、数据序列化 | > 70% |

### 8.2 集成测试

```go
// 示例：Agent管理集成测试
func TestAgentLifecycle(t *testing.T) {
    // 1. 启动Daemon
    // 2. 验证Agent自动启动
    // 3. 模拟Agent崩溃
    // 4. 验证自动重启
    // 5. 验证重启退避策略
    // 6. 优雅停止Daemon
    // 7. 验证Agent继续运行
}
```

### 8.3 性能测试

| 测试场景 | 预期指标 |
|----------|----------|
| 空闲状态CPU占用 | < 1% |
| 空闲状态内存占用 | < 30MB |
| 资源采集耗时 | < 500ms |
| 单次上报延迟 | < 200ms |

---

## 附录

### A. 目录结构

```
daemon/
├── cmd/
│   └── daemon/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── collector/
│   │   ├── collector.go
│   │   ├── cpu.go
│   │   ├── memory.go
│   │   ├── disk.go
│   │   └── network.go
│   ├── agent/
│   │   ├── manager.go
│   │   ├── health.go
│   │   └── heartbeat.go
│   ├── updater/
│   │   ├── updater.go
│   │   ├── downloader.go
│   │   ├── verifier.go
│   │   └── backuper.go
│   ├── comm/
│   │   ├── grpc.go
│   │   └── socket.go
│   └── core/
│       ├── engine.go
│       └── signal.go
├── pkg/
│   └── proto/
│       └── daemon.proto
├── configs/
│   └── daemon.yaml
└── scripts/
    ├── install.sh
    └── update.sh
```

### B. 依赖清单

```go
// go.mod
module daemon

go 1.21

require (
    github.com/shirou/gopsutil/v3 v3.23.0
    go.uber.org/zap v1.26.0
    google.golang.org/grpc v1.59.0
    gopkg.in/yaml.v3 v3.0.1
)
```

---

*— 文档结束 —*
