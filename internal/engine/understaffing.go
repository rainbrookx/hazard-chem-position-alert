package engine

import (
	"fmt"
	"time"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine/geom"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// UnderstaffingRule 缺员报警规则。
// 区域内人数不足或指定人员未到场时触发。
// 个人类去重: (UNDERSTAFFING, person_id, zone_id) for missing required persons
// 区域类去重: (UNDERSTAFFING, zone_id) for count below min
type UnderstaffingRule struct{}

func NewUnderstaffingRule() *UnderstaffingRule { return &UnderstaffingRule{} }
func (r *UnderstaffingRule) ID() string        { return "understaffing" }
func (r *UnderstaffingRule) Type() alertv1.AlertType {
	return alertv1.AlertType_ALERT_TYPE_UNDERSTAFFING
}

func (r *UnderstaffingRule) Evaluate(_ model.PositionData, fences []model.Geofence, state *StateTracker, _ *geom.GridHash) ([]AlertEvent, error) {
	var events []AlertEvent

	for _, fence := range fences {
		if !fence.IsActive {
			continue
		}

		peopleInZone := state.GetPeopleInZone(fence.ZoneID)
		count := len(peopleInZone)

		// 区域级缺员：人数低于最小值
		if fence.MinPeople > 0 && count < int(fence.MinPeople) {
			if state.IsFirstTriggerArea(alertv1.AlertType_ALERT_TYPE_UNDERSTAFFING, fence.ZoneID) {
				state.RecordTriggerArea(alertv1.AlertType_ALERT_TYPE_UNDERSTAFFING, fence.ZoneID, time.Now().UnixMilli())
				events = append(events, AlertEvent{
					AlertID:       NewULID(),
					AlertType:     alertv1.AlertType_ALERT_TYPE_UNDERSTAFFING,
					Severity:      alertv1.Severity_SEVERITY_ORANGE,
					TriggerTimeMs: time.Now().UnixMilli(),
					PersonIDs:     peopleInZone,
					X:             0, Y: 0,
					ZoneID:      fence.ZoneID,
					RuleID:      r.ID(),
					Description: fmt.Sprintf("%s 区域缺员：当前 %d 人，最少需要 %d 人", fence.Name, count, fence.MinPeople),
				})
			}
		} else {
			state.ResetAreaDedup(alertv1.AlertType_ALERT_TYPE_UNDERSTAFFING, fence.ZoneID)
		}

		// 个人级缺员：指定人员未到场
		peopleSet := make(map[string]bool, count)
		for _, id := range peopleInZone {
			peopleSet[id] = true
		}
		for _, requiredID := range fence.RequiredPersonIDs {
			if !peopleSet[requiredID] {
				if state.IsFirstTriggerPersonal(alertv1.AlertType_ALERT_TYPE_UNDERSTAFFING, requiredID, fence.ZoneID) {
					state.RecordTriggerPersonal(alertv1.AlertType_ALERT_TYPE_UNDERSTAFFING, requiredID, fence.ZoneID, time.Now().UnixMilli())
					events = append(events, AlertEvent{
						AlertID:       NewULID(),
						AlertType:     alertv1.AlertType_ALERT_TYPE_UNDERSTAFFING,
						Severity:      alertv1.Severity_SEVERITY_ORANGE,
						TriggerTimeMs: time.Now().UnixMilli(),
						PersonIDs:     []string{requiredID},
						ZoneID:        fence.ZoneID,
						RuleID:        r.ID(),
						Description:   fmt.Sprintf("必需人员 %s 未在 %s 区域到场", requiredID, fence.Name),
					})
				}
			} else {
				state.ResetPersonalDedup(alertv1.AlertType_ALERT_TYPE_UNDERSTAFFING, requiredID, fence.ZoneID)
			}
		}
	}

	return events, nil
}
