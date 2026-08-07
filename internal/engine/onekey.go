package engine

import (
	"fmt"
	"sync"
	"time"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine/geom"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// OneKeyRule 一键报警规则。
// position.sos_flag == true 时立即触发，跳过常规队列，优先推送。
// FR-013: 优先通道 sub-500ms 延迟。
// 冷却期内同一终端不重复触发。
type OneKeyRule struct {
	mu              sync.Mutex
	lastTriggerTime map[string]int64 // key: "personID:terminalID" → 上次触发时间 ms
	cooldownSeconds int              // 去重冷却时间（秒），0 表示无冷却
}

// NewOneKeyRule 创建一键报警规则。
func NewOneKeyRule(cooldownSeconds int) *OneKeyRule {
	if cooldownSeconds <= 0 {
		cooldownSeconds = 60 // 默认 60 秒冷却
	}
	return &OneKeyRule{
		lastTriggerTime: make(map[string]int64),
		cooldownSeconds: cooldownSeconds,
	}
}

func (r *OneKeyRule) ID() string              { return "onekey" }
func (r *OneKeyRule) Type() alertv1.AlertType { return alertv1.AlertType_ALERT_TYPE_ONEKEY }

func (r *OneKeyRule) Evaluate(pos model.PositionData, _ []model.Geofence, _ *StateTracker, _ *geom.GridHash) ([]AlertEvent, error) {
	if !pos.SOSFlag {
		return nil, nil
	}

	dedupKey := fmt.Sprintf("%s:%s", pos.PersonID, pos.TerminalID)
	nowMs := time.Now().UnixMilli()

	r.mu.Lock()
	lastTime, exists := r.lastTriggerTime[dedupKey]
	// 检查冷却期
	if exists && r.cooldownSeconds > 0 {
		cooldownMs := int64(r.cooldownSeconds) * 1000
		if nowMs-lastTime < cooldownMs {
			r.mu.Unlock()
			return nil, nil // 冷却期内，抑制重复触发
		}
	}
	// 记录本次触发时间
	r.lastTriggerTime[dedupKey] = nowMs
	r.mu.Unlock()

	return []AlertEvent{{
		AlertID:       NewULID(),
		AlertType:     alertv1.AlertType_ALERT_TYPE_ONEKEY,
		Severity:      alertv1.Severity_SEVERITY_RED,
		TriggerTimeMs: nowMs,
		PersonIDs:     []string{pos.PersonID},
		X:             pos.X,
		Y:             pos.Y,
		ZoneID:        "", // 一键报警无特定区域
		RuleID:        r.ID(),
		Description:   fmt.Sprintf("终端 %s (%s) 触发一键报警", pos.TerminalID, pos.PersonID),
		Priority:      true, // 优先通道
	}}, nil
}
