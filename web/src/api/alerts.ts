/**
 * 告警查询 API 封装。
 */
import { authHeader } from './auth'

const API_BASE = '/api'

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

export interface AlertQuery {
  types?: string[]
  start_time_ms?: number
  end_time_ms?: number
  zone_id?: string
  limit?: number
  offset?: number
}

export async function fetchActiveAlerts(types?: string[]): Promise<AlertRecord[]> {
  const params = new URLSearchParams()
  if (types && types.length > 0) {
    params.set('types', types.join(','))
  }

  const resp = await fetch(`${API_BASE}/alerts/active?${params}`, {
    headers: { ...authHeader() },
  })
  if (!resp.ok) throw new Error('Failed to fetch active alerts')
  const data = await resp.json()
  return data.alerts || []
}

export async function fetchHistory(
  query: AlertQuery,
): Promise<{ records: AlertRecord[]; total: number }> {
  const resp = await fetch(`${API_BASE}/alerts/history`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(query),
  })
  if (!resp.ok) throw new Error('Failed to fetch alert history')
  return resp.json()
}
