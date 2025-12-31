<script setup lang="ts">
import { ref, computed } from 'vue'
import type { LogEntry, LogLevel } from '@/types'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import CardHeader from '@/components/ui/card/CardHeader.vue'
import CardTitle from '@/components/ui/card/CardTitle.vue'
import { Select, type SelectOption } from '@/components/ui/select'

interface Props {
  logs: LogEntry[]
  isLoading?: boolean
  title?: string
}

const props = withDefaults(defineProps<Props>(), {
  isLoading: false,
  title: 'Logs'
})

const emit = defineEmits<{
  refresh: []
}>()

const levelFilter = ref<LogLevel | 'all'>('all')
const searchQuery = ref('')
const autoScroll = ref(true)
const logContainer = ref<HTMLElement | null>(null)

const levelOptions: SelectOption[] = [
  { label: 'All Levels', value: 'all' },
  { label: 'Error', value: 'error' },
  { label: 'Warn', value: 'warn' },
  { label: 'Info', value: 'info' },
  { label: 'Debug', value: 'debug' },
]

const filteredLogs = computed(() => {
  let result = props.logs

  // Filter by level
  if (levelFilter.value !== 'all') {
    result = result.filter(log => log.level === levelFilter.value)
  }

  // Filter by search query
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(log =>
      log.message.toLowerCase().includes(query) ||
      log.worker_id?.toLowerCase().includes(query) ||
      log.job_id?.toLowerCase().includes(query) ||
      log.snapshot_id?.toLowerCase().includes(query) ||
      log.source_id?.toLowerCase().includes(query) ||
      log.schedule_id?.toLowerCase().includes(query)
    )
  }

  return result
})

const logStats = computed(() => ({
  total: props.logs.length,
  error: props.logs.filter(l => l.level === 'error').length,
  warn: props.logs.filter(l => l.level === 'warn').length,
  info: props.logs.filter(l => l.level === 'info').length,
  debug: props.logs.filter(l => l.level === 'debug').length,
}))

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

