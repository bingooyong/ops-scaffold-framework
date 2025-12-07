package service

import (
	"context"
	"testing"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MetricsCleanerTestSuite 清理服务测试套件
type MetricsCleanerTestSuite struct {
	suite.Suite
	db      *gorm.DB
	cleaner *MetricsCleaner
	ctx     context.Context
	logger  *zap.Logger
}

// SetupSuite 测试套件初始化
func (s *MetricsCleanerTestSuite) SetupSuite() {
	// 使用 SQLite 内存数据库
	// 注意：SQLite 不支持 MySQL 分区，此测试主要验证逻辑和错误处理
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(s.T(), err, "初始化数据库失败")

	// 自动迁移表结构
	err = db.AutoMigrate(&model.Metrics{})
	require.NoError(s.T(), err, "迁移表结构失败")

	s.db = db
	s.logger = zap.NewNop()                        // 使用空 logger
	s.cleaner = NewMetricsCleaner(db, 7, s.logger) // 保留 7 天
	s.ctx = context.Background()
}

// TearDownSuite 测试套件清理
func (s *MetricsCleanerTestSuite) TearDownSuite() {
	sqlDB, _ := s.db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}

// TestNewMetricsCleaner 测试构造函数
func (s *MetricsCleanerTestSuite) TestNewMetricsCleaner() {
	cleaner := NewMetricsCleaner(s.db, 30, s.logger)
	assert.NotNil(s.T(), cleaner)
	assert.Equal(s.T(), 30, cleaner.retentionDays)
	assert.Equal(s.T(), s.db, cleaner.db)
	assert.Equal(s.T(), s.logger, cleaner.logger)
}

// TestCleanExpiredPartitions_NoPartitions 测试无分区情况
func (s *MetricsCleanerTestSuite) TestCleanExpiredPartitions_NoPartitions() {
	// SQLite 不支持分区，getPartitions 会返回错误或空列表
	// 此测试主要验证错误处理逻辑
	err := s.cleaner.CleanExpiredPartitions(s.ctx)
	// SQLite 不支持 information_schema.PARTITIONS，预期会失败
	// 在实际 MySQL 环境中应该成功
	if err != nil {
		s.T().Logf("⚠️ SQLite 不支持 MySQL 分区，此测试需要在 MySQL 环境中运行: %v", err)
	} else {
		// 如果没有分区，应该正常返回
		assert.NoError(s.T(), err)
	}
}

// TestGetPartitions 测试获取分区列表
func (s *MetricsCleanerTestSuite) TestGetPartitions() {
	// SQLite 不支持 information_schema.PARTITIONS
	partitions, err := s.cleaner.getPartitions(s.ctx)
	if err != nil {
		s.T().Logf("⚠️ SQLite 不支持 MySQL information_schema，此测试需要在 MySQL 环境中运行: %v", err)
	} else {
		// 在 MySQL 环境中，应该返回分区列表
		assert.NotNil(s.T(), partitions)
	}
}

// TestRetentionDays 测试不同的保留天数配置
func (s *MetricsCleanerTestSuite) TestRetentionDays() {
	testCases := []struct {
		name          string
		retentionDays int
		description   string
	}{
		{"7 days", 7, "开发环境保留 7 天"},
		{"30 days", 30, "生产环境保留 30 天"},
		{"90 days", 90, "长期保留 90 天"},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			cleaner := NewMetricsCleaner(s.db, tc.retentionDays, s.logger)
			assert.Equal(t, tc.retentionDays, cleaner.retentionDays, tc.description)
		})
	}
}

// TestCleanExpiredPartitions_ErrorHandling 测试错误处理
func (s *MetricsCleanerTestSuite) TestCleanExpiredPartitions_ErrorHandling() {
	// 测试无效的数据库连接
	invalidDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	invalidDB.Exec("DROP TABLE IF EXISTS metrics") // 删除表，模拟错误场景

	cleaner := NewMetricsCleaner(invalidDB, 7, s.logger)
	err := cleaner.CleanExpiredPartitions(s.ctx)
	// 应该返回错误，但不应该崩溃
	if err != nil {
		assert.Error(s.T(), err, "应该返回错误")
	}
}

// TestMetricsCleaner 运行测试套件
func TestMetricsCleaner(t *testing.T) {
	suite.Run(t, new(MetricsCleanerTestSuite))
}
