package daemon

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
)

// startMultiAgentMode 启动多Agent模式
func (d *Daemon) startMultiAgentMode() error {
	d.logger.Info("starting all agents", zap.Int("count", d.multiAgentManager.Count()))

	// 创建工作目录
	agentsWorkDir := fmt.Sprintf("%s/agents", d.config.Daemon.WorkDir)
	if err := os.MkdirAll(agentsWorkDir, 0755); err != nil {
		d.logger.Error("failed to create agents work directory", zap.Error(err))
	}

	// 启动所有Agent
	results := d.multiAgentManager.StartAll(d.ctx)
	successCount := 0
	for agentID, err := range results {
		if err != nil {
			d.logger.Error("failed to start agent",
				zap.String("agent_id", agentID),
				zap.Error(err))
		} else {
			successCount++
		}
	}

	d.logger.Info("agents started",
		zap.Int("total", len(results)),
		zap.Int("success", successCount),
		zap.Int("failed", len(results)-successCount))

	// 启动多Agent健康检查器
	if d.multiHealthChecker != nil {
		d.multiHealthChecker.Start()
		d.logger.Info("multi-agent health checker started")
	}

	// 启动HTTP服务器(用于接收Agent心跳，仅在配置了HTTP端口时启动)
	d.startHTTPHeartbeatReceiver()

	// 启动 Unix Socket 心跳接收器（如果配置了，用于向后兼容）
	d.startUnixSocketHeartbeatReceiver()

	// 启动资源监控器
	if d.resourceMonitor != nil {
		d.resourceMonitor.Start()
		d.logger.Info("resource monitor started")
	}

	// 启动日志清理任务
	if d.logManager != nil {
		d.logManager.StartCleanupTask()
		d.logger.Info("log manager started")
	}

	return nil
}

// startSingleAgentMode 启动单Agent模式（向后兼容）
func (d *Daemon) startSingleAgentMode() error {
	if err := d.agentManager.Start(d.ctx); err != nil {
		d.logger.Error("failed to start agent", zap.Error(err))
		// 不中断启动，继续运行采集器
		return nil
	}

	// 启动心跳接收器
	if d.heartbeatReceiver != nil {
		if err := d.heartbeatReceiver.Start(); err != nil {
			d.logger.Error("failed to start heartbeat receiver", zap.Error(err))
		}
	}

	// 启动健康检查
	if d.healthChecker != nil {
		d.healthChecker.Start()
	}

	return nil
}

// startHTTPHeartbeatReceiver 启动HTTP心跳接收器
func (d *Daemon) startHTTPHeartbeatReceiver() {
	if d.httpHeartbeatReceiver == nil {
		return
	}

	if d.config.Daemon.HTTPPort > 0 {
		if err := d.startHTTPServer(); err != nil {
			d.logger.Error("failed to start HTTP server", zap.Error(err))
			// 不中断启动，继续运行
		} else {
			d.logger.Info("HTTP heartbeat receiver started", zap.Int("port", d.config.Daemon.HTTPPort))
		}
	} else {
		d.logger.Info("HTTP heartbeat receiver disabled (http_port not configured, using Unix Socket only)")
	}
}

// startUnixSocketHeartbeatReceiver 启动Unix Socket心跳接收器
func (d *Daemon) startUnixSocketHeartbeatReceiver() {
	if d.heartbeatReceiver == nil {
		return
	}

	// 注意：在多Agent模式下，不启动对应的 HealthChecker（健康检查由 MultiHealthChecker 管理）
	if err := d.heartbeatReceiver.Start(); err != nil {
		d.logger.Error("failed to start Unix Socket heartbeat receiver", zap.Error(err))
		// 不中断启动，继续运行
	} else {
		d.logger.Info("Unix Socket heartbeat receiver started (backward compatibility mode)")
	}
}

// connectToManager 连接到Manager并启动相关服务
func (d *Daemon) connectToManager() error {
	if d.config.Manager.Address == "" {
		d.logger.Info("manager connection disabled (no address configured)")
		return nil
	}

	// 连接Manager并注册
	if err := d.connectAndRegister(); err != nil {
		d.logger.Error("failed to connect to manager", zap.Error(err))
		// 不中断启动，后台会重试连接
	}

	// 连接ManagerClient(用于上报Agent状态)
	if d.managerClient == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()

	if err := d.managerClient.Connect(ctx); err != nil {
		d.logger.Error("failed to connect manager client for agent state sync", zap.Error(err))
		// 不中断启动，StateSyncer会在下次同步时重试
		return nil
	}

	// 启动StateSyncer
	if d.stateSyncer != nil {
		d.stateSyncer.Start(d.nodeID)
		d.logger.Info("state syncer started")
	}

	return nil
}
