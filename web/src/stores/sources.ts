import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/lib/api'
import type { Source, CreateSourceRequest } from '@/types'

export const useSourcesStore = defineStore('sources', () => {
  const sources = ref<Source[]>([])
  const currentSource = ref<Source | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function fetchSources(): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<{ sources: Source[] }>('/v1/sources')
      sources.value = response.data.sources || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch sources'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function fetchSource(id: string): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<Source>(`/v1/sources/${id}`)
      currentSource.value = response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch source'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function createSource(data: CreateSourceRequest): Promise<Source> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.post<Source>('/v1/sources', data)
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

  async function updateSource(id: string, data: Partial<CreateSourceRequest>): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.put<Source>(`/v1/sources/${id}`, data)
      const updatedSource = response.data

      const index = sources.value.findIndex((s) => s.id === id)
      if (index !== -1) {
        sources.value[index] = updatedSource
      }
      if (currentSource.value?.id === id) {
        currentSource.value = updatedSource
      }
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
      await api.delete(`/v1/sources/${id}`)
      sources.value = sources.value.filter((s) => s.id !== id)
      if (currentSource.value?.id === id) {
        currentSource.value = null
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to delete source'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  function clearCurrentSource(): void {
    currentSource.value = null
  }

  return {
    sources,
    currentSource,
    isLoading,
    error,
    fetchSources,
    fetchSource,
    createSource,
    updateSource,
    deleteSource,
    clearCurrentSource,
  }
})
