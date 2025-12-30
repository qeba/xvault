<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { useAuthStore } from '@/stores/auth'
import type { User } from '@/types'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import Input from '@/components/ui/input/Input.vue'
import Label from '@/components/ui/label/Label.vue'
import Dialog from '@/components/ui/dialog/Dialog.vue'
import { Select, type SelectOption } from '@/components/ui/select'

const router = useRouter()
const adminStore = useAdminStore()
const authStore = useAuthStore()

const searchQuery = ref('')
const roleFilter = ref('')
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
const editingUser = ref<User | null>(null)
const deletingUser = ref<User | null>(null)

const roleOptions: SelectOption[] = [
  { label: 'Member', value: 'member' },
  { label: 'Admin', value: 'admin' },
  { label: 'Owner', value: 'owner' },
]

const roleFilterOptions: SelectOption[] = [
  { label: 'All Roles', value: '' },
  ...roleOptions,
]

const createUserForm = ref({
  email: '',
  password: '',
  name: '',
  role: 'member' as 'owner' | 'admin' | 'member',
})

const updateUserForm = ref({
  email: '',
  role: 'member' as 'owner' | 'admin' | 'member',
})

const filteredUsers = computed(() => {
  let result = adminStore.users

  // Filter by search query
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(u =>
      u.email.toLowerCase().includes(query) ||
      getTenantName(u.tenant_id).toLowerCase().includes(query)
    )
  }

  // Filter by role
  if (roleFilter.value) {
    result = result.filter(u => u.role === roleFilter.value)
  }

  return result
})

const userStats = computed(() => ({
  total: adminStore.users.length,
  admins: adminStore.users.filter(u => u.role === 'admin').length,
  owners: adminStore.users.filter(u => u.role === 'owner').length,
  members: adminStore.users.filter(u => u.role === 'member').length,
}))

onMounted(async () => {
  try {
    // Fetch both users and tenants for lookup
    await Promise.all([
      adminStore.fetchUsers(),
      adminStore.fetchTenants(),
    ])
  } catch (error) {
    console.error('Failed to load data:', error)
  } finally {
    isLoading.value = false
  }
})

function getTenantName(tenantId: string): string {
  const tenant = adminStore.getTenantById(tenantId)
  return tenant?.name || tenantId.slice(0, 8) + '...'
}

function openCreateDialog() {
  createError.value = ''
  createUserForm.value = { email: '', password: '', name: '', role: 'member' }
  showCreateDialog.value = true
}

function openEditDialog(user: User) {
  editError.value = ''
  editingUser.value = user
  updateUserForm.value = {
    email: user.email,
    role: user.role,
  }
  showEditDialog.value = true
}

function openDeleteDialog(user: User) {
  deleteError.value = ''
  deletingUser.value = user
  showDeleteDialog.value = true
}

async function handleCreateUser() {
  createError.value = ''

  if (!createUserForm.value.email || !createUserForm.value.password) {
    createError.value = 'Email and password are required'
    return
  }

  if (createUserForm.value.password.length < 8) {
    createError.value = 'Password must be at least 8 characters'
    return
  }

  isCreating.value = true
  try {
    await adminStore.createUser({
      email: createUserForm.value.email,
      password: createUserForm.value.password,
      name: createUserForm.value.name,
      role: createUserForm.value.role,
    })
    showCreateDialog.value = false
    // Refresh tenants list since a new one may have been created
    await adminStore.fetchTenants()
  } catch (error: unknown) {
    createError.value = error instanceof Error ? error.message : 'Failed to create user'
  } finally {
    isCreating.value = false
  }
}

async function handleUpdateUser() {
  if (!editingUser.value) return

  editError.value = ''

  if (!updateUserForm.value.email) {
    editError.value = 'Email is required'
    return
  }

  isUpdating.value = true
  try {
    await adminStore.updateUser(editingUser.value.id, {
      email: updateUserForm.value.email,
      role: updateUserForm.value.role,
    })
    showEditDialog.value = false
    editingUser.value = null
  } catch (error: unknown) {
    editError.value = error instanceof Error ? error.message : 'Failed to update user'
  } finally {
    isUpdating.value = false
  }
}

