package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/bingooyong/ops-scaffold-framework/daemon/pkg/types"
	"go.uber.org/zap"
)

// HeartbeatReceiver 心跳接收器
type HeartbeatReceiver struct {
	socketPath string
	listener   net.Listener
	healthChecker *HealthChecker
	logger     *zap.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewHeartbeatReceiver 创建心跳接收器
func NewHeartbeatReceiver(socketPath string, healthChecker *HealthChecker, logger *zap.Logger) *HeartbeatReceiver {
	ctx, cancel := context.WithCancel(context.Background())
	return &HeartbeatReceiver{
		socketPath:    socketPath,
		healthChecker: healthChecker,
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start 启动心跳接收
func (r *HeartbeatReceiver) Start() error {
	// 清理旧的socket文件
	os.Remove(r.socketPath)

	// 创建Unix socket监听器
	listener, err := net.Listen("unix", r.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}
	r.listener = listener

	r.logger.Info("heartbeat receiver started", zap.String("socket", r.socketPath))

	r.wg.Add(1)
	go r.acceptLoop()

	return nil
}

// Stop 停止心跳接收
func (r *HeartbeatReceiver) Stop() {
	r.logger.Info("stopping heartbeat receiver")
	r.cancel()

	if r.listener != nil {
		r.listener.Close()
	}

	r.wg.Wait()

	// 清理socket文件
	os.Remove(r.socketPath)

	r.logger.Info("heartbeat receiver stopped")
}

// acceptLoop 接受连接循环
func (r *HeartbeatReceiver) acceptLoop() {
	defer r.wg.Done()

	for {
		conn, err := r.listener.Accept()
		if err != nil {
			select {
			case <-r.ctx.Done():
				return
			default:
				r.logger.Error("failed to accept connection", zap.Error(err))
				continue
			}
		}

		r.wg.Add(1)
		go r.handleConnection(conn)
	}
}

// handleConnection 处理连接
func (r *HeartbeatReceiver) handleConnection(conn net.Conn) {
	defer r.wg.Done()
	defer conn.Close()

	decoder := json.NewDecoder(conn)

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		var hb types.Heartbeat
		if err := decoder.Decode(&hb); err != nil {
			if err.Error() != "EOF" {
				r.logger.Error("failed to decode heartbeat", zap.Error(err))
			}
			return
		}

		// 将心跳传递给健康检查器
		r.healthChecker.ReceiveHeartbeat(&hb)
	}
}
