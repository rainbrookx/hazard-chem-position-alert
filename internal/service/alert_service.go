package service

import (
	"log/slog"
	"sync"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine"
	"google.golang.org/grpc"
)

// subscriber 表示一个 gRPC 告警推送订阅者。
type subscriber struct {
	stream   grpc.ServerStreamingServer[alertv1.AlertEvent]
	clientID string
}

// AlertPushService 实现 gRPC AlertPushServiceServer。
// 管理订阅者注册表并广播告警事件。
type AlertPushService struct {
	alertv1.UnimplementedAlertPushServiceServer
	mu          sync.RWMutex
	subscribers map[string]*subscriber // clientID → subscriber
}

// NewAlertPushService 创建告警推送服务。
func NewAlertPushService() *AlertPushService {
	return &AlertPushService{
		subscribers: make(map[string]*subscriber),
	}
}

// SubscribeAlerts Data Preprocessor 调用此 RPC 建立告警推送流（server-streaming）。
func (s *AlertPushService) SubscribeAlerts(req *alertv1.SubscribeRequest, stream grpc.ServerStreamingServer[alertv1.AlertEvent]) error {
	sub := &subscriber{
		stream:   stream,
		clientID: req.ClientId,
	}

	s.mu.Lock()
	s.subscribers[req.ClientId] = sub
	s.mu.Unlock()

	slog.Info("告警推送订阅者已连接", "client_id", req.ClientId)

	// 阻塞直到客户端断开连接
	<-stream.Context().Done()

	s.mu.Lock()
	delete(s.subscribers, req.ClientId)
	s.mu.Unlock()

	slog.Info("告警推送订阅者已断开", "client_id", req.ClientId)
	return nil
}

// Broadcast 向所有订阅者广播告警事件。
func (s *AlertPushService) Broadcast(event engine.AlertEvent) {
	pbEvent := event.ToProtobuf()

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sub := range s.subscribers {
		if err := sub.stream.Send(pbEvent); err != nil {
			slog.Error("告警推送失败", "client_id", sub.clientID, "alert_id", event.AlertID, "error", err)
		}
	}
}

// SubscriberCount 返回当前订阅者数量。
func (s *AlertPushService) SubscriberCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscribers)
}
