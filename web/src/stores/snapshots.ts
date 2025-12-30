import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/lib/api'
import type { Snapshot } from '@/types'

export const useSnapshotsStore = defineStore('snapshots', () => {
  const snapshots = ref<Snapshot[]>([])
  const currentSnapshot = ref<Snapshot | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function fetchSnapshots(params?: { source_id?: string }): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const queryParams = new URLSearchParams()
      if (params?.source_id) {
        queryParams.append('source_id', params.source_id)
      }

      const url = queryParams.toString()
        ? `/v1/snapshots?${queryParams.toString()}`
        : '/v1/snapshots'

      const response = await api.get<{ snapshots: Snapshot[] }>(url)
      snapshots.value = response.data.snapshots || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch snapshots'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function fetchSnapshot(id: string): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<Snapshot>(`/v1/snapshots/${id}`)
      currentSnapshot.value = response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch snapshot'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function triggerRestore(snapshotId: string): Promise<{ job_id: string }> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.post<{ job_id: string }>(`/v1/snapshots/${snapshotId}/restore`, {})
      return response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to trigger restore'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function generateDownloadLink(snapshotId: string): Promise<{ download_url: string; expires_at: string }> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.post<{ download_url: string; expires_at: string }>(
        `/v1/snapshots/${snapshotId}/download`,
        {}
      )
      return response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to generate download link'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  function clearCurrentSnapshot(): void {
    currentSnapshot.value = null
  }

  return {
    snapshots,
    currentSnapshot,
    isLoading,
    error,
    fetchSnapshots,
    fetchSnapshot,
    triggerRestore,
    generateDownloadLink,
    clearCurrentSnapshot,
  }
})
