package engine

import (
	"fmt"
	"time"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine/geom"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// StationaryRule 静止报警规则。
// 人员在受限/围栏区域内长时间不移动触发告警。
// FR-005: 恢复需持续移动超阈值达 stationary_recovery_seconds。
type StationaryRule struct{}

// NewStationaryRule 创建静止报警规则。
func NewStationaryRule() *StationaryRule {
	return &StationaryRule{}
}

func (r *StationaryRule) ID() string              { return "stationary" }
func (r *StationaryRule) Type() alertv1.AlertType { return alertv1.AlertType_ALERT_TYPE_STATIONARY }

func (r *StationaryRule) Evaluate(pos model.PositionData, fences []model.Geofence, state *StateTracker, _ *geom.GridHash) ([]AlertEvent, error) {
	var events []AlertEvent

	for _, fence := range fences {
		if !fence.IsActive || fence.StationarySeconds <= 0 {
			continue
		}
		if !state.IsPersonInZone(pos.PersonID, fence.ZoneID) {
			continue
		}

		threshold := fence.StationaryThresholdMeters
		if threshold <= 0 {
			threshold = 2.0 // 默认位移阈值 2 米
		}
		recoverySecs := fence.StationaryRecoverySeconds
		if recoverySecs <= 0 {
			recoverySecs = 3 // 默认恢复去抖 3 秒
		}

		prevPos, hasPrev := state.GetPreviousPosition(pos.PersonID)
		if !hasPrev {
			state.UpdatePosition(pos)
			continue
		}

		moved := geom.EuclideanDistance(
			geom.Point{X: pos.X, Y: pos.Y},
			geom.Point{X: prevPos.X, Y: prevPos.Y},
		) >= threshold

		state.RecordMovement(pos.PersonID, fence.ZoneID, moved, threshold, time.Now())

		if moved {
			// 检查恢复去抖：持续移动超过 recoverySecs 才重置
			movingDuration := state.GetMovingDuration(pos.PersonID, fence.ZoneID)
			if movingDuration >= time.Duration(recoverySecs)*time.Second {
				// 条件恢复
				state.ResetPersonalDedup(alertv1.AlertType_ALERT_TYPE_STATIONARY, pos.PersonID, fence.ZoneID)
			}
			continue
		}

		// 未移动 → 检查是否触发
		entryTime := state.GetEntryTime(pos.PersonID, fence.ZoneID)
		if entryTime.IsZero() {
			continue
		}

		elapsed := time.Since(entryTime)
		if elapsed >= time.Duration(fence.StationarySeconds)*time.Second {
			if state.IsFirstTriggerPersonal(alertv1.AlertType_ALERT_TYPE_STATIONARY, pos.PersonID, fence.ZoneID) {
				state.RecordTriggerPersonal(alertv1.AlertType_ALERT_TYPE_STATIONARY, pos.PersonID, fence.ZoneID, time.Now().UnixMilli())

				events = append(events, AlertEvent{
					AlertID:       NewULID(),
					AlertType:     alertv1.AlertType_ALERT_TYPE_STATIONARY,
					Severity:      alertv1.Severity_SEVERITY_ORANGE,
					TriggerTimeMs: time.Now().UnixMilli(),
					PersonIDs:     []string{pos.PersonID},
					X:             pos.X,
					Y:             pos.Y,
					ZoneID:        fence.ZoneID,
					RuleID:        r.ID(),
					Description:   fmt.Sprintf("人员 %s 在 %s 区域静止超过 %d 秒", pos.PersonID, fence.Name, fence.StationarySeconds),
				})
			}
		}
	}

	return events, nil
}
