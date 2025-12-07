# ============================================
# Stage 1: Builder - 编译 Go 应用
# ============================================
FROM golang:1.24-bookworm AS builder

# 设置工作目录
WORKDIR /build

# 设置 Go 工具链自动下载
ENV GOTOOLCHAIN=auto

# 复制整个项目（因为有 replace 依赖到 manager 模块）
COPY . .

# 进入 daemon 目录
WORKDIR /build/daemon

# 下载依赖
RUN go mod download

# 编译二进制文件（启用 CGO 以支持 gopsutil 获取系统信息）
# 让构建器自动检测架构，避免交叉编译问题
RUN CGO_ENABLED=1 go build \
    -ldflags="-w -s" \
    -o /build/daemon-bin \
    ./cmd/daemon

# ============================================
# Stage 2: Runtime - 运行环境
# ============================================
FROM debian:bookworm-slim

# 安装运行时依赖（包含系统监控工具）
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    procps \
    sysstat && \
    rm -rf /var/lib/apt/lists/*

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 创建非 root 用户
RUN groupadd -g 1000 opsdaemon && \
    useradd -u 1000 -g opsdaemon -s /bin/bash -m opsdaemon

# 创建必要的目录
RUN mkdir -p /app/configs \
             /app/logs \
             /app/tmp \
             /etc/daemon/certs && \
    chown -R opsdaemon:opsdaemon /app /etc/daemon

# 设置工作目录
WORKDIR /app

# 从 builder 阶段复制编译好的二进制文件
COPY --from=builder /build/daemon-bin /app/daemon

# 复制配置文件
COPY --from=builder /build/daemon/configs/daemon.container.yaml /app/configs/daemon.yaml

# 切换到非 root 用户
USER opsdaemon

# 健康检查（可选，如果 daemon 提供健康检查接口）
# HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
#   CMD ["/app/daemon", "-health-check"] || exit 1

# 启动命令
ENTRYPOINT ["/app/daemon"]
CMD ["-config", "/app/configs/daemon.yaml"]
