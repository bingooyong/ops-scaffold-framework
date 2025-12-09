package agent

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// MetadataUpdateRequest 元数据更新请求
type MetadataUpdateRequest struct {
	AgentID  string
	Metadata *AgentMetadata
	IsUpdate bool // true=更新, false=保存
}

// AsyncMetadataWriter 异步元数据写入器
// 使用channel队列避免大量并发IO导致的锁竞争
type AsyncMetadataWriter struct {
	store      MetadataStore
	updateChan chan *MetadataUpdateRequest
	wg         sync.WaitGroup
	stopChan   chan struct{}
	logger     *zap.Logger
}

// NewAsyncMetadataWriter 创建异步元数据写入器
func NewAsyncMetadataWriter(store MetadataStore, logger *zap.Logger) *AsyncMetadataWriter {
	writer := &AsyncMetadataWriter{
		store:      store,
		updateChan: make(chan *MetadataUpdateRequest, 100), // 缓冲100个请求
		stopChan:   make(chan struct{}),
		logger:     logger,
	}
	
	// 启动写入goroutine
	writer.wg.Add(1)
	go writer.writeLoop()
	
	return writer
}

// writeLoop 写入循环
func (w *AsyncMetadataWriter) writeLoop() {
	defer w.wg.Done()
	
	// 批量写入缓存
	batch := make([]*MetadataUpdateRequest, 0, 10)
	ticker := time.NewTicker(500 * time.Millisecond) // 每500ms刷新一次
	defer ticker.Stop()
	
	flush := func() {
		if len(batch) == 0 {
			return
		}
		
		// 处理批量写入
		for _, req := range batch {
			var err error
			if req.IsUpdate {
				err = w.store.UpdateMetadata(req.AgentID, req.Metadata)
			} else {
				err = w.store.SaveMetadata(req.AgentID, req.Metadata)
			}
			
			if err != nil {
				w.logger.Warn("async metadata write failed",
					zap.String("agent_id", req.AgentID),
					zap.Bool("is_update", req.IsUpdate),
					zap.Error(err))
			}
		}
		
		w.logger.Debug("flushed metadata batch",
			zap.Int("count", len(batch)))
		
		batch = batch[:0] // 清空batch
	}
	
	for {
		select {
		case req := <-w.updateChan:
			batch = append(batch, req)
			
			// 如果batch满了，立即刷新
			if len(batch) >= 10 {
				flush()
			}
			
		case <-ticker.C:
			// 定期刷新
			flush()
			
		case <-w.stopChan:
			// 停止前刷新所有待处理的请求
			// 先处理当前batch
			flush()
			
			// 再处理channel中剩余的请求
			for {
				select {
				case req := <-w.updateChan:
					batch = append(batch, req)
					if len(batch) >= 10 {
						flush()
					}
				default:
					// channel已空，最后刷新一次
					flush()
					return
				}
			}
		}
	}
}

// SaveMetadata 异步保存元数据
func (w *AsyncMetadataWriter) SaveMetadata(agentID string, metadata *AgentMetadata) {
	select {
	case w.updateChan <- &MetadataUpdateRequest{
		AgentID:  agentID,
		Metadata: metadata,
		IsUpdate: false,
	}:
		// 成功发送
	default:
		// channel满了，记录警告但不阻塞
		w.logger.Warn("metadata update channel full, dropping request",
			zap.String("agent_id", agentID))
	}
}

// UpdateMetadata 异步更新元数据
func (w *AsyncMetadataWriter) UpdateMetadata(agentID string, metadata *AgentMetadata) {
	select {
	case w.updateChan <- &MetadataUpdateRequest{
		AgentID:  agentID,
		Metadata: metadata,
		IsUpdate: true,
	}:
		// 成功发送
	default:
		// channel满了，记录警告但不阻塞
		w.logger.Warn("metadata update channel full, dropping request",
			zap.String("agent_id", agentID))
	}
}

// Stop 停止异步写入器
func (w *AsyncMetadataWriter) Stop() {
	close(w.stopChan)
	w.wg.Wait()
	w.logger.Info("async metadata writer stopped")
}
