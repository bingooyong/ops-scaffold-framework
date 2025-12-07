# Daemon Podman 容器部署指南

本文档说明如何使用 Podman 和 podman-compose 部署多个 daemon 节点，模拟分布式环境。

## 📋 前置要求

1. 安装 Podman
```bash
# macOS
brew install podman

# Linux (Fedora/RHEL/CentOS)
sudo dnf install podman

# Linux (Debian/Ubuntu)
sudo apt install podman
```

2. 安装 podman-compose
```bash
pip3 install podman-compose
```

3. 启动 Podman Machine（仅 macOS/Windows）
```bash
podman machine init
podman machine start
```

## 🚀 快速启动

### 1. 构建镜像并启动所有服务

在项目根目录执行：

```bash
# 构建并启动所有 daemon 节点
podman-compose -f podman-compose.yml up -d --build

# 查看运行状态
podman-compose -f podman-compose.yml ps

# 查看日志
podman-compose -f podman-compose.yml logs -f
```

### 2. 查看特定节点日志

```bash
# 查看 node1 的日志
podman-compose -f podman-compose.yml logs -f daemon-node1

# 查看 node2 的日志
podman-compose -f podman-compose.yml logs -f daemon-node2

# 查看 node3 的日志
podman-compose -f podman-compose.yml logs -f daemon-node3
```

### 3. 停止和清理

```bash
# 停止所有服务
podman-compose -f podman-compose.yml down

# 停止并删除卷（包括日志和数据）
podman-compose -f podman-compose.yml down -v

# 删除镜像
podman rmi ops-scaffold-framework-daemon-node1
```

## 🔧 配置说明

### Dockerfile 说明

位置：`daemon/Dockerfile`

- **多阶段构建**：减小最终镜像大小
- **非 root 用户**：提高安全性
- **时区设置**：默认使用 Asia/Shanghai
- **日志和数据目录**：挂载到容器卷

### 容器配置文件

位置：`daemon/configs/daemon.container.yaml`

关键配置：
- Manager 地址：`manager:9090`（通过容器网络通信）
- 采集间隔：10秒（适合测试）
- TLS：默认禁用（可通过挂载证书启用）

### podman-compose.yml 说明

位置：项目根目录 `podman-compose.yml`

#### 网络
- `ops-network`：自定义桥接网络，所有服务通过此网络通信

#### 卷
每个节点有独立的卷：
- `daemon-nodeX-logs`：日志文件
- `daemon-nodeX-data`：临时数据和 node_id

#### 环境变量
- `NODE_NAME`：节点名称标识
- `MANAGER_ADDRESS`：Manager 服务地址
- `LOG_LEVEL`：日志级别（debug/info/warn/error）

## 📊 连接到 Manager

### 选项 1：Manager 运行在宿主机

如果 manager 在宿主机上运行（例如 localhost:9090），需要修改配置：

1. 编辑 `daemon/configs/daemon.container.yaml`：
```yaml
manager:
  address: "host.containers.internal:9090"
```

2. 或者在 `podman-compose.yml` 中添加 extra_hosts：
```yaml
services:
  daemon-node1:
    extra_hosts:
      - "manager:host-gateway"
```

### 选项 2：Manager 在同一 compose 中

取消 `podman-compose.yml` 中 manager 服务的注释：

```yaml
services:
  manager:
    build:
      context: .
      dockerfile: manager/Dockerfile
    # ... 其他配置
```

并取消 daemon 服务中的 depends_on 注释。

### 选项 3：Manager 在独立网络中

如果 manager 在其他 podman 网络中：

```bash
# 将 daemon 连接到 manager 所在的网络
podman network connect <manager-network> ops-daemon-node1
```

## 🔍 监控和调试

### 进入容器查看

```bash
# 进入 node1 容器
podman exec -it ops-daemon-node1 /bin/sh

# 查看进程
podman exec ops-daemon-node1 ps aux

# 查看日志文件
podman exec ops-daemon-node1 cat /app/logs/daemon.log
```

### 查看资源使用

```bash
# 查看所有容器资源使用
podman stats

# 查看特定容器
podman stats ops-daemon-node1
```

### 查看网络

```bash
# 查看网络列表
podman network ls

# 查看 ops-network 详情
podman network inspect ops-network

# 查看容器 IP
podman inspect ops-daemon-node1 | grep IPAddress
```

## 📈 扩展节点数量

### 方法 1：修改 podman-compose.yml

复制现有节点配置，修改节点编号：

```yaml
daemon-node4:
  build:
    context: .
    dockerfile: daemon/Dockerfile
  container_name: ops-daemon-node4
  hostname: daemon-node4
  networks:
    - ops-network
  environment:
    - NODE_NAME=daemon-node4
    - MANAGER_ADDRESS=manager:9090
  volumes:
    - daemon-node4-logs:/app/logs
    - daemon-node4-data:/app/tmp
  restart: unless-stopped
```

### 方法 2：使用 podman run 手动启动

```bash
# 启动第 4 个节点
podman run -d \
  --name ops-daemon-node4 \
  --hostname daemon-node4 \
  --network ops-network \
  -e NODE_NAME=daemon-node4 \
  -e MANAGER_ADDRESS=manager:9090 \
  -v daemon-node4-logs:/app/logs \
  -v daemon-node4-data:/app/tmp \
  --restart unless-stopped \
  ops-scaffold-framework-daemon-node1
```

## 🛠️ 故障排查

### 问题 1：容器无法连接到 manager

**症状**：日志中显示连接失败

**解决方案**：
1. 检查 manager 是否运行：`podman ps | grep manager`
2. 检查网络连接：`podman network inspect ops-network`
3. 测试连接：`podman exec ops-daemon-node1 ping manager`

### 问题 2：编译失败

**症状**：构建镜像时 Go 编译失败

**解决方案**：
1. 确保 go.mod 依赖正确
2. 检查 replace 指令是否正确
3. 清理缓存重新构建：`podman-compose build --no-cache`

### 问题 3：权限问题

**症状**：无法写入日志或数据文件

**解决方案**：
1. 检查卷权限：`podman volume inspect daemon-node1-logs`
2. 容器内使用非 root 用户（daemon:daemon），确保卷权限正确

## 📝 最佳实践

1. **日志管理**：定期清理日志卷，避免占用过多磁盘空间
2. **资源限制**：为容器设置 CPU 和内存限制
3. **健康检查**：实现健康检查接口，启用 HEALTHCHECK
4. **监控集成**：将容器日志导出到中心化日志系统

## 🔐 安全建议

1. 启用 TLS 通信（生产环境必须）
2. 使用密钥管理（Podman secrets）
3. 定期更新基础镜像
4. 扫描镜像漏洞：`podman scan <image>`

## 📚 相关资源

- [Podman 官方文档](https://docs.podman.io/)
- [podman-compose GitHub](https://github.com/containers/podman-compose)
- [Dockerfile 最佳实践](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)

## 💡 提示

- 使用 `podman-compose` 命令时，可以简写为 `podman compose`（Podman 4.0+）
- 所有 Docker Compose 文件基本兼容 Podman
- macOS/Windows 上 Podman 通过虚拟机运行，性能略低于 Linux 原生
