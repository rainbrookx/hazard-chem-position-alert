/**
 * 定位终端查询 API 封装。
 */
import { authHeader } from './auth'
import type { TerminalInfo, TerminalListResponse } from '@/types'

const API_BASE = '/api'

export type { TerminalInfo, TerminalListResponse }

export async function fetchTerminals(): Promise<TerminalListResponse> {
  const resp = await fetch(`${API_BASE}/terminals`, {
    headers: { ...authHeader() },
  })
  if (!resp.ok) throw new Error('Failed to fetch terminals')
  return resp.json()
}
