package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/config"
	grpcserver "github.com/bingooyong/ops-scaffold-framework/manager/internal/grpc"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/handler"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/logger"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/middleware"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/repository"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/service"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/database"
	"github.com/bingooyong/ops-scaffold-framework/manager/pkg/jwt"
	pb "github.com/bingooyong/ops-scaffold-framework/manager/pkg/proto"
	daemonpb "github.com/bingooyong/ops-scaffold-framework/manager/pkg/proto/daemon"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var (
	configFile = flag.String("config", "configs/manager.yaml", "配置文件路径")
)

func main() {
	flag.Parse()

	// 1. 加载配置
	cfg, err := config.Load(*configFile)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	log, err := logger.Init(&cfg.Log)
	if err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync() // 忽略日志同步错误，程序退出时无法处理
	}()

	log.Info("Manager starting...",
		zap.String("version", "0.1.0"),
		zap.String("mode", cfg.Server.Mode),
	)

	// 3. 初始化数据库
	db, err := database.Init(&cfg.Database, log)
	if err != nil {
		log.Fatal("Failed to init database", zap.Error(err))
	}

	// 自动迁移数据库表结构
	if err := database.AutoMigrate(db, log); err != nil {
		log.Fatal("Failed to migrate database", zap.Error(err))
	}
	log.Info("Database migrated successfully")

	// 4. 初始化JWT管理器
	jwtManager := jwt.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.Issuer,
		cfg.JWT.ExpireTime,
	)

	// 5. 初始化Repository层
	userRepo := repository.NewUserRepository(db)
	nodeRepo := repository.NewNodeRepository(db)
	metricsRepo := repository.NewMetricsRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	versionRepo := repository.NewVersionRepository(db)
	auditRepo := repository.NewAuditLogRepository(db)
	agentRepo := repository.NewAgentRepository(db)

	// 6. 初始化Daemon客户端连接池
	daemonPool := grpcserver.NewDaemonClientPool(log)

	// 7. 初始化Service层
	authService := service.NewAuthService(userRepo, auditRepo, jwtManager, log)
	nodeService := service.NewNodeService(nodeRepo, auditRepo, log)
	metricsService := service.NewMetricsService(metricsRepo, log)
	taskService := service.NewTaskService(taskRepo, nodeRepo, auditRepo, log)
	versionService := service.NewVersionService(versionRepo, auditRepo, log)
	agentService := service.NewAgentService(agentRepo, nodeRepo, daemonPool, log)

	// 避免编译器警告
	_ = taskService
	_ = versionService

	// 8. 初始化Handler层
	authHandler := handler.NewAuthHandler(authService, log)
	nodeHandler := handler.NewNodeHandler(nodeService, log)
	metricsHandler := handler.NewMetricsHandler(metricsService, log)
	agentHandler := handler.NewAgentHandler(agentService, log)

	// 9. 初始化gRPC服务器
	grpcSrv := grpcserver.NewServer(nodeService, metricsService, log)

	// 9.1. 初始化 Metrics 清理服务
	metricsCleaner := service.NewMetricsCleaner(db, cfg.Metrics.RetentionDays, log)

	// 9.2. 初始化 cron 调度器
	var cronScheduler *cron.Cron

	// 9.2.1 Metrics 清理任务
	if cfg.Metrics.CleanupSchedule != "" {
		// 使用标准 cron 表达式（5 字段：分 时 日 月 周）
		// 如果需要秒级精度，可以使用 cron.WithSeconds() 并配置 6 字段表达式
		cronScheduler = cron.New()
		_, err = cronScheduler.AddFunc(cfg.Metrics.CleanupSchedule, func() {
			log.Info("starting scheduled metrics cleanup")
			if err := metricsCleaner.CleanExpiredPartitions(context.Background()); err != nil {
				log.Error("scheduled metrics cleanup failed", zap.Error(err))
			} else {
				log.Info("scheduled metrics cleanup completed")
			}
		})
		if err != nil {
			log.Fatal("failed to add metrics cleanup cron job", zap.Error(err))
		}
		log.Info("metrics cleanup cron job scheduled",
			zap.String("schedule", cfg.Metrics.CleanupSchedule),
			zap.Int("retention_days", cfg.Metrics.RetentionDays))
	}

	// 9.2.2 节点离线检测任务
	if cronScheduler == nil {
		cronScheduler = cron.New()
	}

	if cfg.Node.OfflineCheckSchedule != "" && cfg.Node.OfflineDurationMinutes > 0 {
		offlineDuration := time.Duration(cfg.Node.OfflineDurationMinutes) * time.Minute
		_, err = cronScheduler.AddFunc(cfg.Node.OfflineCheckSchedule, func() {
			log.Info("starting scheduled node offline check",
				zap.Int("offline_duration_minutes", cfg.Node.OfflineDurationMinutes))
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := nodeService.CheckOfflineNodes(ctx, offlineDuration); err != nil {
				log.Error("scheduled node offline check failed", zap.Error(err))
			} else {
				log.Info("scheduled node offline check completed")
			}
		})
		if err != nil {
			log.Fatal("failed to add node offline check cron job", zap.Error(err))
		}
		log.Info("node offline check cron job scheduled",
			zap.String("schedule", cfg.Node.OfflineCheckSchedule),
			zap.Int("offline_duration_minutes", cfg.Node.OfflineDurationMinutes))
	}

	// 10. 初始化Gin引擎
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()

	// 11. 注册全局中间件
	router.Use(middleware.Recovery(log))
	router.Use(middleware.Logger(log))
	router.Use(middleware.CORS())

	// 12. 注册路由
	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 公开API（无需认证）
	public := router.Group("/api/v1")
	{
		// 认证相关
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/register", authHandler.Register)
	}

	// 需要认证的API
	api := router.Group("/api/v1")
	api.Use(middleware.JWTAuth(jwtManager))
	api.Use(middleware.Audit(auditRepo, log))
	{
		// 用户相关
		auth := api.Group("/auth")
		{
			auth.GET("/profile", authHandler.GetProfile)
			auth.POST("/change-password", authHandler.ChangePassword)
		}

		// Agent管理相关（必须在节点路由之前注册，避免路由冲突）
		agents := api.Group("/nodes/:node_id/agents")
		{
			agents.GET("", agentHandler.List)
			agents.POST("/sync", agentHandler.Sync) // 手动同步Agent状态
			agents.POST("/:agent_id/operate", agentHandler.Operate)
			agents.GET("/:agent_id/logs", agentHandler.GetLogs)
			agents.GET("/:agent_id/metrics", agentHandler.GetMetrics)
		}

		// 监控指标相关
		metrics := api.Group("/metrics")
		{
			metrics.GET("/nodes/:node_id/latest", metricsHandler.GetLatestMetrics)
			metrics.GET("/nodes/:node_id/:type/history", metricsHandler.GetMetricsHistory)
			metrics.GET("/nodes/:node_id/summary", metricsHandler.GetMetricsSummary)
			metrics.GET("/cluster/overview", metricsHandler.GetClusterOverview)
		}

		// 节点相关（必须在 agents 路由之后注册，避免路由冲突）
		nodes := api.Group("/nodes")
		{
			nodes.GET("", nodeHandler.List)
			nodes.GET("/statistics", nodeHandler.GetStatistics)
			nodes.GET("/:node_id", nodeHandler.Get) // 使用 :node_id 统一参数名，避免与 agents 路由冲突
		}

		// 管理员相关（需要管理员权限）
		admin := api.Group("/admin")
		admin.Use(middleware.RequireAdmin())
		{
			// 用户管理
			admin.GET("/users", authHandler.ListUsers)
			admin.POST("/users/:id/disable", authHandler.DisableUser)
			admin.POST("/users/:id/enable", authHandler.EnableUser)

			// 节点管理
			admin.DELETE("/nodes/:id", nodeHandler.Delete)
		}
	}

	// 13. 启动HTTP服务器
	httpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: router,
	}

	go func() {
		log.Info("HTTP server starting", zap.String("addr", httpAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// 14. 启动gRPC服务器
	grpcAddr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal("Failed to listen gRPC", zap.Error(err))
	}

	// 配置gRPC服务端keepalive参数(用于Daemon到Manager的连接)
	keepaliveParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,  // 连接空闲5分钟后关闭
		MaxConnectionAge:      30 * time.Minute, // 连接最长生命周期30分钟
		MaxConnectionAgeGrace: 5 * time.Second,  // 关闭前宽限期5秒
		Time:                  60 * time.Second, // 每60秒检查一次客户端keepalive
		Timeout:               20 * time.Second, // keepalive超时20秒
	}
	keepaliveEnforcementPolicy := keepalive.EnforcementPolicy{
		MinTime:             20 * time.Second, // 客户端最小ping间隔(Daemon客户端配置为30秒)
		PermitWithoutStream: true,             // 允许无流时发送ping
	}

	grpcServerInstance := grpc.NewServer(
		grpc.KeepaliveParams(keepaliveParams),
		grpc.KeepaliveEnforcementPolicy(keepaliveEnforcementPolicy),
		grpc.MaxRecvMsgSize(10*1024*1024), // 10MB 最大接收消息
		grpc.MaxSendMsgSize(10*1024*1024), // 10MB 最大发送消息
		grpc.InitialWindowSize(1<<20),     // 1MB 初始窗口
		grpc.InitialConnWindowSize(1<<20), // 1MB 连接窗口
		grpc.UnaryInterceptor(grpcserver.UnaryServerInterceptor(log)),
	)
	pb.RegisterManagerServiceServer(grpcServerInstance, grpcSrv)

	// 注册DaemonService服务器(用于接收Daemon上报的Agent状态)
	daemonSrv := grpcserver.NewDaemonServer(agentService, log)
	daemonpb.RegisterDaemonServiceServer(grpcServerInstance, daemonSrv)

	go func() {
		log.Info("gRPC server starting", zap.String("addr", grpcAddr))
		if err := grpcServerInstance.Serve(grpcListener); err != nil {
			log.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// 14.1. 启动 cron 调度器（在 HTTP 服务器启动后）
	if cronScheduler != nil {
		cronScheduler.Start()
		log.Info("cron scheduler started")
	}

	// 15. 等待信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Manager shutting down...")

	// 16. 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown failed", zap.Error(err))
	}

	// 关闭gRPC服务器
	grpcServerInstance.GracefulStop()
	log.Info("gRPC server stopped")

	// 关闭Daemon客户端连接池
	daemonPool.CloseAll()
	log.Info("Daemon client pool closed")

	// 停止 cron 调度器
	if cronScheduler != nil {
		cronScheduler.Stop()
		log.Info("cron scheduler stopped")
	}

	// 关闭数据库连接
	if err := database.Close(); err != nil {
		log.Error("Database close failed", zap.Error(err))
	}

	log.Info("Manager stopped")
}
