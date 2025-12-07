package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"go.uber.org/zap"
)

// Heartbeat 心跳数据结构（与 daemon/pkg/types/types.go 中的 Heartbeat 结构保持一致）
type Heartbeat struct {
	PID       int       `json:"pid"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Status    string    `json:"status"`
	CPU       float64   `json:"cpu"`
	Memory    uint64    `json:"memory"`
}

// Manager 心跳管理器
type Manager struct {
	socketPath string
	interval   time.Duration
	version    string
	logger     *zap.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	conn       net.Conn
	// 资源使用缓存
	lastCPU    float64
	lastMemory uint64
	// 心跳状态回调
	onHeartbeatSuccess func()
	onHeartbeatFailure func()
}

// NewManager 创建心跳管理器
func NewManager(socketPath string, interval time.Duration, version string, logger *zap.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		socketPath: socketPath,
		interval:   interval,
		version:    version,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// SetHeartbeatCallbacks 设置心跳状态回调
func (m *Manager) SetHeartbeatCallbacks(onSuccess, onFailure func()) {
	m.onHeartbeatSuccess = onSuccess
	m.onHeartbeatFailure = onFailure
}

// GetLastResourceUsage 获取最后一次采集的资源使用情况
func (m *Manager) GetLastResourceUsage() (cpu float64, memory uint64) {
	return m.lastCPU, m.lastMemory
}

// Start 启动心跳发送
func (m *Manager) Start() error {
	m.logger.Info("starting heartbeat manager",
		zap.String("socket_path", m.socketPath),
		zap.Duration("interval", m.interval))

	// 连接到 Daemon 的 Unix Socket
	conn, err := net.Dial("unix", m.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon socket: %w", err)
	}
	m.conn = conn

	// 立即发送一次心跳
	if err := m.sendHeartbeat("running"); err != nil {
		m.logger.Warn("failed to send initial heartbeat", zap.Error(err))
	}

	// 启动心跳循环
	go m.heartbeatLoop()

	return nil
}

// Stop 停止心跳发送
func (m *Manager) Stop() {
	m.logger.Info("stopping heartbeat manager")

	// 发送最后一次心跳（status="stopping"）
	if err := m.sendHeartbeat("stopping"); err != nil {
		m.logger.Warn("failed to send final heartbeat", zap.Error(err))
	}

	// 关闭连接
	if m.conn != nil {
		m.conn.Close()
	}

	// 取消上下文
	m.cancel()
}

// heartbeatLoop 心跳发送循环
func (m *Manager) heartbeatLoop() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if err := m.sendHeartbeat("running"); err != nil {
				m.logger.Error("failed to send heartbeat", zap.Error(err))
				// 心跳失败时尝试重连
				if err := m.reconnect(); err != nil {
					m.logger.Error("failed to reconnect to daemon", zap.Error(err))
				}
			}
		}
	}
}

// sendHeartbeat 发送心跳
func (m *Manager) sendHeartbeat(status string) error {
	if m.conn == nil {
		return fmt.Errorf("not connected to daemon")
	}

	// 采集资源使用情况
	cpu, memory := m.collectResourceUsage()

	// 更新缓存
	m.lastCPU = cpu
	m.lastMemory = memory

	// 构建心跳数据
	hb := Heartbeat{
		PID:       os.Getpid(),
		Timestamp: time.Now(),
		Version:   m.version,
		Status:    status,
		CPU:       cpu,
		Memory:    memory,
	}

	// 序列化为 JSON
	data, err := json.Marshal(hb)
	if err != nil {
		if m.onHeartbeatFailure != nil {
			m.onHeartbeatFailure()
		}
		return fmt.Errorf("failed to marshal heartbeat: %w", err)
	}

	// 发送到 Unix Socket
	if _, err := m.conn.Write(append(data, '\n')); err != nil {
		if m.onHeartbeatFailure != nil {
			m.onHeartbeatFailure()
		}
		return fmt.Errorf("failed to write to socket: %w", err)
	}

	// 调用成功回调
	if m.onHeartbeatSuccess != nil {
		m.onHeartbeatSuccess()
	}

	m.logger.Debug("heartbeat sent",
		zap.String("status", status),
		zap.Float64("cpu", cpu),
		zap.Uint64("memory", memory))

	return nil
}

// collectResourceUsage 采集资源使用情况
func (m *Manager) collectResourceUsage() (cpu float64, memory uint64) {
	pid := os.Getpid()
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		m.logger.Warn("failed to get process info", zap.Error(err))
		return 0, 0
	}

	// 采集 CPU 使用率
	cpuPercent, err := proc.CPUPercent()
	if err != nil {
		m.logger.Debug("failed to get cpu percent", zap.Error(err))
		cpuPercent = 0
	}

	// 采集内存使用
	memInfo, err := proc.MemoryInfo()
	if err != nil {
		m.logger.Debug("failed to get memory info", zap.Error(err))
		return cpuPercent, 0
	}

	return cpuPercent, memInfo.RSS
}

// reconnect 重连到 Daemon
func (m *Manager) reconnect() error {
	m.logger.Info("attempting to reconnect to daemon")

	// 关闭旧连接
	if m.conn != nil {
		m.conn.Close()
	}

	// 重新连接
	conn, err := net.Dial("unix", m.socketPath)
	if err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	m.conn = conn
	m.logger.Info("reconnected to daemon successfully")

	return nil
}
