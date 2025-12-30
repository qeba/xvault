<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useSnapshotsStore } from '@/stores/snapshots'
import { useSourcesStore } from '@/stores/sources'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import Input from '@/components/ui/input/Input.vue'

const snapshotsStore = useSnapshotsStore()
const sourcesStore = useSourcesStore()

const searchQuery = ref('')
const isLoading = ref(true)

const filteredSnapshots = computed(() => {
  if (!searchQuery.value) return snapshotsStore.snapshots
  return snapshotsStore.snapshots.filter(s =>
    s.id.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
    s.source_id.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

onMounted(async () => {
  try {
    await Promise.all([
      snapshotsStore.fetchSnapshots(),
      sourcesStore.fetchSources(),
    ])
  } catch (error) {
    console.error('Failed to load data:', error)
  } finally {
    isLoading.value = false
  }
})

async function handleGenerateDownloadLink(id: string) {
  try {
    const result = await snapshotsStore.generateDownloadLink(id)
    alert(`Download URL: ${result.download_url}\nExpires at: ${result.expires_at}`)
  } catch (error) {
    console.error('Failed to generate download link:', error)
  }
}

function getSourceName(sourceId: string): string {
  const source = sourcesStore.sources.find(s => s.id === sourceId)
  return source?.name || 'Unknown'
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

function formatDate(date: string): string {
  return new Date(date).toLocaleString()
}

function getStatusBadgeClass(status: string): string {
  switch (status) {
    case 'completed':
      return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
    case 'failed':
      return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
    case 'pending':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold">Snapshots</h1>
        <p class="text-muted-foreground">View and manage backup snapshots</p>
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
              placeholder="Search snapshots by ID or source..."
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
          No snapshots found.
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead class="border-b bg-muted/50">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Snapshot ID</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Source</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Status</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Size</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Worker</th>
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
                  <div class="text-sm">{{ getSourceName(snapshot.source_id) }}</div>
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
                  <div class="text-sm text-muted-foreground font-mono text-xs">{{ snapshot.worker_id || 'N/A' }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ formatDate(snapshot.created_at) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <Button
                    variant="ghost"
                    size="sm"
                    @click="handleGenerateDownloadLink(snapshot.id)"
                  >
                    Download
                  </Button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
