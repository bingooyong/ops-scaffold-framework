package collector

import (
	"context"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
)

// Collector 采集器接口
type Collector interface {
	// Name 返回采集器名称
	Name() string

	// Collect 执行采集，返回指标数据
	Collect(ctx context.Context) (*types.Metrics, error)

	// Interval 返回采集间隔
	Interval() time.Duration

	// Enabled 返回是否启用
	Enabled() bool
}
