package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/config"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// DB 全局数据库实例
	DB *gorm.DB
)

// Init 初始化数据库连接
func Init(cfg *config.DatabaseConfig, log *zap.Logger) (*gorm.DB, error) {
	// 配置GORM日志级别
	var logLevel logger.LogLevel
	switch cfg.LogLevel {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	default:
		logLevel = logger.Warn
	}

	// 创建GORM配置
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
		// 禁用外键约束（提高性能，需要应用层保证数据一致性）
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(cfg.DSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 获取底层sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("database connected successfully",
		zap.Int("max_idle_conns", cfg.MaxIdleConns),
		zap.Int("max_open_conns", cfg.MaxOpenConns),
		zap.Duration("conn_max_lifetime", cfg.ConnMaxLifetime),
	)

	// 设置全局DB
	DB = db

	return db, nil
}

// fixConstraintError 尝试修复约束相关的迁移错误
// 返回 (shouldRetry, error)
// shouldRetry: 是否应该重试迁移
// error: 如果shouldRetry为false且error为nil，表示可以安全忽略此错误
func fixConstraintError(db *gorm.DB, err error, log *zap.Logger) (bool, error) {
	errStr := err.Error()

	// 检查是否是 "Can't DROP" 错误（约束不存在）
	if !strings.Contains(errStr, "Can't DROP") {
		return false, err
	}

	if !strings.Contains(errStr, "check that column/key exists") &&
		!strings.Contains(errStr, "Unknown key") {
		return false, err
	}

	// 提取约束名称（例如：uni_users_username）
	// 错误格式通常是：Error 1091 (42000): Can't DROP 'constraint_name'; check that column/key exists
	constraintName := ""
	if idx := strings.Index(errStr, "Can't DROP '"); idx != -1 {
		start := idx + len("Can't DROP '")
		if end := strings.Index(errStr[start:], "'"); end != -1 {
			constraintName = errStr[start : start+end]
		}
	}

	// 尝试从错误信息中提取表名
	// 从 ALTER TABLE `users` DROP FOREIGN KEY 中提取
	tableName := ""
	if idx := strings.Index(errStr, "ALTER TABLE `"); idx != -1 {
		start := idx + len("ALTER TABLE `")
		if end := strings.Index(errStr[start:], "`"); end != -1 {
			tableName = errStr[start : start+end]
		}
	}

	// 如果无法从错误信息中提取表名，尝试从约束名推断
	if tableName == "" && constraintName != "" {
		// 约束名格式通常是：uni_table_column 或 idx_table_column
		parts := strings.Split(constraintName, "_")
		if len(parts) >= 2 {
			// 假设第二个部分是表名（例如：uni_users_username -> users）
			tableName = parts[1]
		}
	}

	if log != nil {
		log.Warn("GORM tried to drop non-existent constraint, verifying...",
			zap.String("constraint", constraintName),
			zap.String("table", tableName),
			zap.String("error", errStr))
	}

	// 如果找到了约束名和表名，检查约束是否真的不存在
	if constraintName != "" && tableName != "" {
		// 检查是否是外键
		var fkCount int64
		checkFKSQL := `
			SELECT COUNT(*) 
			FROM information_schema.KEY_COLUMN_USAGE
			WHERE TABLE_SCHEMA = DATABASE()
			  AND TABLE_NAME = ?
			  AND CONSTRAINT_NAME = ?
			  AND REFERENCED_TABLE_NAME IS NOT NULL
		`
		if err := db.Raw(checkFKSQL, tableName, constraintName).Scan(&fkCount).Error; err == nil {
			if fkCount > 0 {
				// 外键存在，尝试删除它
				if log != nil {
					log.Info("Found existing foreign key, attempting to drop", zap.String("fk", constraintName))
				}
				dropFKSQL := fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `%s`", tableName, constraintName)
				if dropErr := db.Exec(dropFKSQL).Error; dropErr != nil {
					if log != nil {
						log.Error("Failed to drop foreign key",
							zap.String("fk", constraintName),
							zap.Error(dropErr))
					}
					return false, dropErr
				}
				// 成功删除，重试迁移
				return true, nil
			}
		}

		// 检查是否是索引
		var indexCount int64
		checkIndexSQL := `
			SELECT COUNT(*) 
			FROM information_schema.STATISTICS
			WHERE TABLE_SCHEMA = DATABASE()
			  AND TABLE_NAME = ?
			  AND INDEX_NAME = ?
		`
		if err := db.Raw(checkIndexSQL, tableName, constraintName).Scan(&indexCount).Error; err == nil {
			if indexCount > 0 {
				// 索引存在，尝试删除它
				if log != nil {
					log.Info("Found existing index, attempting to drop", zap.String("index", constraintName))
				}
				dropIndexSQL := fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`", tableName, constraintName)
				if dropErr := db.Exec(dropIndexSQL).Error; dropErr != nil {
					if log != nil {
						log.Error("Failed to drop index",
							zap.String("index", constraintName),
							zap.Error(dropErr))
					}
					return false, dropErr
				}
				// 成功删除，重试迁移
				return true, nil
			}
		}

		// 约束确实不存在（这是预期状态）
		if log != nil {
			log.Info("Constraint does not exist (expected state), ignoring error",
				zap.String("constraint", constraintName),
				zap.String("table", tableName),
				zap.String("note", "GORM attempted to drop a constraint that never existed. This is safe to ignore."))
		}
		// 约束不存在，这是预期状态，可以安全忽略
		return false, nil
	}

	// 无法提取约束信息，返回原始错误
	return false, err
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB, log *zap.Logger) error {
	// 定义所有需要迁移的模型
	models := []interface{}{
		&model.User{},
		&model.Node{},
		&model.Metrics{},
		&model.AuditLog{},
		&model.Task{},
		&model.Version{},
		&model.Agent{},
	}

	// 逐个迁移每个模型，这样一个模型的错误不会影响其他模型
	for _, m := range models {
		if err := migrateModel(db, m, log); err != nil {
			return err
		}
	}

	return nil
}

// migrateModel 迁移单个模型，处理可能的约束错误
func migrateModel(db *gorm.DB, model interface{}, log *zap.Logger) error {
	maxRetries := 2
	for attempt := 0; attempt < maxRetries; attempt++ {
		// 执行自动迁移
		err := db.AutoMigrate(model)

		// 迁移成功
		if err == nil {
			if attempt > 0 && log != nil {
				log.Info("Migration succeeded after retry",
					zap.String("model", fmt.Sprintf("%T", model)),
					zap.Int("attempt", attempt+1))
			}
			return nil
		}

		// 处理 GORM 迁移中的已知问题：尝试删除不存在的约束
		shouldRetry, fixErr := fixConstraintError(db, err, log)

		// 如果修复函数返回 nil 且不需要重试，说明错误可以安全忽略
		if !shouldRetry && fixErr == nil {
			if log != nil {
				log.Info("Migration constraint error safely ignored",
					zap.String("model", fmt.Sprintf("%T", model)))
			}
			return nil
		}

		// 如果需要重试，继续循环
		if shouldRetry {
			if log != nil {
				log.Info("Retrying migration after constraint cleanup",
					zap.String("model", fmt.Sprintf("%T", model)),
					zap.Int("attempt", attempt+1))
			}
			continue
		}

		// 其他错误，返回
		if fixErr != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, fixErr)
		}
		return fmt.Errorf("failed to migrate %T: %w", model, err)
	}

	// 如果重试多次仍然失败
	return fmt.Errorf("migration failed for %T after %d attempts", model, maxRetries)
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// Transaction 执行事务
func Transaction(fn func(*gorm.DB) error) error {
	return DB.Transaction(fn)
}
