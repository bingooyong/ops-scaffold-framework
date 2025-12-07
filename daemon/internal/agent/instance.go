package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// AgentInstance Agent实例管理器
// 管理单个Agent进程的生命周期，与AgentInfo关联存储配置和状态
type AgentInstance struct {
	// info Agent信息，存储配置和状态
	info *AgentInfo

	// process Agent进程对象
	process *os.Process

	// logger 日志记录器
	logger *zap.Logger

	// logRotator 日志轮转器（可选）
	logRotator *LogRotator

	// mu 保护进程对象的并发访问锁
	mu sync.Mutex
}

// NewAgentInstance 创建新的Agent实例管理器
func NewAgentInstance(info *AgentInfo, logger *zap.Logger) *AgentInstance {
	return &AgentInstance{
		info:   info,
		logger: logger,
	}
}

// GetInfo 获取Agent信息
func (ai *AgentInstance) GetInfo() *AgentInfo {
	return ai.info
}

// Start 启动Agent进程
func (ai *AgentInstance) Start(ctx context.Context) error {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	// 检查是否已运行
	if ai.isRunningLocked() {
		pid := ai.info.GetPID()
		ai.logger.Info("agent already running",
			zap.String("agent_id", ai.info.ID),
			zap.String("agent_type", string(ai.info.Type)),
			zap.Int("pid", pid))
		return nil
	}

	// 更新状态为启动中
	ai.info.SetStatus(StatusStarting)

	// 生成启动参数
	args := ai.generateArgs()
	cmd := exec.CommandContext(ctx, ai.info.BinaryPath, args...)
	cmd.Dir = ai.info.WorkDir

	// 设置进程组，确保Agent独立运行
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	// 重定向输出到日志文件
	logFilePath := ai.getLogFilePath()

	// 确保日志目录存在
	logDir := fmt.Sprintf("%s/agents/%s/logs", ai.info.WorkDir, ai.info.ID)
	if ai.info.WorkDir == "" {
		logDir = fmt.Sprintf("/tmp/agents/%s/logs", ai.info.ID)
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		ai.info.SetStatus(StatusFailed)
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// 如果启用了日志轮转，检查是否需要轮转
	if ai.logRotator != nil {
		if err := ai.logRotator.RotateIfNeeded(); err != nil {
			ai.logger.Warn("failed to rotate log before start",
				zap.String("agent_id", ai.info.ID),
				zap.Error(err))
			// 不中断启动，继续执行
		}
	}

	logFile, err := os.OpenFile(
		logFilePath,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		ai.info.SetStatus(StatusFailed)
		return fmt.Errorf("failed to open log file: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// 启动定期轮转检查（如果启用了日志轮转）
	if ai.logRotator != nil {
		go ai.periodicRotateCheck()
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		logFile.Close()
		ai.info.SetStatus(StatusFailed)
		return fmt.Errorf("failed to start agent: %w", err)
	}

	ai.process = cmd.Process
	pid := cmd.Process.Pid
	ai.info.SetPID(pid)
	ai.info.SetStatus(StatusRunning)

	ai.logger.Info("agent started",
		zap.String("agent_id", ai.info.ID),
		zap.String("agent_type", string(ai.info.Type)),
		zap.String("agent_name", ai.info.Name),
		zap.Int("pid", pid),
		zap.String("binary", ai.info.BinaryPath),
		zap.Strings("args", args))

	// 在后台等待进程退出
	go func() {
		cmd.Wait()
		logFile.Close()

		// 更新状态
		ai.mu.Lock()
		ai.process = nil
		ai.info.SetPID(0)
		ai.info.SetStatus(StatusStopped)
		ai.mu.Unlock()

		ai.logger.Warn("agent process exited",
			zap.String("agent_id", ai.info.ID),
			zap.String("agent_type", string(ai.info.Type)),
			zap.Int("pid", pid))
	}()

	return nil
}

// periodicRotateCheck 定期检查日志轮转
func (ai *AgentInstance) periodicRotateCheck() {
	if ai.logRotator == nil {
		return
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !ai.IsRunning() {
				return
			}
			if err := ai.logRotator.RotateIfNeeded(); err != nil {
				ai.logger.Warn("failed to rotate log",
					zap.String("agent_id", ai.info.ID),
					zap.Error(err))
			}
		}
	}
}

// SetLogRotator 设置日志轮转器
func (ai *AgentInstance) SetLogRotator(rotator *LogRotator) {
	ai.logRotator = rotator
}

// Stop 停止Agent进程
func (ai *AgentInstance) Stop(ctx context.Context, graceful bool) error {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	if !ai.isRunningLocked() {
		return nil
	}

	pid := ai.info.GetPID()
	ai.logger.Info("stopping agent",
		zap.String("agent_id", ai.info.ID),
		zap.String("agent_type", string(ai.info.Type)),
		zap.Int("pid", pid),
		zap.Bool("graceful", graceful))

	// 更新状态为停止中
	ai.info.SetStatus(StatusStopping)

	if graceful {
		// 发送SIGTERM，等待优雅退出
		if err := ai.process.Signal(syscall.SIGTERM); err != nil {
			ai.info.SetStatus(StatusFailed)
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}

		// 等待最多30秒
		done := make(chan struct{})
		go func() {
			ai.process.Wait()
			close(done)
		}()

		select {
		case <-done:
			ai.logger.Info("agent stopped gracefully",
				zap.String("agent_id", ai.info.ID),
				zap.String("agent_type", string(ai.info.Type)))
		case <-time.After(30 * time.Second):
			ai.logger.Warn("agent graceful shutdown timeout, killing",
				zap.String("agent_id", ai.info.ID),
				zap.String("agent_type", string(ai.info.Type)))
			ai.process.Kill()
		case <-ctx.Done():
			ai.logger.Warn("context cancelled, killing agent",
				zap.String("agent_id", ai.info.ID),
				zap.String("agent_type", string(ai.info.Type)))
			ai.process.Kill()
		}
	} else {
		// 强制杀死
		if err := ai.process.Kill(); err != nil {
			ai.info.SetStatus(StatusFailed)
			return fmt.Errorf("failed to kill agent: %w", err)
		}
	}

	ai.process = nil
	ai.info.SetPID(0)
	ai.info.SetStatus(StatusStopped)

	return nil
}

// Restart 重启Agent
func (ai *AgentInstance) Restart(ctx context.Context) error {
	restartCount := ai.info.GetRestartCount()
	ai.logger.Info("restarting agent",
		zap.String("agent_id", ai.info.ID),
		zap.String("agent_type", string(ai.info.Type)),
		zap.Int("restart_count", restartCount))

	// 更新状态为重启中
	ai.info.SetStatus(StatusRestarting)

	// 计算退避时间
	backoff := ai.calculateBackoff()
	if backoff > 0 {
		ai.logger.Info("waiting before restart",
			zap.String("agent_id", ai.info.ID),
			zap.String("agent_type", string(ai.info.Type)),
			zap.Duration("backoff", backoff))
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			ai.info.SetStatus(StatusStopped)
			return ctx.Err()
		}
	}

	// 停止当前进程
	if err := ai.Stop(ctx, true); err != nil {
		ai.logger.Error("failed to stop agent before restart",
			zap.String("agent_id", ai.info.ID),
			zap.String("agent_type", string(ai.info.Type)),
			zap.Error(err))
		ai.info.SetStatus(StatusFailed)
	}

	// 启动新进程
	if err := ai.Start(ctx); err != nil {
		ai.info.SetStatus(StatusFailed)
		return fmt.Errorf("failed to start agent after restart: %w", err)
	}

	// 增加重启计数
	ai.info.IncrementRestartCount()

	ai.logger.Info("agent restarted",
		zap.String("agent_id", ai.info.ID),
		zap.String("agent_type", string(ai.info.Type)),
		zap.Int("restart_count", ai.info.GetRestartCount()))

	return nil
}

// IsRunning 检查Agent是否运行
func (ai *AgentInstance) IsRunning() bool {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	return ai.isRunningLocked()
}

// isRunningLocked 检查进程是否运行(需要持锁调用)
func (ai *AgentInstance) isRunningLocked() bool {
	if ai.process == nil {
		return false
	}
	// 发送信号0检查进程是否存在
	err := ai.process.Signal(syscall.Signal(0))
	return err == nil
}

// GetPID 获取Agent PID
func (ai *AgentInstance) GetPID() int {
	return ai.info.GetPID()
}

// GetRestartCount 获取重启次数
func (ai *AgentInstance) GetRestartCount() int {
	return ai.info.GetRestartCount()
}

// ResetRestartCount 重置重启计数
func (ai *AgentInstance) ResetRestartCount() {
	ai.info.ResetRestartCount()
}

// generateArgs 根据Agent类型生成启动参数
func (ai *AgentInstance) generateArgs() []string {
	switch ai.info.Type {
	case TypeFilebeat:
		// Filebeat: -c {config_file} -path.home {work_dir}
		args := []string{"-c", ai.info.ConfigFile}
		if ai.info.WorkDir != "" {
			args = append(args, "-path.home", ai.info.WorkDir)
		}
		return args

	case TypeTelegraf:
		// Telegraf: -config {config_file}
		if ai.info.ConfigFile != "" {
			return []string{"-config", ai.info.ConfigFile}
		}
		return []string{}

	case TypeNodeExporter:
		// Node Exporter: 使用默认命令行参数
		// 注意: Node Exporter 不使用配置文件，所有配置通过命令行参数
		args := []string{
			"--web.listen-address=:9100",
			"--path.procfs=/proc",
			"--path.sysfs=/sys",
		}
		return args

	case TypeCustom:
		// 自定义类型: 如果配置了ConfigFile，使用 -config 参数
		// 否则返回空参数列表（由调用者通过args字段提供）
		if ai.info.ConfigFile != "" {
			return []string{"-config", ai.info.ConfigFile}
		}
		return []string{}

	default:
		// 未知类型: 如果配置了ConfigFile，使用 -config 参数
		if ai.info.ConfigFile != "" {
			return []string{"-config", ai.info.ConfigFile}
		}
		return []string{}
	}
}

// getLogFilePath 获取日志文件路径
func (ai *AgentInstance) getLogFilePath() string {
	// 日志文件路径: {work_dir}/agents/{agent_id}/logs/agent.log
	if ai.info.WorkDir != "" {
		return fmt.Sprintf("%s/agents/%s/logs/agent.log", ai.info.WorkDir, ai.info.ID)
	}
	// 如果工作目录为空，使用临时目录
	return fmt.Sprintf("/tmp/agents/%s/logs/agent.log", ai.info.ID)
}

// calculateBackoff 计算退避时间
func (ai *AgentInstance) calculateBackoff() time.Duration {
	lastRestart := ai.info.GetLastRestart()
	restartCount := ai.info.GetRestartCount()

	// 如果距离上次重启超过5分钟，重置计数
	if !lastRestart.IsZero() && time.Since(lastRestart) > 5*time.Minute {
		ai.info.ResetRestartCount()
		return 0
	}

	switch {
	case restartCount < 1:
		return 0
	case restartCount < 3:
		return 10 * time.Second
	case restartCount < 5:
		return 30 * time.Second
	default:
		return 60 * time.Second
	}
}
