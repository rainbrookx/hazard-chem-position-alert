package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// ZoneState 单个围栏区域的状态跟踪。
type ZoneState struct {
	ZoneID            string
	PeopleInZone      map[string]time.Time // person_id → 进入时间
	LastAlertPersonal map[string]int64     // "alert_type:person_id:zone_id" → 上次触发时间 ms
	LastAlertArea     map[string]int64     // "alert_type:zone_id" → 上次触发时间 ms
	EnterHistory      map[string][]int64   // person_id → 进出时间戳历史（去抖用）
	MovingSince       map[string]time.Time // person_id → 持续移动开始时间（静止恢复去抖）
}

// GatheringState 单个网格的聚集预警状态。
type GatheringState struct {
	GridID        string
	SeverityLevel int32          // 当前聚集等级 (1=黄, 2=橙, 3=红)
	PeopleCount   int32          // 当前网格人数
	LastAlertTime int64          // 上次触发时间 ms
	FiredLevels   map[int32]bool // 已触发的等级集合
}

// NewGatheringState 创建新的聚集状态。
func NewGatheringState(gridID string) *GatheringState {
	return &GatheringState{
		GridID:      gridID,
		FiredLevels: make(map[int32]bool),
	}
}

// StateTracker 引擎内部状态跟踪器，线程安全。
// 内存估算: 10000 人 × 10 区域 × ~200 bytes/条目 ≈ 20 MB。
type StateTracker struct {
	mu               sync.RWMutex
	zones            map[string]*ZoneState      // zone_id → ZoneState
	gathering        map[string]*GatheringState // grid_id → GatheringState
	lastPosition     map[string]modelPosition   // person_id → latest position
	previousPosition map[string]modelPosition   // person_id → previous position (for "末次位置" display)
	cleanupTTL       time.Duration
}

type modelPosition struct {
	TerminalID string
	X          float64
	Y          float64
	Timestamp  int64
	Battery    int32
	Online     bool
}

// NewStateTracker 创建新的状态跟踪器。
func NewStateTracker(ttlSeconds int) *StateTracker {
	if ttlSeconds <= 0 {
		ttlSeconds = 300
	}
	return &StateTracker{
		zones:            make(map[string]*ZoneState),
		gathering:        make(map[string]*GatheringState),
		lastPosition:     make(map[string]modelPosition),
		previousPosition: make(map[string]modelPosition),
		cleanupTTL:       time.Duration(ttlSeconds) * time.Second,
	}
}

// --- Zone State Management ---

// GetZoneState 获取或创建指定区域的状态。
func (st *StateTracker) GetZoneState(zoneID string) *ZoneState {
	st.mu.RLock()
	zs, ok := st.zones[zoneID]
	st.mu.RUnlock()
	if ok {
		return zs
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	// 双重检查
	if zs, ok = st.zones[zoneID]; ok {
		return zs
	}
	zs = &ZoneState{
		ZoneID:            zoneID,
		PeopleInZone:      make(map[string]time.Time),
		LastAlertPersonal: make(map[string]int64),
		LastAlertArea:     make(map[string]int64),
		EnterHistory:      make(map[string][]int64),
		MovingSince:       make(map[string]time.Time),
	}
	st.zones[zoneID] = zs
	return zs
}

// RecordEntry 记录人员进入区域。
func (st *StateTracker) RecordEntry(personID, zoneID string, timestamp int64) {
	zs := st.GetZoneState(zoneID)
	st.mu.Lock()
	zs.PeopleInZone[personID] = time.UnixMilli(timestamp)
	st.mu.Unlock()
}

// RecordExit 记录人员离开区域。
func (st *StateTracker) RecordExit(personID, zoneID string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if zs, ok := st.zones[zoneID]; ok {
		delete(zs.PeopleInZone, personID)
		delete(zs.MovingSince, personID)
	}
}

// GetEntryTime 获取人员进入区域的时间。
// 返回 zero time 如果未记录。
func (st *StateTracker) GetEntryTime(personID, zoneID string) time.Time {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if zs, ok := st.zones[zoneID]; ok {
		if t, ok := zs.PeopleInZone[personID]; ok {
			return t
		}
	}
	return time.Time{}
}

// IsPersonInZone 检查人员是否在区域内。
func (st *StateTracker) IsPersonInZone(personID, zoneID string) bool {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if zs, ok := st.zones[zoneID]; ok {
		_, exists := zs.PeopleInZone[personID]
		return exists
	}
	return false
}

// PeopleCountInZone 返回区域内的当前人数。
func (st *StateTracker) PeopleCountInZone(zoneID string) int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if zs, ok := st.zones[zoneID]; ok {
		return len(zs.PeopleInZone)
	}
	return 0
}

// GetPeopleInZone 返回区域内当前人员 ID 列表。
func (st *StateTracker) GetPeopleInZone(zoneID string) []string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if zs, ok := st.zones[zoneID]; ok {
		ids := make([]string, 0, len(zs.PeopleInZone))
		for id := range zs.PeopleInZone {
			ids = append(ids, id)
		}
		return ids
	}
	return nil
}

