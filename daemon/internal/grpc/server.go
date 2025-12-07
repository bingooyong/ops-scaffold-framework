package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/daemon/internal/agent"
	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server gRPC服务端实现
type Server struct {
	proto.UnimplementedDaemonServiceServer

	multiAgentManager *agent.MultiAgentManager
	resourceMonitor   *agent.ResourceMonitor
	logger            *zap.Logger
}

// NewServer 创建新的gRPC服务端实例
func NewServer(
	multiAgentManager *agent.MultiAgentManager,
	resourceMonitor *agent.ResourceMonitor,
	logger *zap.Logger,
) *Server {
	return &Server{
		multiAgentManager: multiAgentManager,
		resourceMonitor:   resourceMonitor,
		logger:            logger,
	}
}

// ListAgents 列举所有Agent
func (s *Server) ListAgents(ctx context.Context, req *proto.ListAgentsRequest) (*proto.ListAgentsResponse, error) {
	// 获取所有Agent实例
	instances := s.multiAgentManager.ListAgents()

	// 转换为protobuf消息
	agents := make([]*proto.AgentInfo, 0, len(instances))
	for _, instance := range instances {
		info := instance.GetInfo()

		// 获取元数据
		metadata, err := s.multiAgentManager.GetAgentMetadata(info.ID)
		if err != nil {
			// 如果元数据不存在，使用默认值
			s.logger.Debug("metadata not found for agent",
				zap.String("agent_id", info.ID),
				zap.Error(err))
			metadata = nil
		}

		agentInfo := convertAgentInfoToProto(info, metadata)
		agents = append(agents, agentInfo)
	}

	s.logger.Info("listed agents",
		zap.Int("count", len(agents)))

	return &proto.ListAgentsResponse{
		Agents: agents,
	}, nil
}

// OperateAgent 操作Agent(启动/停止/重启)
func (s *Server) OperateAgent(ctx context.Context, req *proto.AgentOperationRequest) (*proto.AgentOperationResponse, error) {
	// 验证请求参数
	if req.AgentId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	// 验证操作类型
	validOperations := map[string]bool{
		"start":   true,
		"stop":    true,
		"restart": true,
	}
	if !validOperations[req.Operation] {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid operation: %s, must be one of: start, stop, restart", req.Operation))
	}

	// 检查Agent是否存在
	instance := s.multiAgentManager.GetAgent(req.AgentId)
	if instance == nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("agent not found: %s", req.AgentId))
	}

	// 根据操作类型执行相应操作
	var err error
	switch req.Operation {
	case "start":
		err = s.multiAgentManager.StartAgent(ctx, req.AgentId)
	case "stop":
		err = s.multiAgentManager.StopAgent(ctx, req.AgentId, true)
	case "restart":
		err = s.multiAgentManager.RestartAgent(ctx, req.AgentId)
	default:
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid operation: %s", req.Operation))
	}

	if err != nil {
		s.logger.Error("failed to operate agent",
			zap.String("agent_id", req.AgentId),
			zap.String("operation", req.Operation),
			zap.Error(err))
		return &proto.AgentOperationResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, status.Error(codes.Internal, err.Error())
	}

	s.logger.Info("agent operation completed",
		zap.String("agent_id", req.AgentId),
		zap.String("operation", req.Operation),
		zap.Bool("success", true))

	return &proto.AgentOperationResponse{
		Success: true,
	}, nil
}

