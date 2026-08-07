/**
 * 告警查询 API 封装。
 */
import { authHeader } from './auth'
import type { AlertRecord, AlertQuery } from '@/types'

const API_BASE = '/api'

export type { AlertRecord, AlertQuery }

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