// --- Dedup Methods (FR-008) ---

// personalKey 构建个人类告警去重键。
func personalKey(alertType alertv1.AlertType, personID, zoneID string) string {
	return fmt.Sprintf("%d:%s:%s", int32(alertType), personID, zoneID)
}

// areaKey 构建区域类告警去重键。
func areaKey(alertType alertv1.AlertType, zoneID string) string {
	return fmt.Sprintf("%d:%s", int32(alertType), zoneID)
}

// IsFirstTriggerPersonal 检查个人类告警是否首次触发（未被抑制）。
// 返回 true 表示应触发告警。
func (st *StateTracker) IsFirstTriggerPersonal(alertType alertv1.AlertType, personID, zoneID string) bool {
	key := personalKey(alertType, personID, zoneID)
	st.mu.RLock()
	defer st.mu.RUnlock()
	if zs, ok := st.zones[zoneID]; ok {
		_, fired := zs.LastAlertPersonal[key]
		return !fired
	}
	return true
}

// IsFirstTriggerArea 检查区域类告警是否首次触发。
func (st *StateTracker) IsFirstTriggerArea(alertType alertv1.AlertType, zoneID string) bool {
	key := areaKey(alertType, zoneID)
	st.mu.RLock()
	defer st.mu.RUnlock()
	if zs, ok := st.zones[zoneID]; ok {
		_, fired := zs.LastAlertArea[key]
		return !fired
	}
	return true
}

// IsFirstTriggerGathering 检查聚集预警是否首次在指定等级触发。
func (st *StateTracker) IsFirstTriggerGathering(gridID string, severityLevel int32) bool {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if gs, ok := st.gathering[gridID]; ok {
		return !gs.FiredLevels[severityLevel]
	}
	return true
}

// RecordTriggerPersonal 记录个人类告警已触发。
func (st *StateTracker) RecordTriggerPersonal(alertType alertv1.AlertType, personID, zoneID string, timestampMs int64) {
	key := personalKey(alertType, personID, zoneID)
	zs := st.GetZoneState(zoneID)
	st.mu.Lock()
	zs.LastAlertPersonal[key] = timestampMs
	st.mu.Unlock()
}

// RecordTriggerArea 记录区域类告警已触发。
func (st *StateTracker) RecordTriggerArea(alertType alertv1.AlertType, zoneID string, timestampMs int64) {
	key := areaKey(alertType, zoneID)
	zs := st.GetZoneState(zoneID)
	st.mu.Lock()
	zs.LastAlertArea[key] = timestampMs
	st.mu.Unlock()
}

// RecordTriggerGathering 记录聚集预警已触发。
func (st *StateTracker) RecordTriggerGathering(gridID string, severityLevel int32, timestampMs int64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	gs, ok := st.gathering[gridID]
	if !ok {
		gs = NewGatheringState(gridID)
		st.gathering[gridID] = gs
	}
	gs.FiredLevels[severityLevel] = true
	gs.LastAlertTime = timestampMs
}

// ResetPersonalDedup 重置个人类告警去重（条件恢复时调用）。
func (st *StateTracker) ResetPersonalDedup(alertType alertv1.AlertType, personID, zoneID string) {
	key := personalKey(alertType, personID, zoneID)
	st.mu.Lock()
	defer st.mu.Unlock()
	if zs, ok := st.zones[zoneID]; ok {
		delete(zs.LastAlertPersonal, key)
	}
}

// ResetAreaDedup 重置区域类告警去重。
func (st *StateTracker) ResetAreaDedup(alertType alertv1.AlertType, zoneID string) {
	key := areaKey(alertType, zoneID)
	st.mu.Lock()
	defer st.mu.Unlock()
	if zs, ok := st.zones[zoneID]; ok {
		delete(zs.LastAlertArea, key)
	}
}

// ResetGatheringDedup 重置聚集预警去重（人数降至 <3 时）。
func (st *StateTracker) ResetGatheringDedup(gridID string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.gathering, gridID)
}

// --- Movement / Debounce ---

// RecordMovement 记录人员移动状态，用于静止恢复去抖。
// moved: 本次是否检测到位移超过阈值。
func (st *StateTracker) RecordMovement(personID, zoneID string, moved bool, threshold float64, now time.Time) {
	zs := st.GetZoneState(zoneID)
	st.mu.Lock()
	defer st.mu.Unlock()
	if moved {
		if _, ok := zs.MovingSince[personID]; !ok {
			zs.MovingSince[personID] = now
		}
	} else {
		delete(zs.MovingSince, personID)
	}
}

// GetMovingDuration 返回人员持续移动的时长。
func (st *StateTracker) GetMovingDuration(personID, zoneID string) time.Duration {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if zs, ok := st.zones[zoneID]; ok {
		if start, ok := zs.MovingSince[personID]; ok {
			return time.Since(start)
		}
	}
	return 0
}

