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
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
	defer logger.Sync()

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
	if err := database.AutoMigrate(db); err != nil {
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

	// 6. 初始化Service层
	authService := service.NewAuthService(userRepo, auditRepo, jwtManager, log)
	nodeService := service.NewNodeService(nodeRepo, auditRepo, log)
	metricsService := service.NewMetricsService(metricsRepo, log)
	taskService := service.NewTaskService(taskRepo, nodeRepo, auditRepo, log)
	versionService := service.NewVersionService(versionRepo, auditRepo, log)

	// 避免编译器警告
	_ = taskService
	_ = versionService

	// 7. 初始化Handler层
	authHandler := handler.NewAuthHandler(authService, log)
	nodeHandler := handler.NewNodeHandler(nodeService, log)

	// 8. 初始化gRPC服务器
	grpcSrv := grpcserver.NewServer(nodeService, metricsService, log)

	// 9. 初始化Gin引擎
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()

	// 10. 注册全局中间件
	router.Use(middleware.Recovery(log))
	router.Use(middleware.Logger(log))
	router.Use(middleware.CORS())

	// 11. 注册路由
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

		// 节点相关
		nodes := api.Group("/nodes")
		{
			nodes.GET("", nodeHandler.List)
			nodes.GET("/:id", nodeHandler.Get)
			nodes.GET("/statistics", nodeHandler.GetStatistics)
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

	// 12. 启动HTTP服务器
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

	// 13. 启动gRPC服务器
	grpcAddr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal("Failed to listen gRPC", zap.Error(err))
	}

	grpcServerInstance := grpc.NewServer()
	pb.RegisterManagerServiceServer(grpcServerInstance, grpcSrv)

	go func() {
		log.Info("gRPC server starting", zap.String("addr", grpcAddr))
		if err := grpcServerInstance.Serve(grpcListener); err != nil {
			log.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// 14. 等待信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Manager shutting down...")

	// 15. 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown failed", zap.Error(err))
	}

	// 关闭gRPC服务器
	grpcServerInstance.GracefulStop()
	log.Info("gRPC server stopped")

	// 关闭数据库连接
	if err := database.Close(); err != nil {
		log.Error("Database close failed", zap.Error(err))
	}

	log.Info("Manager stopped")
}
