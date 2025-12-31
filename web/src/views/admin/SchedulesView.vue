<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { useAdminStore } from '@/stores/admin'
import type { Schedule, AdminCreateScheduleRequest, AdminUpdateScheduleRequest } from '@/types'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import Input from '@/components/ui/input/Input.vue'
import Label from '@/components/ui/label/Label.vue'
import Dialog from '@/components/ui/dialog/Dialog.vue'

const adminStore = useAdminStore()

const searchQuery = ref('')
const statusFilter = ref('')
const isLoading = ref(true)
const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showDeleteDialog = ref(false)
const isCreating = ref(false)
const isUpdating = ref(false)
const isDeleting = ref(false)
const createError = ref('')
const editError = ref('')
const deleteError = ref('')
const editingSchedule = ref<Schedule | null>(null)
const deletingSchedule = ref<Schedule | null>(null)

// Clock functionality
const currentTime = ref(new Date())
const timeInterval = ref<number | null>(null)

// Update time every second
const updateTime = () => {
  currentTime.value = new Date()
}

const formatTime = (date: Date, timezone?: string) => {
  return new Intl.DateTimeFormat('en-US', {
    timeZone: timezone,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
    weekday: 'short',
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  }).format(date)
}

const statusFilterOptions = [
  { label: 'All Status', value: '' },
  { label: 'Enabled', value: 'enabled' },
  { label: 'Disabled', value: 'disabled' },
]

// Timezone options for select
const timezoneOptions = [
  { label: 'UTC', value: 'UTC' },
  { label: 'America/New_York (EST/EDT)', value: 'America/New_York' },
  { label: 'America/Chicago (CST/CDT)', value: 'America/Chicago' },
  { label: 'America/Denver (MST/MDT)', value: 'America/Denver' },
  { label: 'America/Los_Angeles (PST/PDT)', value: 'America/Los_Angeles' },
  { label: 'America/Sao_Paulo', value: 'America/Sao_Paulo' },
  { label: 'Europe/London (GMT/BST)', value: 'Europe/London' },
  { label: 'Europe/Paris (CET/CEST)', value: 'Europe/Paris' },
  { label: 'Europe/Berlin (CET/CEST)', value: 'Europe/Berlin' },
  { label: 'Europe/Moscow (MSK)', value: 'Europe/Moscow' },
  { label: 'Asia/Dubai (GST)', value: 'Asia/Dubai' },
  { label: 'Asia/Kolkata (IST)', value: 'Asia/Kolkata' },
  { label: 'Asia/Singapore (SGT)', value: 'Asia/Singapore' },
  { label: 'Asia/Tokyo (JST)', value: 'Asia/Tokyo' },
  { label: 'Asia/Shanghai (CST)', value: 'Asia/Shanghai' },
  { label: 'Asia/Seoul (KST)', value: 'Asia/Seoul' },
  { label: 'Australia/Sydney (AEST/AEDT)', value: 'Australia/Sydney' },
  { label: 'Pacific/Auckland (NZST/NZDT)', value: 'Pacific/Auckland' },
]

// Source options for select
const sourceOptions = computed(() => {
  if (!adminStore.sources || adminStore.sources.length === 0) {
    return []
  }
  return adminStore.sources.map(source => ({
    label: `${source.name} (${getTenantName(source.tenant_id)})`,
    value: source.id,
  }))
})

const createForm = ref({
  source_id: '',
  cron: '0 0 * * *',
  timezone: 'UTC',
  retention_mode: 'latest_n' as 'all' | 'latest_n' | 'within_duration',
  keep_last_n: 7,
  keep_within_duration: '30d',
})

const editForm = ref({
  cron: '',
  timezone: 'UTC',
  status: 'enabled' as 'enabled' | 'disabled',
  retention_mode: 'latest_n' as 'all' | 'latest_n' | 'within_duration',
  keep_last_n: 7,
  keep_within_duration: '30d',
})

