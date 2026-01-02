<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useWorkersStore } from '@/stores/workers'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import CardHeader from '@/components/ui/card/CardHeader.vue'
import CardTitle from '@/components/ui/card/CardTitle.vue'
import Button from '@/components/ui/button/Button.vue'
import type { Worker } from '@/types'

const workersStore = useWorkersStore()

const isLoading = ref(true)
const error = ref<string | null>(null)
const selectedWorker = ref<Worker | null>(null)
const autoRefreshEnabled = ref(true)
let refreshInterval: ReturnType<typeof setInterval> | null = null

// Summary stats
const totalWorkers = computed(() => workersStore.workers.length)
const onlineCount = computed(() => workersStore.onlineWorkers.length)
const offlineCount = computed(() => workersStore.offlineWorkers.length)
const healthyCount = computed(() => workersStore.healthyWorkers.length)
const warningCount = computed(() => workersStore.warningWorkers.length)
const criticalCount = computed(() => workersStore.criticalWorkers.length)

onMounted(async () => {
  await loadWorkers()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})

async function loadWorkers() {
  isLoading.value = true
  error.value = null
  try {
    await workersStore.fetchWorkers()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load workers'
  } finally {
    isLoading.value = false
  }
}

function startAutoRefresh() {
  if (refreshInterval) return
  refreshInterval = setInterval(async () => {
    if (autoRefreshEnabled.value) {
      try {
        await workersStore.fetchWorkers()
      } catch {
        // Silently fail on background refresh
      }
    }
  }, 30000) // Refresh every 30 seconds
}

