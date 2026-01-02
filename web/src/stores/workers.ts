import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/lib/api'
import type { Worker, WorkersResponse } from '@/types'

export const useWorkersStore = defineStore('workers', () => {
  const workers = ref<Worker[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const selectedWorker = ref<Worker | null>(null)

  // Computed
  const onlineWorkers = computed(() => workers.value.filter(w => w.status === 'online'))
  const offlineWorkers = computed(() => workers.value.filter(w => w.status !== 'online'))
  const healthyWorkers = computed(() => workers.value.filter(w => w.health === 'healthy'))
  const warningWorkers = computed(() => workers.value.filter(w => w.health === 'warning'))
  const criticalWorkers = computed(() => workers.value.filter(w => w.health === 'critical'))

  // Actions
  async function fetchWorkers() {
    loading.value = true
    error.value = null
    try {
      const response = await api.get<WorkersResponse>('/v1/admin/workers')
      // Ensure we always assign an array (handle null/undefined)
      workers.value = response.data.workers ?? []
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch workers'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchWorker(id: string) {
    loading.value = true
    error.value = null
    try {
      const response = await api.get<Worker>(`/v1/admin/workers/${id}`)
      selectedWorker.value = response.data
      return response.data
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch worker'
      throw err
    } finally {
      loading.value = false
    }
  }

  function clearError() {
    error.value = null
  }

  return {
    // State
    workers,
    loading,
    error,
    selectedWorker,
    // Computed
    onlineWorkers,
    offlineWorkers,
    healthyWorkers,
    warningWorkers,
    criticalWorkers,
    // Actions
    fetchWorkers,
    fetchWorker,
    clearError,
  }
})
