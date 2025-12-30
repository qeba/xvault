<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useAdminStore } from '@/stores/admin'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import Input from '@/components/ui/input/Input.vue'
import Button from '@/components/ui/button/Button.vue'
import Dialog from '@/components/ui/dialog/Dialog.vue'
import type { Tenant } from '@/types'

const adminStore = useAdminStore()

const searchQuery = ref('')
const isLoading = ref(true)
const deleteError = ref<string | null>(null)
const showViewDialog = ref(false)
const showDeleteDialog = ref(false)
const isDeleting = ref(false)
const viewingTenant = ref<Tenant | null>(null)
const tenantToDelete = ref<Tenant | null>(null)

const filteredTenants = computed(() => {
  if (!searchQuery.value) return adminStore.tenants
  return adminStore.tenants.filter(t =>
    t.name.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

onMounted(async () => {
  try {
    await adminStore.fetchTenants()
  } catch (error) {
    console.error('Failed to load tenants:', error)
  } finally {
    isLoading.value = false
  }
})

function formatDate(date: string): string {
  return new Date(date).toLocaleDateString()
}

function openViewDialog(tenant: Tenant) {
  viewingTenant.value = tenant
  showViewDialog.value = true
}

function openDeleteDialog(tenant: Tenant) {
  deleteError.value = null
  tenantToDelete.value = tenant
  showDeleteDialog.value = true
}

async function handleDelete(): Promise<void> {
  if (!tenantToDelete.value) return

  deleteError.value = null
  isDeleting.value = true

  try {
    await adminStore.deleteTenant(tenantToDelete.value.id)
    showDeleteDialog.value = false
    tenantToDelete.value = null
  } catch (error) {
    deleteError.value = error instanceof Error ? error.message : 'Failed to delete tenant'
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold">Tenants</h1>
        <p class="text-muted-foreground">Manage platform tenants</p>
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
              placeholder="Search tenants by name..."
            />
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Tenants list -->
    <Card>
      <CardContent class="p-0">
        <div v-if="isLoading" class="p-8 text-center text-muted-foreground">
          Loading tenants...
        </div>
        <div v-else-if="filteredTenants.length === 0" class="p-8 text-center text-muted-foreground">
          No tenants found.
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead class="border-b bg-muted/50">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Name</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Tenant ID</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Created</th>
                <th class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr
                v-for="tenant in filteredTenants"
                :key="tenant.id"
                class="hover:bg-muted/50 transition-colors"
              >
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm font-medium">{{ tenant.name }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground font-mono text-xs">
                    {{ tenant.id.slice(0, 8) }}...
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ formatDate(tenant.created_at) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <div class="flex justify-end gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      @click="openViewDialog(tenant)"
                    >
                      View
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      class="text-destructive hover:text-destructive"
                      @click="openDeleteDialog(tenant)"
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

    <!-- View Tenant Dialog -->
    <Dialog v-model:open="showViewDialog">
      <div v-if="viewingTenant" class="p-6">
        <h2 class="text-lg font-semibold mb-4">Tenant Information</h2>
        <div class="space-y-4">
          <div class="grid grid-cols-[120px_1fr] gap-4 items-center">
            <span class="text-sm font-medium text-muted-foreground">Name</span>
            <span class="text-sm">{{ viewingTenant.name }}</span>
          </div>
          <div class="grid grid-cols-[120px_1fr] gap-4 items-center">
            <span class="text-sm font-medium text-muted-foreground">Tenant ID</span>
            <span class="text-sm font-mono text-xs break-all">{{ viewingTenant.id }}</span>
          </div>
          <div class="grid grid-cols-[120px_1fr] gap-4 items-center">
            <span class="text-sm font-medium text-muted-foreground">Created</span>
            <span class="text-sm">{{ formatDate(viewingTenant.created_at) }}</span>
          </div>
        </div>
        <div class="flex justify-end pt-4">
          <Button
            type="button"
            variant="outline"
            @click="showViewDialog = false"
          >
            Close
          </Button>
        </div>
      </div>
    </Dialog>

    <!-- Delete Confirmation Dialog -->
    <Dialog v-model:open="showDeleteDialog">
      <div v-if="tenantToDelete" class="p-6">
        <h2 class="text-lg font-semibold mb-4">Delete Tenant</h2>
        <p class="text-sm text-muted-foreground mb-4">
          Are you sure you want to delete <strong>{{ tenantToDelete.name }}</strong>?
          This action cannot be undone.
        </p>
        <div v-if="deleteError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ deleteError }}
        </div>
        <div class="flex justify-end gap-3">
          <Button
            type="button"
            variant="outline"
            :disabled="isDeleting"
            @click="showDeleteDialog = false; deleteError = null"
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            :disabled="isDeleting"
            @click="handleDelete"
          >
            {{ isDeleting ? 'Deleting...' : 'Delete Tenant' }}
          </Button>
        </div>
      </div>
    </Dialog>
  </div>
</template>
