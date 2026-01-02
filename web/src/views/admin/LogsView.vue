<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useAdminStore } from '@/stores/admin'
import type { LogLevel, SystemLogsParams, AuditEventsParams, AuditAction, AuditTargetType } from '@/types'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import CardHeader from '@/components/ui/card/CardHeader.vue'
import CardTitle from '@/components/ui/card/CardTitle.vue'
import Input from '@/components/ui/input/Input.vue'
import { Select, type SelectOption } from '@/components/ui/select'

const adminStore = useAdminStore()

// Tab state
const activeTab = ref<'logs' | 'audit'>('logs')

// ==================== System Logs State ====================
const isLoading = ref(true)
const searchQuery = ref('')
const levelFilter = ref<LogLevel | 'all'>('all')
const sourceFilter = ref('')
const workerFilter = ref('')
const currentPage = ref(1)
const pageSize = ref('100')
const totalLogs = ref(0)

// Debounce timer for search
let searchTimeout: ReturnType<typeof setTimeout> | null = null

// Filter options
const levelOptions: SelectOption[] = [
  { label: 'All Levels', value: 'all' },
  { label: 'Error', value: 'error' },
  { label: 'Warn', value: 'warn' },
  { label: 'Info', value: 'info' },
  { label: 'Debug', value: 'debug' },
]

// Page size options
const pageSizeOptions: SelectOption[] = [
  { label: '50 per page', value: '50' },
  { label: '100 per page', value: '100' },
  { label: '200 per page', value: '200' },
  { label: '500 per page', value: '500' },
]

// Computed
const displayedLogs = computed(() => adminStore.systemLogs)

const logStats = computed(() => ({
  total: totalLogs.value,
  displayed: displayedLogs.value.length,
  error: displayedLogs.value.filter(l => l.level === 'error').length,
  warn: displayedLogs.value.filter(l => l.level === 'warn').length,
  info: displayedLogs.value.filter(l => l.level === 'info').length,
  debug: displayedLogs.value.filter(l => l.level === 'debug').length,
}))

const totalPages = computed(() => Math.ceil(totalLogs.value / parseInt(pageSize.value)))
const hasNextPage = computed(() => currentPage.value < totalPages.value)
const hasPreviousPage = computed(() => currentPage.value > 1)

// ==================== Audit Events State ====================
const isLoadingAudit = ref(false)
const auditSearchQuery = ref('')
const actionFilter = ref<AuditAction | 'all'>('all')
const targetTypeFilter = ref<AuditTargetType | 'all'>('all')
const auditCurrentPage = ref(1)
const auditPageSize = ref('100')
const totalAuditEvents = ref(0)

// Debounce timer for audit search
let auditSearchTimeout: ReturnType<typeof setTimeout> | null = null

// Action filter options
const actionOptions: SelectOption[] = [
  { label: 'All Actions', value: 'all' },
  { label: 'Create Source', value: 'create_source' },
  { label: 'Update Source', value: 'update_source' },
  { label: 'Delete Source', value: 'delete_source' },
  { label: 'Create Schedule', value: 'create_schedule' },
  { label: 'Update Schedule', value: 'update_schedule' },
  { label: 'Delete Schedule', value: 'delete_schedule' },
  { label: 'Delete Snapshot', value: 'delete_snapshot' },
  { label: 'Trigger Backup', value: 'trigger_backup' },
  { label: 'Create User', value: 'create_user' },
  { label: 'Update User', value: 'update_user' },
  { label: 'Delete User', value: 'delete_user' },
  { label: 'Create Tenant', value: 'create_tenant' },
  { label: 'Delete Tenant', value: 'delete_tenant' },
  { label: 'Update Setting', value: 'update_setting' },
]

// Target type filter options
const targetTypeOptions: SelectOption[] = [
  { label: 'All Types', value: 'all' },
  { label: 'Source', value: 'source' },
  { label: 'Schedule', value: 'schedule' },
  { label: 'Snapshot', value: 'snapshot' },
  { label: 'User', value: 'user' },
  { label: 'Tenant', value: 'tenant' },
  { label: 'Setting', value: 'setting' },
]

