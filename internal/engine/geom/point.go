package geom

import "math"

// Point 二维坐标点（米）。
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Polygon 多边形由有序顶点序列定义（顺时针/逆时针均可）。
type Polygon []Point

// EuclideanDistance 计算两点之间的欧几里得距离。
// 时间复杂度: O(1)。
func EuclideanDistance(a, b Point) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// BoundingBox 返回多边形的轴对齐包围盒。
// 时间复杂度: O(v)，v 为顶点数。
func (p Polygon) BoundingBox() (minX, minY, maxX, maxY float64) {
	if len(p) == 0 {
		return 0, 0, 0, 0
	}
	minX, minY = p[0].X, p[0].Y
	maxX, maxY = p[0].X, p[0].Y
	for _, pt := range p[1:] {
		if pt.X < minX {
			minX = pt.X
		}
		if pt.X > maxX {
			maxX = pt.X
		}
		if pt.Y < minY {
			minY = pt.Y
		}
		if pt.Y > maxY {
			maxY = pt.Y
		}
	}
	return
}
