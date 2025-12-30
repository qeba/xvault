<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Button from '@/components/ui/button/Button.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const sidebarOpen = ref(false)

const navigation = [
  { name: 'Dashboard', href: '/admin/dashboard', icon: 'M3 13h8V3H3v10zm0 8h8v-6H3v6zm10 0h8V11h-8v10zm0-18v6h8V3h-8z' },
  { name: 'Tenants', href: '/admin/tenants', icon: 'M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14l-5-5 1.41-1.41L12 14.17l4.59-4.58L18 11l-6 6z' },
  { name: 'Users', href: '/admin/users', icon: 'M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z' },
  { name: 'Sources', href: '/admin/sources', icon: 'M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 14l-5-5 1.41-1.41L12 14.17l4.59-4.58L18 11l-6 6z' },
  { name: 'Schedules', href: '/admin/schedules', icon: 'M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z' },
  { name: 'Snapshots', href: '/admin/snapshots', icon: 'M19 12h-2v3h-3v2h5v-5zM7 9h2V6h3V4H7v5zm12-5h-5v2h3v3h2V6zm-9 14H7v-3H5v5h5v-2z' },
  { name: 'Settings', href: '/admin/settings', icon: 'M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.09.63-.09.94s.02.64.07.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z' },
]

const currentPage = computed(() => {
  const path = route.path
  return navigation.find((item) => path.startsWith(item.href))?.name || 'Admin'
})

async function handleLogout() {
  await authStore.logout()
  router.push('/auth/login')
}

function isActive(href: string): boolean {
  return route.path === href || route.path.startsWith(href + '/')
}
</script>

<template>
  <div class="min-h-screen bg-background">
    <!-- Mobile sidebar backdrop -->
    <div
      v-if="sidebarOpen"
      class="fixed inset-0 z-40 lg:hidden bg-black/50"
      @click="sidebarOpen = false"
    />

    <!-- Sidebar -->
    <aside
      :class="[
        'fixed inset-y-0 left-0 z-50 w-64 bg-card border-r transform transition-transform duration-200 ease-in-out lg:translate-x-0',
        sidebarOpen ? 'translate-x-0' : '-translate-x-full'
      ]"
    >
      <div class="flex flex-col h-full">
        <!-- Logo -->
        <div class="flex items-center h-16 px-6 border-b">
          <h1 class="text-xl font-bold">xVault</h1>
        </div>

        <!-- Navigation -->
        <nav class="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
          <router-link
            v-for="item in navigation"
            :key="item.href"
            :to="item.href"
            :class="[
              'flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors',
              isActive(item.href)
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
            ]"
            @click="sidebarOpen = false"
          >
            <svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
              <path :d="item.icon" />
            </svg>
            {{ item.name }}
          </router-link>
        </nav>

        <!-- User info -->
        <div class="p-4 border-t">
          <div class="flex items-center gap-3">
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium truncate">{{ authStore.user?.email }}</p>
              <p class="text-xs text-muted-foreground capitalize">{{ authStore.user?.role }}</p>
            </div>
          </div>
        </div>
      </div>
    </aside>

    <!-- Main content -->
    <div class="lg:pl-64">
      <!-- Header -->
      <header class="sticky top-0 z-30 flex items-center h-16 px-4 bg-background border-b">
        <button
          class="lg:hidden p-2 rounded-md hover:bg-accent"
          @click="sidebarOpen = !sidebarOpen"
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>

        <div class="flex-1 px-4">
          <h2 class="text-lg font-semibold">{{ currentPage }}</h2>
        </div>

        <div class="flex items-center gap-2">
          <Button variant="ghost" size="sm" @click="handleLogout">
            Sign out
          </Button>
        </div>
      </header>

      <!-- Page content -->
      <main class="p-6">
        <router-view />
      </main>
    </div>
  </div>
</template>
