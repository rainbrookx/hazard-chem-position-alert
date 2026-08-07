package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine"
)

// TerminalHandler 定位终端查询 HTTP handler。
type TerminalHandler struct {
	engine *engine.Engine
}

// NewTerminalHandler 创建终端查询处理器。
func NewTerminalHandler(eng *engine.Engine) *TerminalHandler {
	return &TerminalHandler{engine: eng}
}

type terminalResponse struct {
	TerminalID string  `json:"terminal_id"`
	PersonID   string  `json:"person_id"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	LastX      float64 `json:"last_x"`
	LastY      float64 `json:"last_y"`
	Battery    int32   `json:"battery"`
	Online     bool    `json:"online"`
}

// GetTerminals GET /api/terminals — 返回最新人员位置快照。
func (h *TerminalHandler) GetTerminals(c *gin.Context) {
	snapshot := h.engine.StateTracker().GetPositionSnapshot()

	tracker := h.engine.StateTracker()
	terminals := make([]terminalResponse, 0, len(snapshot))
	for personID, pos := range snapshot {
		resp := terminalResponse{
			TerminalID: pos.TerminalID,
			PersonID:   personID,
			X:          pos.X,
			Y:          pos.Y,
			Battery:    pos.Battery,
			Online:     pos.Online,
		}
		// 填充末次位置（上一次上报的坐标）
		if prev, ok := tracker.GetPreviousPosition(personID); ok {
			resp.LastX = prev.X
			resp.LastY = prev.Y
		}
		terminals = append(terminals, resp)
	}

	c.JSON(http.StatusOK, gin.H{
		"terminals": terminals,
		"count":     len(terminals),
	})
}
