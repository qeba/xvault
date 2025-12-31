<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useAdminStore } from '@/stores/admin'
import type { AdminSnapshot, LogEntry } from '@/types'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import Input from '@/components/ui/input/Input.vue'
import Dialog from '@/components/ui/dialog/Dialog.vue'
import { Select, type SelectOption } from '@/components/ui/select'
import { LogViewer } from '@/components/ui/log-viewer'
import api from '@/lib/api'

const adminStore = useAdminStore()

const searchQuery = ref('')
const statusFilter = ref('')
const tenantFilter = ref('')
const sourceFilter = ref('')
const isLoading = ref(true)
const showDetailDialog = ref(false)
const showLogsDialog = ref(false)
const selectedSnapshot = ref<AdminSnapshot | null>(null)
const selectedSnapshotLogs = ref<LogEntry[]>([])
const isLoadingLogs = ref(false)
const isDownloading = ref(false)
const downloadResult = ref<{ success: boolean; message: string; url?: string } | null>(null)

// Filter options
const statusFilterOptions: SelectOption[] = [
  { label: 'All Status', value: '' },
  { label: 'Completed', value: 'completed' },
  { label: 'Running', value: 'running' },
  { label: 'Queued', value: 'queued' },
  { label: 'Failed', value: 'failed' },
]

// Tenant filter options (computed from snapshots)
const tenantFilterOptions = computed<SelectOption[]>(() => {
  const uniqueTenants = new Map<string, string>()
  adminStore.snapshots.forEach(s => {
    if (s.tenant_id && !uniqueTenants.has(s.tenant_id)) {
      uniqueTenants.set(s.tenant_id, s.tenant_name || s.tenant_id.slice(0, 8))
    }
  })
  return [
    { label: 'All Tenants', value: '' },
    ...Array.from(uniqueTenants.entries()).map(([id, name]) => ({
      label: name,
      value: id,
    })),
  ]
})

// Source filter options (computed from snapshots)
const sourceFilterOptions = computed<SelectOption[]>(() => {
  const uniqueSources = new Map<string, string>()
  // Filter by tenant if selected
  let snapshots = adminStore.snapshots
  if (tenantFilter.value) {
    snapshots = snapshots.filter(s => s.tenant_id === tenantFilter.value)
  }
  snapshots.forEach(s => {
    if (s.source_id && !uniqueSources.has(s.source_id)) {
      uniqueSources.set(s.source_id, s.source_name || s.source_id.slice(0, 8))
    }
  })
  return [
    { label: 'All Sources', value: '' },
    ...Array.from(uniqueSources.entries()).map(([id, name]) => ({
      label: name,
      value: id,
    })),
  ]
})

// Computed
const filteredSnapshots = computed(() => {
  let result = adminStore.snapshots

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(s =>
      s.id.toLowerCase().includes(query) ||
      s.source_name?.toLowerCase().includes(query) ||
      s.tenant_name?.toLowerCase().includes(query) ||
      s.worker_id?.toLowerCase().includes(query)
    )
  }

  if (statusFilter.value) {
    result = result.filter(s => s.status === statusFilter.value)
  }

  if (tenantFilter.value) {
    result = result.filter(s => s.tenant_id === tenantFilter.value)
  }

  if (sourceFilter.value) {
    result = result.filter(s => s.source_id === sourceFilter.value)
  }

  return result
})

const snapshotStats = computed(() => ({
  total: adminStore.snapshots.length,
  completed: adminStore.snapshots.filter(s => s.status === 'completed').length,
  pending: adminStore.snapshots.filter(s => s.status === 'pending' || s.status === 'running').length,
  failed: adminStore.snapshots.filter(s => s.status === 'failed').length,
  totalSize: adminStore.snapshots.reduce((acc, s) => acc + (s.size_bytes || 0), 0),
}))

