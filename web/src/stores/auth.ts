import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/lib/api'
import type {
  User,
  Tenant,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
  AuthResponse,
} from '@/types'

export const useAuthStore = defineStore('auth', () => {
  // State
  const user = ref<User | null>(null)
  const tenant = ref<Tenant | null>(null)
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(null)
  const isAuthenticated = computed(() => !!user.value && !!accessToken.value)
  const isAdmin = computed(() => user.value?.role === 'admin' || user.value?.role === 'owner')
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // Actions
  async function login(credentials: LoginRequest): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.post<LoginResponse>('/v1/auth/login', credentials)
      const data = response.data

      user.value = data.user
      tenant.value = data.tenant
      accessToken.value = data.access_token
      refreshToken.value = data.refresh_token

      // Store tokens in localStorage
      localStorage.setItem('access_token', data.access_token)
      localStorage.setItem('refresh_token', data.refresh_token)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Login failed'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function register(data: RegisterRequest): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.post<RegisterResponse>('/v1/auth/register', data)
      const resData = response.data

      user.value = resData.user
      tenant.value = resData.tenant
      accessToken.value = resData.access_token
      refreshToken.value = resData.refresh_token

      // Store tokens in localStorage
      localStorage.setItem('access_token', resData.access_token)
      localStorage.setItem('refresh_token', resData.refresh_token)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Registration failed'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function fetchMe(): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<AuthResponse>('/v1/auth/me')
      const data = response.data

      user.value = {
        id: data.user_id,
        tenant_id: data.tenant_id,
        email: data.email,
        role: data.role as 'owner' | 'admin' | 'member',
        created_at: '',
        updated_at: '',
      }
      accessToken.value = localStorage.getItem('access_token')
      refreshToken.value = localStorage.getItem('refresh_token')
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch user info'
      // Clear auth state on error
      logout()
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function logout(): Promise<void> {
    try {
      if (accessToken.value && refreshToken.value) {
        await api.post('/v1/auth/logout', {
          refresh_token: refreshToken.value,
        })
      }
    } catch (err) {
      console.error('Logout error:', err)
    } finally {
      // Clear state regardless of API call success
      user.value = null
      tenant.value = null
      accessToken.value = null
      refreshToken.value = null
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
    }
  }

  function initializeFromStorage(): void {
    const storedAccessToken = localStorage.getItem('access_token')
    const storedRefreshToken = localStorage.getItem('refresh_token')

    if (storedAccessToken && storedRefreshToken) {
      accessToken.value = storedAccessToken
      refreshToken.value = storedRefreshToken
    }
  }

  return {
    // State
    user,
    tenant,
    accessToken,
    refreshToken,
    isAuthenticated,
    isAdmin,
    isLoading,
    error,
    // Actions
    login,
    register,
    fetchMe,
    logout,
    initializeFromStorage,
  }
})