// Computed for audit
const displayedAuditEvents = computed(() => adminStore.auditEvents)

const auditTotalPages = computed(() => Math.ceil(totalAuditEvents.value / parseInt(auditPageSize.value)))
const auditHasNextPage = computed(() => auditCurrentPage.value < auditTotalPages.value)
const auditHasPreviousPage = computed(() => auditCurrentPage.value > 1)

// ==================== Watchers ====================

// Watch for filter changes with debounce
watch([levelFilter, sourceFilter, workerFilter], () => {
  currentPage.value = 1
  fetchLogs()
})

watch(searchQuery, () => {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    currentPage.value = 1
    fetchLogs()
  }, 300)
})

watch(pageSize, () => {
  currentPage.value = 1
  fetchLogs()
})

// Audit watchers
watch([actionFilter, targetTypeFilter], () => {
  auditCurrentPage.value = 1
  fetchAuditEvents()
})

watch(auditSearchQuery, () => {
  if (auditSearchTimeout) clearTimeout(auditSearchTimeout)
  auditSearchTimeout = setTimeout(() => {
    auditCurrentPage.value = 1
    fetchAuditEvents()
  }, 300)
})

watch(auditPageSize, () => {
  auditCurrentPage.value = 1
  fetchAuditEvents()
})

// Tab change handler
watch(activeTab, (newTab) => {
  if (newTab === 'audit' && displayedAuditEvents.value.length === 0) {
    fetchAuditEvents()
  }
})

// ==================== Functions ====================

async function fetchLogs() {
  isLoading.value = true
  try {
    const pageSizeNum = parseInt(pageSize.value)
    const params: SystemLogsParams = {
      limit: pageSizeNum,
      offset: (currentPage.value - 1) * pageSizeNum,
    }
    
    if (levelFilter.value !== 'all') {
      params.level = levelFilter.value
    }
    if (searchQuery.value) {
      params.search = searchQuery.value
    }
    if (sourceFilter.value) {
      params.source_id = sourceFilter.value
    }
    if (workerFilter.value) {
      params.worker_id = workerFilter.value
    }

    const result = await adminStore.fetchSystemLogs(params)
    totalLogs.value = result.total
  } catch (error) {
    console.error('Failed to load logs:', error)
  } finally {
    isLoading.value = false
  }
}

async function fetchAuditEvents() {
  isLoadingAudit.value = true
  try {
    const pageSizeNum = parseInt(auditPageSize.value)
    const params: AuditEventsParams = {
      limit: pageSizeNum,
      offset: (auditCurrentPage.value - 1) * pageSizeNum,
    }
    
    if (actionFilter.value !== 'all') {
      params.action = actionFilter.value
    }
    if (targetTypeFilter.value !== 'all') {
      params.target_type = targetTypeFilter.value
    }
    if (auditSearchQuery.value) {
      params.search = auditSearchQuery.value
    }

    const result = await adminStore.fetchAuditEvents(params)
    totalAuditEvents.value = result.total
  } catch (error) {
    console.error('Failed to load audit events:', error)
  } finally {
    isLoadingAudit.value = false
  }
}

onMounted(async () => {
  await fetchLogs()
})

