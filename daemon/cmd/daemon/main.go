package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/daemon"
	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/logger"
	"go.uber.org/zap"
)

var (
	configFile = flag.String("config", "/etc/daemon/daemon.yaml", "path to config file")
	version    = flag.Bool("version", false, "print version and exit")
)

func main() {
	flag.Parse()

	// 打印版本
	if *version {
		fmt.Println("Daemon v1.0.0")
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.Load(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logCfg := &logger.Config{
		Level:    cfg.Daemon.LogLevel,
		FilePath: cfg.Daemon.LogFile,
	}
	if err := logger.Init(logCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync() // 忽略日志同步错误，程序退出时无法处理
	}()

	logger.Info("starting daemon")

	// 创建Daemon实例
	d, err := daemon.New(cfg, logger.Logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create daemon: %v\n", err)
		logger.Error("failed to create daemon", zap.Error(err))
		os.Exit(1)
	}

	// 启动Daemon
	if err := d.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start daemon: %v\n", err)
		logger.Error("failed to start daemon", zap.Error(err))
		os.Exit(1)
	}

	// 等待退出信号
	d.WaitForSignal()

	logger.Info("daemon exited")
}