async function handleDeleteUser() {
  if (!deletingUser.value) return

  isDeleting.value = true
  deleteError.value = ''

  try {
    await adminStore.deleteUser(deletingUser.value.id)
    showDeleteDialog.value = false
    deletingUser.value = null
  } catch (error: unknown) {
    deleteError.value = error instanceof Error ? error.message : 'Failed to delete user'
  } finally {
    isDeleting.value = false
  }
}

function viewUser(id: string) {
  router.push(`/admin/users/${id}`)
}

function formatDate(date: string): string {
  return new Date(date).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
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

function isCurrentUser(userId: string): boolean {
  return authStore.user?.id === userId
}

// Reset dialogs when closed
watch(showCreateDialog, (open) => {
  if (!open) createError.value = ''
})
watch(showEditDialog, (open) => {
  if (!open) {
    editError.value = ''
    editingUser.value = null
  }
})
watch(showDeleteDialog, (open) => {
  if (!open) {
    deleteError.value = ''
    deletingUser.value = null
  }
})
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold">Users</h1>
        <p class="text-muted-foreground">Manage platform users and their permissions</p>
      </div>
      <Button @click="openCreateDialog">
        <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
        </svg>
        Add User
      </Button>
    </div>

    <!-- Stats cards -->
    <div class="grid gap-4 md:grid-cols-4">
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold">{{ userStats.total }}</div>
          <p class="text-xs text-muted-foreground">Total Users</p>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-blue-600 dark:text-blue-400">{{ userStats.owners }}</div>
          <p class="text-xs text-muted-foreground">Owners</p>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-purple-600 dark:text-purple-400">{{ userStats.admins }}</div>
          <p class="text-xs text-muted-foreground">Admins</p>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-gray-600 dark:text-gray-400">{{ userStats.members }}</div>
          <p class="text-xs text-muted-foreground">Members</p>
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
              placeholder="Search by email or tenant name..."
            />
          </div>
          <div class="w-full sm:w-48">
            <Select
              v-model="roleFilter"
              :options="roleFilterOptions"
              placeholder="Filter by role"
            />
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Users list -->
    <Card>
      <CardContent class="p-0">
        <div v-if="isLoading" class="p-8 text-center text-muted-foreground">
          <svg class="animate-spin h-8 w-8 mx-auto mb-4 text-primary" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          Loading users...
        </div>
        <div v-else-if="filteredUsers.length === 0" class="p-8 text-center text-muted-foreground">
          <svg class="w-12 h-12 mx-auto mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
          </svg>
          <p v-if="searchQuery || roleFilter">No users match your filters.</p>
          <p v-else>No users found. Create your first user to get started.</p>
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead class="border-b bg-muted/50">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">User</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Role</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Tenant</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Created</th>
                <th class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr
                v-for="user in filteredUsers"
                :key="user.id"
                class="hover:bg-muted/50 transition-colors"
              >
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="flex items-center gap-3">
                    <div class="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                      <span class="text-sm font-medium text-primary">
                        {{ user.email.charAt(0).toUpperCase() }}
                      </span>
                    </div>
                    <div>
                      <div class="text-sm font-medium">{{ user.email }}</div>
                      <div class="text-xs text-muted-foreground font-mono">{{ user.id.slice(0, 8) }}...</div>
                    </div>
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <span :class="['px-2 py-1 text-xs rounded-full font-medium', getRoleBadgeClass(user.role)]">
                    {{ user.role }}
                  </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm">{{ getTenantName(user.tenant_id) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ formatDate(user.created_at) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <div class="flex justify-end gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      @click="viewUser(user.id)"
                    >
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                      </svg>
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      @click="openEditDialog(user)"
                    >
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      class="text-destructive hover:text-destructive"
                      :disabled="isCurrentUser(user.id)"
                      :title="isCurrentUser(user.id) ? 'Cannot delete your own account' : 'Delete user'"
                      @click="openDeleteDialog(user)"
                    >
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </Button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <!-- Results count -->
        <div v-if="!isLoading && filteredUsers.length > 0" class="px-6 py-3 border-t bg-muted/50 text-xs text-muted-foreground">
          Showing {{ filteredUsers.length }} of {{ adminStore.users.length }} users
        </div>
      </CardContent>
    </Card>

    <!-- Create User Dialog -->
    <Dialog v-model:open="showCreateDialog">
      <div class="p-6">
        <h2 class="text-lg font-semibold mb-4">Create New User</h2>
        <p class="text-sm text-muted-foreground mb-6">
          Create a new user account and assign them to a new tenant.
        </p>

        <div v-if="createError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ createError }}
        </div>

        <form @submit.prevent="handleCreateUser" class="space-y-4">
          <div class="space-y-2">
            <Label for="create-email">Email <span class="text-destructive">*</span></Label>
            <Input
              id="create-email"
              v-model="createUserForm.email"
              type="email"
              placeholder="user@example.com"
              required
              :disabled="isCreating"
            />
          </div>

          <div class="space-y-2">
            <Label for="create-password">Password <span class="text-destructive">*</span></Label>
            <Input
              id="create-password"
              v-model="createUserForm.password"
              type="password"
              placeholder="Min. 8 characters"
              required
              minlength="8"
              :disabled="isCreating"
            />
          </div>

          <div class="space-y-2">
            <Label for="create-tenant">Tenant Name <span class="text-muted-foreground text-xs">(optional)</span></Label>
            <Input
              id="create-tenant"
              v-model="createUserForm.name"
              type="text"
              placeholder="Leave empty to auto-generate from email"
              :disabled="isCreating"
            />
            <p class="text-xs text-muted-foreground">If left empty, tenant name will be generated as "Username's Workspace".</p>
          </div>

          <div class="space-y-2">
            <Label for="create-role">Role</Label>
            <Select
              id="create-role"
              v-model="createUserForm.role"
              :options="roleOptions"
              :disabled="isCreating"
            />
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
              <svg v-if="isCreating" class="animate-spin -ml-1 mr-2 h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              {{ isCreating ? 'Creating...' : 'Create User' }}
            </Button>
          </div>
        </form>
      </div>
    </Dialog>

    <!-- Edit User Dialog -->
    <Dialog v-model:open="showEditDialog">
      <div class="p-6">
        <h2 class="text-lg font-semibold mb-4">Edit User</h2>
        <p v-if="editingUser" class="text-sm text-muted-foreground mb-6">
          Update user information for {{ editingUser.email }}
        </p>

        <div v-if="editError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ editError }}
        </div>

        <form @submit.prevent="handleUpdateUser" class="space-y-4">
          <div class="space-y-2">
            <Label for="edit-email">Email <span class="text-destructive">*</span></Label>
            <Input
              id="edit-email"
              v-model="updateUserForm.email"
              type="email"
              placeholder="user@example.com"
              required
              :disabled="isUpdating"
            />
          </div>

          <div class="space-y-2">
            <Label for="edit-role">Role</Label>
            <Select
              id="edit-role"
              v-model="updateUserForm.role"
              :options="roleOptions"
              :disabled="isUpdating"
            />
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <Button
              type="button"
              variant="outline"
              :disabled="isUpdating"
              @click="showEditDialog = false"
            >
              Cancel
            </Button>
            <Button type="submit" :disabled="isUpdating">
              <svg v-if="isUpdating" class="animate-spin -ml-1 mr-2 h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              {{ isUpdating ? 'Saving...' : 'Save Changes' }}
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

        <p v-if="deletingUser" class="text-sm mb-6">
          Are you sure you want to delete the user <strong>{{ deletingUser.email }}</strong>?
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
