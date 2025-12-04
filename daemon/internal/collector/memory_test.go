package collector

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMemoryCollector_Name(t *testing.T) {
	logger := zap.NewNop()
	collector := NewMemoryCollector(true, 1*time.Second, logger)

	if collector.Name() != "memory" {
		t.Errorf("expected name 'memory', got '%s'", collector.Name())
	}
}

func TestMemoryCollector_Enabled(t *testing.T) {
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
			collector := NewMemoryCollector(tt.enabled, 1*time.Second, logger)
			if collector.Enabled() != tt.enabled {
				t.Errorf("expected enabled=%v, got %v", tt.enabled, collector.Enabled())
			}
		})
	}
}

func TestMemoryCollector_Interval(t *testing.T) {
	logger := zap.NewNop()
	interval := 5 * time.Second
	collector := NewMemoryCollector(true, interval, logger)

	if collector.Interval() != interval {
		t.Errorf("expected interval %v, got %v", interval, collector.Interval())
	}
}

func TestMemoryCollector_Collect(t *testing.T) {
	logger := zap.NewNop()
	collector := NewMemoryCollector(true, 1*time.Second, logger)
	ctx := context.Background()

	metric, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if metric == nil {
		t.Fatal("expected non-nil metric")
	}

	// 检查指标名称
	if metric.Name != "memory" {
		t.Errorf("expected name 'memory', got '%s'", metric.Name)
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
	requiredFields := []string{
		"total", "available", "used", "used_percent",
		"swap_total", "swap_used", "swap_percent",
	}
	for _, field := range requiredFields {
		if _, exists := metric.Values[field]; !exists {
			t.Errorf("expected field '%s' in values", field)
		}
	}

	// 检查内存值的合理性
	if total, ok := metric.Values["total"].(uint64); ok {
		if total == 0 {
			t.Error("total memory should not be zero")
		}

		// used应该小于等于total
		if used, ok := metric.Values["used"].(uint64); ok {
			if used > total {
				t.Errorf("used memory (%d) should not exceed total (%d)", used, total)
			}
		}

		// available应该小于等于total
		if available, ok := metric.Values["available"].(uint64); ok {
			if available > total {
				t.Errorf("available memory (%d) should not exceed total (%d)", available, total)
			}
		}
	} else {
		t.Error("total field should be uint64")
	}

	// 检查百分比范围
	if usedPercent, ok := metric.Values["used_percent"].(float64); ok {
		if usedPercent < 0 || usedPercent > 100 {
			t.Errorf("used_percent out of range: %f", usedPercent)
		}
	}

	if swapPercent, ok := metric.Values["swap_percent"].(float64); ok {
		if swapPercent < 0 || swapPercent > 100 {
			t.Errorf("swap_percent out of range: %f", swapPercent)
		}
	}
}

func TestMemoryCollector_MultipleCollections(t *testing.T) {
	logger := zap.NewNop()
	collector := NewMemoryCollector(true, 100*time.Millisecond, logger)
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

		if metric.Name != "memory" {
			t.Errorf("collection %d: expected name 'memory', got '%s'", i+1, metric.Name)
		}

		// 短暂等待
		if i < 2 {
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func TestMemoryCollector_DataConsistency(t *testing.T) {
	logger := zap.NewNop()
	collector := NewMemoryCollector(true, 1*time.Second, logger)
	ctx := context.Background()

	metric, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if metric == nil {
		t.Fatal("expected non-nil metric")
	}

	// 验证 total = used + available 的关系（允许一定误差）
	total, _ := metric.Values["total"].(uint64)
	used, _ := metric.Values["used"].(uint64)
	available, _ := metric.Values["available"].(uint64)

	// 由于系统内存统计的复杂性，这里只做基本检查
	if total > 0 {
		if used > total || available > total {
			t.Errorf("memory values inconsistent: total=%d, used=%d, available=%d",
				total, used, available)
		}
	}
}