// Computed
const filteredSchedules = computed(() => {
  // Make sure we have schedules loaded
  if (!adminStore.schedules || adminStore.schedules.length === 0) {
    return []
  }

  let result = [...adminStore.schedules] // Create a copy to avoid mutating original array

  // Apply search filter
  if (searchQuery.value && searchQuery.value.trim()) {
    const query = searchQuery.value.toLowerCase().trim()
    result = result.filter(schedule => {
      // Search in schedule ID
      const scheduleId = schedule.id?.toLowerCase() || ''
      
      // Search in source name
      const sourceName = getSourceName(schedule.source_id).toLowerCase()
      
      // Search in tenant name  
      const tenantName = getTenantName(schedule.tenant_id).toLowerCase()
      
      // Search in cron expression
      const cronExpression = (schedule.cron || '').toLowerCase()
      
      // Search in timezone
      const timezone = (schedule.timezone || '').toLowerCase()
      
      // Search in status
      const status = (schedule.status || '').toLowerCase()

      return scheduleId.includes(query) ||
             sourceName.includes(query) ||
             tenantName.includes(query) ||
             cronExpression.includes(query) ||
             timezone.includes(query) ||
             status.includes(query)
    })
  }

  // Apply status filter
  if (statusFilter.value && statusFilter.value.trim()) {
    result = result.filter(schedule => {
      return schedule.status === statusFilter.value
    })
  }

  return result
})

const scheduleStats = computed(() => {
  // Ensure we have schedules before computing stats
  const schedules = adminStore.schedules || []
  return {
    total: schedules.length,
    enabled: schedules.filter(s => s.status === 'enabled').length,
    disabled: schedules.filter(s => s.status === 'disabled').length,
  }
})

onMounted(async () => {
  // Start the clock
  timeInterval.value = setInterval(updateTime, 1000)
  updateTime() // Initial update
  
  try {
    isLoading.value = true
    // Fetch data in sequence to ensure proper loading
    await adminStore.fetchTenants()
    await adminStore.fetchSources()
    await adminStore.fetchSchedules()
    
    // Debug: log the loaded data
    console.log('Schedules loaded:', adminStore.schedules?.length || 0)
    console.log('Sources loaded:', adminStore.sources?.length || 0)
    console.log('Tenants loaded:', adminStore.tenants?.length || 0)
  } catch (error) {
    console.error('Failed to load data:', error)
  } finally {
    isLoading.value = false
  }
})

onUnmounted(() => {
  if (timeInterval.value) {
    clearInterval(timeInterval.value)
  }
})

function getTenantName(tenantId: string): string {
  if (!tenantId) return 'Unknown'
  const tenant = adminStore.getTenantById(tenantId)
  return tenant?.name || `${tenantId.slice(0, 8)}...`
}

function getSourceName(sourceId: string): string {
  if (!sourceId) return 'Unknown'
  const source = adminStore.sources?.find(s => s.id === sourceId)
  return source?.name || `${sourceId.slice(0, 8)}...`
}

function formatCron(schedule: Schedule): string {
  if (schedule.cron) {
    return schedule.cron
  }
  if (schedule.interval_minutes) {
    return `Every ${schedule.interval_minutes} min`
  }
  return 'N/A'
}

function formatRetention(schedule: Schedule): string {
  if (!schedule.retention_policy) return 'N/A'
  const rp = schedule.retention_policy
  switch (rp.mode) {
    case 'all':
      return 'Keep all'
    case 'latest_n':
      return `Keep last ${rp.keep_last_n}`
    case 'within_duration':
      return `Keep ${rp.keep_within_duration}`
    default:
      return 'N/A'
  }
}

function formatDateTime(date: string | null | undefined, timezone?: string): string {
  if (!date) return 'Never'
  try {
    const dateObj = new Date(date)
    if (timezone) {
      return dateObj.toLocaleString('en-US', {
        timeZone: timezone,
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      })
    }
    return dateObj.toLocaleString()
  } catch {
    // Fallback if timezone is invalid
    return new Date(date).toLocaleString()
  }
}

