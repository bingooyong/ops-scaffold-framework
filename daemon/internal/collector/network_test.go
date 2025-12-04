package collector

import (
	"context"
	"testing"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"go.uber.org/zap"
)

func TestNetworkCollector_Name(t *testing.T) {
	logger := zap.NewNop()
	collector := NewNetworkCollector(true, 1*time.Second, []string{}, logger)

	if collector.Name() != "network" {
		t.Errorf("expected name 'network', got '%s'", collector.Name())
	}
}

func TestNetworkCollector_Enabled(t *testing.T) {
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
			collector := NewNetworkCollector(tt.enabled, 1*time.Second, []string{}, logger)
			if collector.Enabled() != tt.enabled {
				t.Errorf("expected enabled=%v, got %v", tt.enabled, collector.Enabled())
			}
		})
	}
}

func TestNetworkCollector_Interval(t *testing.T) {
	logger := zap.NewNop()
	interval := 5 * time.Second
	collector := NewNetworkCollector(true, interval, []string{}, logger)

	if collector.Interval() != interval {
		t.Errorf("expected interval %v, got %v", interval, collector.Interval())
	}
}

func TestNetworkCollector_Collect_AllInterfaces(t *testing.T) {
	logger := zap.NewNop()
	collector := NewNetworkCollector(true, 1*time.Second, []string{}, logger)
	ctx := context.Background()

	metric, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if metric == nil {
		t.Fatal("expected non-nil metric")
	}

	// 检查指标名称
	if metric.Name != "network" {
		t.Errorf("expected name 'network', got '%s'", metric.Name)
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

func TestNetworkCollector_Collect_SpecificInterfaces(t *testing.T) {
	logger := zap.NewNop()
	collector := NewNetworkCollector(true, 1*time.Second, []string{"lo", "lo0"}, logger)
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

func TestNetworkCollector_MultipleCollections(t *testing.T) {
	logger := zap.NewNop()
	collector := NewNetworkCollector(true, 100*time.Millisecond, []string{}, logger)
	ctx := context.Background()

	var previousMetric *types.Metrics

	// 收集多次，确保每次都能成功
	for i := 0; i < 3; i++ {
		metric, err := collector.Collect(ctx)
		if err != nil {
			t.Fatalf("collection %d failed: %v", i+1, err)
		}
		if metric == nil {
			t.Fatalf("collection %d: expected non-nil metric", i+1)
		}

		if metric.Name != "network" {
			t.Errorf("collection %d: expected name 'network', got '%s'", i+1, metric.Name)
		}

		// 检查时间戳递增
		if previousMetric != nil {
			if !metric.Timestamp.After(previousMetric.Timestamp) {
				t.Errorf("collection %d: timestamp should increase", i+1)
			}
		}

		previousMetric = metric

		if i < 2 {
			time.Sleep(50 * time.Millisecond)
		}
	}
}
