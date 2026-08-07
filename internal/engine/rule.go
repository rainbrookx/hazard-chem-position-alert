package engine

import (
	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine/geom"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// AlertEvent 引擎内部告警事件，在持久化和 gRPC 推送前使用。
type AlertEvent struct {
	AlertID       string
	AlertType     alertv1.AlertType
	Severity      alertv1.Severity
	TriggerTimeMs int64
	PersonIDs     []string
	X             float64
	Y             float64
	ZoneID        string
	RuleID        string
	Description   string
	Priority      bool // true = 优先通道（一键报警等）
}

// ToProtobuf 将内部 AlertEvent 转换为 protobuf AlertEvent。
func (e *AlertEvent) ToProtobuf() *alertv1.AlertEvent {
	return &alertv1.AlertEvent{
		AlertId:       e.AlertID,
		AlertType:     e.AlertType,
		Severity:      e.Severity,
		TriggerTimeMs: e.TriggerTimeMs,
		PersonIds:     e.PersonIDs,
		X:             e.X,
		Y:             e.Y,
		ZoneId:        e.ZoneID,
		RuleId:        e.RuleID,
		Description:   e.Description,
	}
}

// Rule 告警规则接口。所有 7 种告警类型均实现此接口。
type Rule interface {
	// ID 返回规则唯一标识。
	ID() string

	// Type 返回此规则对应的告警类型。
	Type() alertv1.AlertType

	// Evaluate 评估单条定位数据是否触发告警。
	// pos: 当前定位数据
	// fences: 当前活跃围栏列表
	// state: 状态跟踪器（用于去抖、去重、计时）
	// gridHash: 网格分桶（用于聚集检测等空间查询）
	// 返回触发的告警事件列表（同一条数据可能触发多个告警）。
	Evaluate(pos model.PositionData, fences []model.Geofence, state *StateTracker, gridHash *geom.GridHash) ([]AlertEvent, error)
}
