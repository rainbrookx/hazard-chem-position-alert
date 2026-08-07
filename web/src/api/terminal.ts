/**
 * 定位终端查询 API 封装。
 */
import { authHeader } from './auth'

const API_BASE = '/api'

export interface TerminalInfo {
  terminal_id: string
  person_id: string
  x: number
  y: number
  last_x: number
  last_y: number
  battery: number
  online: boolean
}

export interface TerminalListResponse {
  terminals: TerminalInfo[]
  count: number
}

export async function fetchTerminals(): Promise<TerminalListResponse> {
  const resp = await fetch(`${API_BASE}/terminals`, {
    headers: { ...authHeader() },
  })
  if (!resp.ok) throw new Error('Failed to fetch terminals')
  return resp.json()
}
