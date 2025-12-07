package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/agent/internal/api"
	"github.com/bingooyong/ops-scaffold-framework/agent/internal/config"
	"github.com/bingooyong/ops-scaffold-framework/agent/internal/heartbeat"
	"github.com/bingooyong/ops-scaffold-framework/agent/internal/logger"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "", "config file path")
	version    = "1.0.0" // TODO: 从 version 包获取
)

func main() {
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	log, err := logger.InitLogger(cfg.Log.Level, cfg.Log.File, cfg.Log.Format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("agent starting",
		zap.String("agent_id", cfg.AgentID),
		zap.String("version", version),
		zap.Int("pid", os.Getpid()))

	// 创建心跳管理器
	hbManager := heartbeat.NewManager(
		cfg.Heartbeat.SocketPath,
		cfg.Heartbeat.Interval,
		version,
		log,
	)

	// 创建 HTTP API 服务器
	apiServer := api.NewServer(
		cfg.HTTP.Host,
		cfg.HTTP.Port,
		cfg.AgentID,
		version,
		log,
	)

	// 集成心跳管理器与 API 服务器
	// 1. 设置资源使用获取器
	apiServer.SetResourceGetter(hbManager)

	// 2. 设置心跳状态回调
	hbManager.SetHeartbeatCallbacks(
		func() { apiServer.UpdateHeartbeatStatus(true) },
		func() { apiServer.UpdateHeartbeatStatus(false) },
	)

	// 3. 设置配置重载回调
	apiServer.SetReloadCallback(func() error {
		log.Info("reloading configuration...")
		// TODO: 实现配置重载逻辑
		// 1. 重新加载配置文件
		// 2. 验证新配置
		// 3. 更新心跳间隔等参数
		log.Warn("config reload not fully implemented yet")
		return nil
	})

	// 启动心跳管理器
	if err := hbManager.Start(); err != nil {
		log.Error("failed to start heartbeat manager", zap.Error(err))
		os.Exit(1)
	}

	// 启动 HTTP API 服务器
	if err := apiServer.Start(); err != nil {
		log.Error("failed to start HTTP server", zap.Error(err))
		os.Exit(1)
	}

	log.Info("agent started successfully",
		zap.String("agent_id", cfg.AgentID),
		zap.String("http_addr", fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)),
		zap.String("socket_path", cfg.Heartbeat.SocketPath))

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Info("received shutdown signal", zap.String("signal", sig.String()))

	// 优雅退出
	gracefulShutdown(hbManager, apiServer, log)

	log.Info("agent stopped")
}

// gracefulShutdown 优雅退出
func gracefulShutdown(hbManager *heartbeat.Manager, apiServer *api.Server, log *zap.Logger) {
	log.Info("shutting down gracefully")

	// 停止心跳管理器（会发送最后一次心跳）
	hbManager.Stop()

	// 停止 HTTP 服务器（设置超时）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := apiServer.Stop(ctx); err != nil {
		log.Error("failed to stop HTTP server gracefully", zap.Error(err))
	}

	log.Info("graceful shutdown completed")
}
