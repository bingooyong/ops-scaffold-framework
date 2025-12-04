package grpc

import (
	"context"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/bingooyong/ops-scaffold-framework/manager/internal/service"
	pb "github.com/bingooyong/ops-scaffold-framework/manager/pkg/proto"
	"go.uber.org/zap"
)

// Server gRPC服务器
type Server struct {
	pb.UnimplementedManagerServiceServer
	nodeService    service.NodeService
	metricsService service.MetricsService
	logger         *zap.Logger
}

// NewServer 创建gRPC服务器实例
func NewServer(
	nodeService service.NodeService,
	metricsService service.MetricsService,
	logger *zap.Logger,
) *Server {
	return &Server{
		nodeService:    nodeService,
		metricsService: metricsService,
		logger:         logger,
	}
}

// RegisterNode 节点注册
func (s *Server) RegisterNode(ctx context.Context, req *pb.RegisterNodeRequest) (*pb.RegisterNodeResponse, error) {
	s.logger.Info("node registration request",
		zap.String("node_id", req.NodeId),
		zap.String("hostname", req.Hostname),
		zap.String("ip", req.Ip),
	)

	// 构建节点模型
	now := time.Now()
	node := &model.Node{
		NodeID:        req.NodeId,
		Hostname:      req.Hostname,
		IP:            req.Ip,
		OS:            req.Os,
		Arch:          req.Arch,
		Labels:        req.Labels,
		DaemonVersion: req.DaemonVersion,
		AgentVersion:  req.AgentVersion,
		Status:        "online",
		RegisterAt:    now,
	}

	// 注册节点
	if err := s.nodeService.Register(ctx, node); err != nil {
		s.logger.Error("failed to register node", zap.Error(err))
		return &pb.RegisterNodeResponse{
			Success: false,
			Message: "节点注册失败: " + err.Error(),
		}, nil
	}

	s.logger.Info("node registered successfully", zap.String("node_id", req.NodeId))

	return &pb.RegisterNodeResponse{
		Success: true,
		Message: "节点注册成功",
	}, nil
}

// Heartbeat 心跳上报
func (s *Server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	s.logger.Debug("heartbeat received",
		zap.String("node_id", req.NodeId),
		zap.Int64("timestamp", req.Timestamp),
	)

	// 处理心跳
	if err := s.nodeService.Heartbeat(ctx, req.NodeId); err != nil {
		s.logger.Warn("failed to process heartbeat",
			zap.String("node_id", req.NodeId),
			zap.Error(err),
		)
		return &pb.HeartbeatResponse{
			Success: false,
			Message: "心跳处理失败: " + err.Error(),
		}, nil
	}

	return &pb.HeartbeatResponse{
		Success: true,
		Message: "心跳已接收",
	}, nil
}

// ReportMetrics 指标上报
func (s *Server) ReportMetrics(ctx context.Context, req *pb.ReportMetricsRequest) (*pb.ReportMetricsResponse, error) {
	s.logger.Debug("metrics report received",
		zap.String("node_id", req.NodeId),
		zap.Int("count", len(req.Metrics)),
	)

	// 批量创建指标记录
	var metrics []*model.Metrics
	for _, m := range req.Metrics {
		// 转换values map
		values := make(map[string]interface{})
		for k, v := range m.Values {
			values[k] = v
		}

		metrics = append(metrics, &model.Metrics{
			NodeID:    req.NodeId,
			Type:      m.Type,
			Timestamp: time.Unix(m.Timestamp, 0),
			Values:    values,
		})
	}

	// 批量保存指标
	if err := s.metricsService.BatchCreate(ctx, metrics); err != nil {
		s.logger.Error("failed to save metrics",
			zap.String("node_id", req.NodeId),
			zap.Error(err),
		)
		return &pb.ReportMetricsResponse{
			Success: false,
			Message: "指标保存失败: " + err.Error(),
		}, nil
	}

	s.logger.Debug("metrics saved successfully",
		zap.String("node_id", req.NodeId),
		zap.Int("count", len(metrics)),
	)

	return &pb.ReportMetricsResponse{
		Success: true,
		Message: "指标已保存",
	}, nil
}
