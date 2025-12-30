import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/lib/api'
import type { User, Tenant, Setting } from '@/types'

interface CreateUserRequest {
  email: string
  password: string
  name: string
  role?: 'owner' | 'admin' | 'member'
}

export const useAdminStore = defineStore('admin', () => {
  const users = ref<User[]>([])
  const tenants = ref<Tenant[]>([])
  const settings = ref<Setting[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // Users management
  async function fetchUsers(): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<{ users: User[] }>('/v1/admin/users')
      users.value = response.data.users || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch users'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function fetchUser(id: string): Promise<User> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<User>(`/v1/admin/users/${id}`)
      return response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch user'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function createUser(data: CreateUserRequest): Promise<User> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.post<User>('/v1/admin/users', data)
      const newUser = response.data

      users.value.push(newUser)
      return newUser
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to create user'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function updateUser(id: string, data: Partial<CreateUserRequest>): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.put<User>(`/v1/admin/users/${id}`, data)
      const updatedUser = response.data

      const index = users.value.findIndex((u) => u.id === id)
      if (index !== -1) {
        users.value[index] = updatedUser
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to update user'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function deleteUser(id: string): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      await api.delete(`/v1/admin/users/${id}`)
      users.value = users.value.filter((u) => u.id !== id)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to delete user'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  // Tenants management
  async function fetchTenants(): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<{ tenants: Tenant[] }>('/v1/admin/tenants')
      tenants.value = response.data.tenants || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch tenants'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function fetchTenant(id: string): Promise<Tenant> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<Tenant>(`/v1/admin/tenants/${id}`)
      return response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch tenant'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  // Settings management
  async function fetchSettings(): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<{ settings: Setting[] }>('/v1/admin/settings')
      settings.value = response.data.settings || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch settings'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function fetchSetting(key: string): Promise<Setting> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<Setting>(`/v1/admin/settings/${key}`)
      return response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch setting'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function updateSetting(key: string, data: { value: string }): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.put<Setting>(`/v1/admin/settings/${key}`, data)
      const updatedSetting = response.data

      const index = settings.value.findIndex((s) => s.key === key)
      if (index !== -1) {
        settings.value[index] = updatedSetting
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to update setting'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  // Retention management
  async function runRetentionForAllSources(): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      await api.post('/v1/admin/retention/run', {})
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to run retention evaluation'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function runRetentionForSource(sourceId: string): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      await api.post(`/v1/admin/retention/run/${sourceId}`, {})
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to run retention evaluation'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  return {
    // State
    users,
    tenants,
    settings,
    isLoading,
    error,
    // Users
    fetchUsers,
    fetchUser,
    createUser,
    updateUser,
    deleteUser,
    // Tenants
    fetchTenants,
    fetchTenant,
    // Settings
    fetchSettings,
    fetchSetting,
    updateSetting,
    // Retention
    runRetentionForAllSources,
    runRetentionForSource,
  }
})
