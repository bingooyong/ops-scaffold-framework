package grpc

import (
	"context"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/service"
	daemonpb "github.com/bingooyong/ops-scaffold-framework/manager/pkg/proto/daemon"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DaemonServer DaemonService gRPC服务器
// 用于接收Daemon上报的Agent状态等信息
type DaemonServer struct {
	daemonpb.UnimplementedDaemonServiceServer
	agentService *service.AgentService
	logger       *zap.Logger
}

// NewDaemonServer 创建DaemonService服务器实例
func NewDaemonServer(
	agentService *service.AgentService,
	logger *zap.Logger,
) *DaemonServer {
	return &DaemonServer{
		agentService: agentService,
		logger:       logger,
	}
}

// SyncAgentStates 同步Agent状态
func (s *DaemonServer) SyncAgentStates(ctx context.Context, req *daemonpb.SyncAgentStatesRequest) (*daemonpb.SyncAgentStatesResponse, error) {
	// 验证请求参数
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	if len(req.States) == 0 {
		return &daemonpb.SyncAgentStatesResponse{
			Success: true,
			Message: "no states to sync",
		}, nil
	}

	s.logger.Info("syncing agent states",
		zap.String("node_id", req.NodeId),
		zap.Int("count", len(req.States)))

	// 调用AgentService同步状态
	if err := s.agentService.SyncAgentStates(ctx, req.NodeId, req.States); err != nil {
		s.logger.Error("failed to sync agent states",
			zap.String("node_id", req.NodeId),
			zap.Error(err))
		return &daemonpb.SyncAgentStatesResponse{
			Success: false,
			Message: "failed to sync agent states: " + err.Error(),
		}, nil
	}

	s.logger.Info("agent states synced successfully",
		zap.String("node_id", req.NodeId),
		zap.Int("count", len(req.States)))

	return &daemonpb.SyncAgentStatesResponse{
		Success: true,
		Message: "states synced successfully",
	}, nil
}
