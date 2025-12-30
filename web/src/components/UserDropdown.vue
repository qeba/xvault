<script setup lang="ts">
import { LogOut } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Avatar } from '@/components/ui/avatar'
import Button from '@/components/ui/button/Button.vue'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'

const router = useRouter()
const authStore = useAuthStore()

async function handleLogout() {
  await authStore.logout()
  router.push('/auth/login')
}

// Get initials from email
function getInitials(email: string | undefined): string {
  if (!email) return '?'
  const username = email.split('@')[0] || email
  const parts = username.split(/[._-]/)
  if (parts.length >= 2 && parts[0] && parts[1]) {
    return `${parts[0][0] || ''}${parts[1][0] || ''}`.toUpperCase()
  }
  return username.substring(0, 2).toUpperCase()
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger>
      <Button variant="ghost" class="relative h-8 w-8 rounded-full">
        <Avatar
          :alt="authStore.user?.email"
          :fallback="getInitials(authStore.user?.email)"
        />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent class="w-56" align="end">
      <DropdownMenuLabel class="font-normal">
        <div class="flex flex-col space-y-1">
          <p class="text-sm font-medium leading-none">
            {{ authStore.user?.email?.split('@')[0] }}
          </p>
          <p class="text-xs leading-none text-muted-foreground">
            {{ authStore.user?.email }}
          </p>
        </div>
      </DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuItem destructive @select="handleLogout">
        <LogOut class="mr-2 h-4 w-4" />
        <span>Sign out</span>
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
