<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { useAuthStore } from '@/stores/auth'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import CardHeader from '@/components/ui/card/CardHeader.vue'
import CardTitle from '@/components/ui/card/CardTitle.vue'
import Input from '@/components/ui/input/Input.vue'
import Label from '@/components/ui/label/Label.vue'
import Dialog from '@/components/ui/dialog/Dialog.vue'

const route = useRoute()
const router = useRouter()
const adminStore = useAdminStore()
const authStore = useAuthStore()

const userId = computed(() => route.params.id as string)
const user = computed(() => adminStore.users.find(u => u.id === userId.value))

const isLoading = ref(true)
const showEditDialog = ref(false)
const showDeleteDialog = ref(false)
const isSaving = ref(false)
const error = ref('')

const editForm = ref({
  email: '',
  role: 'member' as 'owner' | 'admin' | 'member',
})

onMounted(async () => {
  try {
    await adminStore.fetchUsers()
    if (user.value) {
      editForm.value = {
        email: user.value.email,
        role: user.value.role,
      }
    }
  } catch (err) {
    console.error('Failed to load user:', err)
  } finally {
    isLoading.value = false
  }
})

async function openEditDialog() {
  if (user.value) {
    editForm.value = {
      email: user.value.email,
      role: user.value.role,
    }
    error.value = ''
    showEditDialog.value = true
  }
}

async function handleUpdateUser() {
  if (!user.value) return

  isSaving.value = true
  error.value = ''

  try {
    await adminStore.updateUser(user.value.id, {
      email: editForm.value.email,
      role: editForm.value.role,
    })
    showEditDialog.value = false
    await adminStore.fetchUsers()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to update user'
  } finally {
    isSaving.value = false
  }
}

async function handleDeleteUser() {
  if (!user.value) return

  isSaving.value = true
  try {
    await adminStore.deleteUser(user.value.id)
    showDeleteDialog.value = false
    router.push('/admin/users')
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to delete user'
    isSaving.value = false
  }
}

function getRoleBadgeClass(role: string): string {
  switch (role) {
    case 'admin':
      return 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200'
    case 'owner':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
  }
}

function formatDate(date: string): string {
  return new Date(date).toLocaleString()
}

function goBack() {
  router.push('/admin/users')
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center gap-4">
      <Button variant="ghost" size="icon" @click="goBack">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </Button>
      <div>
        <h1 class="text-3xl font-bold">User Details</h1>
        <p class="text-muted-foreground">View and manage user account</p>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="p-8 text-center text-muted-foreground">
      Loading user details...
    </div>

    <!-- User not found -->
    <Card v-else-if="!user">
      <CardContent class="p-8 text-center text-muted-foreground">
        User not found
      </CardContent>
    </Card>

    <!-- User details -->
    <template v-else>
      <!-- Info card -->
      <Card>
        <CardHeader>
          <div class="flex items-center justify-between">
            <CardTitle>User Information</CardTitle>
            <div class="flex gap-2">
              <Button variant="outline" @click="openEditDialog">
                <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
                Edit
              </Button>
              <Button
                variant="destructive"
                @click="showDeleteDialog = true"
                :disabled="user.id === authStore.user?.id"
              >
                <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
                Delete
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <dl class="grid gap-4 sm:grid-cols-2">
            <div>
              <dt class="text-sm font-medium text-muted-foreground">Email</dt>
              <dd class="text-sm font-mono mt-1">{{ user.email }}</dd>
            </div>
            <div>
              <dt class="text-sm font-medium text-muted-foreground">Role</dt>
              <dd class="mt-1">
                <span :class="['px-2 py-1 text-xs rounded-full', getRoleBadgeClass(user.role)]">
                  {{ user.role }}
                </span>
              </dd>
            </div>
            <div>
              <dt class="text-sm font-medium text-muted-foreground">Tenant ID</dt>
              <dd class="text-sm font-mono text-xs mt-1">{{ user.tenant_id }}</dd>
            </div>
            <div>
              <dt class="text-sm font-medium text-muted-foreground">User ID</dt>
              <dd class="text-sm font-mono text-xs mt-1">{{ user.id }}</dd>
            </div>
            <div>
              <dt class="text-sm font-medium text-muted-foreground">Created</dt>
              <dd class="text-sm mt-1">{{ formatDate(user.created_at) }}</dd>
            </div>
            <div>
              <dt class="text-sm font-medium text-muted-foreground">Last Updated</dt>
              <dd class="text-sm mt-1">{{ formatDate(user.updated_at) }}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>
    </template>

    <!-- Edit User Dialog -->
    <Dialog v-model:open="showEditDialog">
      <div class="p-6">
        <h2 class="text-lg font-semibold mb-4">Edit User</h2>

        <div v-if="error" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ error }}
        </div>

        <form @submit.prevent="handleUpdateUser" class="space-y-4">
          <div class="space-y-2">
            <Label for="edit-email">Email</Label>
            <Input
              id="edit-email"
              v-model="editForm.email"
              type="email"
              required
              :disabled="isSaving"
            />
          </div>

          <div class="space-y-2">
            <Label for="edit-role">Role</Label>
            <select
              id="edit-role"
              v-model="editForm.role"
              class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              :disabled="isSaving"
            >
              <option value="member">Member</option>
              <option value="admin">Admin</option>
              <option value="owner">Owner</option>
            </select>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <Button
              type="button"
              variant="outline"
              :disabled="isSaving"
              @click="showEditDialog = false"
            >
              Cancel
            </Button>
            <Button type="submit" :disabled="isSaving">
              {{ isSaving ? 'Saving...' : 'Save Changes' }}
            </Button>
          </div>
        </form>
      </div>
    </Dialog>

    <!-- Delete Confirmation Dialog -->
    <Dialog v-model:open="showDeleteDialog">
      <div class="p-6">
        <h2 class="text-lg font-semibold mb-4 text-destructive">Delete User</h2>

        <div v-if="error" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ error }}
        </div>

        <p class="text-sm text-muted-foreground mb-6">
          Are you sure you want to delete this user? This action cannot be undone.
        </p>

        <div class="flex justify-end gap-3">
          <Button
            variant="outline"
            :disabled="isSaving"
            @click="showDeleteDialog = false"
          >
            Cancel
          </Button>
          <Button variant="destructive" :disabled="isSaving" @click="handleDeleteUser">
            {{ isSaving ? 'Deleting...' : 'Delete User' }}
          </Button>
        </div>
      </div>
    </Dialog>
  </div>
</template>
