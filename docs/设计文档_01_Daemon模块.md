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
| **AgentMgr** | **多Agent管理** | **AgentRegistry, MultiAgentManager, MultiHealthChecker, AgentInstance** |
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

### 3.5 Agent 管理模块 (Multi-Agent Management)

#### 3.5.1 AgentRegistry 设计

**职责**: 管理所有已注册的 Agent 实例,提供并发安全的注册、查询、列举功能。

**数据结构**:

```go
// AgentInfo Agent信息结构体
type AgentInfo struct {
    // 基础信息
    ID           string      // Agent唯一标识符
    Type         AgentType   // Agent类型(filebeat/telegraf/node_exporter/custom)
    Name         string      // Agent显示名称
    
    // 配置信息
    BinaryPath   string      // Agent可执行文件路径
    ConfigFile   string      // Agent配置文件路径(可为空)
    WorkDir      string      // Agent工作目录
    SocketPath   string      // Unix Socket路径(可为空)
    
    // 运行时状态
    PID          int         // 进程ID,0表示未运行
    Status       AgentStatus // 运行状态
    RestartCount int         // 重启次数
    LastRestart  time.Time   // 上次重启时间
    
    // 时间戳
    CreatedAt    time.Time   // 注册时间
    UpdatedAt    time.Time   // 最后更新时间
    
    // 并发保护
    mu           sync.RWMutex
}

// AgentStatus Agent运行状态
type AgentStatus string

const (
    StatusStopped    AgentStatus = "stopped"     // 已停止
    StatusStarting   AgentStatus = "starting"    // 正在启动
    StatusRunning    AgentStatus = "running"     // 正在运行
    StatusStopping   AgentStatus = "stopping"    // 正在停止
    StatusRestarting AgentStatus = "restarting"  // 正在重启
    StatusFailed     AgentStatus = "failed"      // 失败
)

// AgentType Agent类型
type AgentType string

const (
    TypeFilebeat     AgentType = "filebeat"      // Filebeat日志采集
    TypeTelegraf     AgentType = "telegraf"      // Telegraf指标采集
    TypeNodeExporter AgentType = "node_exporter" // Node Exporter指标暴露
    TypeCustom       AgentType = "custom"        // 自定义Agent
)

// AgentRegistry Agent注册表
type AgentRegistry struct {
    agents map[string]*AgentInfo  // key: Agent ID
    mu     sync.RWMutex
}

func NewAgentRegistry() *AgentRegistry {
    return &AgentRegistry{
        agents: make(map[string]*AgentInfo),
    }
}

// Register 注册Agent
func (r *AgentRegistry) Register(
    id string,
    agentType AgentType,
    name string,
    binaryPath string,
    configFile string,
    workDir string,
    socketPath string,
) (*AgentInfo, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    // 检查ID是否已存在
    if _, exists := r.agents[id]; exists {
        return nil, fmt.Errorf("agent already exists: %s", id)
    }
    
    // 创建AgentInfo
    now := time.Now()
    info := &AgentInfo{
        ID:           id,
        Type:         agentType,
        Name:         name,
        BinaryPath:   binaryPath,
        ConfigFile:   configFile,
        WorkDir:      workDir,
        SocketPath:   socketPath,
        PID:          0,
        Status:       StatusStopped,
        RestartCount: 0,
        CreatedAt:    now,
        UpdatedAt:    now,
    }
    
    r.agents[id] = info
    return info, nil
}

// Get 获取Agent信息
func (r *AgentRegistry) Get(id string) *AgentInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.agents[id]
}

// List 列举所有Agent
func (r *AgentRegistry) List() []*AgentInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    result := make([]*AgentInfo, 0, len(r.agents))
    for _, info := range r.agents {
        result = append(result, info)
    }
    return result
}

// ListByType 根据类型列举Agent
func (r *AgentRegistry) ListByType(agentType AgentType) []*AgentInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    result := make([]*AgentInfo, 0)
    for _, info := range r.agents {
        if info.Type == agentType {
            result = append(result, info)
        }
    }
    return result
}

// Unregister 注销Agent
func (r *AgentRegistry) Unregister(id string) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    info, exists := r.agents[id]
    if !exists {
        return fmt.Errorf("agent not found: %s", id)
    }
    
    // 检查是否正在运行
    info.mu.RLock()
    isRunning := info.Status == StatusRunning || info.Status == StatusStarting
    info.mu.RUnlock()
    
    if isRunning {
        return fmt.Errorf("agent is running, cannot unregister: %s", id)
    }
    
    delete(r.agents, id)
    return nil
}
```

