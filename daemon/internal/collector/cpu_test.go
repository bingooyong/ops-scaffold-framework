package collector

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestCPUCollector_Name(t *testing.T) {
	logger := zap.NewNop()
	collector := NewCPUCollector(true, 1*time.Second, logger)

	if collector.Name() != "cpu" {
		t.Errorf("expected name 'cpu', got '%s'", collector.Name())
	}
}

func TestCPUCollector_Enabled(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled collector", true},
		{"disabled collector", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewCPUCollector(tt.enabled, 1*time.Second, logger)
			if collector.Enabled() != tt.enabled {
				t.Errorf("expected enabled=%v, got %v", tt.enabled, collector.Enabled())
			}
		})
	}
}

func TestCPUCollector_Interval(t *testing.T) {
	logger := zap.NewNop()
	interval := 5 * time.Second
	collector := NewCPUCollector(true, interval, logger)

	if collector.Interval() != interval {
		t.Errorf("expected interval %v, got %v", interval, collector.Interval())
	}
}

func TestCPUCollector_Collect(t *testing.T) {
	logger := zap.NewNop()
	collector := NewCPUCollector(true, 1*time.Second, logger)
	ctx := context.Background()

	// 第一次采集需要等待以计算CPU使用率
	metric, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if metric == nil {
		t.Fatal("expected non-nil metric")
	}

	// 检查指标名称
	if metric.Name != "cpu" {
		t.Errorf("expected name 'cpu', got '%s'", metric.Name)
	}

	// 检查时间戳
	if metric.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	// 检查数据字段
	if metric.Values == nil {
		t.Fatal("expected non-nil values")
	}

	// 检查必需字段
	requiredFields := []string{"usage_percent", "cores"}
	for _, field := range requiredFields {
		if _, exists := metric.Values[field]; !exists {
			t.Errorf("expected field '%s' in values", field)
		}
	}

	// 检查usage_percent范围
	if usagePercent, ok := metric.Values["usage_percent"].(float64); ok {
		if usagePercent < 0 || usagePercent > 100 {
			t.Errorf("usage_percent out of range: %f", usagePercent)
		}
	}

	// 检查cores数量
	if cores, ok := metric.Values["cores"].(int); ok {
		if cores <= 0 {
			t.Errorf("invalid cores count: %d", cores)
		}
	}
}

func TestCPUCollector_MultipleCollections(t *testing.T) {
	logger := zap.NewNop()
	collector := NewCPUCollector(true, 100*time.Millisecond, logger)
	ctx := context.Background()

	// 收集多次，确保每次都能成功
	for i := 0; i < 3; i++ {
		metric, err := collector.Collect(ctx)
		if err != nil {
			t.Fatalf("collection %d failed: %v", i+1, err)
		}
		if metric == nil {
			t.Fatalf("collection %d: expected non-nil metric", i+1)
		}

		if metric.Name != "cpu" {
			t.Errorf("collection %d: expected name 'cpu', got '%s'", i+1, metric.Name)
		}

		// 等待采集间隔
		if i < 2 {
			time.Sleep(150 * time.Millisecond)
		}
	}
}

func TestCPUCollector_DisabledCollector(t *testing.T) {
	logger := zap.NewNop()
	collector := NewCPUCollector(false, 1*time.Second, logger)
	ctx := context.Background()

	// 禁用的采集器仍然可以采集，只是不会被管理器自动调用
	metric, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if metric == nil {
		t.Fatal("expected non-nil metric even when disabled")
	}
}
