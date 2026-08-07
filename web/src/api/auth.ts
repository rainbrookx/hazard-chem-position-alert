/**
 * 认证 API 封装 — 登录、续签、获取公钥。
 */
import { encryptCredentials } from '@/utils/crypto'

const API_BASE = '/api'
let cachedToken: string | null = localStorage.getItem('token')

export function authHeader(): Record<string, string> {
  const token = getToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

export function getToken(): string | null {
  if (!cachedToken) {
    cachedToken = localStorage.getItem('token')
  }
  return cachedToken
}

export function setToken(token: string): void {
  cachedToken = token
  localStorage.setItem('token', token)
}

export function clearToken(): void {
  cachedToken = null
  localStorage.removeItem('token')
}

export async function getPublicKey(): Promise<string> {
  const resp = await fetch(`${API_BASE}/public-key`)
  if (!resp.ok) throw new Error('Failed to fetch public key')
  return resp.text()
}

export async function login(
  username: string,
  password: string,
): Promise<{ token: string; username: string }> {
  const publicKeyPem = await getPublicKey()
  const encrypted = await encryptCredentials(username, password, publicKeyPem)

  const resp = await fetch(`${API_BASE}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(encrypted),
  })
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({ error: 'Login failed' }))
    throw new Error(err.error || 'Login failed')
  }

  const data = await resp.json()
  setToken(data.token)
  return data
}

export async function refresh(): Promise<{ token: string; username: string }> {
  const resp = await fetch(`${API_BASE}/refresh`, {
    method: 'POST',
    headers: { ...authHeader() },
  })
  if (!resp.ok) {
    clearToken()
    throw new Error('Token refresh failed')
  }
  const data = await resp.json()
  setToken(data.token)
  return data
}
