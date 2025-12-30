<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Button from '@/components/ui/button/Button.vue'

const router = useRouter()
const authStore = useAuthStore()

function goToLogin() {
  router.push('/auth/login')
}

function goToRegister() {
  router.push('/auth/register')
}

function goToDashboard() {
  if (authStore.isAdmin) {
    router.push('/admin/dashboard')
  } else {
    router.push('/dashboard')
  }
}
</script>

<template>
  <div class="min-h-screen bg-gradient-to-b from-background to-muted flex items-center justify-center p-4">
    <div class="max-w-4xl w-full text-center space-y-8">
      <div class="space-y-4">
        <h1 class="text-5xl font-bold tracking-tight">xVault</h1>
        <p class="text-xl text-muted-foreground">
          Automated Backup SaaS Platform
        </p>
      </div>

      <div class="flex flex-col sm:flex-row gap-4 justify-center">
        <Button v-if="!authStore.isAuthenticated" size="lg" @click="goToLogin">
          Sign In
        </Button>
        <Button v-if="!authStore.isAuthenticated" variant="outline" size="lg" @click="goToRegister">
          Get Started
        </Button>
        <Button v-if="authStore.isAuthenticated" size="lg" @click="goToDashboard">
          Go to Dashboard
        </Button>
      </div>
    </div>
  </div>
</template>
