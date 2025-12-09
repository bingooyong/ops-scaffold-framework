# Agent 管理配置示例

**版本**: v1.0  
**更新日期**: 2025-01-27  
**适用版本**: Ops Scaffold Framework v0.3.0+

---

## 目录

1. [常见 Agent 类型配置模板](#1-常见-agent-类型配置模板)
2. [资源限制建议](#2-资源限制建议)
3. [健康检查配置](#3-健康检查配置)
4. [多 Agent 部署最佳实践](#4-多-agent-部署最佳实践)
5. [安全配置建议](#5-安全配置建议)

---

## 1. 常见 Agent 类型配置模板

### 1.1 Filebeat 配置

#### 1.1.1 基础配置

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
      threshold_duration: 60s
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always
```

#### 1.1.2 生产环境配置

```yaml
agents:
  - id: filebeat-logs-prod
    type: filebeat
    name: "Filebeat Log Collector (Production)"
    binary_path: /usr/local/filebeat/filebeat
    config_file: /etc/filebeat/filebeat.prod.yml
    work_dir: /var/lib/daemon/agents/filebeat-logs-prod
    enabled: true
    args:
      - "-c"
      - "/etc/filebeat/filebeat.prod.yml"
      - "-path.home"
      - "/var/lib/daemon/agents/filebeat-logs-prod"
      - "-e"  # 输出到 stderr
    health_check:
      interval: 30s
      heartbeat_timeout: 90s
      cpu_threshold: 60.0  # 生产环境允许稍高的 CPU 使用
      memory_threshold: 1048576000  # 1GB
      threshold_duration: 120s  # 生产环境延长阈值持续时间
    restart:
      max_retries: 15
      backoff_base: 15s
      backoff_max: 120s
      policy: always
```

#### 1.1.3 多实例配置（多个 Filebeat 实例）

```yaml
agents:
  # Filebeat 实例 1: 应用日志
  - id: filebeat-app-logs
    type: filebeat
    name: "Filebeat Application Logs"
    binary_path: /usr/bin/filebeat
    config_file: /etc/filebeat/filebeat-app.yml
    work_dir: /var/lib/daemon/agents/filebeat-app-logs
    enabled: true
    args:
      - "-c"
      - "/etc/filebeat/filebeat-app.yml"
      - "-path.home"
      - "/var/lib/daemon/agents/filebeat-app-logs"
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

  # Filebeat 实例 2: 系统日志
  - id: filebeat-sys-logs
    type: filebeat
    name: "Filebeat System Logs"
    binary_path: /usr/bin/filebeat
    config_file: /etc/filebeat/filebeat-sys.yml
    work_dir: /var/lib/daemon/agents/filebeat-sys-logs
    enabled: true
    args:
      - "-c"
      - "/etc/filebeat/filebeat-sys.yml"
      - "-path.home"
      - "/var/lib/daemon/agents/filebeat-sys-logs"
    health_check:
      interval: 30s
      heartbeat_timeout: 90s
      cpu_threshold: 40.0
      memory_threshold: 262144000
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always
```

### 1.2 Telegraf 配置

#### 1.2.1 基础配置

```yaml
agents:
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
      heartbeat_timeout: 90s
      cpu_threshold: 40.0
      memory_threshold: 262144000
      threshold_duration: 60s
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always
```

#### 1.2.2 高负载配置

```yaml
agents:
  - id: telegraf-metrics-heavy
    type: telegraf
    name: "Telegraf Metrics Collector (Heavy Load)"
    binary_path: /usr/bin/telegraf
    config_file: /etc/telegraf/telegraf-heavy.conf
    work_dir: /var/lib/daemon/agents/telegraf-metrics-heavy
    enabled: true
    args:
      - "-config"
      - "/etc/telegraf/telegraf-heavy.conf"
      - "-config-directory"
      - "/etc/telegraf/telegraf.d"
    health_check:
      interval: 30s
      heartbeat_timeout: 90s
      cpu_threshold: 60.0  # 高负载场景允许更高的 CPU
      memory_threshold: 524288000  # 500MB
      threshold_duration: 120s
    restart:
      max_retries: 15
      backoff_base: 15s
      backoff_max: 120s
      policy: always
```

### 1.3 Node Exporter 配置

#### 1.3.1 基础配置

```yaml
agents:
  - id: node-exporter
    type: node_exporter
    name: "Node Exporter Metrics"
    binary_path: /usr/local/bin/node_exporter
    config_file: ""  # Node Exporter 不使用配置文件
    work_dir: /var/lib/daemon/agents/node-exporter
    enabled: true
    args:
      - "--web.listen-address=:9100"
      - "--path.procfs=/proc"
      - "--path.sysfs=/sys"
      - "--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($|/)"
    health_check:
      interval: 30s
      heartbeat_timeout: 0s  # 不使用心跳检查
      cpu_threshold: 30.0
      memory_threshold: 104857600
      threshold_duration: 60s
      http_endpoint: "http://localhost:9100/metrics"  # HTTP 健康检查
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always
```

#### 1.3.2 自定义采集器配置

```yaml
agents:
  - id: node-exporter-custom
    type: node_exporter
    name: "Node Exporter (Custom Collectors)"
    binary_path: /usr/local/bin/node_exporter
    config_file: ""
    work_dir: /var/lib/daemon/agents/node-exporter-custom
    enabled: true
    args:
      - "--web.listen-address=:9100"
      - "--path.procfs=/proc"
      - "--path.sysfs=/sys"
      - "--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($|/)"
      - "--collector.netdev.device-exclude=^(veth.*|docker.*|br-.*)$"
      - "--collector.textfile.directory=/var/lib/node_exporter/textfile_collector"
    health_check:
      interval: 30s
      heartbeat_timeout: 0s
      cpu_threshold: 30.0
      memory_threshold: 104857600
      threshold_duration: 60s
      http_endpoint: "http://localhost:9100/metrics"
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always
```

### 1.4 自定义 Agent 配置

#### 1.4.1 基础自定义 Agent

```yaml
agents:
  - id: custom-agent
    type: custom
    name: "Custom Application Agent"
    binary_path: /usr/local/bin/custom-agent
    config_file: /etc/custom-agent/config.json
    work_dir: /var/lib/daemon/agents/custom-agent
    socket_path: /var/run/custom-agent.sock
    enabled: true
    args:
      - "--config"
      - "/etc/custom-agent/config.json"
      - "--log-level"
      - "info"
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
```

#### 1.4.2 支持 HTTP 健康检查的自定义 Agent

```yaml
agents:
  - id: custom-agent-http
    type: custom
    name: "Custom Agent with HTTP Health Check"
    binary_path: /usr/local/bin/custom-agent
    config_file: /etc/custom-agent/config.json
    work_dir: /var/lib/daemon/agents/custom-agent-http
    enabled: true
    args:
      - "--config"
      - "/etc/custom-agent/config.json"
      - "--http-listen"
      - ":8080"
    health_check:
      interval: 30s
      heartbeat_timeout: 90s
      cpu_threshold: 50.0
      memory_threshold: 524288000
      threshold_duration: 60s
      http_endpoint: "http://localhost:8080/health"  # HTTP 健康检查端点
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always
```

### 1.5 混合部署配置（多个不同类型的 Agent）

```yaml
agents:
  # Filebeat: 日志采集
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

  # Telegraf: 指标采集
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
      heartbeat_timeout: 90s
      cpu_threshold: 40.0
      memory_threshold: 262144000
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always

  # Node Exporter: Prometheus 指标暴露
  - id: node-exporter
    type: node_exporter
    name: "Node Exporter Metrics"
    binary_path: /usr/local/bin/node_exporter
    config_file: ""
    work_dir: /var/lib/daemon/agents/node-exporter
    enabled: true
    args:
      - "--web.listen-address=:9100"
      - "--path.procfs=/proc"
      - "--path.sysfs=/sys"
    health_check:
      interval: 30s
      heartbeat_timeout: 0s
      cpu_threshold: 30.0
      memory_threshold: 104857600
      http_endpoint: "http://localhost:9100/metrics"
    restart:
      max_retries: 10
      backoff_base: 10s
      backoff_max: 60s
      policy: always
```

---

## 2. 资源限制建议

### 2.1 CPU 使用率限制建议

| Agent 类型 | 建议阈值 | 说明 |
|-----------|---------|------|
| Filebeat | 50% | 正常日志采集场景 |
| Filebeat (高负载) | 60-70% | 大量日志采集场景 |
| Telegraf | 40% | 正常指标采集场景 |
| Telegraf (高负载) | 60% | 大量指标采集场景 |
| Node Exporter | 30% | 系统指标暴露，资源占用很低 |
| Custom Agent | 50% | 根据实际业务调整 |

**配置示例**：
```yaml
health_check:
  cpu_threshold: 50.0  # CPU 使用率阈值（%）
  threshold_duration: 60s  # 超过阈值持续时间
```

### 2.2 内存使用限制建议

| Agent 类型 | 建议阈值 | 说明 |
|-----------|---------|------|
| Filebeat | 500MB (524288000 字节) | 正常日志采集场景 |
| Filebeat (高负载) | 1GB (1048576000 字节) | 大量日志采集场景 |
| Telegraf | 250MB (262144000 字节) | 正常指标采集场景 |
| Telegraf (高负载) | 500MB (524288000 字节) | 大量指标采集场景 |
| Node Exporter | 100MB (104857600 字节) | 系统指标暴露，资源占用很低 |
| Custom Agent | 500MB (524288000 字节) | 根据实际业务调整 |

**配置示例**：
```yaml
health_check:
  memory_threshold: 524288000  # 内存使用阈值（字节，500MB）
  threshold_duration: 60s
```

### 2.3 磁盘空间限制建议

虽然 Daemon 不直接限制磁盘空间，但建议：

1. **工作目录大小**
   - 每个 Agent 工作目录建议预留至少 1GB 空间
   - 定期清理旧日志和临时文件

2. **日志文件大小**
   - 启用日志轮转，单个日志文件不超过 100MB
   - 保留最近 10 个日志文件

**配置示例**（在 Agent 自身配置中）：
```yaml
# Filebeat 配置示例
filebeat.registry.flush: 5s
logging.level: info
logging.to_files: true
logging.files:
  path: /var/lib/daemon/agents/filebeat-logs
  name: filebeat
  keepfiles: 7
  permissions: 0644
```

### 2.4 文件描述符限制建议

某些 Agent（如 Filebeat）需要打开大量文件，建议：

1. **系统级别限制**
   ```bash
   # 查看当前限制
   ulimit -n
   
   # 设置限制（在 /etc/security/limits.conf）
   daemon soft nofile 65535
   daemon hard nofile 65535
   ```

2. **Agent 级别限制**
   - 在 Agent 配置中限制并发文件数
   - Filebeat: `filebeat.inputs[].prospector.scanner.max_files`

---

## 3. 健康检查配置

### 3.1 心跳超时时间设置

**建议值**：
- **默认**：90 秒（3 个心跳周期，假设心跳间隔 30 秒）
- **高延迟网络**：120-180 秒
- **本地网络**：60 秒

**配置示例**：
```yaml
health_check:
  heartbeat_timeout: 90s  # 心跳超时时间
```

**说明**：
- 如果 Agent 不支持心跳，设置 `heartbeat_timeout: 0s`
- 心跳超时后，Daemon 会判定 Agent 异常并触发自动重启

### 3.2 自动重启策略配置

#### 3.2.1 生产环境配置

```yaml
restart:
  max_retries: 15  # 生产环境允许更多重试
  backoff_base: 15s  # 基础退避时间稍长
  backoff_max: 120s  # 最大退避时间 2 分钟
  policy: always  # 总是重启，确保服务持续运行
```

#### 3.2.2 开发测试环境配置

```yaml
restart:
  max_retries: 5  # 开发环境减少重试次数
  backoff_base: 5s  # 基础退避时间较短
  backoff_max: 30s  # 最大退避时间 30 秒
  policy: on-failure  # 仅在失败时重启
```

#### 3.2.3 禁用自动重启

```yaml
restart:
  max_retries: 0
  policy: never  # 不自动重启，手动控制
```

### 3.3 最大重试次数设置

**建议值**：
- **生产环境**：10-15 次
- **开发测试环境**：5 次
- **临时禁用**：0 次（配合 `policy: never`）

**配置示例**：
```yaml
restart:
  max_retries: 10  # 最大重启次数
```

**说明**：
- 超过最大重试次数后，Daemon 不再自动重启
- 会触发告警，通知管理员手动处理

### 3.4 回退时间配置

**退避策略**：
- 第 1 次重启：立即执行
- 第 2-3 次重启：等待 `backoff_base`
- 第 4-5 次重启：等待 `backoff_base * 2`
- 第 6-N 次重启：等待 `backoff_max`

**配置示例**：
```yaml
restart:
  backoff_base: 10s  # 基础退避时间
  backoff_max: 60s   # 最大退避时间
```

**建议值**：
- **基础退避时间**：10-15 秒
- **最大退避时间**：60-120 秒

---

## 4. 多 Agent 部署最佳实践

### 4.1 如何规划 Agent 部署

#### 4.1.1 按功能分类部署

**日志采集类**：
- Filebeat（应用日志）
- Filebeat（系统日志）
- 其他日志采集 Agent

**指标采集类**：
- Telegraf（系统指标）
- Node Exporter（Prometheus 指标）
- 其他指标采集 Agent

**自定义 Agent**：
- 业务相关 Agent
- 监控 Agent
- 其他自定义 Agent

#### 4.1.2 按环境分类部署

**生产环境**：
- 启用所有必需的 Agent
- 使用生产级配置（更高的资源限制、更多的重试次数）
- 启用自动重启策略

**测试环境**：
- 启用部分 Agent 用于测试
- 使用测试级配置（较低的资源限制、较少的重试次数）
- 可以禁用自动重启，便于调试

**开发环境**：
- 仅启用必要的 Agent
- 使用开发级配置
- 禁用自动重启

### 4.2 如何避免资源冲突

#### 4.2.1 端口冲突

**问题**：多个 Agent 可能使用相同的端口。

**解决方法**：
1. **为每个 Agent 分配不同的端口**
   ```yaml
   # Node Exporter 实例 1
   - id: node-exporter-1
     args:
       - "--web.listen-address=:9100"
   
   # Node Exporter 实例 2
   - id: node-exporter-2
     args:
       - "--web.listen-address=:9101"
   ```

2. **使用配置文件指定端口**
   - 在 Agent 配置文件中指定不同的端口
   - 避免在启动参数中硬编码端口

#### 4.2.2 工作目录冲突

**问题**：多个 Agent 使用相同的工作目录可能导致文件冲突。

**解决方法**：
1. **为每个 Agent 使用独立的工作目录**
   ```yaml
   agents:
     - id: filebeat-app
       work_dir: /var/lib/daemon/agents/filebeat-app
     
     - id: filebeat-sys
       work_dir: /var/lib/daemon/agents/filebeat-sys
   ```

2. **使用 Agent ID 作为目录名**
   - 默认工作目录：`{daemon.work_dir}/agents/{agent_id}`
   - 确保每个 Agent 有唯一的 ID

#### 4.2.3 配置文件冲突

**问题**：多个 Agent 实例使用相同的配置文件。

**解决方法**：
1. **为每个 Agent 实例创建独立的配置文件**
   ```yaml
   agents:
     - id: filebeat-app
       config_file: /etc/filebeat/filebeat-app.yml
     
     - id: filebeat-sys
       config_file: /etc/filebeat/filebeat-sys.yml
   ```

2. **使用配置模板和环境变量**
   - 使用配置模板生成不同实例的配置文件
   - 使用环境变量区分不同实例

### 4.3 如何优化性能

#### 4.3.1 资源分配优化

1. **合理设置资源限制**
   - 根据 Agent 实际资源使用情况设置阈值
   - 避免设置过低的阈值导致频繁重启
   - 避免设置过高的阈值导致资源泄漏无法检测

2. **调整健康检查间隔**
   ```yaml
   health_check:
     interval: 30s  # 根据实际需求调整，30-60 秒之间
   ```

3. **优化启动参数**
   - 减少不必要的启动参数
   - 使用 Agent 推荐的启动参数

#### 4.3.2 日志优化

1. **启用日志轮转**
   - 限制单个日志文件大小
   - 限制保留的日志文件数量

2. **调整日志级别**
   - 生产环境使用 `info` 级别
   - 开发环境使用 `debug` 级别

3. **日志输出优化**
   - 使用结构化日志（JSON 格式）
   - 减少不必要的日志输出

### 4.4 如何实现高可用

#### 4.4.1 Agent 高可用

1. **启用自动重启**
   ```yaml
   restart:
     policy: always
     max_retries: 15
   ```

2. **配置合理的健康检查**
   ```yaml
   health_check:
     interval: 30s
     heartbeat_timeout: 90s
   ```

3. **监控和告警**
   - 配置告警规则，及时发现 Agent 异常
   - 设置告警通知，确保管理员及时处理

#### 4.4.2 Daemon 高可用

1. **Daemon 进程监控**
   - 使用 systemd 管理 Daemon 进程
   - 配置 systemd 自动重启

2. **Daemon 与 Manager 连接监控**
   - 监控 Daemon 与 Manager 的连接状态
   - 配置连接断开自动重连

3. **数据持久化**
   - Agent 状态数据持久化到文件
   - Daemon 重启后可以恢复 Agent 状态

---

## 5. 安全配置建议

### 5.1 文件权限设置

#### 5.1.1 Agent 二进制文件权限

```bash
# 设置正确的文件权限
sudo chmod 755 /usr/bin/filebeat
sudo chown root:root /usr/bin/filebeat

# 验证权限
ls -l /usr/bin/filebeat
# 输出: -rwxr-xr-x 1 root root ...
```

#### 5.1.2 配置文件权限

```bash
# 设置配置文件权限（仅 root 可写，其他用户只读）
sudo chmod 644 /etc/filebeat/filebeat.yml
sudo chown root:root /etc/filebeat/filebeat.yml

# 验证权限
ls -l /etc/filebeat/filebeat.yml
# 输出: -rw-r--r-- 1 root root ...
```

#### 5.1.3 工作目录权限

```bash
# 创建工作目录
sudo mkdir -p /var/lib/daemon/agents/filebeat-logs

# 设置目录权限（daemon 用户可读写，其他用户无权限）
sudo chmod 750 /var/lib/daemon/agents/filebeat-logs
sudo chown daemon:daemon /var/lib/daemon/agents/filebeat-logs

# 验证权限
ls -ld /var/lib/daemon/agents/filebeat-logs
# 输出: drwxr-x--- 1 daemon daemon ...
```

#### 5.1.4 Socket 文件权限

```bash
# Socket 文件由 Daemon 自动创建，确保目录权限正确
sudo chmod 750 /var/run/daemon/agents
sudo chown daemon:daemon /var/run/daemon/agents
```

### 5.2 网络访问控制

#### 5.2.1 Agent HTTP API 访问控制

如果 Agent 提供 HTTP API，建议：

1. **仅监听本地地址**
   ```yaml
   # Agent 配置中
   http:
     listen: "127.0.0.1:8080"  # 仅本地访问
   ```

2. **使用防火墙规则**
   ```bash
   # 仅允许本地访问
   sudo iptables -A INPUT -p tcp --dport 8080 -s 127.0.0.1 -j ACCEPT
   sudo iptables -A INPUT -p tcp --dport 8080 -j DROP
   ```

3. **使用反向代理**
   - 通过 Nginx 等反向代理访问 Agent API
   - 在反向代理层配置访问控制

#### 5.2.2 Daemon 与 Manager 通信安全

1. **使用 TLS 加密**
   ```yaml
   manager:
     tls:
       cert_file: /etc/daemon/certs/client.crt
       key_file: /etc/daemon/certs/client.key
       ca_file: /etc/daemon/certs/ca.crt
   ```

2. **使用 IP 白名单**（如果支持）
   - 在 Manager 配置 IP 白名单
   - 仅允许指定的 Daemon IP 连接

### 5.3 认证和授权配置

#### 5.3.1 Manager API 认证

Manager API 使用 JWT 认证：

1. **获取 Token**
   ```bash
   curl -X POST "http://manager:8080/api/v1/auth/login" \
     -H "Content-Type: application/json" \
     -d '{
       "username": "admin",
       "password": "password"
     }'
   ```

2. **使用 Token**
   ```bash
   curl -X GET "http://manager:8080/api/v1/nodes" \
     -H "Authorization: Bearer <token>"
   ```

#### 5.3.2 Agent API 认证（如果支持）

如果 Agent 提供 HTTP API，建议：

1. **使用 API Key 认证**
   ```yaml
   # Agent 配置中
   http:
     api_key: "your-api-key-here"
   ```

2. **使用 HMAC 签名认证**
   - 请求签名验证
   - 防止请求被篡改

### 5.4 日志脱敏建议

#### 5.4.1 配置文件中的敏感信息

1. **使用环境变量**
   ```yaml
   # 不直接在配置文件中写密码
   # password: "secret123"  # ❌ 错误
   
   # 使用环境变量
   password: "${DB_PASSWORD}"  # ✅ 正确
   ```

2. **使用密钥管理服务**
   - 使用 Vault、AWS Secrets Manager 等密钥管理服务
   - 在运行时从密钥管理服务获取敏感信息

#### 5.4.2 日志中的敏感信息

1. **配置日志脱敏规则**
   ```yaml
   # Agent 配置中
   logging:
     redact:
       - "password"
       - "api_key"
       - "secret"
   ```

2. **避免记录完整配置**
   - 日志中不记录完整的配置文件内容
   - 仅记录配置变更的关键信息

#### 5.4.3 内存中的敏感信息

1. **及时清除敏感数据**
   - 使用完敏感数据后立即清除
   - 避免敏感数据在内存中长时间驻留

2. **使用安全的内存管理**
   - 使用加密的内存区域存储敏感数据
   - 使用安全的内存分配器

---

## 附录

### A. 配置文件完整示例

完整的配置文件示例请参考：
- `daemon/configs/daemon.multi-agent.example.yaml`

### B. 相关文档

- [Agent 管理功能使用指南](./Agent管理功能使用指南.md)
- [Agent 管理管理员手册](./Agent管理管理员手册.md)
- [Agent 管理开发者文档](./Agent管理开发者文档.md)
- [Daemon 多 Agent 管理架构设计](./设计文档_04_Daemon多Agent管理架构.md)

### C. 配置验证工具

使用以下工具验证配置文件：

1. **YAML 语法验证**
   ```bash
   yamllint daemon.yaml
   ```

2. **配置加载测试**
   ```bash
   # Daemon 启动时会自动验证配置
   daemon --config daemon.yaml --validate
   ```

---

**文档版本**: v1.0  
**最后更新**: 2025-01-27  
**维护者**: Ops Scaffold Framework Team
