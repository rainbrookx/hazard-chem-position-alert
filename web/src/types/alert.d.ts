/**
 * 告警相关类型定义。
 */

/** 告警记录 */
export interface AlertRecord {
  alert_id: string
  alert_type: number
  severity: number
  trigger_time_ms: number
  person_ids: string[]
  zone_id: string
  description: string
  created_at_ms: number
}

/** 告警历史查询参数 */
export interface AlertQuery {
  types?: string[]
  start_time_ms?: number
  end_time_ms?: number
  zone_id?: string
  limit?: number
  offset?: number
}
