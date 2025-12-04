package collector

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDiskCollector_Name(t *testing.T) {
	logger := zap.NewNop()
	collector := NewDiskCollector(true, 1*time.Second, []string{}, logger)

	if collector.Name() != "disk" {
		t.Errorf("expected name 'disk', got '%s'", collector.Name())
	}
}

func TestDiskCollector_Enabled(t *testing.T) {
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
			collector := NewDiskCollector(tt.enabled, 1*time.Second, []string{}, logger)
			if collector.Enabled() != tt.enabled {
				t.Errorf("expected enabled=%v, got %v", tt.enabled, collector.Enabled())
			}
		})
	}
}

func TestDiskCollector_Interval(t *testing.T) {
	logger := zap.NewNop()
	interval := 5 * time.Second
	collector := NewDiskCollector(true, interval, []string{}, logger)

	if collector.Interval() != interval {
		t.Errorf("expected interval %v, got %v", interval, collector.Interval())
	}
}

func TestDiskCollector_Collect_AllMountPoints(t *testing.T) {
	logger := zap.NewNop()
	collector := NewDiskCollector(true, 1*time.Second, []string{}, logger)
	ctx := context.Background()

	metric, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if metric == nil {
		t.Fatal("expected non-nil metric")
	}

	// 检查指标名称
	if metric.Name != "disk" {
		t.Errorf("expected name 'disk', got '%s'", metric.Name)
	}

	// 检查时间戳
	if metric.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	// 检查数据字段
	if metric.Values == nil {
		t.Fatal("expected non-nil values")
	}
}

func TestDiskCollector_Collect_SpecificMountPoints(t *testing.T) {
	logger := zap.NewNop()
	collector := NewDiskCollector(true, 1*time.Second, []string{"/"}, logger)
	ctx := context.Background()

	metric, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if metric == nil {
		t.Fatal("expected non-nil metric")
	}

	if metric.Values == nil {
		t.Fatal("expected non-nil values")
	}
}

func TestDiskCollector_MultipleCollections(t *testing.T) {
	logger := zap.NewNop()
	collector := NewDiskCollector(true, 100*time.Millisecond, []string{}, logger)
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

		if metric.Name != "disk" {
			t.Errorf("collection %d: expected name 'disk', got '%s'", i+1, metric.Name)
		}

		if i < 2 {
			time.Sleep(50 * time.Millisecond)
		}
	}
}
