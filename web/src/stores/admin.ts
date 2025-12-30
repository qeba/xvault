import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/lib/api'
import type { User, Tenant, Setting, Source, Schedule, AdminCreateUserRequest, AdminUpdateUserRequest, AdminCreateSourceRequest, AdminUpdateSourceRequest, TestConnectionRequest, TestConnectionResult, AdminCreateScheduleRequest, AdminUpdateScheduleRequest } from '@/types'

export const useAdminStore = defineStore('admin', () => {
  const users = ref<User[]>([])
  const tenants = ref<Tenant[]>([])
  const sources = ref<Source[]>([])
  const schedules = ref<Schedule[]>([])
  const settings = ref<Setting[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // Computed: Get tenant by ID for lookups
  const getTenantById = computed(() => (id: string) => {
    return tenants.value.find(t => t.id === id)
  })

  // Users management
  async function fetchUsers(): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<{ users: User[] }>('/v1/admin/users')
      // API returns { users: [...] } wrapped format
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
      const response = await api.get<User | { user: User }>(`/v1/admin/users/${id}`)
      // Handle both wrapped { user: {...} } and direct response formats
      const data = response.data
      return 'user' in data ? data.user : data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch user'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function createUser(data: AdminCreateUserRequest): Promise<User> {
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

  async function updateUser(id: string, data: AdminUpdateUserRequest): Promise<User> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.put<User>(`/v1/admin/users/${id}`, data)
      const updatedUser = response.data

      const index = users.value.findIndex((u) => u.id === id)
      if (index !== -1) {
        users.value[index] = updatedUser
      }
      return updatedUser
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
      // API returns { tenants: [...] } wrapped format
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
      const response = await api.get<Tenant | { tenant: Tenant }>(`/v1/admin/tenants/${id}`)
      // Handle both wrapped { tenant: {...} } and direct response formats
      const data = response.data
      return 'tenant' in data ? data.tenant : data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch tenant'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function deleteTenant(id: string): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      await api.delete(`/v1/admin/tenants/${id}`)
      tenants.value = tenants.value.filter((t) => t.id !== id)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to delete tenant'
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

  // Sources management (admin)
  async function fetchSources(): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<{ sources: Source[] }>('/v1/admin/sources')
      sources.value = response.data.sources || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch sources'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function fetchSource(id: string): Promise<Source> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<Source>(`/v1/admin/sources/${id}`)
      return response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch source'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function createSource(data: AdminCreateSourceRequest): Promise<Source> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.post<Source>('/v1/admin/sources', data)
      const newSource = response.data
      sources.value.push(newSource)
      return newSource
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to create source'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function updateSource(id: string, data: AdminUpdateSourceRequest): Promise<Source> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.put<Source>(`/v1/admin/sources/${id}`, data)
      const updatedSource = response.data

      const index = sources.value.findIndex((s) => s.id === id)
      if (index !== -1) {
        sources.value[index] = updatedSource
      }
      return updatedSource
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to update source'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function deleteSource(id: string): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      await api.delete(`/v1/admin/sources/${id}`)
      sources.value = sources.value.filter((s) => s.id !== id)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to delete source'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  // Test connection
  async function testConnection(data: TestConnectionRequest): Promise<TestConnectionResult> {
    try {
      const response = await api.post<TestConnectionResult>('/v1/admin/sources/test-connection', data)
      return response.data
    } catch (err) {
      // Return a failure result instead of throwing
      return {
        success: false,
        message: 'Connection test failed',
        details: err instanceof Error ? err.message : 'Unknown error'
      }
    }
  }

  // Trigger backup
  async function triggerBackup(sourceId: string): Promise<{ message: string; job: { id: string } }> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.post<{ message: string; job: { id: string } }>(`/v1/admin/sources/${sourceId}/backup`, {})
      return response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to trigger backup'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  // Schedules management (admin)
  async function fetchSchedules(): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<{ schedules: Schedule[] }>('/v1/admin/schedules')
      schedules.value = response.data.schedules || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch schedules'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function fetchSchedule(id: string): Promise<Schedule> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<Schedule>(`/v1/admin/schedules/${id}`)
      return response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch schedule'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function createSchedule(data: AdminCreateScheduleRequest): Promise<Schedule> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.post<Schedule>('/v1/admin/schedules', data)
      const newSchedule = response.data
      schedules.value.push(newSchedule)
      return newSchedule
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to create schedule'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function updateSchedule(id: string, data: AdminUpdateScheduleRequest): Promise<Schedule> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.put<Schedule>(`/v1/admin/schedules/${id}`, data)
      const updatedSchedule = response.data

      const index = schedules.value.findIndex((s) => s.id === id)
      if (index !== -1) {
        schedules.value[index] = updatedSchedule
      }
      return updatedSchedule
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to update schedule'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function deleteSchedule(id: string): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      await api.delete(`/v1/admin/schedules/${id}`)
      schedules.value = schedules.value.filter((s) => s.id !== id)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to delete schedule'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  return {
    // State
    users,
    tenants,
    sources,
    schedules,
    settings,
    isLoading,
    error,
    // Computed
    getTenantById,
    // Users
    fetchUsers,
    fetchUser,
    createUser,
    updateUser,
    deleteUser,
    // Tenants
    fetchTenants,
    fetchTenant,
    deleteTenant,
    // Sources
    fetchSources,
    fetchSource,
    createSource,
    updateSource,
    deleteSource,
    testConnection,
    triggerBackup,
    // Schedules
    fetchSchedules,
    fetchSchedule,
    createSchedule,
    updateSchedule,
    deleteSchedule,
    // Settings
    fetchSettings,
    fetchSetting,
    updateSetting,
    // Retention
    runRetentionForAllSources,
    runRetentionForSource,
  }
})
