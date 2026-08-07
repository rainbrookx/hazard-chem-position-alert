package engine

import (
	"fmt"
	"time"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine/geom"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// LoiteringRule 滞留报警规则。
// 人员在区域内停留超过 fence.max_stay_seconds 时触发。
type LoiteringRule struct{}

func NewLoiteringRule() *LoiteringRule           { return &LoiteringRule{} }
func (r *LoiteringRule) ID() string              { return "loitering" }
func (r *LoiteringRule) Type() alertv1.AlertType { return alertv1.AlertType_ALERT_TYPE_LOITERING }

func (r *LoiteringRule) Evaluate(pos model.PositionData, fences []model.Geofence, state *StateTracker, _ *geom.GridHash) ([]AlertEvent, error) {
	var events []AlertEvent

	for _, fence := range fences {
		if !fence.IsActive || fence.MaxStaySeconds <= 0 {
			continue
		}
		if !state.IsPersonInZone(pos.PersonID, fence.ZoneID) {
			continue
		}

		entryTime := state.GetEntryTime(pos.PersonID, fence.ZoneID)
		if entryTime.IsZero() {
			continue
		}

		elapsed := time.Since(entryTime)
		if elapsed >= time.Duration(fence.MaxStaySeconds)*time.Second {
			if state.IsFirstTriggerPersonal(alertv1.AlertType_ALERT_TYPE_LOITERING, pos.PersonID, fence.ZoneID) {
				state.RecordTriggerPersonal(alertv1.AlertType_ALERT_TYPE_LOITERING, pos.PersonID, fence.ZoneID, time.Now().UnixMilli())

				events = append(events, AlertEvent{
					AlertID:       NewULID(),
					AlertType:     alertv1.AlertType_ALERT_TYPE_LOITERING,
					Severity:      alertv1.Severity_SEVERITY_YELLOW,
					TriggerTimeMs: time.Now().UnixMilli(),
					PersonIDs:     []string{pos.PersonID},
					X:             pos.X,
					Y:             pos.Y,
					ZoneID:        fence.ZoneID,
					RuleID:        r.ID(),
					Description:   fmt.Sprintf("人员 %s 在 %s 区域滞留超过 %d 秒", pos.PersonID, fence.Name, fence.MaxStaySeconds),
				})
			}
		}
	}

	return events, nil
}