function getTimezoneAbbr(timezone: string): string {
  try {
    const now = new Date()
    const formatter = new Intl.DateTimeFormat('en-US', {
      timeZone: timezone,
      timeZoneName: 'short',
    })
    const parts = formatter.formatToParts(now)
    const tzPart = parts.find(p => p.type === 'timeZoneName')
    return tzPart?.value || timezone
  } catch {
    return timezone
  }
}

// Dialog handlers
function openCreateDialog() {
  createError.value = ''
  createForm.value = {
    source_id: sourceOptions.value[0]?.value || '',
    cron: '0 0 * * *',
    timezone: 'UTC',
    retention_mode: 'latest_n',
    keep_last_n: 7,
    keep_within_duration: '30d',
  }
  showCreateDialog.value = true
}

function openEditDialog(schedule: Schedule) {
  editError.value = ''
  editingSchedule.value = schedule
  const rp = schedule.retention_policy || { mode: 'latest_n', keep_last_n: 7 }
  editForm.value = {
    cron: schedule.cron || '',
    timezone: schedule.timezone || 'UTC',
    status: schedule.status,
    retention_mode: rp.mode,
    keep_last_n: rp.keep_last_n || 7,
    keep_within_duration: rp.keep_within_duration || '30d',
  }
  showEditDialog.value = true
}

function openDeleteDialog(schedule: Schedule) {
  deleteError.value = ''
  deletingSchedule.value = schedule
  showDeleteDialog.value = true
}

// Form handlers
async function handleCreate() {
  createError.value = ''

  if (!createForm.value.source_id) {
    createError.value = 'Please select a source'
    return
  }

  if (!createForm.value.cron) {
    createError.value = 'Cron expression is required'
    return
  }

  isCreating.value = true
  try {
    const retentionPolicy = {
      mode: createForm.value.retention_mode,
      keep_last_n: createForm.value.retention_mode === 'latest_n' ? createForm.value.keep_last_n : undefined,
      keep_within_duration: createForm.value.retention_mode === 'within_duration' ? createForm.value.keep_within_duration : undefined,
    }

    await adminStore.createSchedule({
      source_id: createForm.value.source_id,
      cron: createForm.value.cron,
      timezone: createForm.value.timezone,
      retention_policy: retentionPolicy,
    } as AdminCreateScheduleRequest)
    showCreateDialog.value = false
  } catch (error: unknown) {
    createError.value = error instanceof Error ? error.message : 'Failed to create schedule'
  } finally {
    isCreating.value = false
  }
}

async function handleUpdate() {
  if (!editingSchedule.value) return

  editError.value = ''
  isUpdating.value = true

  try {
    const retentionPolicy = {
      mode: editForm.value.retention_mode,
      keep_last_n: editForm.value.retention_mode === 'latest_n' ? editForm.value.keep_last_n : undefined,
      keep_within_duration: editForm.value.retention_mode === 'within_duration' ? editForm.value.keep_within_duration : undefined,
    }

    await adminStore.updateSchedule(editingSchedule.value.id, {
      cron: editForm.value.cron || undefined,
      timezone: editForm.value.timezone,
      status: editForm.value.status,
      retention_policy: retentionPolicy,
    } as AdminUpdateScheduleRequest)
    showEditDialog.value = false
    editingSchedule.value = null
  } catch (error: unknown) {
    editError.value = error instanceof Error ? error.message : 'Failed to update schedule'
  } finally {
    isUpdating.value = false
  }
}

async function handleDelete() {
  if (!deletingSchedule.value) return

  deleteError.value = ''
  isDeleting.value = true

  try {
    await adminStore.deleteSchedule(deletingSchedule.value.id)
    showDeleteDialog.value = false
    deletingSchedule.value = null
  } catch (error: unknown) {
    deleteError.value = error instanceof Error ? error.message : 'Failed to delete schedule'
  } finally {
    isDeleting.value = false
  }
}

