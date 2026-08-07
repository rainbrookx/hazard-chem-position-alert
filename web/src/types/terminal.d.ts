/**
 * 定位终端相关类型定义。
 */

/** 终端设备信息 */
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

/** 终端列表查询响应 */
export interface TerminalListResponse {
  terminals: TerminalInfo[]
  count: number
}
