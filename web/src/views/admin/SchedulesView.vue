<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSchedulesStore } from '@/stores/schedules'
import { useSourcesStore } from '@/stores/sources'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import Input from '@/components/ui/input/Input.vue'
import Label from '@/components/ui/label/Label.vue'
import Dialog from '@/components/ui/dialog/Dialog.vue'

const schedulesStore = useSchedulesStore()
const sourcesStore = useSourcesStore()

const searchQuery = ref('')
const isLoading = ref(true)
const showCreateDialog = ref(false)
const isCreating = ref(false)
const createError = ref('')

const createScheduleForm = ref({
  source_id: '',
  schedule: '0 0 * * *', // cron format: daily at midnight
})

onMounted(async () => {
  try {
    await Promise.all([
      schedulesStore.fetchSchedules(),
      sourcesStore.fetchSources(),
    ])
  } catch (error) {
    console.error('Failed to load data:', error)
  } finally {
    isLoading.value = false
  }
})

async function openCreateDialog() {
  createError.value = ''
  createScheduleForm.value = { source_id: '', schedule: '0 0 * * *' }
  showCreateDialog.value = true
}

async function handleCreateSchedule() {
  createError.value = ''

  if (!createScheduleForm.value.source_id) {
    createError.value = 'Source is required'
    return
  }

  isCreating.value = true
  try {
    await schedulesStore.createSchedule({
      source_id: createScheduleForm.value.source_id,
      schedule: createScheduleForm.value.schedule,
      enabled: true,
    })
    showCreateDialog.value = false
  } catch (error: unknown) {
    createError.value = error instanceof Error ? error.message : 'Failed to create schedule'
  } finally {
    isCreating.value = false
  }
}

async function handleDeleteSchedule(id: string) {
  if (!confirm('Are you sure you want to delete this schedule?')) return

  try {
    await schedulesStore.deleteSchedule(id)
  } catch (error) {
    console.error('Failed to delete schedule:', error)
  }
}

function getSourceName(sourceId: string): string {
  const source = sourcesStore.sources.find(s => s.id === sourceId)
  return source?.name || 'Unknown'
}

function formatDate(date: string): string {
  return new Date(date).toLocaleString()
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold">Schedules</h1>
        <p class="text-muted-foreground">Manage backup schedules</p>
      </div>
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
          <Button @click="openCreateDialog">
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
            Add Schedule
          </Button>
        </div>
      </CardContent>
    </Card>

    <!-- Schedules list -->
    <Card>
      <CardContent class="p-0">
        <div v-if="isLoading" class="p-8 text-center text-muted-foreground">
          Loading schedules...
        </div>
        <div v-else-if="schedulesStore.schedules.length === 0" class="p-8 text-center text-muted-foreground">
          No schedules found. Create your first schedule to get started.
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead class="border-b bg-muted/50">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">ID</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Source</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Schedule (Cron)</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Status</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Created</th>
                <th class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr
                v-for="schedule in schedulesStore.schedules.filter(s =>
                  !searchQuery || s.id.includes(searchQuery) || s.source_id.includes(searchQuery)
                )"
                :key="schedule.id"
                class="hover:bg-muted/50 transition-colors"
              >
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm font-mono text-xs">{{ schedule.id.slice(0, 8) }}...</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm">{{ getSourceName(schedule.source_id) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm font-mono">{{ schedule.schedule }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <span
                    :class="[
                      'px-2 py-1 text-xs rounded-full',
                      schedule.enabled
                        ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                        : 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
                    ]"
                  >
                    {{ schedule.enabled ? 'Enabled' : 'Disabled' }}
                  </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ formatDate(schedule.created_at) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <Button
                    variant="ghost"
                    size="sm"
                    class="text-destructive"
                    @click="handleDeleteSchedule(schedule.id)"
                  >
                    Delete
                  </Button>
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

        <form @submit.prevent="handleCreateSchedule" class="space-y-4">
          <div class="space-y-2">
            <Label for="create-source">Source</Label>
            <select
              id="create-source"
              v-model="createScheduleForm.source_id"
              class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              required
              :disabled="isCreating"
            >
              <option value="">Select a source</option>
              <option v-for="source in sourcesStore.sources" :key="source.id" :value="source.id">
                {{ source.name }}
              </option>
            </select>
          </div>

          <div class="space-y-2">
            <Label for="create-schedule">Schedule (Cron Format)</Label>
            <Input
              id="create-schedule"
              v-model="createScheduleForm.schedule"
              type="text"
              placeholder="0 0 * * *"
              required
              :disabled="isCreating"
            />
            <p class="text-xs text-muted-foreground">
              Cron format: minute hour day month weekday (e.g., "0 0 * * *" for daily at midnight)
            </p>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <Button
              type="button"
              variant="outline"
              :disabled="isCreating"
              @click="showCreateDialog = false"
            >
              Cancel
            </Button>
            <Button type="submit" :disabled="isCreating">
              {{ isCreating ? 'Creating...' : 'Create Schedule' }}
            </Button>
          </div>
        </form>
      </div>
    </Dialog>
  </div>
</template>
