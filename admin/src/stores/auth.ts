import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import http from '@/api/http'

interface UserInfo {
  id: number
  name: string
  role: string
  permissions: string[]
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem('access_token') || '')
  const refreshToken = ref<string>(localStorage.getItem('refresh_token') || '')
  const userInfo = ref<UserInfo | null>(
    JSON.parse(localStorage.getItem('user_info') || 'null')
  )

  const isLoggedIn = computed(() => !!token.value)
  const permissions = computed(() => userInfo.value?.permissions || [])

  function hasPermission(perm: string): boolean {
    if (!userInfo.value) return false
    if (userInfo.value.role === '管理员') return true
    return permissions.value.includes(perm)
  }

  function hasAnyPermission(...perms: string[]): boolean {
    if (!userInfo.value) return false
    if (userInfo.value.role === '管理员') return true
    return perms.some(p => permissions.value.includes(p))
  }

  async function login(name: string, password: string) {
    const res: any = await http.post('/auth/login', { name, password })
    token.value = res.access_token
    refreshToken.value = res.refresh_token
    userInfo.value = res.user
    localStorage.setItem('access_token', res.access_token)
    localStorage.setItem('refresh_token', res.refresh_token)
    localStorage.setItem('user_info', JSON.stringify(res.user))
  }

  function logout() {
    token.value = ''
    refreshToken.value = ''
    userInfo.value = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem('user_info')
  }

  return { token, refreshToken, userInfo, isLoggedIn, permissions, hasPermission, hasAnyPermission, login, logout }
})
