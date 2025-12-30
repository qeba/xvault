<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAdminStore } from '@/stores/admin'
import Button from '@/components/ui/button/Button.vue'
import Card from '@/components/ui/card/Card.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import Input from '@/components/ui/input/Input.vue'
import Label from '@/components/ui/label/Label.vue'

const adminStore = useAdminStore()

const isLoading = ref(true)
const isSaving = ref(false)
const saveMessage = ref('')

onMounted(async () => {
  try {
    await adminStore.fetchSettings()
  } catch (error) {
    console.error('Failed to load settings:', error)
  } finally {
    isLoading.value = false
  }
})

function updateSettingValue(key: string, value: string) {
  const setting = adminStore.settings.find(s => s.key === key)
  if (setting) {
    setting.value = value
  }
}

async function handleSaveSettings() {
  isSaving.value = true
  saveMessage.value = ''

  try {
    // Update all settings individually
    for (const setting of adminStore.settings) {
      await adminStore.updateSetting(setting.key, { value: setting.value })
    }
    saveMessage.value = 'Settings saved successfully!'
    setTimeout(() => {
      saveMessage.value = ''
    }, 3000)
  } catch (error) {
    saveMessage.value = error instanceof Error ? error.message : 'Failed to save settings'
  } finally {
    isSaving.value = false
  }
}

function getSettingValue(key: string): string {
  const setting = adminStore.settings.find(s => s.key === key)
  return setting?.value || ''
}
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold">Settings</h1>
        <p class="text-muted-foreground">Manage system-wide configuration</p>
      </div>
      <Button @click="handleSaveSettings" :disabled="isSaving || isLoading">
        <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
        </svg>
        {{ isSaving ? 'Saving...' : 'Save Settings' }}
      </Button>
    </div>

    <!-- Save message -->
    <div
      v-if="saveMessage"
      :class="[
        'p-4 rounded-md text-sm',
        saveMessage.includes('success') ? 'bg-green-50 text-green-800 dark:bg-green-900/20 dark:text-green-200' : 'bg-destructive/10 text-destructive'
      ]"
    >
      {{ saveMessage }}
    </div>

    <!-- Settings groups -->
    <div v-if="isLoading" class="p-8 text-center text-muted-foreground">
      Loading settings...
    </div>

    <div v-else class="grid gap-6">
      <!-- Security Settings -->
      <Card>
        <CardContent class="p-6">
          <h2 class="text-lg font-semibold mb-4">Security</h2>
          <div class="space-y-4">
            <div class="space-y-2">
              <Label for="jwt-expiration">JWT Token Expiration (hours)</Label>
              <Input
                id="jwt-expiration"
                type="number"
                :value="getSettingValue('jwt_expiration_hours')"
                @input="updateSettingValue('jwt_expiration_hours', ($event.target as HTMLInputElement).value)"
              />
              <p class="text-xs text-muted-foreground">How long access tokens are valid</p>
            </div>

            <div class="space-y-2">
              <Label for="min-password-length">Minimum Password Length</Label>
              <Input
                id="min-password-length"
                type="number"
                :value="getSettingValue('min_password_length')"
                @input="updateSettingValue('min_password_length', ($event.target as HTMLInputElement).value)"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Download Settings -->
      <Card>
        <CardContent class="p-6">
          <h2 class="text-lg font-semibold mb-4">Downloads</h2>
          <div class="space-y-4">
            <div class="space-y-2">
              <Label for="download-token-expiration">Download Token Expiration (minutes)</Label>
              <Input
                id="download-token-expiration"
                type="number"
                :value="getSettingValue('download_token_expiration_minutes')"
                @input="updateSettingValue('download_token_expiration_minutes', ($event.target as HTMLInputElement).value)"
              />
              <p class="text-xs text-muted-foreground">How long download links remain valid</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Backup Settings -->
      <Card>
        <CardContent class="p-6">
          <h2 class="text-lg font-semibold mb-4">Backups</h2>
          <div class="space-y-4">
            <div class="space-y-2">
              <Label for="default-retention">Default Retention (days)</Label>
              <Input
                id="default-retention"
                type="number"
                :value="getSettingValue('default_retention_days')"
                @input="updateSettingValue('default_retention_days', ($event.target as HTMLInputElement).value)"
              />
              <p class="text-xs text-muted-foreground">Default retention period for new schedules</p>
            </div>

            <div class="space-y-2">
              <Label for="max-snapshots-per-source">Max Snapshots Per Source</Label>
              <Input
                id="max-snapshots-per-source"
                type="number"
                :value="getSettingValue('max_snapshots_per_source')"
                @input="updateSettingValue('max_snapshots_per_source', ($event.target as HTMLInputElement).value)"
              />
              <p class="text-xs text-muted-foreground">Maximum number of snapshots to keep per source</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Storage Settings -->
      <Card>
        <CardContent class="p-6">
          <h2 class="text-lg font-semibold mb-4">Storage</h2>
          <div class="space-y-4">
            <div class="space-y-2">
              <Label for="storage-backend">Storage Backend</Label>
              <Input
                id="storage-backend"
                type="text"
                :value="getSettingValue('storage_backend')"
                @input="updateSettingValue('storage_backend', ($event.target as HTMLInputElement).value)"
                readonly
                class="bg-muted"
              />
              <p class="text-xs text-muted-foreground">Current storage backend (read-only)</p>
            </div>

            <div class="space-y-2">
              <Label for="max-snapshot-size">Max Snapshot Size (GB)</Label>
              <Input
                id="max-snapshot-size"
                type="number"
                :value="getSettingValue('max_snapshot_size_gb')"
                @input="updateSettingValue('max_snapshot_size_gb', ($event.target as HTMLInputElement).value)"
              />
              <p class="text-xs text-muted-foreground">Maximum size allowed for a single snapshot</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
