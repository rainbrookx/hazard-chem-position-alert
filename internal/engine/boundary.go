package engine

import (
	"fmt"
	"time"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine/geom"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// BoundaryRule 越界报警规则。
// 检测人员进入禁止/受限围栏区域，触发越界告警。
// FR-006: 30 秒第二段检查，若仍在内则生成第二个独立事件。
type BoundaryRule struct {
	debounceSeconds int // 进入去抖窗口（秒）
}

// NewBoundaryRule 创建越界规则实例。
func NewBoundaryRule(debounceSeconds int) *BoundaryRule {
	if debounceSeconds <= 0 {
		debounceSeconds = 3
	}
	return &BoundaryRule{
		debounceSeconds: debounceSeconds,
	}
}

// ID 返回规则标识。
func (r *BoundaryRule) ID() string {
	return "boundary"
}

// Type 返回告警类型。
func (r *BoundaryRule) Type() alertv1.AlertType {
	return alertv1.AlertType_ALERT_TYPE_BOUNDARY
}

// Evaluate 评估越界告警条件。
// 对所有围栏类型追踪人员进出；边界告警仅对 FORBIDDEN/RESTRICTED 触发。
func (r *BoundaryRule) Evaluate(pos model.PositionData, fences []model.Geofence, state *StateTracker, _ *geom.GridHash) ([]AlertEvent, error) {
	var events []AlertEvent

	for _, fence := range fences {
		if !fence.IsActive || len(fence.Vertices) < 3 {
			continue
		}

		point := geom.Point{X: pos.X, Y: pos.Y}
		polygon := geom.Polygon(fence.Vertices)

		inZone := geom.PointInPolygon(point, polygon)
		wasInZone := state.IsPersonInZone(pos.PersonID, fence.ZoneID)

		// FR-005: 对所有围栏类型追踪人员进出（供超员/缺员/滞留等规则使用）
		if inZone && !wasInZone {
			state.RecordEntry(pos.PersonID, fence.ZoneID, pos.Timestamp)
		} else if !inZone && wasInZone {
			state.RecordExit(pos.PersonID, fence.ZoneID)
		}

		// 边界告警仅对 FORBIDDEN 和 RESTRICTED 类型围栏触发
		if fence.Type != model.FenceTypeForbidden && fence.Type != model.FenceTypeRestricted {
			continue
		}

		if inZone {
			if !wasInZone {
				continue // 去抖窗口内不触发（入口已在上面记录）
			}

			// 人员已在区域内 → 检查是否通过去抖窗口
			entryTime := state.GetEntryTime(pos.PersonID, fence.ZoneID)
			if entryTime.IsZero() {
				continue
			}

			elapsedMs := pos.Timestamp - entryTime.UnixMilli()
			if elapsedMs < int64(r.debounceSeconds)*1000 {
				// 去抖中，不触发
				continue
			}

			// 检查是否已触发过（去重 FR-008）
			if !state.IsFirstTriggerPersonal(alertv1.AlertType_ALERT_TYPE_BOUNDARY, pos.PersonID, fence.ZoneID) {
				// 已触发过 → 检查 FR-006 30s 第二段
				// 如果超过 30 秒仍在禁区 → 生成第二个独立事件
				if elapsedMs >= 30000 {
					// 记录第二段已触发（通过重设去重键的变体）
					secondKey := fmt.Sprintf("%d:%s:%s:second", int32(alertv1.AlertType_ALERT_TYPE_BOUNDARY), pos.PersonID, fence.ZoneID)
					// 使用不同的去重键来区分第一段和第二段
					if _, fired := state.getLastAlertPersonal(secondKey); !fired {
						state.setLastAlertPersonal(secondKey, pos.Timestamp)
						event := r.buildEvent(pos, fence, "第二段越界：人员仍在禁区内超过 30 秒")
						events = append(events, event)
					}
				}
				continue
			}

			// 首次触发
			state.RecordTriggerPersonal(alertv1.AlertType_ALERT_TYPE_BOUNDARY, pos.PersonID, fence.ZoneID, pos.Timestamp)

			event := r.buildEvent(pos, fence, fmt.Sprintf("人员 %s 越界进入 %s 区域", pos.PersonID, fence.Name))
			events = append(events, event)

		} else if wasInZone {
			// 人员离开禁/限区域 → 条件恢复
			state.ResetPersonalDedup(alertv1.AlertType_ALERT_TYPE_BOUNDARY, pos.PersonID, fence.ZoneID)
			// 也重置第二段
			secondKey := fmt.Sprintf("%d:%s:%s:second", int32(alertv1.AlertType_ALERT_TYPE_BOUNDARY), pos.PersonID, fence.ZoneID)
			state.deleteLastAlertPersonal(secondKey)
		}
	}

	return events, nil
}

func (r *BoundaryRule) buildEvent(pos model.PositionData, fence model.Geofence, description string) AlertEvent {
	return AlertEvent{
		AlertID:       NewULID(),
		AlertType:     alertv1.AlertType_ALERT_TYPE_BOUNDARY,
		Severity:      alertv1.Severity_SEVERITY_RED,
		TriggerTimeMs: time.Now().UnixMilli(),
		PersonIDs:     []string{pos.PersonID},
		X:             pos.X,
		Y:             pos.Y,
		ZoneID:        fence.ZoneID,
		RuleID:        r.ID(),
		Description:   description,
		Priority:      false,
	}
}

// Helper methods on StateTracker for boundary rule internal use.

func (st *StateTracker) getLastAlertPersonal(key string) (int64, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	// Search all zones for this key (boundary rule uses composite keys)
	for _, zs := range st.zones {
		if ts, ok := zs.LastAlertPersonal[key]; ok {
			return ts, true
		}
	}
	return 0, false
}

func (st *StateTracker) setLastAlertPersonal(key string, timestampMs int64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	// Store in the first available zone (boundary keys are zone-scoped via prefix)
	for _, zs := range st.zones {
		zs.LastAlertPersonal[key] = timestampMs
		return
	}
}

func (st *StateTracker) deleteLastAlertPersonal(key string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, zs := range st.zones {
		delete(zs.LastAlertPersonal, key)
	}
}
