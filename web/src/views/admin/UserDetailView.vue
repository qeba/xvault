<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { useAuthStore } from '@/stores/auth'
import type { User, Tenant } from '@/types'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import CardHeader from '@/components/ui/card/CardHeader.vue'
import CardTitle from '@/components/ui/card/CardTitle.vue'
import Input from '@/components/ui/input/Input.vue'
import Label from '@/components/ui/label/Label.vue'
import Dialog from '@/components/ui/dialog/Dialog.vue'
import { Select, type SelectOption } from '@/components/ui/select'

const route = useRoute()
const router = useRouter()
const adminStore = useAdminStore()
const authStore = useAuthStore()

const userId = computed(() => route.params.id as string)
const user = ref<User | null>(null)
const tenant = ref<Tenant | null>(null)

const isLoading = ref(true)
const showEditDialog = ref(false)
const showDeleteDialog = ref(false)
const isSaving = ref(false)
const isDeleting = ref(false)
const error = ref('')
const editError = ref('')
const deleteError = ref('')

const roleOptions: SelectOption[] = [
  { label: 'Member', value: 'member' },
  { label: 'Admin', value: 'admin' },
  { label: 'Owner', value: 'owner' },
]

const editForm = ref({
  email: '',
  role: 'member' as 'owner' | 'admin' | 'member',
})

onMounted(async () => {
  await loadUserData()
})

