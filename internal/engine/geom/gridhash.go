package geom

// GridHash 网格分桶结构，用于空间邻近查询（人员聚集判定）。
// 时间复杂度: 添加 O(1)，查询邻域 O(1)（自身格 + 8 邻居格）。
type GridHash struct {
	CellSize float64
	cells    map[[2]int][]string // grid coordinate → person IDs
}

// NewGridHash 创建指定单元格大小的网格分桶。
func NewGridHash(cellSize float64) *GridHash {
	if cellSize <= 0 {
		cellSize = 30 // 默认 30 米
	}
	return &GridHash{
		CellSize: cellSize,
		cells:    make(map[[2]int][]string),
	}
}

// cellKey 计算坐标所属的网格坐标。
func (g *GridHash) cellKey(x, y float64) [2]int {
	return [2]int{
		int(x / g.CellSize),
		int(y / g.CellSize),
	}
}

// GetCell 返回坐标所在的网格坐标键。
func (g *GridHash) GetCell(x, y float64) [2]int {
	return g.cellKey(x, y)
}

// Add 向网格中添加一个人员 ID。
func (g *GridHash) Add(x, y float64, personID string) {
	key := g.cellKey(x, y)
	g.cells[key] = append(g.cells[key], personID)
}

// QueryNeighbors 查询坐标所在网格及其 8 个邻居网格中的所有人员 ID。
// 返回去重后的 ID 列表。
func (g *GridHash) QueryNeighbors(x, y float64) []string {
	center := g.cellKey(x, y)
	seen := make(map[string]bool)
	var result []string

	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			key := [2]int{center[0] + dx, center[1] + dy}
			for _, id := range g.cells[key] {
				if !seen[id] {
					seen[id] = true
					result = append(result, id)
				}
			}
		}
	}
	return result
}

// QueryCell 仅查询坐标所在网格的人员 ID。
func (g *GridHash) QueryCell(x, y float64) []string {
	key := g.cellKey(x, y)
	seen := make(map[string]bool)
	var result []string
	for _, id := range g.cells[key] {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

// CellCount 返回指定网格坐标的人员数量。
func (g *GridHash) CellCount(cellKey [2]int) int {
	return len(g.cells[cellKey])
}

// Cells 返回所有非空网格的迭代器。
func (g *GridHash) Cells() map[[2]int][]string {
	return g.cells
}

// Clear 清空所有网格数据。
func (g *GridHash) Clear() {
	g.cells = make(map[[2]int][]string)
}
