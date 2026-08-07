package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine/geom"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/infrastructure/config"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
	"gorm.io/gorm"
)

// BroadcastFn 告警广播回调函数类型。
type BroadcastFn func(event AlertEvent)

// Engine 告警规则引擎主控。
type Engine struct {
	rules      []Rule
	state      *StateTracker
	fenceStore *FenceStore
	alertStore *AlertStore
	config     config.AlertEngineConfig

	alertCh    chan AlertEvent // 常规告警推送通道
	priorityCh chan AlertEvent // 优先告警推送通道（一键报警等）

	mu          sync.Mutex
	started     bool
	ctx         context.Context
	cancel      context.CancelFunc
	onBroadcast BroadcastFn // 外部广播回调（gRPC AlertPushService）
}

// New 创建告警引擎实例。
func New(cfg config.AlertEngineConfig, db *gorm.DB, fenceStore *FenceStore) *Engine {
	if cfg.DebounceSeconds <= 0 {
		cfg.DebounceSeconds = 3
	}
	if cfg.GridCellMeters <= 0 {
		cfg.GridCellMeters = 30
	}
	if cfg.StateCleanupSeconds <= 0 {
		cfg.StateCleanupSeconds = 60
	}
	if cfg.StateTTLSeconds <= 0 {
		cfg.StateTTLSeconds = 300
	}
	if cfg.MQTTTopic == "" {
		cfg.MQTTTopic = "position/cleaned"
	}

	return &Engine{
		rules:      make([]Rule, 0),
		state:      NewStateTracker(cfg.StateTTLSeconds),
		fenceStore: fenceStore,
		alertStore: NewAlertStore(db),
		config:     cfg,
		alertCh:    make(chan AlertEvent, 1024),
		priorityCh: make(chan AlertEvent, 256),
	}
}

// AddRule 注册告警规则。
func (e *Engine) AddRule(r Rule) {
	e.rules = append(e.rules, r)
	slog.Info("告警规则已注册", "rule_id", r.ID(), "type", r.Type().String())
}

// Process 处理单条定位数据：校验 → 空围栏检查 → 运行规则 → 去重 → 持久化 → 推送。
// FR-014: 校验失败跳过错处理，LOG ERROR。
// FR-016: 无围栏时拒绝处理，推送系统告警。
func (e *Engine) Process(ctx context.Context, pos model.PositionData) {
	// FR-014: 校验位置数据
	if err := pos.Validate(); err != nil {
		slog.Warn("定位数据校验失败", "terminal_id", pos.TerminalID, "person_id", pos.PersonID, "error", err)
		return
	}

	// 更新人员最新位置
	e.state.UpdatePosition(pos)

	// FR-016: 空围栏配置检查
	if !e.fenceStore.HasFences() {
		slog.Error("围栏配置为空，拒绝处理定位数据",
			"terminal_id", pos.TerminalID,
			"person_id", pos.PersonID,
		)
		// 推送系统告警
		e.sendSystemAlert("配置缺失：围栏配置为空，无法进行告警判定。请检查 Data Preprocessor 围栏同步状态。")
		return
	}

	// 获取活跃围栏
	fences := e.fenceStore.GetActive()
	if len(fences) == 0 {
		slog.Debug("无活跃围栏，跳过告警判定")
		return
	}

	// 构建 GridHash 用于聚集检测
	gridHash := geom.NewGridHash(float64(e.config.GridCellMeters))
	for personID, lastPos := range e.state.GetPositionSnapshot() {
		gridHash.Add(lastPos.X, lastPos.Y, personID)
	}

	// 运行所有注册的告警规则
	for _, rule := range e.rules {
		events, err := rule.Evaluate(pos, fences, e.state, gridHash)
		if err != nil {
			slog.Error("规则评估失败", "rule_id", rule.ID(), "error", err)
			continue
		}
		for _, event := range events {
			e.dispatch(event)
		}
	}
}

// Start 启动引擎广播协程。在任何规则注册前调用。
func (e *Engine) Start(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		return
	}

	e.ctx, e.cancel = context.WithCancel(ctx)

	// 启动后台清理协程
	e.state.StartCleanup(e.ctx, e.config.StateCleanupSeconds)

	// 启动广播 goroutine：优先通道优先（FR-013）
	go e.broadcastLoop()

	e.started = true
	slog.Info("告警引擎已启动",
		"mqtt_topic", e.config.MQTTTopic,
		"debounce_s", e.config.DebounceSeconds,
		"grid_cell_m", e.config.GridCellMeters,
	)
}

// Stop 停止引擎，关闭通道并清理状态。
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.started {
		return
	}

	if e.cancel != nil {
		e.cancel()
	}
	close(e.alertCh)
	close(e.priorityCh)
	e.started = false
	slog.Info("告警引擎已停止")
}

// AlertCh 返回常规告警通道（供 gRPC AlertPushService 消费）。
func (e *Engine) AlertCh() <-chan AlertEvent {
	return e.alertCh
}

// PriorityCh 返回优先告警通道。
func (e *Engine) PriorityCh() <-chan AlertEvent {
	return e.priorityCh
}

// FenceStore 返回围栏缓存（供 HTTP handler 查询）。
func (e *Engine) FenceStore() *FenceStore {
	return e.fenceStore
}

// AlertStore 返回告警持久化存储。
func (e *Engine) AlertStore() *AlertStore {
	return e.alertStore
}