async function loadUserData() {
  isLoading.value = true
  error.value = ''

  try {
    // Fetch user details
    const fetchedUser = await adminStore.fetchUser(userId.value)
    user.value = fetchedUser

    // Fetch tenant details
    if (fetchedUser?.tenant_id) {
      try {
        const fetchedTenant = await adminStore.fetchTenant(fetchedUser.tenant_id)
        tenant.value = fetchedTenant
      } catch {
        // Tenant fetch may fail if not found, that's ok
        console.warn('Could not fetch tenant details')
      }
    }

    // Initialize edit form
    if (fetchedUser) {
      editForm.value = {
        email: fetchedUser.email,
        role: fetchedUser.role,
      }
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load user'
    console.error('Failed to load user:', err)
  } finally {
    isLoading.value = false
  }
}

function openEditDialog() {
  if (user.value) {
    editForm.value = {
      email: user.value.email,
      role: user.value.role,
    }
    editError.value = ''
    showEditDialog.value = true
  }
}

async function handleUpdateUser() {
  if (!user.value) return

  isSaving.value = true
  editError.value = ''

  try {
    const updatedUser = await adminStore.updateUser(user.value.id, {
      email: editForm.value.email,
      role: editForm.value.role,
    })
    user.value = updatedUser
    showEditDialog.value = false
  } catch (err) {
    editError.value = err instanceof Error ? err.message : 'Failed to update user'
  } finally {
    isSaving.value = false
  }
}

async function handleDeleteUser() {
  if (!user.value) return

  isDeleting.value = true
  deleteError.value = ''

  try {
    await adminStore.deleteUser(user.value.id)
    showDeleteDialog.value = false
    router.push('/admin/users')
  } catch (err) {
    deleteError.value = err instanceof Error ? err.message : 'Failed to delete user'
  } finally {
    isDeleting.value = false
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
  return new Date(date).toLocaleString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatShortDate(date: string): string {
  return new Date(date).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function goBack() {
  router.push('/admin/users')
}

function isCurrentUser(): boolean {
  return authStore.user?.id === user.value?.id
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text)
}

// Reset dialogs when closed
watch(showEditDialog, (open) => {
  if (!open) editError.value = ''
})
watch(showDeleteDialog, (open) => {
  if (!open) deleteError.value = ''
})
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
      <div class="flex-1">
        <h1 class="text-3xl font-bold">User Details</h1>
        <p class="text-muted-foreground">View and manage user account</p>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="p-8 text-center text-muted-foreground">
      <svg class="animate-spin h-8 w-8 mx-auto mb-4 text-primary" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
      Loading user details...
    </div>

    <!-- Error state -->
    <Card v-else-if="error">
      <CardContent class="p-8 text-center">
        <svg class="w-12 h-12 mx-auto mb-4 text-destructive opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <p class="text-destructive mb-4">{{ error }}</p>
        <Button @click="loadUserData">Try Again</Button>
      </CardContent>
    </Card>

    <!-- User not found -->
    <Card v-else-if="!user">
      <CardContent class="p-8 text-center text-muted-foreground">
        <svg class="w-12 h-12 mx-auto mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
        <p class="mb-4">User not found</p>
        <Button variant="outline" @click="goBack">Go Back</Button>
      </CardContent>
    </Card>

    <!-- User details -->
    <template v-else>
      <!-- User header card -->
      <Card>
        <CardContent class="pt-6">
          <div class="flex flex-col sm:flex-row sm:items-center gap-4">
            <div class="h-16 w-16 rounded-full bg-primary/10 flex items-center justify-center">
              <span class="text-2xl font-bold text-primary">
                {{ user.email.charAt(0).toUpperCase() }}
              </span>
            </div>
            <div class="flex-1">
              <div class="flex items-center gap-3 mb-1">
                <h2 class="text-xl font-semibold">{{ user.email }}</h2>
                <span :class="['px-2 py-1 text-xs rounded-full font-medium', getRoleBadgeClass(user.role)]">
                  {{ user.role }}
                </span>
                <span v-if="isCurrentUser()" class="px-2 py-1 text-xs rounded-full bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                  You
                </span>
              </div>
              <p class="text-sm text-muted-foreground">
                Member since {{ formatShortDate(user.created_at) }}
              </p>
            </div>
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
                :disabled="isCurrentUser()"
                :title="isCurrentUser() ? 'Cannot delete your own account' : 'Delete user'"
              >
                <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
                Delete
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Info cards -->
      <div class="grid gap-6 md:grid-cols-2">
        <!-- Account Information -->
        <Card>
          <CardHeader>
            <CardTitle class="flex items-center gap-2">
              <svg class="w-5 h-5 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
              </svg>
              Account Information
            </CardTitle>
          </CardHeader>
          <CardContent>
            <dl class="space-y-4">
              <div>
                <dt class="text-sm font-medium text-muted-foreground">User ID</dt>
                <dd class="mt-1 flex items-center gap-2">
                  <code class="text-xs font-mono bg-muted px-2 py-1 rounded">{{ user.id }}</code>
                  <button
                    @click="copyToClipboard(user.id)"
                    class="text-muted-foreground hover:text-foreground"
                    title="Copy to clipboard"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                  </button>
                </dd>
              </div>
              <div>
                <dt class="text-sm font-medium text-muted-foreground">Email</dt>
                <dd class="mt-1 text-sm">{{ user.email }}</dd>
              </div>
              <div>
                <dt class="text-sm font-medium text-muted-foreground">Role</dt>
                <dd class="mt-1">
                  <span :class="['px-2 py-1 text-xs rounded-full font-medium', getRoleBadgeClass(user.role)]">
                    {{ user.role }}
                  </span>
                </dd>
              </div>
            </dl>
          </CardContent>
        </Card>

        <!-- Tenant Information -->
        <Card>
          <CardHeader>
            <CardTitle class="flex items-center gap-2">
              <svg class="w-5 h-5 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
              </svg>
              Tenant Information
            </CardTitle>
          </CardHeader>
          <CardContent>
            <dl class="space-y-4">
              <div>
                <dt class="text-sm font-medium text-muted-foreground">Tenant ID</dt>
                <dd class="mt-1 flex items-center gap-2">
                  <code class="text-xs font-mono bg-muted px-2 py-1 rounded">{{ user.tenant_id }}</code>
                  <button
                    @click="copyToClipboard(user.tenant_id)"
                    class="text-muted-foreground hover:text-foreground"
                    title="Copy to clipboard"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                  </button>
                </dd>
              </div>
              <div>
                <dt class="text-sm font-medium text-muted-foreground">Tenant Name</dt>
                <dd class="mt-1 text-sm">
                  <template v-if="tenant">{{ tenant.name || 'Unnamed Tenant' }}</template>
                  <template v-else>Loading...</template>
                </dd>
              </div>
              <div v-if="tenant">
                <dt class="text-sm font-medium text-muted-foreground">Tenant Created</dt>
                <dd class="mt-1 text-sm">{{ formatShortDate(tenant.created_at) }}</dd>
              </div>
            </dl>
          </CardContent>
        </Card>
      </div>

      <!-- Timestamps card -->
      <Card>
        <CardHeader>
          <CardTitle class="flex items-center gap-2">
            <svg class="w-5 h-5 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            Timestamps
          </CardTitle>
        </CardHeader>
        <CardContent>
          <dl class="grid gap-4 sm:grid-cols-2">
            <div>
              <dt class="text-sm font-medium text-muted-foreground">Created At</dt>
              <dd class="mt-1 text-sm">{{ formatDate(user.created_at) }}</dd>
            </div>
            <div>
              <dt class="text-sm font-medium text-muted-foreground">Last Updated</dt>
              <dd class="mt-1 text-sm">{{ formatDate(user.updated_at) }}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>
    </template>

    <!-- Edit User Dialog -->
    <Dialog v-model:open="showEditDialog">
      <div class="p-6">
        <h2 class="text-lg font-semibold mb-4">Edit User</h2>
        <p v-if="user" class="text-sm text-muted-foreground mb-6">
          Update user information for {{ user.email }}
        </p>

        <div v-if="editError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ editError }}
        </div>

        <form @submit.prevent="handleUpdateUser" class="space-y-4">
          <div class="space-y-2">
            <Label for="edit-email">Email <span class="text-destructive">*</span></Label>
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
            <Select
              id="edit-role"
              v-model="editForm.role"
              :options="roleOptions"
              :disabled="isSaving"
            />
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
              <svg v-if="isSaving" class="animate-spin -ml-1 mr-2 h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              {{ isSaving ? 'Saving...' : 'Save Changes' }}
            </Button>
          </div>
        </form>
      </div>
    </Dialog>

    <!-- Delete Confirmation Dialog -->
    <Dialog v-model:open="showDeleteDialog">
      <div class="p-6">
        <div class="flex items-center gap-4 mb-4">
          <div class="h-10 w-10 rounded-full bg-destructive/10 flex items-center justify-center">
            <svg class="w-5 h-5 text-destructive" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h2 class="text-lg font-semibold text-destructive">Delete User</h2>
            <p class="text-sm text-muted-foreground">This action cannot be undone</p>
          </div>
        </div>

        <div v-if="deleteError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ deleteError }}
        </div>

        <p v-if="user" class="text-sm mb-6">
          Are you sure you want to delete the user <strong>{{ user.email }}</strong>?
          This will permanently remove their account and all associated data.
        </p>

        <div class="flex justify-end gap-3">
          <Button
            variant="outline"
            :disabled="isDeleting"
            @click="showDeleteDialog = false"
          >
            Cancel
          </Button>
          <Button variant="destructive" :disabled="isDeleting" @click="handleDeleteUser">
            <svg v-if="isDeleting" class="animate-spin -ml-1 mr-2 h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            {{ isDeleting ? 'Deleting...' : 'Delete User' }}
          </Button>
        </div>
      </div>
    </Dialog>
  </div>
</template>
