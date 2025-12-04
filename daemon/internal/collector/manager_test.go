package collector

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewManager(t *testing.T) {
	logger := zap.NewNop()
	collectors := []Collector{
		NewCPUCollector(true, 1*time.Second, logger),
		NewMemoryCollector(true, 1*time.Second, logger),
	}

	manager := NewManager(collectors, logger)
	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestManager_Start_Stop(t *testing.T) {
	logger := zap.NewNop()
	collectors := []Collector{
		NewCPUCollector(true, 100*time.Millisecond, logger),
		NewMemoryCollector(true, 100*time.Millisecond, logger),
	}

	manager := NewManager(collectors, logger)

	// 启动管理器
	manager.Start()

	// 等待采集器运行
	time.Sleep(300 * time.Millisecond)

	// 停止管理器
	manager.Stop()

	// 测试通过意味着没有panic或死锁
}

func TestManager_GetLatest(t *testing.T) {
	logger := zap.NewNop()
	collectors := []Collector{
		NewCPUCollector(true, 100*time.Millisecond, logger),
		NewMemoryCollector(true, 100*time.Millisecond, logger),
	}

	manager := NewManager(collectors, logger)
	manager.Start()
	defer manager.Stop()

	// 等待至少采集一次
	time.Sleep(300 * time.Millisecond)

	// 获取最新指标
	metrics := manager.GetLatest()

	// 应该有两个采集器的指标
	if len(metrics) == 0 {
		t.Error("expected at least some metrics")
	}

	// 检查指标类型
	foundCPU := false
	foundMemory := false
	for _, metric := range metrics {
		if metric.Name == "cpu" {
			foundCPU = true
		}
		if metric.Name == "memory" {
			foundMemory = true
		}
	}

	if !foundCPU {
		t.Error("expected to find CPU metric")
	}
	if !foundMemory {
		t.Error("expected to find memory metric")
	}
}

func TestManager_GetLatest_BeforeStart(t *testing.T) {
	logger := zap.NewNop()
	collectors := []Collector{
		NewCPUCollector(true, 1*time.Second, logger),
	}

	manager := NewManager(collectors, logger)

	// 启动前获取指标应该返回空
	metrics := manager.GetLatest()
	if len(metrics) != 0 {
		t.Error("expected empty metrics before start")
	}
}

func TestManager_DisabledCollector(t *testing.T) {
	logger := zap.NewNop()
	collectors := []Collector{
		NewCPUCollector(true, 100*time.Millisecond, logger),
		NewMemoryCollector(false, 100*time.Millisecond, logger), // 禁用
	}

	manager := NewManager(collectors, logger)
	manager.Start()
	defer manager.Stop()

	// 等待采集
	time.Sleep(300 * time.Millisecond)

	metrics := manager.GetLatest()

	// 应该只有启用的采集器的指标
	foundMemory := false
	for _, metric := range metrics {
		if metric.Name == "memory" {
			foundMemory = true
		}
	}

	if foundMemory {
		t.Error("should not find metrics from disabled collector")
	}
}

func TestManager_EmptyCollectors(t *testing.T) {
	logger := zap.NewNop()
	collectors := []Collector{}

	manager := NewManager(collectors, logger)
	manager.Start()
	defer manager.Stop()

	// 不应该panic
	metrics := manager.GetLatest()
	if len(metrics) != 0 {
		t.Error("expected empty metrics for empty collectors")
	}
}

func TestManager_MultipleCycles(t *testing.T) {
	logger := zap.NewNop()
	collectors := []Collector{
		NewCPUCollector(true, 50*time.Millisecond, logger),
	}

	manager := NewManager(collectors, logger)
	manager.Start()
	defer manager.Stop()

	// 等待多个采集周期
	time.Sleep(200 * time.Millisecond)

	// 应该能持续获取指标
	for i := 0; i < 3; i++ {
		metrics := manager.GetLatest()
		if len(metrics) == 0 {
			t.Errorf("cycle %d: expected metrics", i+1)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	logger := zap.NewNop()
	collectors := []Collector{
		NewCPUCollector(true, 50*time.Millisecond, logger),
		NewMemoryCollector(true, 50*time.Millisecond, logger),
	}

	manager := NewManager(collectors, logger)
	manager.Start()
	defer manager.Stop()

	// 并发读取指标
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_ = manager.GetLatest()
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 5; i++ {
		<-done
	}

	// 测试通过意味着没有数据竞争
}

func TestManager_StopBeforeStart(t *testing.T) {
	logger := zap.NewNop()
	collectors := []Collector{
		NewCPUCollector(true, 1*time.Second, logger),
	}

	manager := NewManager(collectors, logger)

	// 未启动就停止，不应该panic
	manager.Stop()
}

func TestManager_AllCollectorTypes(t *testing.T) {
	logger := zap.NewNop()
	collectors := []Collector{
		NewCPUCollector(true, 100*time.Millisecond, logger),
		NewMemoryCollector(true, 100*time.Millisecond, logger),
		NewDiskCollector(true, 100*time.Millisecond, []string{}, logger),
		NewNetworkCollector(true, 100*time.Millisecond, []string{}, logger),
	}

	manager := NewManager(collectors, logger)
	manager.Start()
	defer manager.Stop()

	// 等待所有采集器至少采集一次
	time.Sleep(400 * time.Millisecond)

	metrics := manager.GetLatest()

	// 应该有所有类型的指标
	foundTypes := make(map[string]bool)
	for _, metric := range metrics {
		foundTypes[metric.Name] = true
	}

	expectedTypes := []string{"cpu", "memory", "disk", "network"}
	for _, expectedType := range expectedTypes {
		if !foundTypes[expectedType] {
			t.Errorf("expected to find %s metric", expectedType)
		}
	}
}
