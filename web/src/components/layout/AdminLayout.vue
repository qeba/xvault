<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import {
  LayoutDashboard,
  Building2,
  Users,
  Server,
  Clock,
  Archive,
  Settings,
  Menu,
  ScrollText,
  Cpu,
} from 'lucide-vue-next'
import ThemeSwitch from '@/components/ThemeSwitch.vue'
import UserDropdown from '@/components/UserDropdown.vue'
import { Separator } from '@/components/ui/separator'

const route = useRoute()
const authStore = useAuthStore()

const sidebarOpen = ref(false)

const navigation = [
  { name: 'Dashboard', href: '/admin/dashboard', icon: LayoutDashboard },
  { name: 'Tenants', href: '/admin/tenants', icon: Building2 },
  { name: 'Users', href: '/admin/users', icon: Users },
  { name: 'Sources', href: '/admin/sources', icon: Server },
  { name: 'Schedules', href: '/admin/schedules', icon: Clock },
  { name: 'Snapshots', href: '/admin/snapshots', icon: Archive },
  { name: 'Logs', href: '/admin/logs', icon: ScrollText },
  { name: 'Workers', href: '/admin/workers', icon: Cpu },
  { name: 'Settings', href: '/admin/settings', icon: Settings },
]

const currentPage = computed(() => {
  const path = route.path
  return navigation.find((item) => path.startsWith(item.href))?.name || 'Admin'
})

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
        'fixed inset-y-0 left-0 z-50 w-64 bg-sidebar border-r border-sidebar-border transform transition-transform duration-200 ease-in-out lg:translate-x-0',
        sidebarOpen ? 'translate-x-0' : '-translate-x-full'
      ]"
    >
      <div class="flex flex-col h-full">
        <!-- Logo -->
        <div class="flex items-center h-16 px-6 border-b border-sidebar-border">
          <div class="flex items-center gap-2">
            <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Archive class="h-4 w-4" />
            </div>
            <div class="grid flex-1 text-left text-sm leading-tight">
              <span class="truncate font-semibold">xVault</span>
              <span class="truncate text-xs text-muted-foreground">Backup Platform</span>
            </div>
          </div>
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
                ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
            ]"
            @click="sidebarOpen = false"
          >
            <component :is="item.icon" class="h-4 w-4" />
            {{ item.name }}
          </router-link>
        </nav>

        <!-- User info -->
        <div class="p-4 border-t border-sidebar-border">
          <div class="flex items-center gap-3">
            <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-muted text-sm font-medium">
              {{ authStore.user?.email?.substring(0, 2).toUpperCase() || '?' }}
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium truncate">{{ authStore.user?.email?.split('@')[0] }}</p>
              <p class="text-xs text-muted-foreground capitalize">{{ authStore.user?.role }}</p>
            </div>
          </div>
        </div>
      </div>
    </aside>

    <!-- Main content -->
    <div class="lg:pl-64">
      <!-- Header -->
      <header class="sticky top-0 z-30 h-16 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div class="flex h-full items-center gap-4 px-4">
          <!-- Mobile menu button -->
          <button
            class="lg:hidden inline-flex items-center justify-center rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
            @click="sidebarOpen = !sidebarOpen"
          >
            <Menu class="h-5 w-5" />
          </button>

          <Separator orientation="vertical" class="h-6 lg:hidden" />

          <!-- Page title -->
          <div class="flex-1">
            <h1 class="text-lg font-semibold">{{ currentPage }}</h1>
          </div>

          <!-- Right side actions -->
          <div class="flex items-center gap-2">
            <ThemeSwitch />
            <UserDropdown />
          </div>
        </div>
      </header>

      <!-- Page content -->
      <main class="p-6">
        <router-view />
      </main>
    </div>
  </div>
</template>
