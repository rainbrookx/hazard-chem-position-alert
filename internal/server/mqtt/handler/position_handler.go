package handler

import (
	"context"
	"encoding/json"
	"log/slog"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// PositionHandler 处理从 MQTT position/cleaned topic 收到的清洗定位数据。
type PositionHandler struct {
	engine *engine.Engine
}

// NewPositionHandler 创建定位数据处理器。
func NewPositionHandler(eng *engine.Engine) *PositionHandler {
	return &PositionHandler{engine: eng}
}

// Handle 处理 MQTT 消息：解析 JSON → 调用 engine.Process()。
// 符合 Mochi MQTT v2 InlineSubFn 签名（无返回值）。
func (h *PositionHandler) Handle(cl *mqtt.Client, sub packets.Subscription, pk packets.Packet) {
	slog.Debug("收到 MQTT 定位数据", "topic", pk.TopicName, "payload_len", len(pk.Payload))

	var pos model.PositionData
	if err := json.Unmarshal(pk.Payload, &pos); err != nil {
		slog.Warn("MQTT 载荷 JSON 解析失败", "topic", pk.TopicName, "error", err)
		return
	}

	ctx := context.Background()
	h.engine.Process(ctx, pos)
}
