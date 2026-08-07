package geom

// PointInPolygon 使用射线法（Ray Casting / Even-Odd Rule）判断点是否在多边形内部。
// 点在边上或在多边形内部时返回 true（保守策略，宁可误报不可漏报）。
// 时间复杂度: O(v)，v 为多边形顶点数。
func PointInPolygon(point Point, polygon Polygon) bool {
	n := len(polygon)
	if n < 3 {
		return false
	}

	// 快速包围盒预判
	minX, minY, maxX, maxY := polygon.BoundingBox()
	if point.X < minX || point.X > maxX || point.Y < minY || point.Y > maxY {
		return false
	}

	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		vi, vj := polygon[i], polygon[j]

		// 检查点是否恰好在边上（保守策略：边上视为内部）
		if isPointOnSegment(point, vi, vj) {
			return true
		}

		// 经典射线法判断
		// 检查射线是否穿过边 vi→vj
		if (vi.Y > point.Y) != (vj.Y > point.Y) &&
			point.X < (vj.X-vi.X)*(point.Y-vi.Y)/(vj.Y-vi.Y)+vi.X {
			inside = !inside
		}
		j = i
	}
	return inside
}

// isPointOnSegment 判断点是否在水平线段 vi-vj 上（含端点）。
// 用于保守的边界判定。
func isPointOnSegment(p, a, b Point) bool {
	// 共线性检查：叉积 (b-a) × (p-a) 应接近零
	cross := (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X)
	if abs(cross) > 1e-12 {
		return false
	}
	// 投影在线段范围内
	minX, maxX := a.X, b.X
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	minY, maxY := a.Y, b.Y
	if minY > maxY {
		minY, maxY = maxY, minY
	}
	return p.X >= minX && p.X <= maxX && p.Y >= minY && p.Y <= maxY
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
