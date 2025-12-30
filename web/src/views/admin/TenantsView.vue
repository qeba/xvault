<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useAdminStore } from '@/stores/admin'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import Input from '@/components/ui/input/Input.vue'
import Button from '@/components/ui/button/Button.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { Tenant } from '@/types'

const adminStore = useAdminStore()

const searchQuery = ref('')
const isLoading = ref(true)
const deleteError = ref<string | null>(null)
const viewTenant = ref<Tenant | null>(null)
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

async function handleDelete(tenant: Tenant): Promise<void> {
  deleteError.value = null
  try {
    await adminStore.deleteTenant(tenant.id)
    tenantToDelete.value = null
  } catch (error) {
    deleteError.value = error instanceof Error ? error.message : 'Failed to delete tenant'
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
                  <Button
                    variant="ghost"
                    size="sm"
                    @click="viewTenant = tenant"
                  >
                    View
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    class="text-destructive hover:text-destructive"
                    @click="tenantToDelete = tenant"
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

    <!-- View Tenant Dialog -->
    <Dialog v-model:open="!!viewTenant">
      <DialogContent v-if="viewTenant">
        <DialogHeader>
          <DialogTitle>Tenant Information</DialogTitle>
          <DialogDescription>
            View detailed information about this tenant
          </DialogDescription>
        </DialogHeader>
        <div class="space-y-4 py-4">
          <div class="grid grid-cols-[120px_1fr] gap-4 items-center">
            <span class="text-sm font-medium text-muted-foreground">Name</span>
            <span class="text-sm">{{ viewTenant.name }}</span>
          </div>
          <div class="grid grid-cols-[120px_1fr] gap-4 items-center">
            <span class="text-sm font-medium text-muted-foreground">Tenant ID</span>
            <span class="text-sm font-mono text-xs">{{ viewTenant.id }}</span>
          </div>
          <div class="grid grid-cols-[120px_1fr] gap-4 items-center">
            <span class="text-sm font-medium text-muted-foreground">Created</span>
            <span class="text-sm">{{ formatDate(viewTenant.created_at) }}</span>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="viewTenant = null">Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Delete Confirmation Dialog -->
    <Dialog v-model:open="!!tenantToDelete">
      <DialogContent v-if="tenantToDelete">
        <DialogHeader>
          <DialogTitle>Delete Tenant</DialogTitle>
          <DialogDescription>
            Are you sure you want to delete <strong>{{ tenantToDelete.name }}</strong>?
            This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <div v-if="deleteError" class="rounded-md bg-destructive/15 p-3 text-sm text-destructive">
          {{ deleteError }}
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            @click="tenantToDelete = null; deleteError = null"
            :disabled="adminStore.isLoading"
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            @click="handleDelete(tenantToDelete)"
            :disabled="adminStore.isLoading"
          >
            {{ adminStore.isLoading ? 'Deleting...' : 'Delete Tenant' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
