package http

import (
	"net/http"
	"time"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine"

	"github.com/gin-gonic/gin"
)

// parsePersonIDsJSON 简单解析 JSON 数组字符串为 []string。
func parsePersonIDsJSON(jsonStr string) []string {
	if jsonStr == "" || jsonStr == "[]" {
		return nil
	}
	s := jsonStr
	if len(s) < 2 {
		return nil
	}
	s = s[1 : len(s)-1] // 去掉 []
	if s == "" {
		return nil
	}
	var ids []string
	current := ""
	inQuote := false
	for _, ch := range s {
		if ch == '"' {
			inQuote = !inQuote
			if !inQuote && current != "" {
				ids = append(ids, current)
				current = ""
			}
		} else if inQuote {
			current += string(ch)
		}
	}
	return ids
}

// AlertHandler 告警查询 HTTP handler。
type AlertHandler struct {
	alertStore *engine.AlertStore
}

// NewAlertHandler 创建告警查询处理器。
func NewAlertHandler(alertStore *engine.AlertStore) *AlertHandler {
	return &AlertHandler{alertStore: alertStore}
}

// GetActiveAlerts GET /api/alerts/active — 返回当前活跃告警（最近 5 分钟）。
func (h *AlertHandler) GetActiveAlerts(c *gin.Context) {
	typeStrs := c.QueryArray("types")
	var types []alertv1.AlertType
	for _, s := range typeStrs {
		// 简单解析整数类型值
		var val int32
		for _, ch := range s {
			if ch >= '0' && ch <= '9' {
				val = val*10 + int32(ch-'0')
			}
		}
		if val > 0 {
			types = append(types, alertv1.AlertType(val))
		}
	}

	records, err := h.alertStore.ListActive(5*time.Minute, types)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	type activeAlert struct {
		AlertID       string   `json:"alert_id"`
		AlertType     int32    `json:"alert_type"`
		Severity      int32    `json:"severity"`
		TriggerTimeMs int64    `json:"trigger_time_ms"`
		PersonIDs     []string `json:"person_ids"`
		ZoneID        string   `json:"zone_id"`
		Description   string   `json:"description"`
	}

	alerts := make([]activeAlert, 0, len(records))
	for _, r := range records {
		alerts = append(alerts, activeAlert{
			AlertID:       r.ID,
			AlertType:     r.AlertType,
			Severity:      r.Severity,
			TriggerTimeMs: r.TriggerTimeMs,
			PersonIDs:     parsePersonIDsJSON(r.PersonIDs),
			ZoneID:        r.ZoneID,
			Description:   r.Description,
		})
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts, "count": len(alerts)})
}

// QueryHistory POST /api/alerts/history — 查询历史告警（支持类型/时间/区域过滤 + 分页）。
func (h *AlertHandler) QueryHistory(c *gin.Context) {
	var req struct {
		Types       []int32 `json:"types"`
		StartTimeMs int64   `json:"start_time_ms"`
		EndTimeMs   int64   `json:"end_time_ms"`
		ZoneID      string  `json:"zone_id"`
		Limit       int     `json:"limit"`
		Offset      int     `json:"offset"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空 body
		req.Limit = 100
	}

	if req.Limit <= 0 {
		req.Limit = 100
	}

	var types []alertv1.AlertType
	for _, t := range req.Types {
		types = append(types, alertv1.AlertType(t))
	}

	result, err := h.alertStore.QueryHistory(types, req.StartTimeMs, req.EndTimeMs, req.ZoneID, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	type historyRecord struct {
		AlertID       string   `json:"alert_id"`
		AlertType     int32    `json:"alert_type"`
		Severity      int32    `json:"severity"`
		TriggerTimeMs int64    `json:"trigger_time_ms"`
		PersonIDs     []string `json:"person_ids"`
		X             float64  `json:"x"`
		Y             float64  `json:"y"`
		ZoneID        string   `json:"zone_id"`
		RuleID        string   `json:"rule_id"`
		Description   string   `json:"description"`
		CreatedAtMs   int64    `json:"created_at_ms"`
	}

	records := make([]historyRecord, 0, len(result.Records))
	for _, r := range result.Records {
		records = append(records, historyRecord{
			AlertID:       r.ID,
			AlertType:     r.AlertType,
			Severity:      r.Severity,
			TriggerTimeMs: r.TriggerTimeMs,
			PersonIDs:     parsePersonIDsJSON(r.PersonIDs),
			X:             r.X,
			Y:             r.Y,
			ZoneID:        r.ZoneID,
			RuleID:        r.RuleID,
			Description:   r.Description,
			CreatedAtMs:   r.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
		"total":   result.Total,
	})
}