**注册流程**:

```
1. Daemon启动
   ↓
2. 读取daemon.yaml配置文件
   ↓
3. 解析agents数组配置
   ↓
4. 遍历每个Agent配置
   ↓
5. 调用AgentRegistry.Register()注册Agent
   ↓
6. 验证Agent配置(二进制文件存在、工作目录权限等)
   ↓
7. 创建AgentInfo并存储到注册表
   ↓
8. 返回AgentInfo指针供后续使用
```

#### 3.5.2 MultiAgentManager 设计

**职责**: 管理多个Agent实例的生命周期,提供批量操作和单个操作接口。

**数据结构**:

```go
// MultiAgentManager 多Agent管理器
type MultiAgentManager struct {
    registry   *AgentRegistry
    instances  map[string]*AgentInstance  // key: Agent ID
    mu         sync.RWMutex
    logger     *zap.Logger
}

func NewMultiAgentManager(registry *AgentRegistry, logger *zap.Logger) *MultiAgentManager {
    return &MultiAgentManager{
        registry:  registry,
        instances: make(map[string]*AgentInstance),
        logger:    logger,
    }
}

// RegisterAgent 注册并创建Agent实例
func (m *MultiAgentManager) RegisterAgent(info *AgentInfo) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // 检查是否已存在
    if _, exists := m.instances[info.ID]; exists {
        return fmt.Errorf("agent instance already exists: %s", info.ID)
    }
    
    // 创建AgentInstance
    instance := NewAgentInstance(info, m.logger)
    m.instances[info.ID] = instance
    
    return nil
}

// StartAgent 启动指定Agent
func (m *MultiAgentManager) StartAgent(ctx context.Context, agentID string) error {
    m.mu.RLock()
    instance, exists := m.instances[agentID]
    m.mu.RUnlock()
    
    if !exists {
        return fmt.Errorf("agent not found: %s", agentID)
    }
    
    return instance.Start(ctx)
}

// StopAgent 停止指定Agent
func (m *MultiAgentManager) StopAgent(ctx context.Context, agentID string, graceful bool) error {
    m.mu.RLock()
    instance, exists := m.instances[agentID]
    m.mu.RUnlock()
    
    if !exists {
        return fmt.Errorf("agent not found: %s", agentID)
    }
    
    return instance.Stop(ctx, graceful)
}

// RestartAgent 重启指定Agent
func (m *MultiAgentManager) RestartAgent(ctx context.Context, agentID string) error {
    m.mu.RLock()
    instance, exists := m.instances[agentID]
    m.mu.RUnlock()
    
    if !exists {
        return fmt.Errorf("agent not found: %s", agentID)
    }
    
    return instance.Restart(ctx)
}

// StartAll 启动所有Agent
func (m *MultiAgentManager) StartAll(ctx context.Context) error {
    m.mu.RLock()
    instances := make([]*AgentInstance, 0, len(m.instances))
    for _, instance := range m.instances {
        instances = append(instances, instance)
    }
    m.mu.RUnlock()
    
    var errs []error
    for _, instance := range instances {
        if err := instance.Start(ctx); err != nil {
            errs = append(errs, err)
        }
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("failed to start some agents: %v", errs)
    }
    return nil
}

// StopAll 停止所有Agent
func (m *MultiAgentManager) StopAll(ctx context.Context, graceful bool) error {
    m.mu.RLock()
    instances := make([]*AgentInstance, 0, len(m.instances))
    for _, instance := range m.instances {
        instances = append(instances, instance)
    }
    m.mu.RUnlock()
    
    var errs []error
    for _, instance := range instances {
        if err := instance.Stop(ctx, graceful); err != nil {
            errs = append(errs, err)
        }
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("failed to stop some agents: %v", errs)
    }
    return nil
}
```