async function toggleStatus(schedule: Schedule) {
  try {
    const newStatus = schedule.status === 'enabled' ? 'disabled' : 'enabled'
    await adminStore.updateSchedule(schedule.id, { status: newStatus })
  } catch (error) {
    console.error('Failed to toggle schedule status:', error)
  }
}

</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold">Schedules</h1>
        <p class="text-muted-foreground">Manage backup schedules across all tenants</p>
        
        <!-- Clocks -->
        <div class="flex gap-6 mt-4 text-sm">
          <div class="bg-muted/50 px-3 py-2 rounded-lg">
            <div class="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-1">UTC Time</div>
            <div class="font-mono">{{ formatTime(currentTime, 'UTC') }}</div>
          </div>
          <div class="bg-muted/50 px-3 py-2 rounded-lg">
            <div class="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-1">Local Time</div>
            <div class="font-mono">{{ formatTime(currentTime) }}</div>
          </div>
        </div>
      </div>
      <Button @click="openCreateDialog">
        <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
        </svg>
        Add Schedule
      </Button>
    </div>

    <!-- Stats cards -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold">{{ scheduleStats.total }}</div>
          <div class="text-sm text-muted-foreground">Total Schedules</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-green-600">{{ scheduleStats.enabled }}</div>
          <div class="text-sm text-muted-foreground">Enabled</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-gray-500">{{ scheduleStats.disabled }}</div>
          <div class="text-sm text-muted-foreground">Disabled</div>
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
              placeholder="Search schedules..."
            />
          </div>
          <select 
            v-model="statusFilter" 
            class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <option v-for="option in statusFilterOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </div>
      </CardContent>
    </Card>

    <!-- Schedules list -->
    <Card>
      <CardContent class="p-0">
        <div v-if="isLoading" class="p-8 text-center text-muted-foreground">
          Loading schedules...
        </div>
        <div v-else-if="filteredSchedules.length === 0 && !isLoading" class="p-8 text-center text-muted-foreground">
          <div v-if="searchQuery || statusFilter">
            No schedules match your current filters.
            <button 
              @click="searchQuery = ''; statusFilter = ''" 
              class="text-primary hover:underline ml-2"
            >
              Clear filters
            </button>
          </div>
          <div v-else>
            No schedules found. Create your first schedule to get started.
          </div>
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead class="border-b bg-muted/50">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Source</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Tenant</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Schedule</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Last Run</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Next Run</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Retention</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Status</th>
                <th class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr
                v-for="schedule in filteredSchedules"
                :key="schedule.id"
                class="hover:bg-muted/50 transition-colors"
              >
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="font-medium">{{ getSourceName(schedule.source_id) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ getTenantName(schedule.tenant_id) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm font-mono">{{ formatCron(schedule) }}</div>
                  <div class="text-xs text-muted-foreground">{{ schedule.timezone }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ formatDateTime(schedule.last_run_at, schedule.timezone) }}</div>
                  <div v-if="schedule.last_run_at" class="text-xs text-muted-foreground/70">{{ getTimezoneAbbr(schedule.timezone) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ formatDateTime(schedule.next_run_at, schedule.timezone) }}</div>
                  <div v-if="schedule.next_run_at" class="text-xs text-muted-foreground/70">{{ getTimezoneAbbr(schedule.timezone) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm">{{ formatRetention(schedule) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <button
                    :class="[
                      'px-2 py-1 text-xs rounded-full cursor-pointer transition-colors',
                      schedule.status === 'enabled'
                        ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 hover:bg-green-200 dark:hover:bg-green-800'
                        : 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700'
                    ]"
                    @click="toggleStatus(schedule)"
                  >
                    {{ schedule.status === 'enabled' ? 'Enabled' : 'Disabled' }}
                  </button>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <div class="flex justify-end gap-2">
                    <Button variant="ghost" size="sm" @click="openEditDialog(schedule)">
                      Edit
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      class="text-destructive hover:text-destructive"
                      @click="openDeleteDialog(schedule)"
                    >
                      Delete
                    </Button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>

    <!-- Create Schedule Dialog -->
    <Dialog v-model:open="showCreateDialog">
      <div class="p-6">
        <h2 class="text-lg font-semibold mb-4">Create New Schedule</h2>

        <div v-if="createError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ createError }}
        </div>

        <form @submit.prevent="handleCreate" class="space-y-4">
          <!-- Source Selection -->
          <div class="space-y-2">
            <Label>Source</Label>
            <select 
              v-model="createForm.source_id" 
              :disabled="isCreating"
              class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option value="" disabled>Select a source...</option>
              <option v-for="option in sourceOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          </div>

          <!-- Cron Expression -->
          <div class="space-y-2">
            <Label for="create-cron">Cron Expression</Label>
            <Input
              id="create-cron"
              v-model="createForm.cron"
              placeholder="0 0 * * *"
              :disabled="isCreating"
            />
            <p class="text-xs text-muted-foreground">
              Format: minute hour day month weekday (e.g., "0 0 * * *" for daily at midnight)
            </p>
          </div>

          <!-- Timezone -->
          <div class="space-y-2">
            <Label for="create-timezone">Timezone</Label>
            <select 
              id="create-timezone"
              v-model="createForm.timezone" 
              :disabled="isCreating"
              class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option v-for="option in timezoneOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
            <p class="text-xs text-muted-foreground">
              Backups will run at the scheduled time in this timezone
            </p>
          </div>

          <!-- Retention Policy -->
          <div class="border-t pt-4 mt-4">
            <h3 class="font-medium mb-3">Retention Policy</h3>

            <div class="space-y-3">
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  v-model="createForm.retention_mode"
                  value="all"
                  class="h-4 w-4"
                  :disabled="isCreating"
                />
                <span>Keep all snapshots</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  v-model="createForm.retention_mode"
                  value="latest_n"
                  class="h-4 w-4"
                  :disabled="isCreating"
                />
                <span>Keep last N snapshots</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  v-model="createForm.retention_mode"
                  value="within_duration"
                  class="h-4 w-4"
                  :disabled="isCreating"
                />
                <span>Keep snapshots within duration</span>
              </label>
            </div>

            <div v-if="createForm.retention_mode === 'latest_n'" class="mt-3">
              <Label for="create-keep-n">Number of snapshots to keep</Label>
              <Input
                id="create-keep-n"
                v-model.number="createForm.keep_last_n"
                type="number"
                min="1"
                :disabled="isCreating"
              />
            </div>

            <div v-if="createForm.retention_mode === 'within_duration'" class="mt-3">
              <Label for="create-duration">Duration (e.g., 30d, 7d, 24h)</Label>
              <Input
                id="create-duration"
                v-model="createForm.keep_within_duration"
                placeholder="30d"
                :disabled="isCreating"
              />
            </div>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" :disabled="isCreating" @click="showCreateDialog = false">
              Cancel
            </Button>
            <Button type="submit" :disabled="isCreating">
              {{ isCreating ? 'Creating...' : 'Create Schedule' }}
            </Button>
          </div>
        </form>
      </div>
    </Dialog>

    <!-- Edit Schedule Dialog -->
    <Dialog v-model:open="showEditDialog">
      <div class="p-6">
        <h2 class="text-lg font-semibold mb-4">Edit Schedule</h2>

        <div v-if="editError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ editError }}
        </div>

        <form v-if="editingSchedule" @submit.prevent="handleUpdate" class="space-y-4">
          <!-- Schedule Info (read-only) -->
          <div class="p-3 bg-muted/50 rounded-md text-sm">
            <div><strong>Source:</strong> {{ getSourceName(editingSchedule.source_id) }}</div>
            <div><strong>Tenant:</strong> {{ getTenantName(editingSchedule.tenant_id) }}</div>
          </div>

          <!-- Status -->
          <div class="space-y-2">
            <Label>Status</Label>
            <div class="flex gap-4">
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  v-model="editForm.status"
                  value="enabled"
                  class="h-4 w-4"
                  :disabled="isUpdating"
                />
                <span>Enabled</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  v-model="editForm.status"
                  value="disabled"
                  class="h-4 w-4"
                  :disabled="isUpdating"
                />
                <span>Disabled</span>
              </label>
            </div>
          </div>

          <!-- Cron Expression -->
          <div class="space-y-2">
            <Label for="edit-cron">Cron Expression</Label>
            <Input
              id="edit-cron"
              v-model="editForm.cron"
              placeholder="0 0 * * *"
              :disabled="isUpdating"
            />
          </div>

          <!-- Timezone -->
          <div class="space-y-2">
            <Label for="edit-timezone">Timezone</Label>
            <select 
              id="edit-timezone"
              v-model="editForm.timezone" 
              :disabled="isUpdating"
              class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option v-for="option in timezoneOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
            <p class="text-xs text-muted-foreground">
              Backups will run at the scheduled time in this timezone
            </p>
          </div>

          <!-- Retention Policy -->
          <div class="border-t pt-4 mt-4">
            <h3 class="font-medium mb-3">Retention Policy</h3>

            <div class="space-y-3">
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  v-model="editForm.retention_mode"
                  value="all"
                  class="h-4 w-4"
                  :disabled="isUpdating"
                />
                <span>Keep all snapshots</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  v-model="editForm.retention_mode"
                  value="latest_n"
                  class="h-4 w-4"
                  :disabled="isUpdating"
                />
                <span>Keep last N snapshots</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  v-model="editForm.retention_mode"
                  value="within_duration"
                  class="h-4 w-4"
                  :disabled="isUpdating"
                />
                <span>Keep snapshots within duration</span>
              </label>
            </div>

            <div v-if="editForm.retention_mode === 'latest_n'" class="mt-3">
              <Label for="edit-keep-n">Number of snapshots to keep</Label>
              <Input
                id="edit-keep-n"
                v-model.number="editForm.keep_last_n"
                type="number"
                min="1"
                :disabled="isUpdating"
              />
            </div>

            <div v-if="editForm.retention_mode === 'within_duration'" class="mt-3">
              <Label for="edit-duration">Duration (e.g., 30d, 7d, 24h)</Label>
              <Input
                id="edit-duration"
                v-model="editForm.keep_within_duration"
                placeholder="30d"
                :disabled="isUpdating"
              />
            </div>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" :disabled="isUpdating" @click="showEditDialog = false">
              Cancel
            </Button>
            <Button type="submit" :disabled="isUpdating">
              {{ isUpdating ? 'Updating...' : 'Update Schedule' }}
            </Button>
          </div>
        </form>
      </div>
    </Dialog>

    <!-- Delete Confirmation Dialog -->
    <Dialog v-model:open="showDeleteDialog">
      <div class="p-6">
        <h2 class="text-lg font-semibold mb-4">Delete Schedule</h2>

        <div v-if="deleteError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ deleteError }}
        </div>

        <p class="mb-4">
          Are you sure you want to delete the schedule for source
          <strong>{{ deletingSchedule ? getSourceName(deletingSchedule.source_id) : '' }}</strong>?
        </p>

        <p class="text-sm text-muted-foreground mb-4">
          This will stop future backups from running. Existing snapshots will not be affected.
        </p>

        <div class="flex justify-end gap-3">
          <Button variant="outline" :disabled="isDeleting" @click="showDeleteDialog = false">
            Cancel
          </Button>
          <Button variant="destructive" :disabled="isDeleting" @click="handleDelete">
            {{ isDeleting ? 'Deleting...' : 'Delete Schedule' }}
          </Button>
        </div>
      </div>
    </Dialog>
  </div>
</template>