function formatTimestamp(timestamp: string): { date: string; time: string } {
  const d = new Date(timestamp)
  return {
    date: d.toLocaleString('en-US', { month: 'short', day: '2-digit' }),
    time: d.toLocaleString('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
  }
}

function handleRefresh() {
  emit('refresh')
}

function copyLogs() {
  const logText = filteredLogs.value
    .map(log => `[${log.timestamp}] [${log.level.toUpperCase()}] ${log.message}`)
    .join('\n')
  
  navigator.clipboard.writeText(logText)
}

function downloadLogs() {
  const logText = filteredLogs.value
    .map(log => `[${log.timestamp}] [${log.level.toUpperCase()}] ${log.message}`)
    .join('\n')
  
  const blob = new Blob([logText], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `logs-${new Date().toISOString().slice(0, 10)}.txt`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

function clearFilters() {
  levelFilter.value = 'all'
  searchQuery.value = ''
}

// Auto-scroll to bottom when logs are added
function scrollToBottom() {
  if (autoScroll.value && logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

// Watch for logs changes to auto-scroll
import { watch } from 'vue'
watch(() => props.logs, () => {
  if (autoScroll.value) {
    setTimeout(scrollToBottom, 50)
  }
}, { deep: true })
</script>

<template>
  <Card class="w-full overflow-hidden">
    <CardHeader class="pb-4">
      <div class="flex items-center justify-between gap-4">
        <div class="flex items-center gap-3">
          <div class="flex items-center justify-center w-8 h-8 rounded-lg bg-primary/10">
            <svg class="w-4 h-4 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
          </div>
          <CardTitle class="text-lg font-semibold">{{ title }}</CardTitle>
        </div>
        <div class="flex items-center gap-2">
          <Button variant="outline" size="sm" @click="handleRefresh" :disabled="isLoading" class="gap-2">
            <svg class="w-4 h-4" :class="{ 'animate-spin': isLoading }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span class="hidden sm:inline">Refresh</span>
          </Button>
          <Button variant="outline" size="sm" @click="copyLogs" :disabled="filteredLogs.length === 0" class="gap-2">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
            <span class="hidden sm:inline">Copy</span>
          </Button>
          <Button variant="outline" size="sm" @click="downloadLogs" :disabled="filteredLogs.length === 0" class="gap-2">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            <span class="hidden sm:inline">Download</span>
          </Button>
        </div>
      </div>
      
      <!-- Stats Pills -->
      <div v-if="logs.length > 0" class="flex items-center gap-2 flex-wrap mt-4">
        <div class="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-muted text-muted-foreground text-xs font-medium">
          <span class="text-foreground font-semibold">{{ logStats.total }}</span>
          <span>total</span>
        </div>
        <div v-if="logStats.error > 0" :class="['flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium', getStatBadgeClass('error')]">
          <span class="font-semibold">{{ logStats.error }}</span>
          <span>errors</span>
        </div>
        <div v-if="logStats.warn > 0" :class="['flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium', getStatBadgeClass('warn')]">
          <span class="font-semibold">{{ logStats.warn }}</span>
          <span>warnings</span>
        </div>
        <div v-if="logStats.info > 0" :class="['flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium', getStatBadgeClass('info')]">
          <span class="font-semibold">{{ logStats.info }}</span>
          <span>info</span>
        </div>
        <div v-if="logStats.debug > 0" :class="['flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium', getStatBadgeClass('debug')]">
          <span class="font-semibold">{{ logStats.debug }}</span>
          <span>debug</span>
        </div>
      </div>
      
      <!-- Filters -->
      <div class="flex flex-col sm:flex-row gap-3 mt-4">
        <div class="relative flex-1">
          <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            v-model="searchQuery"
            type="search"
            placeholder="Search logs..."
            class="flex w-full rounded-md border border-input bg-background pl-10 pr-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          />
        </div>
        <div class="w-full sm:w-40">
          <Select v-model="levelFilter" :options="levelOptions" />
        </div>
        <Button v-if="levelFilter !== 'all' || searchQuery" variant="ghost" size="sm" @click="clearFilters" class="gap-1.5">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
          Clear
        </Button>
        <label class="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer hover:text-foreground transition-colors whitespace-nowrap">
          <input
            v-model="autoScroll"
            type="checkbox"
            class="h-4 w-4 rounded border-input accent-primary"
          />
          Auto-scroll
        </label>
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
      <div v-else-if="filteredLogs.length === 0" class="flex flex-col items-center justify-center p-12 text-muted-foreground">
        <svg class="w-12 h-12 mb-3 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
        <span class="text-sm font-medium">{{ logs.length === 0 ? 'No logs available' : 'No logs match your filters' }}</span>
        <span v-if="logs.length > 0" class="text-xs mt-1">Try adjusting your search or filter</span>
      </div>
      
      <!-- Log Entries -->
      <div
        v-else
        ref="logContainer"
        class="h-[60vh] min-h-[500px] max-h-[70vh] overflow-y-auto font-mono text-sm border-t"
      >
        <div class="divide-y divide-border/50">
          <div
            v-for="(log, index) in filteredLogs"
            :key="log.id || index"
            :class="[
              'group flex gap-4 px-4 py-3 border-l-[3px] transition-colors',
              getLogRowClass(log.level)
            ]"
          >
            <!-- Timestamp Column -->
            <div class="flex flex-col items-end flex-shrink-0 w-[72px] text-[11px] tabular-nums">
              <span class="text-muted-foreground/70">{{ formatTimestamp(log.timestamp).date }}</span>
              <span class="text-foreground font-medium">{{ formatTimestamp(log.timestamp).time }}</span>
            </div>
            
            <!-- Level Badge -->
            <div class="flex-shrink-0 pt-0.5">
              <span :class="['inline-flex items-center justify-center w-14 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide rounded', getLevelBadgeClass(log.level)]">
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
    </CardContent>
  </Card>
</template>
