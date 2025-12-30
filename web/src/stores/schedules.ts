import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/lib/api'
import type { Schedule, CreateScheduleRequest, UpdateScheduleRequest } from '@/types'

export const useSchedulesStore = defineStore('schedules', () => {
  const schedules = ref<Schedule[]>([])
  const currentSchedule = ref<Schedule | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function fetchSchedules(params?: { source_id?: string }): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const queryParams = new URLSearchParams()
      if (params?.source_id) {
        queryParams.append('source_id', params.source_id)
      }

      const url = queryParams.toString()
        ? `/v1/schedules?${queryParams.toString()}`
        : '/v1/schedules'

      const response = await api.get<{ schedules: Schedule[] }>(url)
      schedules.value = response.data.schedules || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch schedules'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function fetchSchedule(id: string): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<Schedule>(`/v1/schedules/${id}`)
      currentSchedule.value = response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch schedule'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function createSchedule(data: CreateScheduleRequest): Promise<Schedule> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.post<Schedule>('/v1/schedules', data)
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

  async function updateSchedule(id: string, data: UpdateScheduleRequest): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.put<Schedule>(`/v1/schedules/${id}`, data)
      const updatedSchedule = response.data

      const index = schedules.value.findIndex((s) => s.id === id)
      if (index !== -1) {
        schedules.value[index] = updatedSchedule
      }
      if (currentSchedule.value?.id === id) {
        currentSchedule.value = updatedSchedule
      }
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
      await api.delete(`/v1/schedules/${id}`)
      schedules.value = schedules.value.filter((s) => s.id !== id)
      if (currentSchedule.value?.id === id) {
        currentSchedule.value = null
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to delete schedule'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  function clearCurrentSchedule(): void {
    currentSchedule.value = null
  }

  return {
    schedules,
    currentSchedule,
    isLoading,
    error,
    fetchSchedules,
    fetchSchedule,
    createSchedule,
    updateSchedule,
    deleteSchedule,
    clearCurrentSchedule,
  }
})