**启动流程**:

```
1. 接收启动请求(StartAgent或StartAll)
   ↓
2. 从instances映射中获取AgentInstance
   ↓
3. 调用AgentInstance.Start()
   ↓
4. AgentInstance根据AgentType生成启动参数
   ↓
5. 执行exec.Command启动Agent进程
   ↓
6. 设置进程组,确保Agent独立运行
   ↓
7. 更新AgentInfo状态为StatusRunning
   ↓
8. 记录PID到AgentInfo
   ↓
9. 启动Agent监控goroutine
   ↓
10. 返回启动结果
```

#### 3.5.3 MultiHealthChecker 设计

**职责**: 监控所有Agent的健康状态,执行健康检查并触发自动恢复。

**检查策略**:

```go
// MultiHealthChecker 多Agent健康检查器
type MultiHealthChecker struct {
    registry      *AgentRegistry
    manager       *MultiAgentManager
    config        *HealthCheckConfig
    heartbeatMgr  *HeartbeatManager
    logger        *zap.Logger
}

type HealthCheckConfig struct {
    CheckInterval     time.Duration  // 检查间隔,默认30秒
    HeartbeatTimeout  time.Duration  // 心跳超时,默认90秒
    CPUThreshold      float64        // CPU阈值,默认50%
    MemoryThreshold   uint64         // 内存阈值,默认500MB
    ThresholdDuration time.Duration  // 超限持续时间,默认60秒
    HTTPEndpoint      string         // HTTP健康检查端点(可选)
}

func (hc *MultiHealthChecker) Start(ctx context.Context) {
    ticker := time.NewTicker(hc.config.CheckInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            hc.checkAllAgents(ctx)
        }
    }
}

func (hc *MultiHealthChecker) checkAllAgents(ctx context.Context) {
    agents := hc.registry.List()
    
    for _, agent := range agents {
        go hc.checkAgent(ctx, agent)
    }
}

func (hc *MultiHealthChecker) checkAgent(ctx context.Context, agent *AgentInfo) {
    // 1. 检查进程是否存在
    if !hc.isProcessRunning(agent.PID) {
        hc.logger.Warn("agent process not running",
            zap.String("agent_id", agent.ID),
            zap.Int("pid", agent.PID))
        hc.recoverAgent(ctx, agent)
        return
    }
    
    // 2. 检查心跳(如果配置了心跳)
    if hc.config.HeartbeatTimeout > 0 {
        if hc.isHeartbeatTimeout(agent) {
            hc.logger.Warn("agent heartbeat timeout",
                zap.String("agent_id", agent.ID))
            hc.recoverAgent(ctx, agent)
            return
        }
    }
    
    // 3. 检查资源使用(如果配置了阈值)
    if hc.config.CPUThreshold > 0 || hc.config.MemoryThreshold > 0 {
        if hc.isResourceOverThreshold(agent) {
            hc.logger.Warn("agent resource over threshold",
                zap.String("agent_id", agent.ID))
            hc.recoverAgent(ctx, agent)
            return
        }
    }
    
    // 4. HTTP端点检查(如果配置了HTTP端点)
    if hc.config.HTTPEndpoint != "" {
        if !hc.checkHTTPEndpoint(agent) {
            hc.logger.Warn("agent http endpoint check failed",
                zap.String("agent_id", agent.ID))
            hc.recoverAgent(ctx, agent)
            return
        }
    }
}

func (hc *MultiHealthChecker) recoverAgent(ctx context.Context, agent *AgentInfo) {
    // 自动重启Agent
    hc.logger.Info("recovering agent", zap.String("agent_id", agent.ID))
    if err := hc.manager.RestartAgent(ctx, agent.ID); err != nil {
        hc.logger.Error("failed to recover agent",
            zap.String("agent_id", agent.ID),
            zap.Error(err))
    }
}
```

