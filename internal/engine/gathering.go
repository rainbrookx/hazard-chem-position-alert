package engine

import (
	"fmt"
	"strconv"
	"time"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine/geom"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// GatheringRule 人员聚集预警规则。
// 使用 Grid-Hash 检测网格内人数 ≥ 3 时触发预警。
// FR-007: 三级黄色(3人)/橙色(4-6人)/红色(>6人)，等级跨级时生成新告警。
// FR-008: 网格级别去重 (GATHERING, grid_id, severity_level)。
type GatheringRule struct {
	defaultCellSize int
}

// NewGatheringRule 创建聚集预警规则。
func NewGatheringRule(defaultCellSize int) *GatheringRule {
	if defaultCellSize <= 0 {
		defaultCellSize = 30
	}
	return &GatheringRule{defaultCellSize: defaultCellSize}
}

func (r *GatheringRule) ID() string              { return "gathering" }
func (r *GatheringRule) Type() alertv1.AlertType { return alertv1.AlertType_ALERT_TYPE_GATHERING }

func (r *GatheringRule) Evaluate(_ model.PositionData, fences []model.Geofence, state *StateTracker, gridHash *geom.GridHash) ([]AlertEvent, error) {
	var events []AlertEvent

	if gridHash == nil {
		return nil, nil
	}

	// 收集所有需要处理的网格大小（去重）
	cellSizes := map[int]bool{r.defaultCellSize: true}
	for _, f := range fences {
		if f.IsActive && f.GridCellMeters > 0 && int(f.GridCellMeters) != r.defaultCellSize {
			cellSizes[int(f.GridCellMeters)] = true
		}
	}

	// 获取所有人员位置快照用于重建 GridHash
	allPositions := state.GetPositionSnapshot()

	for cellSize := range cellSizes {
		var gh *geom.GridHash
		if cellSize == r.defaultCellSize {
			gh = gridHash // 使用预构建的默认尺寸 GridHash
		} else {
			// 为自定义网格大小重建 GridHash
			gh = geom.NewGridHash(float64(cellSize))
			for personID, pos := range allPositions {
				gh.Add(pos.X, pos.Y, personID)
			}
		}

		r.evaluateCells(gh, cellSize, state, &events)
	}

	return events, nil
}

// evaluateCells 对指定 GridHash 的非空网格进行聚集判定。
func (r *GatheringRule) evaluateCells(gh *geom.GridHash, cellSize int, state *StateTracker, events *[]AlertEvent) {
	for cellKey, personIDs := range gh.Cells() {
		count := len(personIDs)
		if count < 3 {
			gridID := cellKeyToString(cellKey)
			state.ResetGatheringDedup(gridID)
			continue
		}

		severity := alertv1.Severity_SEVERITY_YELLOW
		if count > 6 {
			severity = alertv1.Severity_SEVERITY_RED
		} else if count >= 4 {
			severity = alertv1.Severity_SEVERITY_ORANGE
		}

		gridID := cellKeyToString(cellKey)

		if state.IsFirstTriggerGathering(gridID, int32(severity)) {
			state.RecordTriggerGathering(gridID, int32(severity), time.Now().UnixMilli())

			*events = append(*events, AlertEvent{
				AlertID:       NewULID(),
				AlertType:     alertv1.AlertType_ALERT_TYPE_GATHERING,
				Severity:      severity,
				TriggerTimeMs: time.Now().UnixMilli(),
				PersonIDs:     personIDs,
				X:             float64(cellKey[0]) * float64(cellSize),
				Y:             float64(cellKey[1]) * float64(cellSize),
				ZoneID:        "",
				RuleID:        r.ID(),
				Description:   fmt.Sprintf("网格 %s 人员聚集：%d 人，等级 %s（网格 %d m）", gridID, count, severity.String(), cellSize),
			})

			gs := state.GetGatheringState(gridID)
			gs.PeopleCount = int32(count)
		}
	}
}

// cellKeyToString 将网格坐标键转为可读字符串标识。
func cellKeyToString(key [2]int) string {
	return strconv.Itoa(key[0]) + "," + strconv.Itoa(key[1])
}
