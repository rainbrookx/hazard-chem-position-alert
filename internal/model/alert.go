package model

// AlertRecord GORM 持久化模型，告警事件持久化记录。
// 创建后不可变（仅追加，不更新）。消警状态由下游管理。
type AlertRecord struct {
	ID            string  `gorm:"column:id;primaryKey" json:"id"`                                                                // ULID, 26 chars
	AlertType     int32   `gorm:"column:alert_type;index:idx_alert_type;index:idx_alert_type_time,priority:1" json:"alert_type"` // 告警类型枚举
	Severity      int32   `gorm:"column:severity" json:"severity"`                                                               // 严重级别枚举
	TriggerTimeMs int64   `gorm:"column:trigger_time_ms;index:idx_trigger_time;index:idx_alert_type_time,priority:2" json:"trigger_time_ms"`
	PersonIDs     string  `gorm:"column:person_ids" json:"person_ids"` // JSON array string
	X             float64 `gorm:"column:x" json:"x"`
	Y             float64 `gorm:"column:y" json:"y"`
	ZoneID        string  `gorm:"column:zone_id;index:idx_zone_id" json:"zone_id"` // nullable for one-key alarm
	RuleID        string  `gorm:"column:rule_id" json:"rule_id"`
	Description   string  `gorm:"column:description" json:"description"`
	CreatedAt     int64   `gorm:"column:created_at;autoCreateTime:milli" json:"created_at"`
}

// TableName 指定 GORM 表名。
func (AlertRecord) TableName() string {
	return "alert_records"
}