**健康检查类型**:

1. **进程检查**: 检查Agent进程是否存在(所有Agent都支持)
2. **HTTP端点检查**: 定期请求HTTP端点,检查响应状态(如Node Exporter的/metrics)
3. **心跳检查**: 检查Agent心跳是否超时(需Agent支持)
4. **资源检查**: 检查CPU、内存使用是否超过阈值

#### 3.5.4 ConfigManager 设计

**职责**: 管理Agent配置加载、验证和热更新。

**配置加载**:

```go
// AgentConfig Agent配置结构
type AgentConfig struct {
    ID           string            `yaml:"id"`
    Type         string            `yaml:"type"`
    Name         string            `yaml:"name"`
    BinaryPath   string            `yaml:"binary_path"`
    ConfigFile   string            `yaml:"config_file"`
    WorkDir      string            `yaml:"work_dir"`
    SocketPath   string            `yaml:"socket_path"`
    Enabled      bool              `yaml:"enabled"`
    Args         []string          `yaml:"args"`
    HealthCheck  HealthCheckConfig `yaml:"health_check"`
    Restart      RestartConfig     `yaml:"restart"`
}

type RestartConfig struct {
    MaxRetries  int           `yaml:"max_retries"`
    BackoffBase time.Duration `yaml:"backoff_base"`
    BackoffMax  time.Duration `yaml:"backoff_max"`
    Policy      string        `yaml:"policy"` // always, on-failure, never
}

// ConfigManager 配置管理器
type ConfigManager struct {
    configPath string
    agents     []AgentConfig
    mu         sync.RWMutex
}

func (cm *ConfigManager) LoadAgents() ([]AgentConfig, error) {
    data, err := os.ReadFile(cm.configPath)
    if err != nil {
        return nil, err
    }
    
    var config struct {
        Agents []AgentConfig `yaml:"agents"`
    }
    
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }
    
    // 验证配置
    for i, agent := range config.Agents {
        if err := cm.validateAgentConfig(&agent); err != nil {
            return nil, fmt.Errorf("agent[%d] validation failed: %w", i, err)
        }
    }
    
    cm.mu.Lock()
    cm.agents = config.Agents
    cm.mu.Unlock()
    
    return config.Agents, nil
}

func (cm *ConfigManager) validateAgentConfig(config *AgentConfig) error {
    // 1. 验证必需字段
    if config.ID == "" {
        return fmt.Errorf("agent id is required")
    }
    if config.Type == "" {
        return fmt.Errorf("agent type is required")
    }
    if config.BinaryPath == "" {
        return fmt.Errorf("binary_path is required")
    }
    
    // 2. 验证二进制文件存在
    if _, err := os.Stat(config.BinaryPath); os.IsNotExist(err) {
        return fmt.Errorf("binary file not found: %s", config.BinaryPath)
    }
    
    // 3. 验证配置文件存在(如果指定)
    if config.ConfigFile != "" {
        if _, err := os.Stat(config.ConfigFile); os.IsNotExist(err) {
            return fmt.Errorf("config file not found: %s", config.ConfigFile)
        }
    }
    
    // 4. 创建工作目录(如果不存在)
    if config.WorkDir != "" {
        if err := os.MkdirAll(config.WorkDir, 0755); err != nil {
            return fmt.Errorf("failed to create work dir: %w", err)
        }
    }
    
    return nil
}
```

**配置示例**(daemon.yaml):

