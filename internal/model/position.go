package model

import "math"

// PositionData MQTT 消费的已清洗定位数据，仅在引擎内存中流转。
type PositionData struct {
	TerminalID string  `json:"terminal_id"`
	Timestamp  int64   `json:"timestamp"` // 定位时间戳（毫秒 Unix epoch）
	X          float64 `json:"x"`         // X 坐标（米）
	Y          float64 `json:"y"`         // Y 坐标（米）
	PersonID   string  `json:"person_id"` // 绑定对象/人员 ID
	Battery    int32   `json:"battery"`   // 电量百分比 (0-100)
	Online     bool    `json:"online"`    // 在线/离线状态
	SOSFlag    bool    `json:"sos_flag"`  // 一键报警标志
}

// Validate 校验 PositionData 字段合法性。
// 返回 nil 表示校验通过，非 nil 表示校验失败原因。
func (p *PositionData) Validate() error {
	if p.TerminalID == "" {
		return ErrEmptyTerminalID
	}
	if p.PersonID == "" {
		return ErrEmptyPersonID
	}
	if math.IsNaN(p.X) || math.IsInf(p.X, 0) {
		return ErrInvalidCoordinate
	}
	if math.IsNaN(p.Y) || math.IsInf(p.Y, 0) {
		return ErrInvalidCoordinate
	}
	if p.Battery < 0 || p.Battery > 100 {
		return ErrBatteryOutOfRange
	}
	if p.Timestamp <= 0 {
		return ErrInvalidTimestamp
	}
	return nil
}