// ShouldDebounce 检查是否在去抖窗口内（进入区域需持续满足条件才触发）。
func (st *StateTracker) ShouldDebounce(personID, zoneID string, windowSeconds int) bool {
	if windowSeconds <= 0 {
		return false
	}
	st.mu.RLock()
	defer st.mu.RUnlock()
	if zs, ok := st.zones[zoneID]; ok {
		history := zs.EnterHistory[personID]
		if len(history) == 0 {
			return false
		}
		nowMs := time.Now().UnixMilli()
		windowMs := int64(windowSeconds) * 1000
		// 所有历史记录都在去抖窗口内 → 应抑制（去抖中）
		for _, ts := range history {
			if nowMs-ts > windowMs {
				return false // 有记录超出窗口，条件已持续满足
			}
		}
		return nowMs-history[len(history)-1] < windowMs
	}
	return false
}

// UpdatePosition 更新人员最新位置信息（含终端元数据），并保留上一位置。
func (st *StateTracker) UpdatePosition(pos model.PositionData) {
	st.mu.Lock()
	defer st.mu.Unlock()

	// 保存上一位置（用于 Web UI "末次位置" 展示）
	if prev, ok := st.lastPosition[pos.PersonID]; ok {
		st.previousPosition[pos.PersonID] = prev
	}

	st.lastPosition[pos.PersonID] = modelPosition{
		TerminalID: pos.TerminalID,
		X:          pos.X,
		Y:          pos.Y,
		Timestamp:  pos.Timestamp,
		Battery:    pos.Battery,
		Online:     pos.Online,
	}
}

// GetLastPosition 获取人员最新位置。
func (st *StateTracker) GetLastPosition(personID string) (modelPosition, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	pos, ok := st.lastPosition[personID]
	return pos, ok
}

// GetPreviousPosition 返回人员的上一位置记录。
func (st *StateTracker) GetPreviousPosition(personID string) (modelPosition, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	pos, ok := st.previousPosition[personID]
	return pos, ok
}

// GetPositionSnapshot 返回所有人员的最新位置快照，供 Web UI 查询。
func (st *StateTracker) GetPositionSnapshot() map[string]modelPosition {
	st.mu.RLock()
	defer st.mu.RUnlock()
	snapshot := make(map[string]modelPosition, len(st.lastPosition))
	for k, v := range st.lastPosition {
		snapshot[k] = v
	}
	return snapshot
}

// --- Gathering State ---

// GetGatheringState 获取指定网格的聚集状态。
func (st *StateTracker) GetGatheringState(gridID string) *GatheringState {
	st.mu.RLock()
	gs, ok := st.gathering[gridID]
	st.mu.RUnlock()
	if ok {
		return gs
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if gs, ok = st.gathering[gridID]; ok {
		return gs
	}
	gs = NewGatheringState(gridID)
	st.gathering[gridID] = gs
	return gs
}

// --- Cleanup ---

// StartCleanup 启动后台清理协程，定期扫描并清理过期状态。
func (st *StateTracker) StartCleanup(ctx context.Context, intervalSeconds int) {
	if intervalSeconds <= 0 {
		intervalSeconds = 60
	}
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				slog.Debug("状态清理协程已退出")
				return
			case <-ticker.C:
				st.cleanup()
			}
		}
	}()
}

func (st *StateTracker) cleanup() {
	st.mu.Lock()
	defer st.mu.Unlock()

	now := time.Now()
	removedZones := 0
	removedPeople := 0

	for zoneID, zs := range st.zones {
		// 清理已离开超时的人员记录
		for personID, entryTime := range zs.PeopleInZone {
			// 如果人员不再活跃（超过 TTL 未更新位置）
			if pos, ok := st.lastPosition[personID]; ok {
				lastSeen := time.UnixMilli(pos.Timestamp)
				if now.Sub(lastSeen) > st.cleanupTTL {
					delete(zs.PeopleInZone, personID)
					delete(zs.MovingSince, personID)
					removedPeople++
				}
			}
			_ = entryTime // suppress unused warning
		}

		// 清理空的区域状态
		if len(zs.PeopleInZone) == 0 {
			// 仅当所有去重键也过期时清理
			allExpired := true
			cutoff := now.Add(-st.cleanupTTL).UnixMilli()
			for _, lastTime := range zs.LastAlertPersonal {
				if lastTime > cutoff {
					allExpired = false
					break
				}
			}
			for _, lastTime := range zs.LastAlertArea {
				if lastTime > cutoff {
					allExpired = false
					break
				}
			}
			if allExpired {
				delete(st.zones, zoneID)
				removedZones++
			}
		}
	}

	// 清理聚集状态（网格已无人的）
	for gridID, gs := range st.gathering {
		if gs.PeopleCount == 0 {
			if now.UnixMilli()-gs.LastAlertTime > st.cleanupTTL.Milliseconds() {
				delete(st.gathering, gridID)
			}
		}
	}

	if removedZones > 0 || removedPeople > 0 {
		slog.Debug("状态清理完成", "removed_zones", removedZones, "removed_people", removedPeople)
	}
}