```yaml
agents:
  - id: filebeat-logs
    type: filebeat
    name: "Filebeat Log Collector"
    binary_path: /usr/bin/filebeat
    config_file: /etc/filebeat/filebeat.yml
    work_dir: /var/lib/daemon/agents/filebeat-logs
    enabled: true
    args:
      - "-c"
      - "/etc/filebeat/filebeat.yml"
      - "-path.home"
      - "/var/lib/daemon/agents/filebeat-logs"
    health_check:
      interval: 30s
      heartbeat_timeout: 90s
      cpu_threshold: 50.0
      memory_threshold: 524288000
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always

  - id: telegraf-metrics
    type: telegraf
    name: "Telegraf Metrics Collector"
    binary_path: /usr/bin/telegraf
    config_file: /etc/telegraf/telegraf.conf
    work_dir: /var/lib/daemon/agents/telegraf-metrics
    enabled: true
    args:
      - "-config"
      - "/etc/telegraf/telegraf.conf"
    health_check:
      interval: 30s
      cpu_threshold: 40.0
      memory_threshold: 262144000
    restart:
      policy: always

  - id: node-exporter
    type: node_exporter
    name: "Node Exporter Metrics"
    binary_path: /usr/local/bin/node_exporter
    work_dir: /var/lib/daemon/agents/node-exporter
    enabled: true
    args:
      - "--web.listen-address=:9100"
      - "--path.procfs=/proc"
      - "--path.sysfs=/sys"
    health_check:
      interval: 30s
      http_endpoint: "http://localhost:9100/metrics"
    restart:
      policy: always
```

#### 3.5.5 ResourceMonitor 设计

**职责**: 监控Agent资源使用情况,触发告警。

```go
// ResourceMonitor 资源监控器
type ResourceMonitor struct {
    registry *AgentRegistry
    config   *ResourceMonitorConfig
    logger   *zap.Logger
}

type ResourceMonitorConfig struct {
    CheckInterval time.Duration
    CPUThreshold  float64
    MemThreshold  uint64
    AlertCallback func(agentID string, alert *ResourceAlert)
}

type ResourceAlert struct {
    AgentID     string
    Type        string  // "cpu" or "memory"
    CurrentValue interface{}
    Threshold   interface{}
    Timestamp   time.Time
}

func (rm *ResourceMonitor) Start(ctx context.Context) {
    ticker := time.NewTicker(rm.config.CheckInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            rm.monitorAllAgents()
        }
    }
}

func (rm *ResourceMonitor) monitorAllAgents() {
    agents := rm.registry.List()
    
    for _, agent := range agents {
        if agent.PID == 0 {
            continue
        }
        
        // 检查CPU使用率
        cpuPercent := rm.getCPUPercent(agent.PID)
        if cpuPercent > rm.config.CPUThreshold {
            alert := &ResourceAlert{
                AgentID:      agent.ID,
                Type:         "cpu",
                CurrentValue: cpuPercent,
                Threshold:    rm.config.CPUThreshold,
                Timestamp:    time.Now(),
            }
            rm.config.AlertCallback(agent.ID, alert)
        }
        
        // 检查内存使用
        memBytes := rm.getMemoryBytes(agent.PID)
        if memBytes > rm.config.MemThreshold {
            alert := &ResourceAlert{
                AgentID:      agent.ID,
                Type:         "memory",
                CurrentValue: memBytes,
                Threshold:    rm.config.MemThreshold,
                Timestamp:    time.Now(),
            }
            rm.config.AlertCallback(agent.ID, alert)
        }
    }
}
```

### 3.6 通信模块 (Communication)

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

### 4.1 Agent 数据结构

#### 4.1.1 AgentInfo - Agent 完整信息

