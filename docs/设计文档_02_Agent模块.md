# Agent模块设计文档

**版本：** 1.0  
**日期：** 2025-12-01  
**模块编号：** MOD-AGENT

---

## 目录

1. [模块概述](#1-模块概述)
2. [架构设计](#2-架构设计)
3. [详细设计](#3-详细设计)
4. [数据结构](#4-数据结构)
5. [API设计](#5-api设计)
6. [插件系统](#6-插件系统)
7. [安全设计](#7-安全设计)
8. [配置管理](#8-配置管理)
9. [错误处理](#9-错误处理)
10. [测试设计](#10-测试设计)

---

## 1. 模块概述

### 1.1 模块职责

Agent是运行在每台被管主机上的任务执行进程，主要职责包括：

- 接收并执行运维任务（脚本执行、文件操作、服务管理等）
- 提供HTTP/HTTPS API服务供Manager/Daemon调用
- 向Daemon上报心跳和自身状态
- 支持任务队列和并发控制
- 支持插件扩展

### 1.2 设计目标

| 目标 | 指标 |
|------|------|
| 轻量级 | CPU < 1%（空闲），内存 < 50MB |
| 高性能 | API响应 < 100ms，支持10并发任务 |
| 安全性 | 请求签名验证，IP白名单 |
| 可扩展 | 插件化任务执行器 |
| 独立性 | 不依赖Daemon运行 |

### 1.3 运行模式

```
┌─────────────────────────────────────────────────────────────┐
│                      Agent 运行模式                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  模式1: 标准模式 (推荐)                                      │
│  ┌─────────┐  心跳上报  ┌─────────┐  任务调用  ┌─────────┐ │
│  │  Agent  │◄─────────►│ Daemon  │◄──────────│ Manager │ │
│  └─────────┘            └─────────┘            └─────────┘ │
│                                                             │
│  模式2: 直连模式 (网络隔离场景)                              │
│  ┌─────────┐  心跳上报  ┌─────────┐                        │
│  │  Agent  │◄─────────►│ Daemon  │                        │
│  └────┬────┘            └─────────┘                        │
│       │ 任务调用                                            │
│  ┌────▼────┐                                                │
│  │ Manager │                                                │
│  └─────────┘                                                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. 架构设计

### 2.1 整体架构

```
┌────────────────────────────────────────────────────────────────┐
│                         Agent Process                           │
├────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                     HTTP/HTTPS Server                     │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐     │  │
│  │  │ Router  │  │  Auth   │  │  Rate   │  │ Handler │     │  │
│  │  │         │  │Middleware│ │ Limiter │  │         │     │  │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘     │  │
│  └──────────────────────────────────────────────────────────┘  │
├────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                     Task Engine                           │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐     │  │
│  │  │  Queue  │  │Scheduler│  │ Worker  │  │ Result  │     │  │
│  │  │ Manager │  │         │  │  Pool   │  │ Store   │     │  │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘     │  │
│  └──────────────────────────────────────────────────────────┘  │
├────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐              │
│  │   Script    │ │    File     │ │   Service   │              │
│  │  Executor   │ │  Executor   │ │  Executor   │              │
│  ├─────────────┤ ├─────────────┤ ├─────────────┤              │
│  │ • Shell     │ │ • Upload    │ │ • Start     │              │
│  │ • Python    │ │ • Download  │ │ • Stop      │              │
│  │ • PowerShell│ │ • Copy      │ │ • Restart   │              │
│  │ • [Plugin]  │ │ • Delete    │ │ • Status    │              │
│  └─────────────┘ └─────────────┘ └─────────────┘              │
├────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   Heartbeat Module                        │  │
│  │  ┌───────────────┐  ┌───────────────┐                    │  │
│  │  │ Status Collect│  │ Socket Client │                    │  │
│  │  │               │  │ (to Daemon)   │                    │  │
│  │  └───────────────┘  └───────────────┘                    │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
```

### 2.2 模块划分

| 模块 | 职责 | 关键组件 |
|------|------|----------|
| HTTP Server | HTTP/HTTPS服务 | Router, AuthMiddleware, RateLimiter, Handlers |
| Task Engine | 任务调度执行 | QueueManager, Scheduler, WorkerPool, ResultStore |
| Executors | 任务执行器 | ScriptExecutor, FileExecutor, ServiceExecutor |
| Heartbeat | 心跳上报 | StatusCollector, SocketClient |
| Plugin | 插件管理 | PluginLoader, PluginRegistry |

### 2.3 线程模型

```
Main Goroutine
    │
    ├── HTTP Server Goroutine
    │       └── 处理HTTP请求（Gin框架管理）
    │
    ├── Task Scheduler Goroutine
    │       └── 从队列取任务分发到Worker
    │
    ├── Worker Pool (N个 Goroutine)
    │       └── 并发执行任务
    │
    ├── Heartbeat Goroutine
    │       └── 每30秒上报心跳
    │
    ├── Result Cleaner Goroutine
    │       └── 定期清理过期结果
    │
    └── Signal Handler Goroutine
            └── 处理退出信号
```

---

## 3. 详细设计

### 3.1 HTTP服务模块

#### 3.1.1 路由设计

```go
func SetupRouter(engine *gin.Engine, handlers *Handlers) {
    // 健康检查（无需认证）
    engine.GET("/api/v1/health", handlers.Health)
    
    // 需要认证的接口
    auth := engine.Group("/api/v1")
    auth.Use(AuthMiddleware())
    auth.Use(RateLimitMiddleware())
    {
        // 状态接口
        auth.GET("/status", handlers.GetStatus)
        
        // 任务接口
        auth.POST("/task/execute", handlers.ExecuteTask)
        auth.GET("/task/:id/status", handlers.GetTaskStatus)
        auth.POST("/task/:id/cancel", handlers.CancelTask)
        auth.GET("/tasks", handlers.ListTasks)
        
        // 文件接口
        auth.POST("/file/upload", handlers.UploadFile)
        auth.GET("/file/download", handlers.DownloadFile)
        auth.DELETE("/file", handlers.DeleteFile)
        
        // 服务管理接口
        auth.POST("/service/:name/start", handlers.StartService)
        auth.POST("/service/:name/stop", handlers.StopService)
        auth.POST("/service/:name/restart", handlers.RestartService)
        auth.GET("/service/:name/status", handlers.ServiceStatus)
    }
}
```

#### 3.1.2 认证中间件

```go
type AuthConfig struct {
    SecretKey    string
    IPWhitelist  []string
    TokenExpiry  time.Duration
}

func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. IP白名单检查
        clientIP := c.ClientIP()
        if !isIPAllowed(clientIP) {
            c.JSON(http.StatusForbidden, Response{
                Code:    1002,
                Message: "IP not allowed",
            })
            c.Abort()
            return
        }
        
        // 2. 签名验证
        timestamp := c.GetHeader("X-Timestamp")
        nonce := c.GetHeader("X-Nonce")
        signature := c.GetHeader("X-Signature")
        
        if !verifySignature(c.Request, timestamp, nonce, signature) {
            c.JSON(http.StatusUnauthorized, Response{
                Code:    1003,
                Message: "Invalid signature",
            })
            c.Abort()
            return
        }
        
        // 3. 时间戳检查（防重放）
        ts, _ := strconv.ParseInt(timestamp, 10, 64)
        if time.Now().Unix()-ts > 300 { // 5分钟有效期
            c.JSON(http.StatusUnauthorized, Response{
                Code:    1004,
                Message: "Request expired",
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}

func verifySignature(req *http.Request, timestamp, nonce, signature string) bool {
    // 构建签名字符串
    // StringToSign = Method + "\n" + Path + "\n" + Timestamp + "\n" + Nonce + "\n" + Body
    body, _ := io.ReadAll(req.Body)
    req.Body = io.NopCloser(bytes.NewBuffer(body))
    
    stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
        req.Method, req.URL.Path, timestamp, nonce, string(body))
    
    // HMAC-SHA256签名
    h := hmac.New(sha256.New, []byte(config.SecretKey))
    h.Write([]byte(stringToSign))
    expectedSig := base64.StdEncoding.EncodeToString(h.Sum(nil))
    
    return hmac.Equal([]byte(signature), []byte(expectedSig))
}
```

#### 3.1.3 限流中间件

```go
type RateLimiter struct {
    limiter *rate.Limiter
    mu      sync.Mutex
    clients map[string]*rate.Limiter
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
    return &RateLimiter{
        limiter: rate.NewLimiter(r, b),
        clients: make(map[string]*rate.Limiter),
    }
}

func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    if limiter, exists := rl.clients[key]; exists {
        return limiter
    }
    
    limiter := rate.NewLimiter(100, 10) // 每秒100请求，突发10
    rl.clients[key] = limiter
    return limiter
}

func RateLimitMiddleware() gin.HandlerFunc {
    rl := NewRateLimiter(100, 10)
    
    return func(c *gin.Context) {
        limiter := rl.GetLimiter(c.ClientIP())
        if !limiter.Allow() {
            c.JSON(http.StatusTooManyRequests, Response{
                Code:    1005,
                Message: "Rate limit exceeded",
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 3.2 任务引擎模块

#### 3.2.1 任务队列管理器

```go
type QueueManager struct {
    queues   map[string]*TaskQueue
    mu       sync.RWMutex
    maxSize  int
    taskCh   chan *Task
}

type TaskQueue struct {
    name     string
    tasks    []*Task
    mu       sync.Mutex
    priority int
}

type Task struct {
    ID          string                 `json:"id"`
    Type        string                 `json:"type"`        // script, file, service
    Action      string                 `json:"action"`      // execute, upload, start, etc.
    Params      map[string]interface{} `json:"params"`
    Timeout     time.Duration          `json:"timeout"`
    Priority    int                    `json:"priority"`
    Status      TaskStatus             `json:"status"`
    Result      *TaskResult            `json:"result,omitempty"`
    CreatedAt   time.Time              `json:"created_at"`
    StartedAt   *time.Time             `json:"started_at,omitempty"`
    FinishedAt  *time.Time             `json:"finished_at,omitempty"`
    CancelFunc  context.CancelFunc     `json:"-"`
}

type TaskStatus string

const (
    TaskStatusPending   TaskStatus = "pending"
    TaskStatusRunning   TaskStatus = "running"
    TaskStatusCompleted TaskStatus = "completed"
    TaskStatusFailed    TaskStatus = "failed"
    TaskStatusCancelled TaskStatus = "cancelled"
    TaskStatusTimeout   TaskStatus = "timeout"
)

func (qm *QueueManager) Enqueue(task *Task) error {
    qm.mu.Lock()
    defer qm.mu.Unlock()
    
    queue, exists := qm.queues[task.Type]
    if !exists {
        queue = &TaskQueue{name: task.Type}
        qm.queues[task.Type] = queue
    }
    
    queue.mu.Lock()
    defer queue.mu.Unlock()
    
    if len(queue.tasks) >= qm.maxSize {
        return fmt.Errorf("queue full")
    }
    
    task.Status = TaskStatusPending
    task.CreatedAt = time.Now()
    queue.tasks = append(queue.tasks, task)
    
    // 按优先级排序
    sort.Slice(queue.tasks, func(i, j int) bool {
        return queue.tasks[i].Priority > queue.tasks[j].Priority
    })
    
    // 通知调度器
    select {
    case qm.taskCh <- task:
    default:
    }
    
    return nil
}

func (qm *QueueManager) Dequeue() *Task {
    qm.mu.RLock()
    defer qm.mu.RUnlock()
    
    // 优先级队列取任务
    var highestPriority *Task
    var highestQueue *TaskQueue
    
    for _, queue := range qm.queues {
        queue.mu.Lock()
        if len(queue.tasks) > 0 {
            task := queue.tasks[0]
            if highestPriority == nil || task.Priority > highestPriority.Priority {
                highestPriority = task
                highestQueue = queue
            }
        }
        queue.mu.Unlock()
    }
    
    if highestQueue != nil {
        highestQueue.mu.Lock()
        highestQueue.tasks = highestQueue.tasks[1:]
        highestQueue.mu.Unlock()
    }
    
    return highestPriority
}
```

#### 3.2.2 Worker Pool

```go
type WorkerPool struct {
    size        int
    taskCh      chan *Task
    resultCh    chan *TaskResult
    executors   map[string]Executor
    wg          sync.WaitGroup
}

func NewWorkerPool(size int, executors map[string]Executor) *WorkerPool {
    return &WorkerPool{
        size:      size,
        taskCh:    make(chan *Task, size*2),
        resultCh:  make(chan *TaskResult, size*2),
        executors: executors,
    }
}

func (wp *WorkerPool) Start(ctx context.Context) {
    for i := 0; i < wp.size; i++ {
        wp.wg.Add(1)
        go wp.worker(ctx, i)
    }
}

func (wp *WorkerPool) worker(ctx context.Context, id int) {
    defer wp.wg.Done()
    
    for {
        select {
        case <-ctx.Done():
            return
        case task := <-wp.taskCh:
            result := wp.executeTask(ctx, task)
            wp.resultCh <- result
        }
    }
}

func (wp *WorkerPool) executeTask(ctx context.Context, task *Task) *TaskResult {
    result := &TaskResult{
        TaskID:    task.ID,
        StartedAt: time.Now(),
    }
    
    // 设置超时
    taskCtx, cancel := context.WithTimeout(ctx, task.Timeout)
    task.CancelFunc = cancel
    defer cancel()
    
    // 更新状态
    now := time.Now()
    task.StartedAt = &now
    task.Status = TaskStatusRunning
    
    // 获取执行器
    executor, exists := wp.executors[task.Type]
    if !exists {
        result.Success = false
        result.Error = fmt.Sprintf("unknown task type: %s", task.Type)
        task.Status = TaskStatusFailed
        return result
    }
    
    // 执行任务
    output, err := executor.Execute(taskCtx, task)
    
    finishedAt := time.Now()
    task.FinishedAt = &finishedAt
    result.FinishedAt = finishedAt
    result.Output = output
    
    if err != nil {
        result.Success = false
        result.Error = err.Error()
        if taskCtx.Err() == context.DeadlineExceeded {
            task.Status = TaskStatusTimeout
        } else if taskCtx.Err() == context.Canceled {
            task.Status = TaskStatusCancelled
        } else {
            task.Status = TaskStatusFailed
        }
    } else {
        result.Success = true
        task.Status = TaskStatusCompleted
    }
    
    return result
}

func (wp *WorkerPool) Submit(task *Task) {
    wp.taskCh <- task
}

func (wp *WorkerPool) Stop() {
    close(wp.taskCh)
    wp.wg.Wait()
}
```

#### 3.2.3 结果存储

```go
type ResultStore struct {
    results map[string]*TaskResult
    mu      sync.RWMutex
    ttl     time.Duration
}

type TaskResult struct {
    TaskID     string      `json:"task_id"`
    Success    bool        `json:"success"`
    Output     interface{} `json:"output,omitempty"`
    Error      string      `json:"error,omitempty"`
    ExitCode   int         `json:"exit_code,omitempty"`
    StartedAt  time.Time   `json:"started_at"`
    FinishedAt time.Time   `json:"finished_at"`
}

func NewResultStore(ttl time.Duration) *ResultStore {
    rs := &ResultStore{
        results: make(map[string]*TaskResult),
        ttl:     ttl,
    }
    go rs.cleanup()
    return rs
}

func (rs *ResultStore) Store(result *TaskResult) {
    rs.mu.Lock()
    defer rs.mu.Unlock()
    rs.results[result.TaskID] = result
}

func (rs *ResultStore) Get(taskID string) (*TaskResult, bool) {
    rs.mu.RLock()
    defer rs.mu.RUnlock()
    result, exists := rs.results[taskID]
    return result, exists
}

func (rs *ResultStore) cleanup() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        rs.mu.Lock()
        now := time.Now()
        for id, result := range rs.results {
            if now.Sub(result.FinishedAt) > rs.ttl {
                delete(rs.results, id)
            }
        }
        rs.mu.Unlock()
    }
}
```

### 3.3 执行器模块

#### 3.3.1 执行器接口

```go
type Executor interface {
    // Name 返回执行器名称
    Name() string
    // Execute 执行任务
    Execute(ctx context.Context, task *Task) (interface{}, error)
    // Validate 验证任务参数
    Validate(task *Task) error
}

// ExecutorRegistry 执行器注册表
type ExecutorRegistry struct {
    executors map[string]Executor
    mu        sync.RWMutex
}

func (r *ExecutorRegistry) Register(executor Executor) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.executors[executor.Name()] = executor
}

func (r *ExecutorRegistry) Get(name string) (Executor, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    e, ok := r.executors[name]
    return e, ok
}
```

#### 3.3.2 脚本执行器

```go
type ScriptExecutor struct {
    workDir     string
    env         []string
    maxOutput   int64
    interpreters map[string]string
}

type ScriptParams struct {
    Script      string            `json:"script"`       // 脚本内容
    ScriptType  string            `json:"script_type"`  // shell, python, powershell
    Args        []string          `json:"args"`
    Env         map[string]string `json:"env"`
    WorkDir     string            `json:"work_dir"`
    User        string            `json:"user"`
}

func (se *ScriptExecutor) Name() string {
    return "script"
}

func (se *ScriptExecutor) Execute(ctx context.Context, task *Task) (interface{}, error) {
    params := &ScriptParams{}
    if err := mapstructure.Decode(task.Params, params); err != nil {
        return nil, fmt.Errorf("invalid params: %w", err)
    }
    
    // 获取解释器
    interpreter, ok := se.interpreters[params.ScriptType]
    if !ok {
        return nil, fmt.Errorf("unsupported script type: %s", params.ScriptType)
    }
    
    // 创建临时脚本文件
    tmpFile, err := se.createScriptFile(params)
    if err != nil {
        return nil, err
    }
    defer os.Remove(tmpFile)
    
    // 构建命令
    args := append([]string{tmpFile}, params.Args...)
    cmd := exec.CommandContext(ctx, interpreter, args...)
    
    // 设置工作目录
    if params.WorkDir != "" {
        cmd.Dir = params.WorkDir
    } else {
        cmd.Dir = se.workDir
    }
    
    // 设置环境变量
    cmd.Env = append(os.Environ(), se.env...)
    for k, v := range params.Env {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }
    
    // 捕获输出
    var stdout, stderr bytes.Buffer
    cmd.Stdout = io.LimitReader(&stdout, se.maxOutput)
    cmd.Stderr = io.LimitReader(&stderr, se.maxOutput)
    
    // 执行
    err = cmd.Run()
    
    output := &ScriptOutput{
        Stdout:   stdout.String(),
        Stderr:   stderr.String(),
        ExitCode: 0,
    }
    
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            output.ExitCode = exitErr.ExitCode()
        } else {
            return output, err
        }
    }
    
    return output, nil
}

type ScriptOutput struct {
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"`
    ExitCode int    `json:"exit_code"`
}

func (se *ScriptExecutor) createScriptFile(params *ScriptParams) (string, error) {
    ext := ".sh"
    switch params.ScriptType {
    case "python":
        ext = ".py"
    case "powershell":
        ext = ".ps1"
    }
    
    tmpFile, err := os.CreateTemp("", "script-*"+ext)
    if err != nil {
        return "", err
    }
    
    if _, err := tmpFile.WriteString(params.Script); err != nil {
        tmpFile.Close()
        os.Remove(tmpFile.Name())
        return "", err
    }
    
    tmpFile.Close()
    os.Chmod(tmpFile.Name(), 0755)
    
    return tmpFile.Name(), nil
}
```

#### 3.3.3 文件执行器

```go
type FileExecutor struct {
    baseDir   string
    maxSize   int64
    allowDirs []string
}

type FileParams struct {
    Action   string `json:"action"`    // upload, download, copy, delete
    Path     string `json:"path"`
    DestPath string `json:"dest_path"` // for copy
    Content  string `json:"content"`   // base64 encoded for upload
    Mode     int    `json:"mode"`
}

func (fe *FileExecutor) Name() string {
    return "file"
}

func (fe *FileExecutor) Execute(ctx context.Context, task *Task) (interface{}, error) {
    params := &FileParams{}
    if err := mapstructure.Decode(task.Params, params); err != nil {
        return nil, fmt.Errorf("invalid params: %w", err)
    }
    
    // 路径安全检查
    if !fe.isPathAllowed(params.Path) {
        return nil, fmt.Errorf("path not allowed: %s", params.Path)
    }
    
    switch params.Action {
    case "upload":
        return fe.upload(ctx, params)
    case "download":
        return fe.download(ctx, params)
    case "copy":
        return fe.copy(ctx, params)
    case "delete":
        return fe.delete(ctx, params)
    case "stat":
        return fe.stat(ctx, params)
    default:
        return nil, fmt.Errorf("unknown action: %s", params.Action)
    }
}

func (fe *FileExecutor) upload(ctx context.Context, params *FileParams) (interface{}, error) {
    content, err := base64.StdEncoding.DecodeString(params.Content)
    if err != nil {
        return nil, fmt.Errorf("decode content failed: %w", err)
    }
    
    if int64(len(content)) > fe.maxSize {
        return nil, fmt.Errorf("file too large: %d > %d", len(content), fe.maxSize)
    }
    
    // 确保目录存在
    dir := filepath.Dir(params.Path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, err
    }
    
    mode := os.FileMode(0644)
    if params.Mode != 0 {
        mode = os.FileMode(params.Mode)
    }
    
    if err := os.WriteFile(params.Path, content, mode); err != nil {
        return nil, err
    }
    
    return map[string]interface{}{
        "path": params.Path,
        "size": len(content),
    }, nil
}

func (fe *FileExecutor) download(ctx context.Context, params *FileParams) (interface{}, error) {
    info, err := os.Stat(params.Path)
    if err != nil {
        return nil, err
    }
    
    if info.Size() > fe.maxSize {
        return nil, fmt.Errorf("file too large: %d > %d", info.Size(), fe.maxSize)
    }
    
    content, err := os.ReadFile(params.Path)
    if err != nil {
        return nil, err
    }
    
    return map[string]interface{}{
        "path":    params.Path,
        "content": base64.StdEncoding.EncodeToString(content),
        "size":    len(content),
    }, nil
}

func (fe *FileExecutor) copy(ctx context.Context, params *FileParams) (interface{}, error) {
    if !fe.isPathAllowed(params.DestPath) {
        return nil, fmt.Errorf("dest path not allowed: %s", params.DestPath)
    }
    
    src, err := os.Open(params.Path)
    if err != nil {
        return nil, err
    }
    defer src.Close()
    
    // 确保目标目录存在
    dir := filepath.Dir(params.DestPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, err
    }
    
    dst, err := os.Create(params.DestPath)
    if err != nil {
        return nil, err
    }
    defer dst.Close()
    
    n, err := io.Copy(dst, src)
    if err != nil {
        return nil, err
    }
    
    return map[string]interface{}{
        "src":  params.Path,
        "dest": params.DestPath,
        "size": n,
    }, nil
}

func (fe *FileExecutor) delete(ctx context.Context, params *FileParams) (interface{}, error) {
    if err := os.RemoveAll(params.Path); err != nil {
        return nil, err
    }
    
    return map[string]interface{}{
        "path":    params.Path,
        "deleted": true,
    }, nil
}

func (fe *FileExecutor) stat(ctx context.Context, params *FileParams) (interface{}, error) {
    info, err := os.Stat(params.Path)
    if err != nil {
        return nil, err
    }
    
    return map[string]interface{}{
        "path":     params.Path,
        "size":     info.Size(),
        "mode":     info.Mode().String(),
        "mod_time": info.ModTime(),
        "is_dir":   info.IsDir(),
    }, nil
}

func (fe *FileExecutor) isPathAllowed(path string) bool {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return false
    }
    
    for _, allowed := range fe.allowDirs {
        if strings.HasPrefix(absPath, allowed) {
            return true
        }
    }
    return false
}
```

#### 3.3.4 服务执行器

```go
type ServiceExecutor struct {
    serviceManager ServiceManager
}

type ServiceManager interface {
    Start(name string) error
    Stop(name string) error
    Restart(name string) error
    Status(name string) (*ServiceStatus, error)
    Enable(name string) error
    Disable(name string) error
}

type ServiceStatus struct {
    Name      string `json:"name"`
    Status    string `json:"status"` // running, stopped, unknown
    PID       int    `json:"pid,omitempty"`
    StartTime string `json:"start_time,omitempty"`
    Enabled   bool   `json:"enabled"`
}

type ServiceParams struct {
    Action string `json:"action"` // start, stop, restart, status, enable, disable
    Name   string `json:"name"`
}

func (se *ServiceExecutor) Name() string {
    return "service"
}

func (se *ServiceExecutor) Execute(ctx context.Context, task *Task) (interface{}, error) {
    params := &ServiceParams{}
    if err := mapstructure.Decode(task.Params, params); err != nil {
        return nil, fmt.Errorf("invalid params: %w", err)
    }
    
    switch params.Action {
    case "start":
        return nil, se.serviceManager.Start(params.Name)
    case "stop":
        return nil, se.serviceManager.Stop(params.Name)
    case "restart":
        return nil, se.serviceManager.Restart(params.Name)
    case "status":
        return se.serviceManager.Status(params.Name)
    case "enable":
        return nil, se.serviceManager.Enable(params.Name)
    case "disable":
        return nil, se.serviceManager.Disable(params.Name)
    default:
        return nil, fmt.Errorf("unknown action: %s", params.Action)
    }
}

// SystemdManager Linux systemd实现
type SystemdManager struct{}

func (sm *SystemdManager) Start(name string) error {
    return exec.Command("systemctl", "start", name).Run()
}

func (sm *SystemdManager) Stop(name string) error {
    return exec.Command("systemctl", "stop", name).Run()
}

func (sm *SystemdManager) Restart(name string) error {
    return exec.Command("systemctl", "restart", name).Run()
}

func (sm *SystemdManager) Status(name string) (*ServiceStatus, error) {
    status := &ServiceStatus{Name: name}
    
    // 检查运行状态
    cmd := exec.Command("systemctl", "is-active", name)
    output, _ := cmd.Output()
    status.Status = strings.TrimSpace(string(output))
    
    // 检查启用状态
    cmd = exec.Command("systemctl", "is-enabled", name)
    output, _ = cmd.Output()
    status.Enabled = strings.TrimSpace(string(output)) == "enabled"
    
    // 获取PID
    cmd = exec.Command("systemctl", "show", name, "--property=MainPID")
    output, _ = cmd.Output()
    if parts := strings.Split(string(output), "="); len(parts) == 2 {
        status.PID, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
    }
    
    return status, nil
}
```

### 3.4 心跳模块

```go
type HeartbeatSender struct {
    socketPath string
    interval   time.Duration
    conn       net.Conn
    mu         sync.Mutex
}

func (hs *HeartbeatSender) Start(ctx context.Context) {
    ticker := time.NewTicker(hs.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := hs.sendHeartbeat(); err != nil {
                log.Warn("send heartbeat failed", zap.Error(err))
                hs.reconnect()
            }
        }
    }
}

func (hs *HeartbeatSender) sendHeartbeat() error {
    hs.mu.Lock()
    defer hs.mu.Unlock()
    
    if hs.conn == nil {
        if err := hs.connect(); err != nil {
            return err
        }
    }
    
    // 收集状态
    status := hs.collectStatus()
    
    // 发送心跳
    data, _ := json.Marshal(status)
    hs.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
    _, err := hs.conn.Write(append(data, '\n'))
    return err
}

func (hs *HeartbeatSender) collectStatus() *Heartbeat {
    proc, _ := process.NewProcess(int32(os.Getpid()))
    cpu, _ := proc.CPUPercent()
    mem, _ := proc.MemoryInfo()
    
    return &Heartbeat{
        PID:       os.Getpid(),
        Timestamp: time.Now(),
        Version:   Version,
        Status:    "running",
        CPU:       cpu,
        Memory:    mem.RSS,
    }
}

func (hs *HeartbeatSender) connect() error {
    conn, err := net.Dial("unix", hs.socketPath)
    if err != nil {
        return err
    }
    hs.conn = conn
    return nil
}

func (hs *HeartbeatSender) reconnect() {
    hs.mu.Lock()
    defer hs.mu.Unlock()
    
    if hs.conn != nil {
        hs.conn.Close()
        hs.conn = nil
    }
}
```

---

## 4. 数据结构

### 4.1 请求/响应结构

```go
// 通用响应
type Response struct {
    Code      int         `json:"code"`
    Message   string      `json:"message"`
    Data      interface{} `json:"data,omitempty"`
    Timestamp time.Time   `json:"timestamp"`
}

// 任务执行请求
type ExecuteRequest struct {
    Type     string                 `json:"type" binding:"required"`
    Action   string                 `json:"action" binding:"required"`
    Params   map[string]interface{} `json:"params"`
    Timeout  int                    `json:"timeout"`  // 秒
    Priority int                    `json:"priority"` // 0-10
    Async    bool                   `json:"async"`    // 是否异步执行
}

// 任务执行响应
type ExecuteResponse struct {
    TaskID string      `json:"task_id"`
    Status TaskStatus  `json:"status"`
    Result interface{} `json:"result,omitempty"`
}

// Agent状态
type AgentStatus struct {
    Version    string    `json:"version"`
    Uptime     int64     `json:"uptime"`      // 秒
    StartTime  time.Time `json:"start_time"`
    CPU        float64   `json:"cpu_percent"`
    Memory     uint64    `json:"memory_bytes"`
    Goroutines int       `json:"goroutines"`
    Tasks      struct {
        Pending   int `json:"pending"`
        Running   int `json:"running"`
        Completed int `json:"completed"`
        Failed    int `json:"failed"`
    } `json:"tasks"`
}
```

---

## 5. API设计

### 5.1 API列表

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | /api/v1/health | 健康检查 | 否 |
| GET | /api/v1/status | Agent状态 | 是 |
| POST | /api/v1/task/execute | 执行任务 | 是 |
| GET | /api/v1/task/:id/status | 查询任务状态 | 是 |
| POST | /api/v1/task/:id/cancel | 取消任务 | 是 |
| GET | /api/v1/tasks | 任务列表 | 是 |
| POST | /api/v1/file/upload | 上传文件 | 是 |
| GET | /api/v1/file/download | 下载文件 | 是 |
| DELETE | /api/v1/file | 删除文件 | 是 |
| POST | /api/v1/service/:name/start | 启动服务 | 是 |
| POST | /api/v1/service/:name/stop | 停止服务 | 是 |
| POST | /api/v1/service/:name/restart | 重启服务 | 是 |
| GET | /api/v1/service/:name/status | 服务状态 | 是 |

### 5.2 API示例

#### 执行脚本

```http
POST /api/v1/task/execute HTTP/1.1
Host: agent:8080
Content-Type: application/json
X-Timestamp: 1701388800
X-Nonce: abc123
X-Signature: base64_signature

{
    "type": "script",
    "action": "execute",
    "params": {
        "script": "#!/bin/bash\necho 'Hello World'",
        "script_type": "shell",
        "args": [],
        "timeout": 60
    },
    "timeout": 120,
    "async": false
}
```

响应：
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "task_id": "task-123456",
        "status": "completed",
        "result": {
            "stdout": "Hello World\n",
            "stderr": "",
            "exit_code": 0
        }
    },
    "timestamp": "2025-12-01T10:00:00Z"
}
```

---

## 6. 插件系统

### 6.1 插件接口

```go
// Plugin 插件接口
type Plugin interface {
    // Name 插件名称
    Name() string
    // Version 插件版本
    Version() string
    // Init 初始化插件
    Init(config map[string]interface{}) error
    // GetExecutor 获取执行器
    GetExecutor() Executor
    // Shutdown 关闭插件
    Shutdown() error
}

// PluginLoader 插件加载器
type PluginLoader struct {
    pluginDir string
    plugins   map[string]Plugin
    mu        sync.RWMutex
}

func (pl *PluginLoader) Load(name string) error {
    pluginPath := filepath.Join(pl.pluginDir, name+".so")
    
    p, err := plugin.Open(pluginPath)
    if err != nil {
        return fmt.Errorf("open plugin failed: %w", err)
    }
    
    sym, err := p.Lookup("NewPlugin")
    if err != nil {
        return fmt.Errorf("lookup NewPlugin failed: %w", err)
    }
    
    newPlugin, ok := sym.(func() Plugin)
    if !ok {
        return fmt.Errorf("invalid plugin interface")
    }
    
    plug := newPlugin()
    
    pl.mu.Lock()
    pl.plugins[plug.Name()] = plug
    pl.mu.Unlock()
    
    return plug.Init(nil)
}
```

### 6.2 插件示例

```go
// 自定义插件示例：数据库执行器
package main

type MySQLPlugin struct {
    db *sql.DB
}

func NewPlugin() Plugin {
    return &MySQLPlugin{}
}

func (p *MySQLPlugin) Name() string {
    return "mysql"
}

func (p *MySQLPlugin) Version() string {
    return "1.0.0"
}

func (p *MySQLPlugin) Init(config map[string]interface{}) error {
    dsn := config["dsn"].(string)
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return err
    }
    p.db = db
    return nil
}

func (p *MySQLPlugin) GetExecutor() Executor {
    return &MySQLExecutor{db: p.db}
}

type MySQLExecutor struct {
    db *sql.DB
}

func (e *MySQLExecutor) Name() string {
    return "mysql"
}

func (e *MySQLExecutor) Execute(ctx context.Context, task *Task) (interface{}, error) {
    query := task.Params["query"].(string)
    rows, err := e.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    // ... 处理结果
    return results, nil
}
```

---

## 7. 安全设计

### 7.1 请求签名

```
签名算法: HMAC-SHA256
签名字符串格式:
    StringToSign = Method + "\n" + Path + "\n" + Timestamp + "\n" + Nonce + "\n" + Body

请求头:
    X-Timestamp: Unix时间戳（秒）
    X-Nonce: 随机字符串（防重放）
    X-Signature: Base64(HMAC-SHA256(StringToSign, SecretKey))

验证规则:
    1. 时间戳在5分钟内有效
    2. Nonce不能重复（缓存1小时）
    3. 签名匹配
```

### 7.2 IP白名单

```yaml
security:
  ip_whitelist:
    enabled: true
    ips:
      - "192.168.1.0/24"    # Manager网段
      - "127.0.0.1"         # 本地
      - "::1"               # IPv6本地
```

### 7.3 路径安全

```go
// 文件操作路径限制
file:
  allowed_dirs:
    - "/var/lib/agent"
    - "/tmp/agent"
  denied_patterns:
    - "/etc/passwd"
    - "/etc/shadow"
    - "*.key"
    - "*.pem"
```

---

## 8. 配置管理

### 8.1 配置文件

```yaml
# agent.yaml

# 基础配置
agent:
  id: ""  # 留空则自动生成
  version: "1.0.0"
  log_level: info
  log_file: /var/log/agent/agent.log
  pid_file: /var/run/agent.pid
  work_dir: /var/lib/agent

# HTTP服务配置
server:
  address: "0.0.0.0:8080"
  tls:
    enabled: true
    cert_file: /etc/agent/certs/server.crt
    key_file: /etc/agent/certs/server.key
  read_timeout: 30s
  write_timeout: 30s
  max_body_size: 104857600  # 100MB

# 安全配置
security:
  secret_key: "your-secret-key"
  ip_whitelist:
    enabled: true
    ips:
      - "192.168.1.0/24"
  rate_limit:
    enabled: true
    requests_per_second: 100
    burst: 10

# Daemon连接配置
daemon:
  socket_path: /var/run/agent.sock
  heartbeat_interval: 30s

# 任务配置
task:
  queue_size: 100
  worker_count: 10
  default_timeout: 300s
  result_ttl: 1h

# 执行器配置
executors:
  script:
    enabled: true
    work_dir: /var/lib/agent/scripts
    max_output: 10485760  # 10MB
    interpreters:
      shell: /bin/bash
      python: /usr/bin/python3
      powershell: /usr/bin/pwsh
  file:
    enabled: true
    max_size: 104857600  # 100MB
    allowed_dirs:
      - /var/lib/agent
      - /tmp/agent
  service:
    enabled: true

# 插件配置
plugins:
  dir: /var/lib/agent/plugins
  autoload: []
```

---

## 9. 错误处理

### 9.1 错误码定义

| 错误码 | 名称 | 说明 |
|--------|------|------|
| A1001 | InvalidParams | 参数无效 |
| A1002 | Unauthorized | 认证失败 |
| A1003 | Forbidden | 权限不足 |
| A1004 | RateLimited | 请求过于频繁 |
| A2001 | TaskNotFound | 任务不存在 |
| A2002 | TaskTimeout | 任务超时 |
| A2003 | TaskFailed | 任务执行失败 |
| A2004 | TaskCancelled | 任务已取消 |
| A2005 | QueueFull | 任务队列已满 |
| A3001 | FileNotFound | 文件不存在 |
| A3002 | FileTooLarge | 文件过大 |
| A3003 | PathNotAllowed | 路径不允许 |
| A4001 | ServiceNotFound | 服务不存在 |
| A4002 | ServiceError | 服务操作失败 |

---

## 10. 测试设计

### 10.1 单元测试

```go
func TestScriptExecutor_Execute(t *testing.T) {
    executor := &ScriptExecutor{
        workDir: "/tmp",
        interpreters: map[string]string{
            "shell": "/bin/bash",
        },
    }
    
    task := &Task{
        ID:   "test-1",
        Type: "script",
        Params: map[string]interface{}{
            "script":      "echo 'hello'",
            "script_type": "shell",
        },
        Timeout: 10 * time.Second,
    }
    
    result, err := executor.Execute(context.Background(), task)
    assert.NoError(t, err)
    
    output := result.(*ScriptOutput)
    assert.Equal(t, "hello\n", output.Stdout)
    assert.Equal(t, 0, output.ExitCode)
}
```

### 10.2 集成测试

```go
func TestTaskExecution_Integration(t *testing.T) {
    // 启动Agent
    // 发送任务请求
    // 验证任务执行结果
    // 验证状态查询
    // 测试任务取消
}
```

### 10.3 性能测试

| 测试场景 | 预期指标 |
|----------|----------|
| API响应时间 | < 100ms (P99) |
| 并发任务数 | 10个 |
| 任务队列容量 | 100个 |
| 文件传输速度 | > 10MB/s |

---

## 附录

### A. 目录结构

```
agent/
├── cmd/
│   └── agent/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── server/
│   │   ├── server.go
│   │   ├── router.go
│   │   ├── middleware.go
│   │   └── handlers.go
│   ├── task/
│   │   ├── queue.go
│   │   ├── worker.go
│   │   └── result.go
│   ├── executor/
│   │   ├── executor.go
│   │   ├── script.go
│   │   ├── file.go
│   │   └── service.go
│   ├── heartbeat/
│   │   └── sender.go
│   └── plugin/
│       └── loader.go
├── pkg/
│   └── types/
│       └── types.go
├── plugins/
│   └── mysql/
│       └── mysql.go
├── configs/
│   └── agent.yaml
└── scripts/
    └── install.sh
```

---

*— 文档结束 —*