// StateTracker 返回状态跟踪器。
func (e *Engine) StateTracker() *StateTracker {
	return e.state
}

// Config 返回引擎配置。
func (e *Engine) Config() config.AlertEngineConfig {
	return e.config
}

// SetBroadcastFn 设置外部告警广播回调（供 gRPC AlertPushService 注册）。
func (e *Engine) SetBroadcastFn(fn BroadcastFn) {
	e.onBroadcast = fn
}

// GetSeverity 根据告警类型和上下文返回告警严重级别（T024 严重级别映射）。
// Boundary→RED, Stationary→ORANGE, Overcrowding→RED, Understaffing→ORANGE,
// Loitering→YELLOW, OneKey→RED, Gathering→(3=YELLOW, 4-6=ORANGE, >6=RED).
func GetSeverity(alertType alertv1.AlertType, peopleCount int32) alertv1.Severity {
	switch alertType {
	case alertv1.AlertType_ALERT_TYPE_BOUNDARY:
		return alertv1.Severity_SEVERITY_RED
	case alertv1.AlertType_ALERT_TYPE_STATIONARY:
		return alertv1.Severity_SEVERITY_ORANGE
	case alertv1.AlertType_ALERT_TYPE_OVERCROWDING:
		return alertv1.Severity_SEVERITY_RED
	case alertv1.AlertType_ALERT_TYPE_UNDERSTAFFING:
		return alertv1.Severity_SEVERITY_ORANGE
	case alertv1.AlertType_ALERT_TYPE_LOITERING:
		return alertv1.Severity_SEVERITY_YELLOW
	case alertv1.AlertType_ALERT_TYPE_ONEKEY:
		return alertv1.Severity_SEVERITY_RED
	case alertv1.AlertType_ALERT_TYPE_GATHERING:
		if peopleCount <= 3 {
			return alertv1.Severity_SEVERITY_YELLOW
		} else if peopleCount <= 6 {
			return alertv1.Severity_SEVERITY_ORANGE
		}
		return alertv1.Severity_SEVERITY_RED
	case alertv1.AlertType_ALERT_TYPE_SYSTEM:
		return alertv1.Severity_SEVERITY_RED
	default:
		return alertv1.Severity_SEVERITY_UNSPECIFIED
	}
}

// dispatch 根据事件优先级分发到对应通道。
func (e *Engine) dispatch(event AlertEvent) {
	if event.Priority {
		select {
		case e.priorityCh <- event:
		default:
			slog.Error("优先告警通道已满，丢弃事件", "alert_id", event.AlertID)
		}
	} else {
		select {
		case e.alertCh <- event:
		default:
			slog.Error("告警通道已满，丢弃事件", "alert_id", event.AlertID)
		}
	}
}

// broadcastLoop 广播协程：优先通道始终优先消费（FR-013）。
func (e *Engine) broadcastLoop() {
	for {
		select {
		case event, ok := <-e.priorityCh:
			if !ok {
				// 通道关闭，转去消费常规通道
				for event := range e.alertCh {
					e.persistAndLog(event)
				}
				return
			}
			e.persistAndLog(event)
		default:
			select {
			case event, ok := <-e.priorityCh:
				if !ok {
					for event := range e.alertCh {
						e.persistAndLog(event)
					}
					return
				}
				e.persistAndLog(event)
			case event, ok := <-e.alertCh:
				if !ok {
					return
				}
				e.persistAndLog(event)
			}
		}
	}
}

// persistAndLog 持久化告警并记录日志。
func (e *Engine) persistAndLog(event AlertEvent) {
	// 持久化到 SQLite
	record := &model.AlertRecord{
		ID:            event.AlertID,
		AlertType:     int32(event.AlertType),
		Severity:      int32(event.Severity),
		TriggerTimeMs: event.TriggerTimeMs,
		PersonIDs:     personIDsToJSON(event.PersonIDs),
		X:             event.X,
		Y:             event.Y,
		ZoneID:        event.ZoneID,
		RuleID:        event.RuleID,
		Description:   event.Description,
		CreatedAt:     time.Now().UnixMilli(),
	}

	if err := e.alertStore.Insert(record); err != nil {
		slog.Error("持久化告警失败", "alert_id", event.AlertID, "error", err)
	}

	slog.Info("告警已生成",
		"alert_id", event.AlertID,
		"type", event.AlertType.String(),
		"severity", event.Severity.String(),
		"person_ids", event.PersonIDs,
		"zone_id", event.ZoneID,
	)

	// 调用外部广播回调（gRPC AlertPushService）
	if e.onBroadcast != nil {
		e.onBroadcast(event)
	}
}

// sendSystemAlert 发送系统告警。
func (e *Engine) sendSystemAlert(description string) {
	event := AlertEvent{
		AlertID:       NewULID(),
		AlertType:     alertv1.AlertType_ALERT_TYPE_SYSTEM,
		Severity:      alertv1.Severity_SEVERITY_RED,
		TriggerTimeMs: time.Now().UnixMilli(),
		PersonIDs:     []string{},
		X:             0,
		Y:             0,
		ZoneID:        "",
		RuleID:        "system",
		Description:   description,
		Priority:      true,
	}
	e.dispatch(event)
}

// personIDsToJSON 将人员 ID 列表转为 JSON 数组字符串。
func personIDsToJSON(ids []string) string {
	if len(ids) == 0 {
		return "[]"
	}
	result := "["
	for i, id := range ids {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`"%s"`, id)
	}
	result += "]"
	return result
}
