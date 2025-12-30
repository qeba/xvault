<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useAdminStore } from '@/stores/admin'
import type { Source, SourceType, SourceConfig } from '@/types'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import Input from '@/components/ui/input/Input.vue'
import Label from '@/components/ui/label/Label.vue'
import Dialog from '@/components/ui/dialog/Dialog.vue'
import { Select, type SelectOption } from '@/components/ui/select'

const adminStore = useAdminStore()

const searchQuery = ref('')
const typeFilter = ref('')
const statusFilter = ref('')
const isLoading = ref(true)
const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showDeleteDialog = ref(false)
const isCreating = ref(false)
const isUpdating = ref(false)
const isDeleting = ref(false)
const isTesting = ref(false)
const isTriggering = ref(false)
const createError = ref('')
const editError = ref('')
const deleteError = ref('')
const testResult = ref<{ success: boolean; message: string; details?: string } | null>(null)
const triggerResult = ref<{ success: boolean; message: string } | null>(null)
const editingSource = ref<Source | null>(null)
const deletingSource = ref<Source | null>(null)

// Source type options
const sourceTypeOptions: SelectOption[] = [
  { label: 'SSH/SFTP', value: 'ssh' },
  { label: 'FTP', value: 'ftp' },
  { label: 'MySQL', value: 'mysql' },
  { label: 'PostgreSQL', value: 'postgresql' },
]

const typeFilterOptions: SelectOption[] = [
  { label: 'All Types', value: '' },
  ...sourceTypeOptions,
]

const statusFilterOptions: SelectOption[] = [
  { label: 'All Status', value: '' },
  { label: 'Active', value: 'active' },
  { label: 'Disabled', value: 'disabled' },
]

// Tenant select options
const tenantOptions = computed<SelectOption[]>(() => {
  return adminStore.tenants.map(t => ({
    label: t.name || t.id.slice(0, 8),
    value: t.id,
  }))
})

// Create form state
const createForm = ref({
  tenant_id: '',
  name: '',
  type: 'ssh' as SourceType,
  // SSH/SFTP/FTP config
  host: '',
  port: 22,
  username: '',
  paths: '',
  // MySQL/PostgreSQL config
  database: '',
  tables: '',
  schemas: '',
  // Credential (password or private key)
  credential: '',
  credentialType: 'password' as 'password' | 'privateKey',
})

// Edit form state
const editForm = ref({
  name: '',
  status: 'active' as 'active' | 'disabled',
  host: '',
  port: 22,
  username: '',
  paths: '',
  database: '',
  tables: '',
  schemas: '',
  credential: '',
  credentialType: 'password' as 'password' | 'privateKey',
})

// Computed
const filteredSources = computed(() => {
  let result = adminStore.sources

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(s =>
      s.name.toLowerCase().includes(query) ||
      getTenantName(s.tenant_id).toLowerCase().includes(query) ||
      s.config.host?.toLowerCase().includes(query)
    )
  }

  if (typeFilter.value) {
    result = result.filter(s => s.type === typeFilter.value)
  }

  if (statusFilter.value) {
    result = result.filter(s => s.status === statusFilter.value)
  }

  return result
})

const sourceStats = computed(() => ({
  total: adminStore.sources.length,
  active: adminStore.sources.filter(s => s.status === 'active').length,
  disabled: adminStore.sources.filter(s => s.status === 'disabled').length,
  ssh: adminStore.sources.filter(s => s.type === 'ssh' || s.type === 'sftp').length,
  database: adminStore.sources.filter(s => s.type === 'mysql' || s.type === 'postgresql').length,
}))

