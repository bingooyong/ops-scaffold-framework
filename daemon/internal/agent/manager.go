package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/config"
	"go.uber.org/zap"
)

// Manager Agent进程管理器
type Manager struct {
	config       *config.AgentConfig
	process      *os.Process
	pid          int
	restartCount int
	lastRestart  time.Time
	mu           sync.Mutex
	logger       *zap.Logger
}

// NewManager 创建Agent进程管理器
func NewManager(cfg *config.AgentConfig, logger *zap.Logger) *Manager {
	return &Manager{
		config: cfg,
		logger: logger,
	}
}

// Start 启动Agent进程
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已运行
	if m.isRunningLocked() {
		m.logger.Info("agent already running", zap.Int("pid", m.pid))
		return nil
	}

	// 构建启动命令
	args := []string{"-config", m.config.ConfigFile}
	cmd := exec.CommandContext(ctx, m.config.BinaryPath, args...)
	cmd.Dir = m.config.WorkDir

	// 设置进程组，确保Agent独立运行
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	// 重定向输出到日志
	logFile, err := os.OpenFile(
		fmt.Sprintf("%s/agent.log", m.config.WorkDir),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// 启动进程
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start agent: %w", err)
	}

	m.process = cmd.Process
	m.pid = cmd.Process.Pid

	m.logger.Info("agent started",
		zap.Int("pid", m.pid),
		zap.String("binary", m.config.BinaryPath))

	// 在后台等待进程退出
	go func() {
		cmd.Wait()
		logFile.Close()
		m.logger.Warn("agent process exited", zap.Int("pid", m.pid))
	}()

	return nil
}

// Stop 停止Agent进程
func (m *Manager) Stop(ctx context.Context, graceful bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunningLocked() {
		return nil
	}

	m.logger.Info("stopping agent", zap.Int("pid", m.pid), zap.Bool("graceful", graceful))

	if graceful {
		// 发送SIGTERM，等待优雅退出
		if err := m.process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}

		// 等待最多30秒
		done := make(chan struct{})
		go func() {
			m.process.Wait()
			close(done)
		}()

		select {
		case <-done:
			m.logger.Info("agent stopped gracefully")
		case <-time.After(30 * time.Second):
			m.logger.Warn("agent graceful shutdown timeout, killing")
			m.process.Kill()
		case <-ctx.Done():
			m.logger.Warn("context cancelled, killing agent")
			m.process.Kill()
		}
	} else {
		// 强制杀死
		if err := m.process.Kill(); err != nil {
			return fmt.Errorf("failed to kill agent: %w", err)
		}
	}

	m.process = nil
	m.pid = 0

	return nil
}

// Restart 重启Agent
func (m *Manager) Restart(ctx context.Context) error {
	m.logger.Info("restarting agent", zap.Int("restart_count", m.restartCount))

	// 计算退避时间
	backoff := m.calculateBackoff()
	if backoff > 0 {
		m.logger.Info("waiting before restart", zap.Duration("backoff", backoff))
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// 停止当前进程
	if err := m.Stop(ctx, true); err != nil {
		m.logger.Error("failed to stop agent before restart", zap.Error(err))
	}

	// 启动新进程
	if err := m.Start(ctx); err != nil {
		return fmt.Errorf("failed to start agent after restart: %w", err)
	}

	m.restartCount++
	m.lastRestart = time.Now()

	// 如果重启次数过多，记录告警
	if m.restartCount > m.config.Restart.MaxRetries {
		m.logger.Error("agent restart count exceeds threshold",
			zap.Int("restart_count", m.restartCount),
			zap.Int("threshold", m.config.Restart.MaxRetries))
	}

	return nil
}

// IsRunning 检查Agent是否运行
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isRunningLocked()
}

// isRunningLocked 检查进程是否运行(需要持锁调用)
func (m *Manager) isRunningLocked() bool {
	if m.process == nil {
		return false
	}
	// 发送信号0检查进程是否存在
	err := m.process.Signal(syscall.Signal(0))
	return err == nil
}

// GetPID 获取Agent PID
func (m *Manager) GetPID() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pid
}

// GetRestartCount 获取重启次数
func (m *Manager) GetRestartCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.restartCount
}

// ResetRestartCount 重置重启计数
func (m *Manager) ResetRestartCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restartCount = 0
}

// calculateBackoff 计算退避时间
func (m *Manager) calculateBackoff() time.Duration {
	// 如果距离上次重启超过5分钟，重置计数
	if time.Since(m.lastRestart) > 5*time.Minute {
		m.restartCount = 0
		return 0
	}

	switch {
	case m.restartCount < 1:
		return 0
	case m.restartCount < 3:
		return 10 * time.Second
	case m.restartCount < 5:
		return 30 * time.Second
	default:
		return 60 * time.Second
	}
}
