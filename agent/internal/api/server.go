package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ResourceGetter 资源使用获取接口
type ResourceGetter interface {
	GetLastResourceUsage() (cpu float64, memory uint64)
}

// Server HTTP API 服务器
type Server struct {
	host       string
	port       int
	agentID    string
	version    string
	startTime  time.Time
	logger     *zap.Logger
	engine     *gin.Engine
	httpServer *http.Server
	// 心跳状态
	lastHeartbeat     time.Time
	heartbeatCount    int64
	heartbeatFailures int64
	// 资源使用获取器
	resourceGetter ResourceGetter
	// 配置重载回调
	onReload func() error
}

// NewServer 创建 HTTP API 服务器
func NewServer(host string, port int, agentID, version string, logger *zap.Logger) *Server {
	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	s := &Server{
		host:      host,
		port:      port,
		agentID:   agentID,
		version:   version,
		startTime: time.Now(),
		logger:    logger,
		engine:    gin.New(),
	}

	// 配置中间件
	s.engine.Use(gin.Recovery())
	s.engine.Use(s.loggingMiddleware())

	// 注册路由
	s.registerRoutes()

	return s
}

// Start 启动 HTTP 服务器
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}

	s.logger.Info("starting HTTP server", zap.String("addr", addr))

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop 停止 HTTP 服务器
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping HTTP server")

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

// registerRoutes 注册路由
func (s *Server) registerRoutes() {
	s.engine.GET("/health", s.handleHealth)
	s.engine.POST("/reload", s.handleReload)
	s.engine.GET("/metrics", s.handleMetrics)
}

// loggingMiddleware 日志中间件
func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		s.logger.Debug("http request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency))
	}
}

// handleHealth 健康检查接口
func (s *Server) handleHealth(c *gin.Context) {
	uptime := int64(time.Since(s.startTime).Seconds())

	status := "healthy"
	if time.Since(s.lastHeartbeat) > 90*time.Second && !s.lastHeartbeat.IsZero() {
		status = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":         status,
			"uptime":         uptime,
			"last_heartbeat": s.lastHeartbeat.Format(time.RFC3339),
			"agent_id":       s.agentID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         status,
		"uptime":         uptime,
		"last_heartbeat": s.lastHeartbeat.Format(time.RFC3339),
		"agent_id":       s.agentID,
	})
}

// handleReload 配置重载接口
func (s *Server) handleReload(c *gin.Context) {
	s.logger.Info("received reload request")

	// 调用配置重载回调
	if s.onReload != nil {
		if err := s.onReload(); err != nil {
			s.logger.Error("config reload failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"success":     false,
				"message":     fmt.Sprintf("config reload failed: %v", err),
				"reloaded_at": time.Now().Format(time.RFC3339),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "config reloaded successfully",
		"reloaded_at": time.Now().Format(time.RFC3339),
	})
}

// handleMetrics 指标暴露接口
func (s *Server) handleMetrics(c *gin.Context) {
	uptime := int64(time.Since(s.startTime).Seconds())

	// 获取当前资源使用情况
	var cpuPercent float64
	var memoryBytes uint64
	if s.resourceGetter != nil {
		cpuPercent, memoryBytes = s.resourceGetter.GetLastResourceUsage()
	}

	// 确定运行状态
	status := "running"
	if time.Since(s.lastHeartbeat) > 90*time.Second && !s.lastHeartbeat.IsZero() {
		status = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id":           s.agentID,
		"version":            s.version,
		"uptime":             uptime,
		"heartbeat_count":    s.heartbeatCount,
		"heartbeat_failures": s.heartbeatFailures,
		"last_heartbeat":     s.lastHeartbeat.Format(time.RFC3339),
		"cpu_percent":        cpuPercent,
		"memory_bytes":       memoryBytes,
		"status":             status,
	})
}

// UpdateHeartbeatStatus 更新心跳状态（供 heartbeat manager 调用）
func (s *Server) UpdateHeartbeatStatus(success bool) {
	s.lastHeartbeat = time.Now()
	if success {
		s.heartbeatCount++
	} else {
		s.heartbeatFailures++
	}
}

// SetResourceGetter 设置资源使用获取器
func (s *Server) SetResourceGetter(getter ResourceGetter) {
	s.resourceGetter = getter
}

// SetReloadCallback 设置配置重载回调
func (s *Server) SetReloadCallback(callback func() error) {
	s.onReload = callback
}