function stopAutoRefresh() {
  if (refreshInterval) {
    clearInterval(refreshInterval)
    refreshInterval = null
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

function formatLastSeen(dateStr: string | undefined): string {
  if (!dateStr) return 'Never'
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffSecs = Math.floor(diffMs / 1000)
  
  if (diffSecs < 60) return 'Just now'
  if (diffSecs < 3600) return `${Math.floor(diffSecs / 60)}m ago`
  if (diffSecs < 86400) return `${Math.floor(diffSecs / 3600)}h ago`
  return `${Math.floor(diffSecs / 86400)}d ago`
}

function getHealthBadgeClasses(health: string): string {
  switch (health) {
    case 'healthy': return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    case 'warning': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
    case 'critical': return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
    case 'offline': return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400'
    default: return 'bg-gray-100 text-gray-600'
  }
}

function getProgressColor(percent: number): string {
  if (percent >= 95) return 'bg-red-500'
  if (percent >= 80) return 'bg-yellow-500'
  return 'bg-green-500'
}

function viewWorkerDetails(worker: Worker) {
  selectedWorker.value = worker
}

function closeDetails() {
  selectedWorker.value = null
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold">Workers</h1>
        <p class="text-muted-foreground">Monitor worker nodes and system resources</p>
      </div>
      <div class="flex items-center gap-4">
        <label class="flex items-center gap-2 text-sm text-muted-foreground">
          <input
            v-model="autoRefreshEnabled"
            type="checkbox"
            class="rounded border-gray-300"
          />
          Auto-refresh
        </label>
        <Button @click="loadWorkers" :disabled="isLoading">
          <svg v-if="isLoading" class="animate-spin -ml-1 mr-2 h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          Refresh
        </Button>
      </div>
    </div>

    <!-- Summary Cards -->
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <Card>
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium">Total Workers</CardTitle>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
          </svg>
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold">{{ totalWorkers }}</div>
          <p class="text-xs text-muted-foreground">
            {{ onlineCount }} online, {{ offlineCount }} offline
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium">Healthy</CardTitle>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-green-500">{{ healthyCount }}</div>
          <p class="text-xs text-muted-foreground">All systems nominal</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium">Warning</CardTitle>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-yellow-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-yellow-500">{{ warningCount }}</div>
          <p class="text-xs text-muted-foreground">Resources &gt; 80%</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium">Critical</CardTitle>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold text-red-500">{{ criticalCount }}</div>
          <p class="text-xs text-muted-foreground">Resources &gt; 95%</p>
        </CardContent>
      </Card>
    </div>

    <!-- Error message -->
    <div v-if="error" class="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-4">
      <div class="flex">
        <svg class="h-5 w-5 text-red-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
        </svg>
        <div class="ml-3">
          <p class="text-sm text-red-700 dark:text-red-400">{{ error }}</p>
        </div>
      </div>
    </div>

    <!-- Workers List -->
    <Card>
      <CardHeader>
        <CardTitle>Worker Nodes</CardTitle>
      </CardHeader>
      <CardContent>
        <!-- Loading State -->
        <div v-if="isLoading && workersStore.workers.length === 0" class="text-center py-8 text-muted-foreground">
          <svg class="animate-spin h-8 w-8 mx-auto mb-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          Loading workers...
        </div>

        <!-- Empty State -->
        <div v-else-if="workersStore.workers.length === 0" class="text-center py-8 text-muted-foreground">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 mx-auto mb-4 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
          </svg>
          <p>No workers registered</p>
          <p class="text-sm mt-1">Workers will appear here once they connect to the hub</p>
        </div>

        <!-- Workers Table -->
        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead>
              <tr class="border-b">
                <th class="text-left py-3 px-4 font-medium">Worker</th>
                <th class="text-left py-3 px-4 font-medium">Status</th>
                <th class="text-left py-3 px-4 font-medium">CPU</th>
                <th class="text-left py-3 px-4 font-medium">Memory</th>
                <th class="text-left py-3 px-4 font-medium">Disk</th>
                <th class="text-left py-3 px-4 font-medium">Active Jobs</th>
                <th class="text-left py-3 px-4 font-medium">Last Seen</th>
                <th class="text-right py-3 px-4 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="worker in workersStore.workers"
                :key="worker.id"
                class="border-b last:border-0 hover:bg-muted/50 cursor-pointer"
                @click="viewWorkerDetails(worker)"
              >
                <td class="py-3 px-4">
                  <div class="font-medium">{{ worker.name }}</div>
                  <div class="text-xs text-muted-foreground font-mono">{{ worker.id }}</div>
                </td>
                <td class="py-3 px-4">
                  <span :class="['px-2 py-1 text-xs font-medium rounded-full', getHealthBadgeClasses(worker.health)]">
                    {{ worker.health }}
                  </span>
                </td>
                <td class="py-3 px-4">
                  <div v-if="worker.system_metrics" class="space-y-1">
                    <div class="text-sm">{{ worker.system_metrics.cpu_percent.toFixed(1) }}%</div>
                    <div class="w-20 h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                      <div
                        :class="['h-full transition-all', getProgressColor(worker.system_metrics.cpu_percent)]"
                        :style="{ width: `${Math.min(worker.system_metrics.cpu_percent, 100)}%` }"
                      ></div>
                    </div>
                  </div>
                  <span v-else class="text-muted-foreground">-</span>
                </td>
                <td class="py-3 px-4">
                  <div v-if="worker.system_metrics" class="space-y-1">
                    <div class="text-sm">{{ worker.system_metrics.memory_percent.toFixed(1) }}%</div>
                    <div class="w-20 h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                      <div
                        :class="['h-full transition-all', getProgressColor(worker.system_metrics.memory_percent)]"
                        :style="{ width: `${Math.min(worker.system_metrics.memory_percent, 100)}%` }"
                      ></div>
                    </div>
                  </div>
                  <span v-else class="text-muted-foreground">-</span>
                </td>
                <td class="py-3 px-4">
                  <div v-if="worker.system_metrics" class="space-y-1">
                    <div class="text-sm">{{ worker.system_metrics.disk_percent.toFixed(1) }}%</div>
                    <div class="w-20 h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                      <div
                        :class="['h-full transition-all', getProgressColor(worker.system_metrics.disk_percent)]"
                        :style="{ width: `${Math.min(worker.system_metrics.disk_percent, 100)}%` }"
                      ></div>
                    </div>
                  </div>
                  <span v-else class="text-muted-foreground">-</span>
                </td>
                <td class="py-3 px-4">
                  <span v-if="worker.system_metrics" class="text-sm">
                    {{ worker.system_metrics.active_jobs }}
                  </span>
                  <span v-else class="text-muted-foreground">-</span>
                </td>
                <td class="py-3 px-4">
                  <span class="text-sm" :class="worker.health === 'offline' ? 'text-red-500' : ''">
                    {{ formatLastSeen(worker.last_seen_at) }}
                  </span>
                </td>
                <td class="py-3 px-4 text-right">
                  <Button variant="ghost" size="sm" @click.stop="viewWorkerDetails(worker)">
                    Details
                  </Button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>

    <!-- Worker Details Modal -->
    <div
      v-if="selectedWorker"
      class="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
      @click.self="closeDetails"
    >
      <Card class="w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <CardHeader class="flex flex-row items-center justify-between">
          <CardTitle>Worker Details</CardTitle>
          <Button variant="ghost" size="sm" @click="closeDetails">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
            </svg>
          </Button>
        </CardHeader>
        <CardContent class="space-y-6">
          <!-- Worker Info -->
          <div class="grid grid-cols-2 gap-4">
            <div>
              <div class="text-sm text-muted-foreground">Name</div>
              <div class="font-medium">{{ selectedWorker.name }}</div>
            </div>
            <div>
              <div class="text-sm text-muted-foreground">ID</div>
              <div class="font-mono text-sm">{{ selectedWorker.id }}</div>
            </div>
            <div>
              <div class="text-sm text-muted-foreground">Status</div>
              <span :class="['px-2 py-1 text-xs font-medium rounded-full', getHealthBadgeClasses(selectedWorker.health)]">
                {{ selectedWorker.health }}
              </span>
            </div>
            <div>
              <div class="text-sm text-muted-foreground">Storage Path</div>
              <div class="font-mono text-sm">{{ selectedWorker.storage_base_path }}</div>
            </div>
            <div>
              <div class="text-sm text-muted-foreground">Last Seen</div>
              <div>{{ formatLastSeen(selectedWorker.last_seen_at) }}</div>
            </div>
            <div v-if="selectedWorker.system_metrics">
              <div class="text-sm text-muted-foreground">Uptime</div>
              <div>{{ formatUptime(selectedWorker.system_metrics.uptime_seconds) }}</div>
            </div>
          </div>

          <!-- System Metrics -->
          <div v-if="selectedWorker.system_metrics" class="space-y-4">
            <h3 class="font-medium">System Resources</h3>
            
            <!-- CPU -->
            <div class="space-y-2">
              <div class="flex justify-between text-sm">
                <span>CPU</span>
                <span>{{ selectedWorker.system_metrics.cpu_percent.toFixed(1) }}%</span>
              </div>
              <div class="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                <div
                  :class="['h-full transition-all', getProgressColor(selectedWorker.system_metrics.cpu_percent)]"
                  :style="{ width: `${Math.min(selectedWorker.system_metrics.cpu_percent, 100)}%` }"
                ></div>
              </div>
            </div>

            <!-- Memory -->
            <div class="space-y-2">
              <div class="flex justify-between text-sm">
                <span>Memory</span>
                <span>
                  {{ formatBytes(selectedWorker.system_metrics.memory_used_bytes) }} / 
                  {{ formatBytes(selectedWorker.system_metrics.memory_total_bytes) }}
                  ({{ selectedWorker.system_metrics.memory_percent.toFixed(1) }}%)
                </span>
              </div>
              <div class="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                <div
                  :class="['h-full transition-all', getProgressColor(selectedWorker.system_metrics.memory_percent)]"
                  :style="{ width: `${Math.min(selectedWorker.system_metrics.memory_percent, 100)}%` }"
                ></div>
              </div>
            </div>

            <!-- Disk -->
            <div class="space-y-2">
              <div class="flex justify-between text-sm">
                <span>Disk</span>
                <span>
                  {{ formatBytes(selectedWorker.system_metrics.disk_used_bytes) }} / 
                  {{ formatBytes(selectedWorker.system_metrics.disk_total_bytes) }}
                  ({{ selectedWorker.system_metrics.disk_percent.toFixed(1) }}%)
                </span>
              </div>
              <div class="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                <div
                  :class="['h-full transition-all', getProgressColor(selectedWorker.system_metrics.disk_percent)]"
                  :style="{ width: `${Math.min(selectedWorker.system_metrics.disk_percent, 100)}%` }"
                ></div>
              </div>
              <div class="text-xs text-muted-foreground">
                {{ formatBytes(selectedWorker.system_metrics.disk_free_bytes) }} free
              </div>
            </div>

            <!-- Active Jobs -->
            <div class="flex justify-between items-center pt-2 border-t">
              <span class="text-sm">Active Jobs</span>
              <span class="text-lg font-medium">{{ selectedWorker.system_metrics.active_jobs }}</span>
            </div>
          </div>

          <!-- No Metrics -->
          <div v-else class="text-center py-4 text-muted-foreground">
            <p>No system metrics available</p>
            <p class="text-sm">Worker may be offline or not reporting metrics</p>
          </div>

          <!-- Capabilities -->
          <div v-if="selectedWorker.capabilities" class="space-y-2">
            <h3 class="font-medium">Capabilities</h3>
            <pre class="bg-muted p-3 rounded-md text-sm overflow-x-auto">{{ JSON.stringify(selectedWorker.capabilities, null, 2) }}</pre>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