onMounted(async () => {
  try {
    await adminStore.fetchSnapshots(200)
  } catch (error) {
    console.error('Failed to load snapshots:', error)
  } finally {
    isLoading.value = false
  }
})

function getStatusBadgeClass(status: string): string {
  switch (status) {
    case 'completed':
      return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
    case 'failed':
      return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
    case 'queued':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
    case 'pending':
    case 'running':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
  }
}

function getTypeBadgeClass(type: string): string {
  switch (type) {
    case 'ssh':
    case 'sftp':
      return 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200'
    case 'ftp':
      return 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200'
    case 'mysql':
      return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
    case 'postgresql':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
  }
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

function formatDate(date: string | null | undefined): string {
  if (!date) return 'N/A'
  return new Date(date).toLocaleString()
}

function formatDuration(ms: number | null | undefined): string {
  if (!ms) return 'N/A'
  if (ms < 1000) return `${ms}ms`
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return `${minutes}m ${remainingSeconds}s`
}

function openDetailDialog(snapshot: AdminSnapshot) {
  selectedSnapshot.value = snapshot
  downloadResult.value = null
  showDetailDialog.value = true
}

async function openLogsDialog(snapshot: AdminSnapshot) {
  selectedSnapshot.value = snapshot
  isLoadingLogs.value = true
  showLogsDialog.value = true
  
  try {
    selectedSnapshotLogs.value = await adminStore.fetchLogsForSnapshot(snapshot.id, 200)
  } catch (error) {
    console.error('Failed to fetch logs:', error)
  } finally {
    isLoadingLogs.value = false
  }
}

function refreshLogs() {
  if (selectedSnapshot.value) {
    openLogsDialog(selectedSnapshot.value)
  }
}

async function handleDownload(snapshot: AdminSnapshot) {
  isDownloading.value = true
  downloadResult.value = null

  try {
    // Call the restore endpoint to initiate download
    const response = await api.post<{ download_url: string; expires_at: string }>(
      `/v1/snapshots/${snapshot.id}/download`,
      {}
    )
    downloadResult.value = {
      success: true,
      message: `Download link generated! Expires at: ${new Date(response.data.expires_at).toLocaleString()}`,
      url: response.data.download_url,
    }
  } catch (error: unknown) {
    downloadResult.value = {
      success: false,
      message: error instanceof Error ? error.message : 'Failed to generate download link',
    }
  } finally {
    isDownloading.value = false
  }
}

async function handleRefresh() {
  isLoading.value = true
  try {
    await adminStore.fetchSnapshots(200)
  } catch (error) {
    console.error('Failed to refresh snapshots:', error)
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold">Snapshots</h1>
        <p class="text-muted-foreground">View backup snapshots across all tenants</p>
      </div>
      <Button variant="outline" @click="handleRefresh" :disabled="isLoading">
        <svg class="w-4 h-4 mr-2" :class="{ 'animate-spin': isLoading }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        Refresh
      </Button>
    </div>

    <!-- Stats cards -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold">{{ snapshotStats.total }}</div>
          <div class="text-sm text-muted-foreground">Total Snapshots</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-green-600">{{ snapshotStats.completed }}</div>
          <div class="text-sm text-muted-foreground">Completed</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-yellow-600">{{ snapshotStats.pending }}</div>
          <div class="text-sm text-muted-foreground">Pending/Running</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-red-600">{{ snapshotStats.failed }}</div>
          <div class="text-sm text-muted-foreground">Failed</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-blue-600">{{ formatBytes(snapshotStats.totalSize) }}</div>
          <div class="text-sm text-muted-foreground">Total Size</div>
        </CardContent>
      </Card>
    </div>

    <!-- Filters and search -->
    <Card>
      <CardContent class="pt-6">
        <div class="flex flex-col sm:flex-row gap-4">
          <div class="flex-1">
            <Input
              v-model="searchQuery"
              type="search"
              placeholder="Search by ID, source, tenant, or worker..."
            />
          </div>
          <div class="w-36">
            <Select
              v-model="statusFilter"
              :options="statusFilterOptions"
              placeholder="Status"
            />
          </div>
          <div class="w-40">
            <Select
              v-model="tenantFilter"
              :options="tenantFilterOptions"
              placeholder="Tenant"
            />
          </div>
          <div class="w-40">
            <Select
              v-model="sourceFilter"
              :options="sourceFilterOptions"
              placeholder="Source"
            />
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Snapshots list -->
    <Card>
      <CardContent class="p-0">
        <div v-if="isLoading" class="p-8 text-center text-muted-foreground">
          Loading snapshots...
        </div>
        <div v-else-if="filteredSnapshots.length === 0" class="p-8 text-center text-muted-foreground">
          No snapshots found. Backups will appear here once created.
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead class="border-b bg-muted/50">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">ID</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Tenant</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Source</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Status</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Size</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Duration</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Created</th>
                <th class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr
                v-for="snapshot in filteredSnapshots"
                :key="snapshot.id"
                class="hover:bg-muted/50 transition-colors"
              >
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm font-mono text-xs">{{ snapshot.id.slice(0, 8) }}...</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm">{{ snapshot.tenant_name || snapshot.tenant_id.slice(0, 8) + '...' }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="flex items-center gap-2">
                    <span :class="['px-2 py-0.5 text-xs rounded-full', getTypeBadgeClass(snapshot.source_type)]">
                      {{ snapshot.source_type?.toUpperCase() || 'N/A' }}
                    </span>
                    <span class="text-sm">{{ snapshot.source_name || snapshot.source_id.slice(0, 8) + '...' }}</span>
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <span :class="['px-2 py-1 text-xs rounded-full', getStatusBadgeClass(snapshot.status)]">
                    {{ snapshot.status }}
                  </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ formatBytes(snapshot.size_bytes) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ formatDuration(snapshot.duration_ms) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ formatDate(snapshot.created_at) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <div class="flex justify-end gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      @click="openLogsDialog(snapshot)"
                    >
                      Logs
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      @click="openDetailDialog(snapshot)"
                    >
                      Details
                    </Button>
                    <Button 
                      variant="secondary" 
                      size="sm" 
                      :disabled="snapshot.status !== 'completed' || isDownloading"
                      @click="handleDownload(snapshot)"
                    >
                      Download
                    </Button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>

    <!-- Detail Dialog -->
    <Dialog v-model:open="showDetailDialog">
      <div class="p-6 max-h-[80vh] overflow-y-auto">
        <h2 class="text-lg font-semibold mb-4">Snapshot Details</h2>

        <div v-if="selectedSnapshot" class="space-y-4">
          <!-- Basic Info -->
          <div class="grid grid-cols-2 gap-4">
            <div>
              <div class="text-sm text-muted-foreground">Snapshot ID</div>
              <div class="font-mono text-sm break-all">{{ selectedSnapshot.id }}</div>
            </div>
            <div>
              <div class="text-sm text-muted-foreground">Status</div>
              <span :class="['px-2 py-1 text-xs rounded-full', getStatusBadgeClass(selectedSnapshot.status)]">
                {{ selectedSnapshot.status }}
              </span>
            </div>
          </div>

          <div class="border-t pt-4">
            <h3 class="font-medium mb-3">Tenant & Source</h3>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <div class="text-sm text-muted-foreground">Tenant</div>
                <div class="text-sm">{{ selectedSnapshot.tenant_name || selectedSnapshot.tenant_id }}</div>
              </div>
              <div>
                <div class="text-sm text-muted-foreground">Source</div>
                <div class="text-sm flex items-center gap-2">
                  <span :class="['px-2 py-0.5 text-xs rounded-full', getTypeBadgeClass(selectedSnapshot.source_type)]">
                    {{ selectedSnapshot.source_type?.toUpperCase() }}
                  </span>
                  {{ selectedSnapshot.source_name || selectedSnapshot.source_id }}
                </div>
              </div>
            </div>
          </div>

          <div class="border-t pt-4">
            <h3 class="font-medium mb-3">Timing</h3>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <div class="text-sm text-muted-foreground">Created At</div>
                <div class="text-sm">{{ formatDate(selectedSnapshot.created_at) }}</div>
              </div>
              <div>
                <div class="text-sm text-muted-foreground">Started At</div>
                <div class="text-sm">{{ formatDate(selectedSnapshot.started_at) }}</div>
              </div>
              <div>
                <div class="text-sm text-muted-foreground">Finished At</div>
                <div class="text-sm">{{ formatDate(selectedSnapshot.finished_at) }}</div>
              </div>
              <div>
                <div class="text-sm text-muted-foreground">Duration</div>
                <div class="text-sm">{{ formatDuration(selectedSnapshot.duration_ms) }}</div>
              </div>
            </div>
          </div>

          <div class="border-t pt-4">
            <h3 class="font-medium mb-3">Storage</h3>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <div class="text-sm text-muted-foreground">Size</div>
                <div class="text-sm">{{ formatBytes(selectedSnapshot.size_bytes) }}</div>
              </div>
              <div>
                <div class="text-sm text-muted-foreground">Storage Backend</div>
                <div class="text-sm">{{ selectedSnapshot.storage_backend || 'N/A' }}</div>
              </div>
              <div>
                <div class="text-sm text-muted-foreground">Worker ID</div>
                <div class="text-sm font-mono text-xs">{{ selectedSnapshot.worker_id || 'N/A' }}</div>
              </div>
              <div>
                <div class="text-sm text-muted-foreground">Job ID</div>
                <div class="text-sm font-mono text-xs">{{ selectedSnapshot.job_id || 'N/A' }}</div>
              </div>
            </div>
          </div>

          <!-- Download Section -->
          <div v-if="selectedSnapshot.status === 'completed'" class="border-t pt-4">
            <h3 class="font-medium mb-3">Download</h3>
            
            <div v-if="downloadResult" :class="[
              'p-3 rounded-md text-sm mb-4',
              downloadResult.success ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
            ]">
              <div class="font-medium">{{ downloadResult.message }}</div>
              <a 
                v-if="downloadResult.url" 
                :href="downloadResult.url"
                target="_blank"
                class="text-blue-600 dark:text-blue-400 underline break-all mt-2 block"
              >
                {{ downloadResult.url }}
              </a>
            </div>

            <Button 
              variant="secondary"
              :disabled="isDownloading"
              @click="handleDownload(selectedSnapshot)"
            >
              {{ isDownloading ? 'Generating...' : 'Generate Download Link' }}
            </Button>
          </div>
        </div>

        <div class="flex justify-end gap-3 pt-4 mt-4 border-t">
          <Button variant="outline" @click="showDetailDialog = false">
            Close
          </Button>
        </div>
      </div>
    </Dialog>

    <!-- Logs Dialog -->
    <Dialog v-model:open="showLogsDialog" size="xl">
      <div class="p-6">
        <div class="flex items-center justify-between mb-4">
          <div>
            <h2 class="text-lg font-semibold">Snapshot Logs</h2>
            <p v-if="selectedSnapshot" class="text-sm text-muted-foreground">
              {{ selectedSnapshot.source_name }} ({{ selectedSnapshot.tenant_name }})
            </p>
          </div>
          <Button variant="outline" size="sm" @click="showLogsDialog = false">
            Close
          </Button>
        </div>
        
        <LogViewer
          v-if="selectedSnapshot"
          :logs="selectedSnapshotLogs"
          :is-loading="isLoadingLogs"
          :title="`Logs for ${selectedSnapshot.source_name}`"
          @refresh="refreshLogs"
        />
      </div>
    </Dialog>
  </div>
</template>
