package engine

import (
	"fmt"
	"time"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine/geom"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// OvercrowdingRule 超员报警规则。
// 区域内人数超过 fence.max_people 时触发（区域级去重）。
type OvercrowdingRule struct{}

func NewOvercrowdingRule() *OvercrowdingRule        { return &OvercrowdingRule{} }
func (r *OvercrowdingRule) ID() string              { return "overcrowding" }
func (r *OvercrowdingRule) Type() alertv1.AlertType { return alertv1.AlertType_ALERT_TYPE_OVERCROWDING }

func (r *OvercrowdingRule) Evaluate(_ model.PositionData, fences []model.Geofence, state *StateTracker, _ *geom.GridHash) ([]AlertEvent, error) {
	var events []AlertEvent

	for _, fence := range fences {
		if !fence.IsActive || fence.MaxPeople <= 0 {
			continue
		}

		count := state.PeopleCountInZone(fence.ZoneID)
		if count > int(fence.MaxPeople) {
			if state.IsFirstTriggerArea(alertv1.AlertType_ALERT_TYPE_OVERCROWDING, fence.ZoneID) {
				state.RecordTriggerArea(alertv1.AlertType_ALERT_TYPE_OVERCROWDING, fence.ZoneID, time.Now().UnixMilli())

				peopleInZone := state.GetPeopleInZone(fence.ZoneID)
				events = append(events, AlertEvent{
					AlertID:       NewULID(),
					AlertType:     alertv1.AlertType_ALERT_TYPE_OVERCROWDING,
					Severity:      alertv1.Severity_SEVERITY_RED,
					TriggerTimeMs: time.Now().UnixMilli(),
					PersonIDs:     peopleInZone,
					X:             0, Y: 0, // 区域级别，无单点坐标
					ZoneID:      fence.ZoneID,
					RuleID:      r.ID(),
					Description: fmt.Sprintf("%s 区域超员：当前 %d 人，上限 %d 人", fence.Name, count, fence.MaxPeople),
				})
			}
		} else {
			// 条件恢复：人数降至阈值以下
			state.ResetAreaDedup(alertv1.AlertType_ALERT_TYPE_OVERCROWDING, fence.ZoneID)
		}
	}

	return events, nil
}