onMounted(async () => {
  try {
    await Promise.all([
      adminStore.fetchSources(),
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

function getTypeBadgeClass(type: string): string {
  switch (type) {
    case 'ssh':
    case 'sftp':
      return 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200'
    case 'ftp':
      return 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200'
    case 'mysql':
      return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
    case 'postgresql':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
  }
}

function getStatusBadgeClass(status: string): string {
  return status === 'active'
    ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
    : 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
}

function formatDate(date: string): string {
  return new Date(date).toLocaleDateString()
}

function getSourceSummary(source: Source): string {
  if (source.config.host) {
    return `${source.config.host}:${source.config.port || 22}`
  }
  if (source.config.database) {
    return source.config.database
  }
  return 'N/A'
}

// Dialog handlers
function openCreateDialog() {
  createError.value = ''
  testResult.value = null
  createForm.value = {
    tenant_id: tenantOptions.value[0]?.value || '',
    name: '',
    type: 'ssh',
    host: '',
    port: 22,
    username: '',
    paths: '',
    database: '',
    tables: '',
    schemas: '',
    credential: '',
    credentialType: 'password',
  }
  showCreateDialog.value = true
}

function openEditDialog(source: Source) {
  editError.value = ''
  testResult.value = null
  editingSource.value = source
  editForm.value = {
    name: source.name,
    status: source.status,
    host: source.config.host || '',
    port: source.config.port || 22,
    username: source.config.username || '',
    paths: source.config.paths?.join(', ') || '',
    database: source.config.database || '',
    tables: source.config.tables?.join(', ') || '',
    schemas: source.config.schemas?.join(', ') || '',
    credential: '',
    credentialType: 'password',
  }
  showEditDialog.value = true
}

function openDeleteDialog(source: Source) {
  deleteError.value = ''
  deletingSource.value = source
  showDeleteDialog.value = true
}

// Build config from form
function buildConfig(type: SourceType, form: typeof createForm.value | typeof editForm.value): SourceConfig {
  if (type === 'ssh' || type === 'sftp' || type === 'ftp') {
    return {
      host: form.host,
      port: form.port,
      username: form.username,
      paths: form.paths.split(',').map(p => p.trim()).filter(Boolean),
      use_password: form.credentialType === 'password',
    }
  }
  if (type === 'mysql') {
    return {
      host: form.host,
      port: form.port || 3306,
      username: form.username,
      database: form.database,
      tables: form.tables.split(',').map(t => t.trim()).filter(Boolean),
    }
  }
  if (type === 'postgresql') {
    return {
      host: form.host,
      port: form.port || 5432,
      username: form.username,
      database: form.database,
      schemas: form.schemas.split(',').map(s => s.trim()).filter(Boolean),
    }
  }
  return {}
}

// Form handlers
async function handleCreate() {
  createError.value = ''

  if (!createForm.value.tenant_id) {
    createError.value = 'Please select a tenant'
    return
  }

  if (!createForm.value.name) {
    createError.value = 'Name is required'
    return
  }

  if (!createForm.value.host) {
    createError.value = 'Host is required'
    return
  }

  if (!createForm.value.username) {
    createError.value = 'Username is required'
    return
  }

  if (!createForm.value.credential) {
    createError.value = 'Password or private key is required'
    return
  }

  isCreating.value = true
  try {
    const config = buildConfig(createForm.value.type, createForm.value)
    
    // Base64 encode the credential
    const credentialBase64 = btoa(createForm.value.credential)

    await adminStore.createSource({
      tenant_id: createForm.value.tenant_id,
      type: createForm.value.type,
      name: createForm.value.name,
      config,
      credential: credentialBase64,
    })
    showCreateDialog.value = false
  } catch (error: unknown) {
    createError.value = error instanceof Error ? error.message : 'Failed to create source'
  } finally {
    isCreating.value = false
  }
}

async function handleUpdate() {
  if (!editingSource.value) return

  editError.value = ''

  if (!editForm.value.name) {
    editError.value = 'Name is required'
    return
  }

  isUpdating.value = true
  try {
    const config = buildConfig(editingSource.value.type, editForm.value)
    
    // Only include credential if user entered a new one
    const credentialBase64 = editForm.value.credential 
      ? btoa(editForm.value.credential) 
      : undefined

    await adminStore.updateSource(editingSource.value.id, {
      name: editForm.value.name,
      status: editForm.value.status,
      config,
      credential: credentialBase64,
    })
    showEditDialog.value = false
    editingSource.value = null
  } catch (error: unknown) {
    editError.value = error instanceof Error ? error.message : 'Failed to update source'
  } finally {
    isUpdating.value = false
  }
}

async function handleDelete() {
  if (!deletingSource.value) return

  deleteError.value = ''
  isDeleting.value = true

  try {
    await adminStore.deleteSource(deletingSource.value.id)
    showDeleteDialog.value = false
    deletingSource.value = null
  } catch (error: unknown) {
    deleteError.value = error instanceof Error ? error.message : 'Failed to delete source'
  } finally {
    isDeleting.value = false
  }
}

// Test connection handler
async function handleTestConnection(isEditMode: boolean = false) {
  const form = isEditMode ? editForm.value : createForm.value
  const type = isEditMode && editingSource.value ? editingSource.value.type : createForm.value.type
  
  testResult.value = null
  
  if (!form.host || !form.username || !form.credential) {
    testResult.value = {
      success: false,
      message: 'Missing required fields',
      details: 'Please fill in host, username, and password/private key before testing'
    }
    return
  }

  isTesting.value = true
  
  try {
    const result = await adminStore.testConnection({
      type: type,
      host: form.host,
      port: form.port,
      username: form.username,
      credential: btoa(form.credential),
      use_private_key: form.credentialType === 'privateKey',
      database: form.database || undefined,
    })
    testResult.value = result
  } catch (error: unknown) {
    testResult.value = {
      success: false,
      message: 'Test failed',
      details: error instanceof Error ? error.message : 'Unknown error'
    }
  } finally {
    isTesting.value = false
  }
}

// Trigger backup handler
async function handleTriggerBackup(source: Source) {
  triggerResult.value = null
  isTriggering.value = true
  
  try {
    const result = await adminStore.triggerBackup(source.id)
    triggerResult.value = {
      success: true,
      message: `Backup job created: ${result.job.id.slice(0, 8)}...`
    }
    // Auto-hide after 5 seconds
    setTimeout(() => {
      triggerResult.value = null
    }, 5000)
  } catch (error: unknown) {
    triggerResult.value = {
      success: false,
      message: error instanceof Error ? error.message : 'Failed to trigger backup'
    }
  } finally {
    isTriggering.value = false
  }
}

// Watch type changes to set default ports
function onTypeChange(type: SourceType) {
  createForm.value.type = type
  switch (type) {
    case 'ssh':
    case 'sftp':
      createForm.value.port = 22
      break
    case 'ftp':
      createForm.value.port = 21
      break
    case 'mysql':
      createForm.value.port = 3306
      break
    case 'postgresql':
      createForm.value.port = 5432
      break
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold">Sources</h1>
        <p class="text-muted-foreground">Manage backup sources across all tenants</p>
      </div>
      <Button @click="openCreateDialog">
        <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
        </svg>
        Add Source
      </Button>
    </div>

    <!-- Backup Trigger Toast -->
    <div 
      v-if="triggerResult" 
      :class="[
        'fixed bottom-4 right-4 p-4 rounded-lg shadow-lg z-50 max-w-sm',
        triggerResult.success 
          ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 border border-green-200 dark:border-green-800' 
          : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 border border-red-200 dark:border-red-800'
      ]"
    >
      <div class="flex items-center justify-between">
        <div>
          <div class="font-medium">{{ triggerResult.success ? 'Backup Started' : 'Backup Failed' }}</div>
          <div class="text-sm opacity-80">{{ triggerResult.message }}</div>
        </div>
        <button @click="triggerResult = null" class="ml-4 p-1 hover:opacity-70">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Stats cards -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold">{{ sourceStats.total }}</div>
          <div class="text-sm text-muted-foreground">Total Sources</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-green-600">{{ sourceStats.active }}</div>
          <div class="text-sm text-muted-foreground">Active</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-gray-500">{{ sourceStats.disabled }}</div>
          <div class="text-sm text-muted-foreground">Disabled</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-indigo-600">{{ sourceStats.ssh }}</div>
          <div class="text-sm text-muted-foreground">SSH/SFTP</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="pt-6">
          <div class="text-2xl font-bold text-orange-600">{{ sourceStats.database }}</div>
          <div class="text-sm text-muted-foreground">Databases</div>
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
              placeholder="Search by name, tenant, or host..."
            />
          </div>
          <div class="w-40">
            <Select
              v-model="typeFilter"
              :options="typeFilterOptions"
              placeholder="Type"
            />
          </div>
          <div class="w-40">
            <Select
              v-model="statusFilter"
              :options="statusFilterOptions"
              placeholder="Status"
            />
          </div>
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
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Tenant</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Type</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Connection</th>
                <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">Status</th>
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
                  <div class="font-medium">{{ source.name }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ getTenantName(source.tenant_id) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <span :class="['px-2 py-1 text-xs rounded-full', getTypeBadgeClass(source.type)]">
                    {{ source.type.toUpperCase() }}
                  </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground font-mono">{{ getSourceSummary(source) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <span :class="['px-2 py-1 text-xs rounded-full', getStatusBadgeClass(source.status)]">
                    {{ source.status }}
                  </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="text-sm text-muted-foreground">{{ formatDate(source.created_at) }}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                  <div class="flex justify-end gap-2">
                    <Button 
                      variant="secondary" 
                      size="sm" 
                      :disabled="source.status !== 'active' || isTriggering"
                      @click="handleTriggerBackup(source)"
                    >
                      {{ isTriggering ? '...' : 'Backup' }}
                    </Button>
                    <Button variant="ghost" size="sm" @click="openEditDialog(source)">
                      Edit
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      class="text-destructive hover:text-destructive"
                      @click="openDeleteDialog(source)"
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

    <!-- Create Source Dialog -->
    <Dialog v-model:open="showCreateDialog">
      <div class="p-6 max-h-[80vh] overflow-y-auto">
        <h2 class="text-lg font-semibold mb-4">Create New Source</h2>

        <div v-if="createError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ createError }}
        </div>

        <form @submit.prevent="handleCreate" class="space-y-4">
          <!-- Tenant Selection -->
          <div class="space-y-2">
            <Label>Tenant</Label>
            <Select
              v-model="createForm.tenant_id"
              :options="tenantOptions"
              placeholder="Select tenant..."
              :disabled="isCreating"
            />
          </div>

          <!-- Source Name -->
          <div class="space-y-2">
            <Label for="create-name">Source Name</Label>
            <Input
              id="create-name"
              v-model="createForm.name"
              placeholder="Production Server"
              :disabled="isCreating"
            />
          </div>

          <!-- Source Type -->
          <div class="space-y-2">
            <Label>Source Type</Label>
            <Select
              :model-value="createForm.type"
              @update:model-value="onTypeChange($event as SourceType)"
              :options="sourceTypeOptions"
              :disabled="isCreating"
            />
          </div>

          <!-- Connection Settings -->
          <div class="border-t pt-4 mt-4">
            <h3 class="font-medium mb-3">Connection Settings</h3>

            <div class="grid grid-cols-2 gap-4">
              <div class="space-y-2">
                <Label for="create-host">Host</Label>
                <Input
                  id="create-host"
                  v-model="createForm.host"
                  placeholder="192.168.1.100"
                  :disabled="isCreating"
                />
              </div>
              <div class="space-y-2">
                <Label for="create-port">Port</Label>
                <Input
                  id="create-port"
                  v-model.number="createForm.port"
                  type="number"
                  :disabled="isCreating"
                />
              </div>
            </div>

            <div class="space-y-2 mt-4">
              <Label for="create-username">Username</Label>
              <Input
                id="create-username"
                v-model="createForm.username"
                placeholder="root"
                :disabled="isCreating"
              />
            </div>

            <!-- SSH/SFTP/FTP specific: Paths -->
            <div v-if="['ssh', 'sftp', 'ftp'].includes(createForm.type)" class="space-y-2 mt-4">
              <Label for="create-paths">Paths to backup (comma-separated)</Label>
              <Input
                id="create-paths"
                v-model="createForm.paths"
                placeholder="/var/www, /home/app/data"
                :disabled="isCreating"
              />
            </div>

            <!-- Database specific: Database name -->
            <div v-if="['mysql', 'postgresql'].includes(createForm.type)" class="space-y-2 mt-4">
              <Label for="create-database">Database Name</Label>
              <Input
                id="create-database"
                v-model="createForm.database"
                placeholder="myapp_production"
                :disabled="isCreating"
              />
            </div>

            <!-- MySQL specific: Tables -->
            <div v-if="createForm.type === 'mysql'" class="space-y-2 mt-4">
              <Label for="create-tables">Tables (comma-separated, leave empty for all)</Label>
              <Input
                id="create-tables"
                v-model="createForm.tables"
                placeholder="users, orders, products"
                :disabled="isCreating"
              />
            </div>

            <!-- PostgreSQL specific: Schemas -->
            <div v-if="createForm.type === 'postgresql'" class="space-y-2 mt-4">
              <Label for="create-schemas">Schemas (comma-separated, leave empty for all)</Label>
              <Input
                id="create-schemas"
                v-model="createForm.schemas"
                placeholder="public, app"
                :disabled="isCreating"
              />
            </div>
          </div>

          <!-- Credential Section -->
          <div class="border-t pt-4 mt-4">
            <h3 class="font-medium mb-3">Authentication</h3>

            <!-- Credential Type for SSH -->
            <div v-if="['ssh', 'sftp'].includes(createForm.type)" class="space-y-2 mb-4">
              <Label>Authentication Method</Label>
              <div class="flex gap-4">
                <label class="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    v-model="createForm.credentialType"
                    value="password"
                    class="h-4 w-4"
                    :disabled="isCreating"
                  />
                  <span>Password</span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    v-model="createForm.credentialType"
                    value="privateKey"
                    class="h-4 w-4"
                    :disabled="isCreating"
                  />
                  <span>SSH Private Key</span>
                </label>
              </div>
            </div>

            <div class="space-y-2">
              <Label for="create-credential">
                {{ createForm.credentialType === 'privateKey' ? 'SSH Private Key' : 'Password' }}
              </Label>
              <textarea
                v-if="createForm.credentialType === 'privateKey'"
                id="create-credential"
                v-model="createForm.credential"
                rows="6"
                class="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring font-mono"
                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----
...
-----END OPENSSH PRIVATE KEY-----"
                :disabled="isCreating"
              ></textarea>
              <Input
                v-else
                id="create-credential"
                v-model="createForm.credential"
                type="password"
                placeholder="Enter password"
                :disabled="isCreating"
              />
              <p class="text-xs text-muted-foreground">
                {{ createForm.credentialType === 'privateKey' 
                   ? 'Paste your SSH private key. It will be encrypted before storage.'
                   : 'The password will be encrypted before storage.'
                }}
              </p>
            </div>

            <!-- Test Connection Result -->
            <div v-if="testResult" :class="[
              'mt-4 p-3 rounded-md text-sm',
              testResult.success ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
            ]">
              <div class="font-medium">{{ testResult.message }}</div>
              <div v-if="testResult.details" class="text-xs mt-1 opacity-80">{{ testResult.details }}</div>
            </div>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" :disabled="isCreating || isTesting" @click="showCreateDialog = false">
              Cancel
            </Button>
            <Button type="button" variant="secondary" :disabled="isCreating || isTesting" @click="handleTestConnection(false)">
              {{ isTesting ? 'Testing...' : 'Test Connection' }}
            </Button>
            <Button type="submit" :disabled="isCreating || isTesting">
              {{ isCreating ? 'Creating...' : 'Create Source' }}
            </Button>
          </div>
        </form>
      </div>
    </Dialog>

    <!-- Edit Source Dialog -->
    <Dialog v-model:open="showEditDialog">
      <div class="p-6 max-h-[80vh] overflow-y-auto">
        <h2 class="text-lg font-semibold mb-4">Edit Source</h2>

        <div v-if="editError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ editError }}
        </div>

        <form v-if="editingSource" @submit.prevent="handleUpdate" class="space-y-4">
          <!-- Source Info (read-only) -->
          <div class="p-3 bg-muted/50 rounded-md text-sm">
            <div><strong>Type:</strong> {{ editingSource.type.toUpperCase() }}</div>
            <div><strong>Tenant:</strong> {{ getTenantName(editingSource.tenant_id) }}</div>
          </div>

          <!-- Source Name -->
          <div class="space-y-2">
            <Label for="edit-name">Source Name</Label>
            <Input
              id="edit-name"
              v-model="editForm.name"
              :disabled="isUpdating"
            />
          </div>

          <!-- Status -->
          <div class="space-y-2">
            <Label>Status</Label>
            <Select
              v-model="editForm.status"
              :options="[
                { label: 'Active', value: 'active' },
                { label: 'Disabled', value: 'disabled' },
              ]"
              :disabled="isUpdating"
            />
          </div>

          <!-- Connection Settings -->
          <div class="border-t pt-4 mt-4">
            <h3 class="font-medium mb-3">Connection Settings</h3>

            <div class="grid grid-cols-2 gap-4">
              <div class="space-y-2">
                <Label for="edit-host">Host</Label>
                <Input id="edit-host" v-model="editForm.host" :disabled="isUpdating" />
              </div>
              <div class="space-y-2">
                <Label for="edit-port">Port</Label>
                <Input id="edit-port" v-model.number="editForm.port" type="number" :disabled="isUpdating" />
              </div>
            </div>

            <div class="space-y-2 mt-4">
              <Label for="edit-username">Username</Label>
              <Input id="edit-username" v-model="editForm.username" :disabled="isUpdating" />
            </div>

            <!-- Type-specific fields -->
            <div v-if="['ssh', 'sftp', 'ftp'].includes(editingSource.type)" class="space-y-2 mt-4">
              <Label for="edit-paths">Paths to backup</Label>
              <Input id="edit-paths" v-model="editForm.paths" :disabled="isUpdating" />
            </div>

            <div v-if="['mysql', 'postgresql'].includes(editingSource.type)" class="space-y-2 mt-4">
              <Label for="edit-database">Database Name</Label>
              <Input id="edit-database" v-model="editForm.database" :disabled="isUpdating" />
            </div>
          </div>

          <!-- Credential Rotation -->
          <div class="border-t pt-4 mt-4">
            <h3 class="font-medium mb-3">Rotate Credential (Optional)</h3>
            <p class="text-sm text-muted-foreground mb-3">
              Leave empty to keep the existing credential. Enter a new value to rotate.
            </p>

            <div v-if="['ssh', 'sftp'].includes(editingSource.type)" class="space-y-2 mb-4">
              <div class="flex gap-4">
                <label class="flex items-center gap-2 cursor-pointer">
                  <input type="radio" v-model="editForm.credentialType" value="password" class="h-4 w-4" :disabled="isUpdating" />
                  <span>Password</span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer">
                  <input type="radio" v-model="editForm.credentialType" value="privateKey" class="h-4 w-4" :disabled="isUpdating" />
                  <span>SSH Private Key</span>
                </label>
              </div>
            </div>

            <div class="space-y-2">
              <Label for="edit-credential">
                New {{ editForm.credentialType === 'privateKey' ? 'SSH Private Key' : 'Password' }}
              </Label>
              <textarea
                v-if="editForm.credentialType === 'privateKey'"
                id="edit-credential"
                v-model="editForm.credential"
                rows="4"
                class="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring font-mono"
                placeholder="Leave empty to keep existing key"
                :disabled="isUpdating"
              ></textarea>
              <Input
                v-else
                id="edit-credential"
                v-model="editForm.credential"
                type="password"
                placeholder="Leave empty to keep existing"
                :disabled="isUpdating"
              />
            </div>

            <!-- Test Connection Result for Edit -->
            <div v-if="testResult" :class="[
              'mt-4 p-3 rounded-md text-sm',
              testResult.success ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
            ]">
              <div class="font-medium">{{ testResult.message }}</div>
              <div v-if="testResult.details" class="text-xs mt-1 opacity-80">{{ testResult.details }}</div>
            </div>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" :disabled="isUpdating || isTesting" @click="showEditDialog = false">
              Cancel
            </Button>
            <Button 
              v-if="editForm.credential" 
              type="button" 
              variant="secondary" 
              :disabled="isUpdating || isTesting" 
              @click="handleTestConnection(true)"
            >
              {{ isTesting ? 'Testing...' : 'Test Connection' }}
            </Button>
            <Button type="submit" :disabled="isUpdating || isTesting">
              {{ isUpdating ? 'Updating...' : 'Update Source' }}
            </Button>
          </div>
        </form>
      </div>
    </Dialog>

    <!-- Delete Confirmation Dialog -->
    <Dialog v-model:open="showDeleteDialog">
      <div class="p-6">
        <h2 class="text-lg font-semibold mb-4">Delete Source</h2>

        <div v-if="deleteError" class="mb-4 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
          {{ deleteError }}
        </div>

        <p class="mb-4">
          Are you sure you want to delete the source
          <strong>{{ deletingSource?.name }}</strong>?
        </p>

        <div class="p-3 bg-destructive/10 rounded-md text-sm mb-4">
          <strong>Warning:</strong> This will also delete all associated schedules. 
          Existing snapshots will remain but will be orphaned.
        </div>

        <div class="flex justify-end gap-3">
          <Button variant="outline" :disabled="isDeleting" @click="showDeleteDialog = false">
            Cancel
          </Button>
          <Button variant="destructive" :disabled="isDeleting" @click="handleDelete">
            {{ isDeleting ? 'Deleting...' : 'Delete Source' }}
          </Button>
        </div>
      </div>
    </Dialog>
  </div>
</template>
