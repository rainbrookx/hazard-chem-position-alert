import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as apiLogin, refresh as apiRefresh, getToken, clearToken } from '@/api/auth'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(getToken())
  const username = ref<string>('')

  const isAuthenticated = computed(() => !!token.value)

  async function login(usernameInput: string, password: string) {
    const result = await apiLogin(usernameInput, password)
    token.value = result.token
    username.value = result.username
  }

  async function refreshToken() {
    try {
      const result = await apiRefresh()
      token.value = result.token
      username.value = result.username
    } catch {
      logout()
      throw new Error('Session expired')
    }
  }

  function logout() {
    clearToken()
    token.value = null
    username.value = ''
    router.push('/login')
  }

  return { token, username, isAuthenticated, login, refreshToken, logout }
})