```go
// AgentInfo Agent信息结构体
// 存储单个Agent实例的完整信息,包括配置、状态和运行时数据
type AgentInfo struct {
    // 唯一标识
    ID string `json:"id"` // Agent唯一标识符,格式: {type}-{name}
    
    // 类型信息
    Type AgentType `json:"type"` // Agent类型
    Name string    `json:"name"` // Agent显示名称
    
    // 配置信息
    BinaryPath string `json:"binary_path"` // 可执行文件路径
    ConfigFile string `json:"config_file"` // 配置文件路径(可为空)
    WorkDir    string `json:"work_dir"`    // 工作目录
    SocketPath string `json:"socket_path"` // Unix Socket路径(可为空)
    
    // 运行时状态
    PID          int         `json:"pid"`           // 进程ID,0表示未运行
    Status       AgentStatus `json:"status"`        // 运行状态
    RestartCount int         `json:"restart_count"` // 重启次数
    LastRestart  time.Time   `json:"last_restart"`  // 上次重启时间
    
    // 时间戳
    CreatedAt time.Time `json:"created_at"` // 注册时间
    UpdatedAt time.Time `json:"updated_at"` // 最后更新时间
    
    // 并发保护
    mu sync.RWMutex `json:"-"`
}

// AgentStatus Agent运行状态常量
type AgentStatus string

const (
    StatusStopped    AgentStatus = "stopped"     // Agent已停止
    StatusStarting   AgentStatus = "starting"    // Agent正在启动中
    StatusRunning    AgentStatus = "running"     // Agent正在运行
    StatusStopping   AgentStatus = "stopping"    // Agent正在停止中
    StatusRestarting AgentStatus = "restarting"  // Agent正在重启中
    StatusFailed     AgentStatus = "failed"      // Agent启动失败或运行异常
)

// AgentType Agent类型常量
type AgentType string

const (
    TypeFilebeat     AgentType = "filebeat"      // Filebeat日志采集Agent
    TypeTelegraf     AgentType = "telegraf"      // Telegraf指标采集Agent
    TypeNodeExporter AgentType = "node_exporter" // Node Exporter指标采集Agent
    TypeCustom       AgentType = "custom"        // 自定义Agent类型
)
```

#### 4.1.2 AgentConfig - Agent 配置结构

```go
// AgentConfig Agent配置结构(从daemon.yaml加载)
type AgentConfig struct {
    ID          string            `yaml:"id"`
    Type        string            `yaml:"type"`
    Name        string            `yaml:"name"`
    BinaryPath  string            `yaml:"binary_path"`
    ConfigFile  string            `yaml:"config_file"`
    WorkDir     string            `yaml:"work_dir"`
    SocketPath  string            `yaml:"socket_path"`
    Enabled     bool              `yaml:"enabled"`
    Args        []string          `yaml:"args"`
    HealthCheck HealthCheckConfig `yaml:"health_check"`
    Restart     RestartConfig     `yaml:"restart"`
}

type HealthCheckConfig struct {
    Interval          time.Duration `yaml:"interval"`
    HeartbeatTimeout  time.Duration `yaml:"heartbeat_timeout"`
    CPUThreshold      float64       `yaml:"cpu_threshold"`
    MemoryThreshold   uint64        `yaml:"memory_threshold"`
    ThresholdDuration time.Duration `yaml:"threshold_duration"`
    HTTPEndpoint      string        `yaml:"http_endpoint"`
}

type RestartConfig struct {
    MaxRetries  int           `yaml:"max_retries"`
    BackoffBase time.Duration `yaml:"backoff_base"`
    BackoffMax  time.Duration `yaml:"backoff_max"`
    Policy      string        `yaml:"policy"` // always, on-failure, never
}
```

### 4.2 节点信息

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

### 4.3 Agent状态

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

### 4.4 上报数据结构

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
  address: "manager.example.com:9090"
  tls:
    cert_file: /etc/daemon/certs/client.crt
    key_file: /etc/daemon/certs/client.key
    ca_file: /etc/daemon/certs/ca.crt
  heartbeat_interval: 60s
  reconnect_interval: 10s
  timeout: 30s

