package collector

import (
	"context"
	"sync"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"go.uber.org/zap"
)

// Manager 采集器管理器
type Manager struct {
	collectors []Collector
	latest     map[string]*types.Metrics
	mu         sync.RWMutex
	logger     *zap.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewManager 创建采集器管理器
func NewManager(collectors []Collector, logger *zap.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		collectors: collectors,
		latest:     make(map[string]*types.Metrics),
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动所有采集器
func (m *Manager) Start() {
	m.logger.Info("starting collector manager")

	for _, collector := range m.collectors {
		if !collector.Enabled() {
			m.logger.Info("collector disabled, skipping",
				zap.String("collector", collector.Name()))
			continue
		}

		m.wg.Add(1)
		go m.runCollector(collector)

		m.logger.Info("collector started",
			zap.String("collector", collector.Name()),
			zap.Duration("interval", collector.Interval()))
	}
}

// Stop 停止所有采集器
func (m *Manager) Stop() {
	m.logger.Info("stopping collector manager")
	m.cancel()
	m.wg.Wait()
	m.logger.Info("collector manager stopped")
}

// runCollector 运行单个采集器
func (m *Manager) runCollector(collector Collector) {
	defer m.wg.Done()

	ticker := time.NewTicker(collector.Interval())
	defer ticker.Stop()

	// 立即执行一次采集
	m.collect(collector)

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.collect(collector)
		}
	}
}

// collect 执行采集
func (m *Manager) collect(collector Collector) {
	start := time.Now()

	metrics, err := collector.Collect(m.ctx)
	if err != nil {
		m.logger.Error("failed to collect metrics",
			zap.String("collector", collector.Name()),
			zap.Error(err))
		return
	}

	// 保存最新指标
	m.mu.Lock()
	m.latest[collector.Name()] = metrics
	m.mu.Unlock()

	duration := time.Since(start)
	m.logger.Debug("metrics collected",
		zap.String("collector", collector.Name()),
		zap.Duration("duration", duration))
}

// GetLatest 获取最新的所有指标
func (m *Manager) GetLatest() map[string]*types.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*types.Metrics)
	for k, v := range m.latest {
		result[k] = v
	}
	return result
}

// GetLatestByName 获取指定采集器的最新指标
func (m *Manager) GetLatestByName(name string) *types.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latest[name]
}
