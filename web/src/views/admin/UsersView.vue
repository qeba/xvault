<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import Input from '@/components/ui/input/Input.vue'
import Label from '@/components/ui/label/Label.vue'
import Dialog from '@/components/ui/dialog/Dialog.vue'

const router = useRouter()
const adminStore = useAdminStore()

const searchQuery = ref('')
const isLoading = ref(true)
const showCreateDialog = ref(false)
const isCreating = ref(false)
const createError = ref('')

const createUserForm = ref({
  email: '',
  password: '',
  name: '',
  role: 'member' as 'owner' | 'admin' | 'member',
})

const filteredUsers = computed(() => {
  if (!searchQuery.value) return adminStore.users
  return adminStore.users.filter(u =>
    u.email.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

onMounted(async () => {
  try {
    await adminStore.fetchUsers()
  } catch (error) {
    console.error('Failed to load users:', error)
  } finally {
    isLoading.value = false
  }
})

async function openCreateDialog() {
  createError.value = ''
  createUserForm.value = { email: '', password: '', name: '', role: 'member' }
  showCreateDialog.value = true
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
  } catch (error: unknown) {
    createError.value = error instanceof Error ? error.message : 'Failed to create user'
  } finally {
    isCreating.value = false
  }
}

function viewUser(id: string) {
  router.push(`/admin/users/${id}`)
}

function formatDate(date: string): string {
  return new Date(date).toLocaleDateString()
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
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold">Users</h1>
        <p class="text-muted-foreground">Manage platform users</p>
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
              placeholder="Search users by email..."
            />
          </div>
          <Button @click="openCreateDialog">
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
            Add User
          </Button>
        </div>
      </CardContent>
    </Card>

    <!-- Users list -->
    <Card>
      <CardContent class="p-0">
        <div v-if="isLoading" class="p-8 text-center text-muted-foreground">
          Loading users...
        </div>
        <div v-else-if="filteredUsers.length === 0" class="p-8 text-center text-muted-foreground">
          No users found. Create your first user to get started.
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead class="border-b bg-muted/50">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Email</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Role</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Tenant ID</th>
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
                  <div class="text-sm font-medium">{{ user.email }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <span :class="['px-2 py-1 text-xs rounded-full', getRoleBadgeClass(user.role)]">
                    {{ user.role }}
                  </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground font-mono text-xs">
                    {{ user.tenant_id.slice(0, 8) }}...
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ formatDate(user.created_at) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <Button
                    variant="ghost"
                    size="sm"
                    @click="viewUser(user.id)"
                  >
                    View
                  </Button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>

    <!-- Create User Dialog -->
    <Dialog v-model:open="showCreateDialog">
      <div class="p-6">
        <h2 class="text-lg font-semibold mb-4">Create New User</h2>

        <div v-if="createError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ createError }}
        </div>

        <form @submit.prevent="handleCreateUser" class="space-y-4">
          <div class="space-y-2">
            <Label for="create-name">Name</Label>
            <Input
              id="create-name"
              v-model="createUserForm.name"
              type="text"
              placeholder="John Doe"
              :disabled="isCreating"
            />
          </div>

          <div class="space-y-2">
            <Label for="create-email">Email</Label>
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
            <Label for="create-password">Password</Label>
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
            <Label for="create-role">Role</Label>
            <select
              id="create-role"
              v-model="createUserForm.role"
              class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              :disabled="isCreating"
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
              :disabled="isCreating"
              @click="showCreateDialog = false"
            >
              Cancel
            </Button>
            <Button type="submit" :disabled="isCreating">
              {{ isCreating ? 'Creating...' : 'Create User' }}
            </Button>
          </div>
        </form>
      </div>
    </Dialog>
  </div>
</template>
