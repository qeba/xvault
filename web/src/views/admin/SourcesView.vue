<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useSourcesStore } from '@/stores/sources'
import type { SourceType } from '@/types'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import Input from '@/components/ui/input/Input.vue'
import Label from '@/components/ui/label/Label.vue'
import Dialog from '@/components/ui/dialog/Dialog.vue'

const sourcesStore = useSourcesStore()

const searchQuery = ref('')
const isLoading = ref(true)
const showCreateDialog = ref(false)
const isCreating = ref(false)
const createError = ref('')

const createSourceForm = ref({
  name: '',
  type: 'sftp' as SourceType,
  config: {} as Record<string, string>,
})

const filteredSources = computed(() => {
  if (!searchQuery.value) return sourcesStore.sources
  return sourcesStore.sources.filter(s =>
    s.name.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

onMounted(async () => {
  try {
    await sourcesStore.fetchSources()
  } catch (error) {
    console.error('Failed to load sources:', error)
  } finally {
    isLoading.value = false
  }
})

async function openCreateDialog() {
  createError.value = ''
  createSourceForm.value = { name: '', type: 'sftp', config: {} }
  showCreateDialog.value = true
}

async function handleCreateSource() {
  createError.value = ''

  if (!createSourceForm.value.name) {
    createError.value = 'Name is required'
    return
  }

  // Build config from form fields
  const config: Record<string, string> = {}
  const type = createSourceForm.value.type

  if (type === 'sftp') {
    config.host = createSourceForm.value.config.host || ''
    config.port = createSourceForm.value.config.port || '22'
    config.username = createSourceForm.value.config.username || ''
    config.path = createSourceForm.value.config.path || ''
  }

  isCreating.value = true
  try {
    await sourcesStore.createSource({
      name: createSourceForm.value.name,
      type: createSourceForm.value.type,
      config,
    })
    showCreateDialog.value = false
  } catch (error: unknown) {
    createError.value = error instanceof Error ? error.message : 'Failed to create source'
  } finally {
    isCreating.value = false
  }
}

async function handleDeleteSource(id: string) {
  if (!confirm('Are you sure you want to delete this source?')) return

  try {
    await sourcesStore.deleteSource(id)
  } catch (error) {
    console.error('Failed to delete source:', error)
  }
}

function getTypeBadgeClass(type: string): string {
  switch (type) {
    case 'sftp':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
    case 'ssh':
      return 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200'
    case 'ftp':
      return 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200'
    case 'mysql':
      return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
    case 'postgresql':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
    case 'mongodb':
      return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
  }
}

function formatDate(date: string): string {
  return new Date(date).toLocaleDateString()
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold">Sources</h1>
        <p class="text-muted-foreground">Manage backup sources</p>
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
              placeholder="Search sources by name..."
            />
          </div>
          <Button @click="openCreateDialog">
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
            Add Source
          </Button>
        </div>
      </CardContent>
    </Card>

    <!-- Sources list -->
    <Card>
      <CardContent class="p-0">
        <div v-if="isLoading" class="p-8 text-center text-muted-foreground">
          Loading sources...
        </div>
        <div v-else-if="filteredSources.length === 0" class="p-8 text-center text-muted-foreground">
          No sources found. Create your first source to get started.
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead class="border-b bg-muted/50">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Name</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Type</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Config</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Created</th>
                <th class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr
                v-for="source in filteredSources"
                :key="source.id"
                class="hover:bg-muted/50 transition-colors"
              >
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm font-medium">{{ source.name }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <span :class="['px-2 py-1 text-xs rounded-full', getTypeBadgeClass(source.type)]">
                    {{ source.type.toUpperCase() }}
                  </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">
                    {{ source.config.host || source.config.bucket || 'N/A' }}
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ formatDate(source.created_at) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <Button
                    variant="ghost"
                    size="sm"
                    class="text-destructive"
                    @click="handleDeleteSource(source.id)"
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

    <!-- Create Source Dialog -->
    <Dialog v-model:open="showCreateDialog">
      <div class="p-6">
        <h2 class="text-lg font-semibold mb-4">Create New Source</h2>

        <div v-if="createError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ createError }}
        </div>

        <form @submit.prevent="handleCreateSource" class="space-y-4">
          <div class="space-y-2">
            <Label for="create-name">Name</Label>
            <Input
              id="create-name"
              v-model="createSourceForm.name"
              type="text"
              placeholder="Production Database"
              required
              :disabled="isCreating"
            />
          </div>

          <div class="space-y-2">
            <Label for="create-type">Type</Label>
            <select
              id="create-type"
              v-model="createSourceForm.type"
              class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              :disabled="isCreating"
            >
              <option value="sftp">SFTP</option>
              <option value="s3">S3</option>
              <option value="local">Local</option>
            </select>
          </div>

          <template v-if="createSourceForm.type === 'sftp'">
            <div class="space-y-2">
              <Label for="sftp-host">Host</Label>
              <Input
                id="sftp-host"
                v-model="createSourceForm.config.host"
                type="text"
                placeholder="example.com"
                :disabled="isCreating"
              />
            </div>

            <div class="space-y-2">
              <Label for="sftp-port">Port</Label>
              <Input
                id="sftp-port"
                v-model="createSourceForm.config.port"
                type="number"
                placeholder="22"
                :disabled="isCreating"
              />
            </div>

            <div class="space-y-2">
              <Label for="sftp-username">Username</Label>
              <Input
                id="sftp-username"
                v-model="createSourceForm.config.username"
                type="text"
                placeholder="user"
                :disabled="isCreating"
              />
            </div>

            <div class="space-y-2">
              <Label for="sftp-path">Path</Label>
              <Input
                id="sftp-path"
                v-model="createSourceForm.config.path"
                type="text"
                placeholder="/var/backups"
                :disabled="isCreating"
              />
            </div>
          </template>

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
              {{ isCreating ? 'Creating...' : 'Create Source' }}
            </Button>
          </div>
        </form>
      </div>
    </Dialog>
  </div>
</template>
