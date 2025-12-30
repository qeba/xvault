<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Card from '@/components/ui/card/Card.vue'
import CardHeader from '@/components/ui/card/CardHeader.vue'
import CardTitle from '@/components/ui/card/CardTitle.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import Button from '@/components/ui/button/Button.vue'

const router = useRouter()
const authStore = useAuthStore()

const stats = ref({
  sources: 0,
  snapshots: 0,
  schedules: 0,
})

function goToSources() {
  router.push('/sources')
}

function goToSchedules() {
  router.push('/schedules')
}

function goToSnapshots() {
  router.push('/snapshots')
}
</script>

<template>
  <div class="min-h-screen bg-background">
    <!-- Header -->
    <header class="border-b">
      <div class="container mx-auto px-4 py-4 flex items-center justify-between">
        <h1 class="text-xl font-bold">xVault</h1>
        <div class="flex items-center gap-4">
          <span class="text-sm text-muted-foreground">{{ authStore.user?.email }}</span>
          <Button variant="ghost" size="sm" @click="authStore.logout(); router.push('/auth/login')">
            Sign out
          </Button>
        </div>
      </div>
    </header>

    <!-- Main content -->
    <main class="container mx-auto px-4 py-8">
      <div class="space-y-8">
        <!-- Welcome -->
        <div>
          <h2 class="text-3xl font-bold">Welcome back!</h2>
          <p class="text-muted-foreground">Manage your backups and schedules</p>
        </div>

        <!-- Stats -->
        <div class="grid gap-4 md:grid-cols-3">
          <Card>
            <CardHeader class="flex flex-row items-center justify-between pb-2">
              <CardTitle class="text-sm font-medium">Sources</CardTitle>
              <svg class="w-4 h-4 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
              </svg>
            </CardHeader>
            <CardContent>
              <div class="text-2xl font-bold">{{ stats.sources }}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader class="flex flex-row items-center justify-between pb-2">
              <CardTitle class="text-sm font-medium">Snapshots</CardTitle>
              <svg class="w-4 h-4 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </CardHeader>
            <CardContent>
              <div class="text-2xl font-bold">{{ stats.snapshots }}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader class="flex flex-row items-center justify-between pb-2">
              <CardTitle class="text-sm font-medium">Schedules</CardTitle>
              <svg class="w-4 h-4 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </CardHeader>
            <CardContent>
              <div class="text-2xl font-bold">{{ stats.schedules }}</div>
            </CardContent>
          </Card>
        </div>

        <!-- Quick actions -->
        <div class="grid gap-4 md:grid-cols-3">
          <Card class="cursor-pointer hover:bg-accent/50 transition-colors" @click="goToSources">
            <CardContent class="pt-6">
              <div class="flex items-center gap-4">
                <div class="p-3 rounded-lg bg-primary/10">
                  <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                  </svg>
                </div>
                <div>
                  <p class="font-medium">Add Source</p>
                  <p class="text-sm text-muted-foreground">Configure backup source</p>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card class="cursor-pointer hover:bg-accent/50 transition-colors" @click="goToSchedules">
            <CardContent class="pt-6">
              <div class="flex items-center gap-4">
                <div class="p-3 rounded-lg bg-primary/10">
                  <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                  </svg>
                </div>
                <div>
                  <p class="font-medium">Create Schedule</p>
                  <p class="text-sm text-muted-foreground">Automate backups</p>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card class="cursor-pointer hover:bg-accent/50 transition-colors" @click="goToSnapshots">
            <CardContent class="pt-6">
              <div class="flex items-center gap-4">
                <div class="p-3 rounded-lg bg-primary/10">
                  <svg class="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                </div>
                <div>
                  <p class="font-medium">View Snapshots</p>
                  <p class="text-sm text-muted-foreground">Browse backups</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        <!-- Placeholder note -->
        <Card>
          <CardContent class="pt-6">
            <p class="text-center text-muted-foreground">
              User dashboard is currently a placeholder. Admin functionality is the primary focus.
            </p>
          </CardContent>
        </Card>
      </div>
    </main>
  </div>
</template>