// GetAgentMetrics 获取Agent资源使用指标
func (s *Server) GetAgentMetrics(ctx context.Context, req *proto.AgentMetricsRequest) (*proto.AgentMetricsResponse, error) {
	// 验证请求参数
	if req.AgentId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	// 验证时间范围
	durationSeconds := req.DurationSeconds
	if durationSeconds <= 0 {
		durationSeconds = 3600 // 默认1小时
	}

	// 检查Agent是否存在
	instance := s.multiAgentManager.GetAgent(req.AgentId)
	if instance == nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("agent not found: %s", req.AgentId))
	}

	// 获取资源历史数据
	duration := time.Duration(durationSeconds) * time.Second
	dataPoints, err := s.resourceMonitor.GetResourceHistory(req.AgentId, duration)
	if err != nil {
		s.logger.Error("failed to get resource history",
			zap.String("agent_id", req.AgentId),
			zap.Error(err))
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get resource history: %v", err))
	}

	// 转换为protobuf消息
	protoDataPoints := make([]*proto.ResourceDataPoint, 0, len(dataPoints))
	for _, dp := range dataPoints {
		protoDataPoints = append(protoDataPoints, convertResourceDataPointToProto(&dp))
	}

	s.logger.Debug("retrieved agent metrics",
		zap.String("agent_id", req.AgentId),
		zap.Int("data_points", len(protoDataPoints)),
		zap.Int64("duration_seconds", durationSeconds))

	return &proto.AgentMetricsResponse{
		AgentId:    req.AgentId,
		DataPoints: protoDataPoints,
	}, nil
}

// SyncAgentStates 同步Agent状态(用于Daemon向Manager上报状态)
func (s *Server) SyncAgentStates(ctx context.Context, req *proto.SyncAgentStatesRequest) (*proto.SyncAgentStatesResponse, error) {
	// 验证请求参数
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	// 空状态列表是合法的，表示该节点没有Agent
	if len(req.States) == 0 {
		s.logger.Info("synced agent states (empty)",
			zap.String("node_id", req.NodeId),
			zap.Int("total_states", 0))
		return &proto.SyncAgentStatesResponse{
			Success: true,
			Message: "synced 0 agent states (empty list)",
		}, nil
	}

	// 遍历状态列表，更新每个Agent的状态
	// 注意：当前实现仅记录日志，完整的状态同步逻辑在Task 3.4中实现
	syncedCount := 0
	for _, state := range req.States {
		instance := s.multiAgentManager.GetAgent(state.AgentId)
		if instance != nil {
			// Agent存在，可以在这里更新状态（如果需要）
			// 当前实现仅记录日志
			syncedCount++
		}
	}

	s.logger.Info("synced agent states",
		zap.String("node_id", req.NodeId),
		zap.Int("total_states", len(req.States)),
		zap.Int("synced_count", syncedCount))

	return &proto.SyncAgentStatesResponse{
		Success: true,
		Message: fmt.Sprintf("synced %d agent states", syncedCount),
	}, nil
}

// convertAgentInfoToProto 将AgentInfo和AgentMetadata转换为protobuf AgentInfo消息
func convertAgentInfoToProto(info *agent.AgentInfo, metadata *agent.AgentMetadata) *proto.AgentInfo {
	protoInfo := &proto.AgentInfo{
		Id:            info.ID,
		Type:          string(info.Type),
		Status:        string(info.GetStatus()),
		Pid:           int32(info.GetPID()),
		Version:       "",
		StartTime:     0,
		RestartCount:  int32(info.GetRestartCount()),
		LastHeartbeat: 0,
	}

	// 从元数据获取额外信息
	if metadata != nil {
		if metadata.Version != "" {
			protoInfo.Version = metadata.Version
		}
		if !metadata.StartTime.IsZero() {
			protoInfo.StartTime = metadata.StartTime.Unix()
		}
		if !metadata.LastHeartbeat.IsZero() {
			protoInfo.LastHeartbeat = metadata.LastHeartbeat.Unix()
		}
		protoInfo.RestartCount = int32(metadata.RestartCount)
	}

	return protoInfo
}

// convertResourceDataPointToProto 将ResourceDataPoint转换为protobuf ResourceDataPoint消息
func convertResourceDataPointToProto(dp *agent.ResourceDataPoint) *proto.ResourceDataPoint {
	return &proto.ResourceDataPoint{
		Timestamp:      dp.Timestamp.Unix(),
		Cpu:            dp.CPU,
		MemoryRss:      dp.MemoryRSS,
		MemoryVms:      dp.MemoryVMS,
		DiskReadBytes:  dp.DiskReadBytes,
		DiskWriteBytes: dp.DiskWriteBytes,
		OpenFiles:      int32(dp.OpenFiles),
	}
}
