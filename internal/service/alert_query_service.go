package service

import (
	"context"
	"time"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	alertqueryv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert_query/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine"
)

// AlertQueryService 实现 gRPC AlertQueryServiceServer。
type AlertQueryService struct {
	alertqueryv1.UnimplementedAlertQueryServiceServer
	alertStore *engine.AlertStore
}

// NewAlertQueryService 创建告警查询服务。
func NewAlertQueryService(alertStore *engine.AlertStore) *AlertQueryService {
	return &AlertQueryService{alertStore: alertStore}
}

// QueryHistory 查询历史告警记录（支持类型/时间/区域过滤 + 分页）。
func (s *AlertQueryService) QueryHistory(ctx context.Context, req *alertqueryv1.AlertQuery) (*alertqueryv1.QueryHistoryResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 100
	}
	offset := int(req.Offset)

	result, err := s.alertStore.QueryHistory(req.Types, req.StartTimeMs, req.EndTimeMs, req.ZoneId, limit, offset)
	if err != nil {
		return nil, err
	}

	records := make([]*alertqueryv1.AlertRecord, 0, len(result.Records))
	for _, r := range result.Records {
		records = append(records, &alertqueryv1.AlertRecord{
			Event: &alertv1.AlertEvent{
				AlertId:       r.ID,
				AlertType:     alertv1.AlertType(r.AlertType),
				Severity:      alertv1.Severity(r.Severity),
				TriggerTimeMs: r.TriggerTimeMs,
				PersonIds:     parsePersonIDsJSON(r.PersonIDs),
				X:             r.X,
				Y:             r.Y,
				ZoneId:        r.ZoneID,
				RuleId:        r.RuleID,
				Description:   r.Description,
			},
			CreatedAtMs: r.CreatedAt,
		})
	}

	return &alertqueryv1.QueryHistoryResponse{
		Records: records,
		Total:   int32(result.Total),
	}, nil
}

// ListActive 查询当前活跃告警（最近 5 分钟内的告警）。
func (s *AlertQueryService) ListActive(ctx context.Context, req *alertqueryv1.ListActiveRequest) (*alertqueryv1.ListActiveResponse, error) {
	records, err := s.alertStore.ListActive(5*time.Minute, req.Types)
	if err != nil {
		return nil, err
	}

	alerts := make([]*alertqueryv1.ActiveAlert, 0, len(records))
	for _, r := range records {
		alerts = append(alerts, &alertqueryv1.ActiveAlert{
			AlertId:       r.ID,
			AlertType:     alertv1.AlertType(r.AlertType),
			Severity:      alertv1.Severity(r.Severity),
			TriggerTimeMs: r.TriggerTimeMs,
			PersonIds:     parsePersonIDsJSON(r.PersonIDs),
			ZoneId:        r.ZoneID,
			Description:   r.Description,
		})
	}

	return &alertqueryv1.ListActiveResponse{Alerts: alerts}, nil
}

// parsePersonIDsJSON 简单解析 JSON 数组字符串为 []string。
func parsePersonIDsJSON(jsonStr string) []string {
	if jsonStr == "" || jsonStr == "[]" {
		return nil
	}
	// 简单解析: ["id1","id2"]
	var ids []string
	s := jsonStr
	if len(s) < 2 {
		return nil
	}
	s = s[1 : len(s)-1] // 去掉 []
	if s == "" {
		return nil
	}
	// 按逗号分割并去引号
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
