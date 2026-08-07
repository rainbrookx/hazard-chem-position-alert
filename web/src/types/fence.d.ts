/**
 * 电子围栏相关类型定义。
 */

/** 二维坐标点 */
export interface Point {
  x: number
  y: number
}

/** 电子围栏信息 */
export interface FenceInfo {
  zone_id: string
  name: string
  type: number
  source: number
  vertices: Point[]
  max_people: number
  min_people: number
  max_stay_seconds: number
  stationary_seconds: number
  stationary_threshold_meters: number
  stationary_recovery_seconds: number
  required_person_ids: string[]
  grid_cell_meters: number
  is_active: boolean
  version: number
}