# 多Agent管理配置(新设计)
agents:
  # 示例 1: Filebeat 日志采集 Agent
  - id: filebeat-logs
    type: filebeat
    name: "Filebeat Log Collector"
    binary_path: /usr/bin/filebeat
    config_file: /etc/filebeat/filebeat.yml
    work_dir: /var/lib/daemon/agents/filebeat-logs
    socket_path: ""
    enabled: true
    args:
      - "-c"
      - "/etc/filebeat/filebeat.yml"
      - "-path.home"
      - "/var/lib/daemon/agents/filebeat-logs"
    health_check:
      interval: 30s
      heartbeat_timeout: 90s
      cpu_threshold: 50.0
      memory_threshold: 524288000  # 500MB
      threshold_duration: 60s
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always

  # 示例 2: Telegraf 指标采集 Agent
  - id: telegraf-metrics
    type: telegraf
    name: "Telegraf Metrics Collector"
    binary_path: /usr/bin/telegraf
    config_file: /etc/telegraf/telegraf.conf
    work_dir: /var/lib/daemon/agents/telegraf-metrics
    socket_path: ""
    enabled: true
    args:
      - "-config"
      - "/etc/telegraf/telegraf.conf"
    health_check:
      interval: 30s
      heartbeat_timeout: 90s
      cpu_threshold: 40.0
      memory_threshold: 262144000  # 250MB
      threshold_duration: 60s
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always

  # 示例 3: Node Exporter 指标采集 Agent
  - id: node-exporter
    type: node_exporter
    name: "Node Exporter Metrics"
    binary_path: /usr/local/bin/node_exporter
    config_file: ""
    work_dir: /var/lib/daemon/agents/node-exporter
    socket_path: ""
    enabled: true
    args:
      - "--web.listen-address=:9100"
      - "--path.procfs=/proc"
      - "--path.sysfs=/sys"
      - "--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($|/)"
    health_check:
      interval: 30s
      heartbeat_timeout: 0s  # Node Exporter不使用心跳
      cpu_threshold: 30.0
      memory_threshold: 104857600  # 100MB
      threshold_duration: 60s
      http_endpoint: "http://localhost:9100/metrics"
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always

# 全局Agent默认配置(可选)
agent_defaults:
  health_check:
    interval: 30s
    heartbeat_timeout: 90s
    cpu_threshold: 50.0
    memory_threshold: 524288000
    threshold_duration: 60s
  restart:
    max_retries: 10
    backoff_base: 10s
    backoff_max: 60s
    policy: always

# 采集器配置(Daemon 自身的资源采集)
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

**配置项说明**:

**agents 数组配置**:
- `id`: Agent唯一标识符(必填)
- `type`: Agent类型,支持: filebeat, telegraf, node_exporter, custom
- `name`: Agent显示名称(可选)
- `binary_path`: Agent可执行文件路径(必填)
- `config_file`: Agent配置文件路径(可选,某些Agent不需要)
- `work_dir`: Agent工作目录(可选,默认: {daemon.work_dir}/agents/{id})
- `socket_path`: Unix Socket路径(可选)
- `enabled`: 是否启用,默认true
- `args`: 启动参数列表(可选,会覆盖默认参数)
- `health_check`: 健康检查配置(可选,继承全局默认值)
- `restart`: 重启策略配置(可选,继承全局默认值)

**向后兼容性**: 系统同时支持旧格式(单Agent)和新格式(多Agent),如果同时存在,优先使用新格式并发出警告。

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
│   ├── agent/                    # 多Agent管理模块
│   │   ├── registry.go           # Agent注册表
│   │   ├── multi_manager.go      # 多Agent管理器
│   │   ├── multi_health_checker.go  # 多Agent健康检查器
│   │   ├── instance.go           # Agent实例管理
│   │   ├── config_loader.go      # 配置加载器
│   │   ├── config_manager.go     # 配置管理器
│   │   ├── heartbeat_receiver.go # 心跳接收器
│   │   ├── resource_monitor.go   # 资源监控器
│   │   └── log_manager.go        # 日志管理器
│   ├── updater/
│   │   ├── updater.go
│   │   ├── downloader.go
│   │   ├── verifier.go
│   │   └── backuper.go
│   ├── comm/
│   │   ├── grpc.go
│   │   └── socket.go
│   ├── grpc/                     # gRPC服务
│   │   ├── client.go             # gRPC客户端
│   │   ├── server.go             # gRPC服务端
│   │   └── state_syncer.go       # 状态同步器
│   └── daemon/
│       ├── daemon.go
│       └── signal.go
├── pkg/
│   └── proto/
│       ├── daemon.proto
│       ├── daemon.pb.go
│       └── daemon_grpc.pb.go
├── configs/
│   ├── daemon.yaml              # 主配置文件
│   └── daemon.multi-agent.example.yaml  # 多Agent配置示例
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