function getLevelBadgeClass(level: LogLevel): string {
  switch (level) {
    case 'error':
      return 'bg-destructive text-destructive-foreground'
    case 'warn':
      return 'bg-amber-500 text-white dark:bg-amber-600'
    case 'info':
      return 'bg-primary text-primary-foreground'
    case 'debug':
      return 'bg-muted text-muted-foreground'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

function getLogRowClass(level: LogLevel): string {
  switch (level) {
    case 'error':
      return 'border-l-destructive bg-destructive/5 hover:bg-destructive/10'
    case 'warn':
      return 'border-l-amber-500 bg-amber-500/5 hover:bg-amber-500/10'
    case 'info':
      return 'border-l-primary bg-transparent hover:bg-muted/50'
    case 'debug':
      return 'border-l-muted-foreground/30 bg-transparent hover:bg-muted/30'
    default:
      return 'border-l-muted-foreground/30 bg-transparent hover:bg-muted/30'
  }
}

function getStatBadgeClass(level: LogLevel): string {
  switch (level) {
    case 'error':
      return 'bg-destructive/10 text-destructive dark:bg-destructive/20'
    case 'warn':
      return 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
    case 'info':
      return 'bg-primary/10 text-primary'
    case 'debug':
      return 'bg-muted text-muted-foreground'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

function getActionBadgeClass(action: string): string {
  if (action.startsWith('delete')) {
    return 'bg-destructive text-destructive-foreground'
  }
  if (action.startsWith('create')) {
    return 'bg-green-500 text-white dark:bg-green-600'
  }
  if (action.startsWith('update')) {
    return 'bg-blue-500 text-white dark:bg-blue-600'
  }
  if (action === 'trigger_backup') {
    return 'bg-purple-500 text-white dark:bg-purple-600'
  }
  return 'bg-muted text-muted-foreground'
}

function formatActionLabel(action: string): string {
  return action.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ')
}

function formatTimestamp(timestamp: string): { date: string; time: string } {
  const d = new Date(timestamp)
  return {
    date: d.toLocaleString('en-US', { month: 'short', day: '2-digit' }),
    time: d.toLocaleString('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
  }
}

function formatFullTimestamp(timestamp: string): string {
  const d = new Date(timestamp)
  return d.toLocaleString('en-US', { 
    month: 'short', 
    day: '2-digit', 
    year: 'numeric',
    hour: '2-digit', 
    minute: '2-digit', 
    second: '2-digit', 
    hour12: false 
  })
}

function clearFilters() {
  levelFilter.value = 'all'
  searchQuery.value = ''
  sourceFilter.value = ''
  workerFilter.value = ''
  currentPage.value = 1
  fetchLogs()
}

function clearAuditFilters() {
  actionFilter.value = 'all'
  targetTypeFilter.value = 'all'
  auditSearchQuery.value = ''
  auditCurrentPage.value = 1
  fetchAuditEvents()
}

function nextPage() {
  if (hasNextPage.value) {
    currentPage.value++
    fetchLogs()
  }
}

function previousPage() {
  if (hasPreviousPage.value) {
    currentPage.value--
    fetchLogs()
  }
}

function auditNextPage() {
  if (auditHasNextPage.value) {
    auditCurrentPage.value++
    fetchAuditEvents()
  }
}

function auditPreviousPage() {
  if (auditHasPreviousPage.value) {
    auditCurrentPage.value--
    fetchAuditEvents()
  }
}

function copyLogs() {
  const logText = displayedLogs.value
    .map(log => `[${log.timestamp}] [${log.level.toUpperCase()}] ${log.message}`)
    .join('\n')
  
  navigator.clipboard.writeText(logText)
}

function downloadLogs() {
  const logText = displayedLogs.value
    .map(log => `[${log.timestamp}] [${log.level.toUpperCase()}] ${log.message}`)
    .join('\n')
  
  const blob = new Blob([logText], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `system-logs-${new Date().toISOString().slice(0, 10)}.txt`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

function downloadAuditEvents() {
  const auditText = displayedAuditEvents.value
    .map(e => `[${e.created_at}] ${e.action} - ${e.target_type || ''}:${e.target_name || e.target_id || ''} by ${e.actor_email || 'system'} from ${e.ip_address || 'unknown'}`)
    .join('\n')
  
  const blob = new Blob([auditText], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `audit-trail-${new Date().toISOString().slice(0, 10)}.txt`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold">Logs & Audit</h1>
        <p class="text-muted-foreground">View system logs and track administrative actions</p>
      </div>
    </div>

    <!-- Tabs -->
    <div class="flex gap-1 p-1 bg-muted rounded-lg w-fit">
      <button
        :class="[
          'px-4 py-2 rounded-md text-sm font-medium transition-colors',
          activeTab === 'logs' 
            ? 'bg-background text-foreground shadow-sm' 
            : 'text-muted-foreground hover:text-foreground'
        ]"
        @click="activeTab = 'logs'"
      >
        <svg class="w-4 h-4 inline-block mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
        System Logs
      </button>
      <button
        :class="[
          'px-4 py-2 rounded-md text-sm font-medium transition-colors',
          activeTab === 'audit' 
            ? 'bg-background text-foreground shadow-sm' 
            : 'text-muted-foreground hover:text-foreground'
        ]"
        @click="activeTab = 'audit'"
      >
        <svg class="w-4 h-4 inline-block mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
        </svg>
        Audit Trail
      </button>
    </div>

    <!-- ==================== System Logs Tab ==================== -->
    <template v-if="activeTab === 'logs'">
      <!-- Action buttons -->
      <div class="flex items-center justify-end gap-2">
        <Button variant="outline" size="sm" @click="copyLogs" :disabled="displayedLogs.length === 0">
          <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
          </svg>
          Copy
        </Button>
        <Button variant="outline" size="sm" @click="downloadLogs" :disabled="displayedLogs.length === 0">
          <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          Download
        </Button>
        <Button variant="outline" @click="fetchLogs" :disabled="isLoading">
          <svg class="w-4 h-4 mr-2" :class="{ 'animate-spin': isLoading }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          Refresh
        </Button>
      </div>

      <!-- Stats cards -->
      <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-4">
        <Card>
          <CardContent class="pt-6">
            <div class="text-2xl font-bold">{{ logStats.total.toLocaleString() }}</div>
            <div class="text-sm text-muted-foreground">Total Logs</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent class="pt-6">
            <div class="text-2xl font-bold">{{ logStats.displayed }}</div>
            <div class="text-sm text-muted-foreground">Displayed</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent class="pt-6">
            <div class="text-2xl font-bold text-destructive">{{ logStats.error }}</div>
            <div class="text-sm text-muted-foreground">Errors</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent class="pt-6">
            <div class="text-2xl font-bold text-amber-600">{{ logStats.warn }}</div>
            <div class="text-sm text-muted-foreground">Warnings</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent class="pt-6">
            <div class="text-2xl font-bold text-primary">{{ logStats.info }}</div>
            <div class="text-sm text-muted-foreground">Info</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent class="pt-6">
            <div class="text-2xl font-bold text-muted-foreground">{{ logStats.debug }}</div>
            <div class="text-sm text-muted-foreground">Debug</div>
          </CardContent>
        </Card>
      </div>

      <!-- Filters -->
      <Card>
        <CardContent class="pt-6">
          <div class="flex flex-col lg:flex-row gap-4">
            <div class="flex-1">
              <Input
                v-model="searchQuery"
                type="search"
                placeholder="Search log messages, job IDs, worker IDs, snapshot IDs..."
              />
            </div>
            <div class="w-full lg:w-36">
              <Select
                v-model="levelFilter"
                :options="levelOptions"
                placeholder="Level"
              />
            </div>
            <div class="w-full lg:w-40">
              <Input
                v-model="sourceFilter"
                type="text"
                placeholder="Source ID..."
              />
            </div>
            <div class="w-full lg:w-40">
              <Input
                v-model="workerFilter"
                type="text"
                placeholder="Worker ID..."
              />
            </div>
            <div class="w-full lg:w-40">
              <Select
                v-model="pageSize"
                :options="pageSizeOptions"
              />
            </div>
            <Button 
              v-if="levelFilter !== 'all' || searchQuery || sourceFilter || workerFilter" 
              variant="ghost" 
              size="sm" 
              @click="clearFilters"
              class="gap-1.5"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
              Clear
            </Button>
          </div>
        </CardContent>
      </Card>

      <!-- Logs list -->
      <Card class="overflow-hidden">
        <CardHeader class="pb-4">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div class="flex items-center justify-center w-8 h-8 rounded-lg bg-primary/10">
                <svg class="w-4 h-4 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
              </div>
              <CardTitle class="text-lg font-semibold">Log Entries</CardTitle>
            </div>
            
            <!-- Stats Pills -->
            <div v-if="displayedLogs.length > 0" class="flex items-center gap-2 flex-wrap">
              <div v-if="logStats.error > 0" :class="['flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium', getStatBadgeClass('error')]">
                <span class="font-semibold">{{ logStats.error }}</span>
                <span>errors</span>
              </div>
              <div v-if="logStats.warn > 0" :class="['flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium', getStatBadgeClass('warn')]">
                <span class="font-semibold">{{ logStats.warn }}</span>
                <span>warnings</span>
              </div>
            </div>
          </div>
        </CardHeader>

        <CardContent class="p-0">
          <!-- Loading State -->
          <div v-if="isLoading" class="flex flex-col items-center justify-center p-12 text-muted-foreground">
            <svg class="w-8 h-8 animate-spin mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span class="text-sm">Loading logs...</span>
          </div>

          <!-- Empty State -->
          <div v-else-if="displayedLogs.length === 0" class="flex flex-col items-center justify-center p-12 text-muted-foreground">
            <svg class="w-12 h-12 mb-3 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            <span class="text-sm font-medium">{{ totalLogs === 0 ? 'No logs available' : 'No logs match your filters' }}</span>
            <span v-if="totalLogs > 0" class="text-xs mt-1">Try adjusting your search or filter</span>
          </div>

          <!-- Log Entries -->
          <div v-else class="max-h-[60vh] overflow-y-auto font-mono text-sm border-t">
            <div class="divide-y divide-border/50">
              <div
                v-for="(log, index) in displayedLogs"
                :key="log.id || index"
                :class="[
                  'group flex gap-4 px-4 py-3 border-l-[3px] transition-colors',
                  getLogRowClass(log.level as LogLevel)
                ]"
              >
                <!-- Timestamp Column -->
                <div class="flex flex-col items-end flex-shrink-0 w-[72px] text-[11px] tabular-nums">
                  <span class="text-muted-foreground/70">{{ formatTimestamp(log.timestamp).date }}</span>
                  <span class="text-foreground font-medium">{{ formatTimestamp(log.timestamp).time }}</span>
                </div>

                <!-- Level Badge -->
                <div class="flex-shrink-0 pt-0.5">
                  <span :class="['inline-flex items-center justify-center w-14 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide rounded', getLevelBadgeClass(log.level as LogLevel)]">
                    {{ log.level }}
                  </span>
                </div>

                <!-- Message Content -->
                <div class="flex-1 min-w-0 space-y-2">
                  <p class="text-foreground leading-relaxed break-words whitespace-pre-wrap">{{ log.message }}</p>

                  <!-- Context Tags -->
                  <div v-if="log.worker_id || log.job_id || log.snapshot_id || log.source_id || log.schedule_id" class="flex gap-2 flex-wrap">
                    <span v-if="log.worker_id" class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-muted text-[10px] text-muted-foreground">
                      <span class="opacity-60">worker</span>
                      <span class="font-medium text-foreground/80">{{ log.worker_id.slice(0, 8) }}</span>
                    </span>
                    <span v-if="log.job_id" class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-muted text-[10px] text-muted-foreground">
                      <span class="opacity-60">job</span>
                      <span class="font-medium text-foreground/80">{{ log.job_id.slice(0, 8) }}</span>
                    </span>
                    <span v-if="log.snapshot_id" class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-muted text-[10px] text-muted-foreground">
                      <span class="opacity-60">snapshot</span>
                      <span class="font-medium text-foreground/80">{{ log.snapshot_id.slice(0, 8) }}</span>
                    </span>
                    <span v-if="log.source_id" class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-muted text-[10px] text-muted-foreground">
                      <span class="opacity-60">source</span>
                      <span class="font-medium text-foreground/80">{{ log.source_id.slice(0, 8) }}</span>
                    </span>
                    <span v-if="log.schedule_id" class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-muted text-[10px] text-muted-foreground">
                      <span class="opacity-60">schedule</span>
                      <span class="font-medium text-foreground/80">{{ log.schedule_id.slice(0, 8) }}</span>
                    </span>
                  </div>

                  <!-- Details Expandable -->
                  <div v-if="log.details && Object.keys(log.details).length > 0" class="mt-2">
                    <pre class="p-3 rounded-md bg-muted/50 border text-[11px] text-muted-foreground overflow-x-auto leading-relaxed">{{ JSON.stringify(log.details, null, 2) }}</pre>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Pagination -->
          <div v-if="totalLogs > parseInt(pageSize)" class="px-4 py-3 bg-muted/30 border-t flex items-center justify-between">
            <div class="text-sm text-muted-foreground">
              Showing {{ ((currentPage - 1) * parseInt(pageSize)) + 1 }} - {{ Math.min(currentPage * parseInt(pageSize), totalLogs) }} of {{ totalLogs.toLocaleString() }} logs
            </div>
            <div class="flex items-center gap-2">
              <Button 
                variant="outline" 
                size="sm" 
                :disabled="!hasPreviousPage || isLoading"
                @click="previousPage"
              >
                <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
                </svg>
                Previous
              </Button>
              <span class="text-sm text-muted-foreground px-2">
                Page {{ currentPage }} of {{ totalPages }}
              </span>
              <Button 
                variant="outline" 
                size="sm" 
                :disabled="!hasNextPage || isLoading"
                @click="nextPage"
              >
                Next
                <svg class="w-4 h-4 ml-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </template>

    <!-- ==================== Audit Trail Tab ==================== -->
    <template v-if="activeTab === 'audit'">
      <!-- Action buttons -->
      <div class="flex items-center justify-end gap-2">
        <Button variant="outline" size="sm" @click="downloadAuditEvents" :disabled="displayedAuditEvents.length === 0">
          <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          Download
        </Button>
        <Button variant="outline" @click="fetchAuditEvents" :disabled="isLoadingAudit">
          <svg class="w-4 h-4 mr-2" :class="{ 'animate-spin': isLoadingAudit }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          Refresh
        </Button>
      </div>

      <!-- Stats cards -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <Card>
          <CardContent class="pt-6">
            <div class="text-2xl font-bold">{{ totalAuditEvents.toLocaleString() }}</div>
            <div class="text-sm text-muted-foreground">Total Events</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent class="pt-6">
            <div class="text-2xl font-bold">{{ displayedAuditEvents.length }}</div>
            <div class="text-sm text-muted-foreground">Displayed</div>
          </CardContent>
        </Card>
      </div>

      <!-- Filters -->
      <Card>
        <CardContent class="pt-6">
          <div class="flex flex-col lg:flex-row gap-4">
            <div class="flex-1">
              <Input
                v-model="auditSearchQuery"
                type="search"
                placeholder="Search by actor email, target name..."
              />
            </div>
            <div class="w-full lg:w-48">
              <Select
                v-model="actionFilter"
                :options="actionOptions"
                placeholder="Action"
              />
            </div>
            <div class="w-full lg:w-40">
              <Select
                v-model="targetTypeFilter"
                :options="targetTypeOptions"
                placeholder="Target Type"
              />
            </div>
            <div class="w-full lg:w-40">
              <Select
                v-model="auditPageSize"
                :options="pageSizeOptions"
              />
            </div>
            <Button 
              v-if="actionFilter !== 'all' || targetTypeFilter !== 'all' || auditSearchQuery" 
              variant="ghost" 
              size="sm" 
              @click="clearAuditFilters"
              class="gap-1.5"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
              Clear
            </Button>
          </div>
        </CardContent>
      </Card>

      <!-- Audit Events list -->
      <Card class="overflow-hidden">
        <CardHeader class="pb-4">
          <div class="flex items-center gap-3">
            <div class="flex items-center justify-center w-8 h-8 rounded-lg bg-primary/10">
              <svg class="w-4 h-4 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
              </svg>
            </div>
            <CardTitle class="text-lg font-semibold">Audit Events</CardTitle>
          </div>
        </CardHeader>

        <CardContent class="p-0">
          <!-- Loading State -->
          <div v-if="isLoadingAudit" class="flex flex-col items-center justify-center p-12 text-muted-foreground">
            <svg class="w-8 h-8 animate-spin mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span class="text-sm">Loading audit events...</span>
          </div>

          <!-- Empty State -->
          <div v-else-if="displayedAuditEvents.length === 0" class="flex flex-col items-center justify-center p-12 text-muted-foreground">
            <svg class="w-12 h-12 mb-3 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
            </svg>
            <span class="text-sm font-medium">{{ totalAuditEvents === 0 ? 'No audit events yet' : 'No events match your filters' }}</span>
            <span class="text-xs mt-1 text-center">Audit events are created when administrators<br/>modify sources, schedules, or snapshots</span>
          </div>

          <!-- Audit Entries -->
          <div v-else class="max-h-[60vh] overflow-y-auto border-t">
            <div class="divide-y divide-border/50">
              <div
                v-for="event in displayedAuditEvents"
                :key="event.id"
                class="group flex gap-4 px-4 py-4 hover:bg-muted/50 transition-colors"
              >
                <!-- Action Badge -->
                <div class="flex-shrink-0">
                  <span :class="['inline-flex items-center px-2.5 py-1 text-xs font-medium rounded', getActionBadgeClass(event.action)]">
                    {{ formatActionLabel(event.action) }}
                  </span>
                </div>

                <!-- Content -->
                <div class="flex-1 min-w-0 space-y-1">
                  <!-- Target info -->
                  <div class="flex items-center gap-2 flex-wrap">
                    <span v-if="event.target_type" class="text-sm text-muted-foreground capitalize">{{ event.target_type }}:</span>
                    <span class="text-sm font-medium">{{ event.target_name || (event.target_id ? event.target_id.slice(0, 8) + '...' : 'N/A') }}</span>
                  </div>

                  <!-- Actor and IP -->
                  <div class="flex items-center gap-4 text-xs text-muted-foreground">
                    <span class="flex items-center gap-1">
                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                      </svg>
                      {{ event.actor_email || 'System' }}
                    </span>
                    <span v-if="event.ip_address" class="flex items-center gap-1">
                      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
                      </svg>
                      {{ event.ip_address }}
                    </span>
                  </div>
                </div>

                <!-- Timestamp -->
                <div class="flex-shrink-0 text-right">
                  <div class="text-xs text-muted-foreground">
                    {{ formatFullTimestamp(event.created_at) }}
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Pagination -->
          <div v-if="totalAuditEvents > parseInt(auditPageSize)" class="px-4 py-3 bg-muted/30 border-t flex items-center justify-between">
            <div class="text-sm text-muted-foreground">
              Showing {{ ((auditCurrentPage - 1) * parseInt(auditPageSize)) + 1 }} - {{ Math.min(auditCurrentPage * parseInt(auditPageSize), totalAuditEvents) }} of {{ totalAuditEvents.toLocaleString() }} events
            </div>
            <div class="flex items-center gap-2">
              <Button 
                variant="outline" 
                size="sm" 
                :disabled="!auditHasPreviousPage || isLoadingAudit"
                @click="auditPreviousPage"
              >
                <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
                </svg>
                Previous
              </Button>
              <span class="text-sm text-muted-foreground px-2">
                Page {{ auditCurrentPage }} of {{ auditTotalPages }}
              </span>
              <Button 
                variant="outline" 
                size="sm" 
                :disabled="!auditHasNextPage || isLoadingAudit"
                @click="auditNextPage"
              >
                Next
                <svg class="w-4 h-4 ml-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </template>
  </div>
</template>
