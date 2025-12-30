<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Archive } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import Button from '@/components/ui/button/Button.vue'
import Input from '@/components/ui/input/Input.vue'
import Label from '@/components/ui/label/Label.vue'
import Card from '@/components/ui/card/Card.vue'
import CardHeader from '@/components/ui/card/CardHeader.vue'
import CardTitle from '@/components/ui/card/CardTitle.vue'
import CardContent from '@/components/ui/card/CardContent.vue'
import CardFooter from '@/components/ui/card/CardFooter.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const isLoading = ref(false)
const error = ref('')

async function handleSubmit() {
  error.value = ''
  isLoading.value = true

  try {
    await authStore.login({
      email: email.value,
      password: password.value,
    })

    // Redirect to intended page or dashboard
    const redirect = route.query.redirect as string
    if (authStore.isAdmin) {
      router.push(redirect || '/admin/dashboard')
    } else {
      router.push(redirect || '/dashboard')
    }
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : 'Login failed'
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div class="container grid h-svh max-w-none items-center justify-center lg:grid-cols-2 lg:px-0">
    <!-- Left side - Branding -->
    <div class="relative hidden h-full flex-col bg-muted p-10 text-foreground lg:flex">
      <div class="absolute inset-0 bg-primary/5" />
      <div class="relative z-20 flex items-center text-lg font-medium">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground mr-2">
          <Archive class="h-4 w-4" />
        </div>
        xVault
      </div>
      <div class="relative z-20 mt-auto">
        <blockquote class="space-y-2">
          <p class="text-lg">
            &ldquo;Secure, reliable backup solutions for your critical data. Set it and forget it.&rdquo;
          </p>
          <footer class="text-sm text-muted-foreground">Enterprise Backup Platform</footer>
        </blockquote>
      </div>
    </div>

    <!-- Right side - Login form -->
    <div class="lg:p-8">
      <div class="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-[400px]">
        <!-- Logo for mobile -->
        <div class="flex flex-col space-y-2 text-center lg:hidden">
          <div class="mx-auto flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Archive class="h-5 w-5" />
          </div>
          <h1 class="text-2xl font-semibold tracking-tight">xVault</h1>
        </div>

        <Card class="border-0 shadow-none lg:border lg:shadow-sm">
          <CardHeader class="space-y-1">
            <CardTitle class="text-2xl font-semibold tracking-tight">
              Sign in
            </CardTitle>
            <p class="text-sm text-muted-foreground">
              Enter your credentials to access your account
            </p>
          </CardHeader>
          <CardContent class="space-y-4">
            <div v-if="error" class="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
              {{ error }}
            </div>

            <form @submit.prevent="handleSubmit" class="space-y-4">
              <div class="space-y-2">
                <Label for="email">Email</Label>
                <Input
                  id="email"
                  v-model="email"
                  type="email"
                  placeholder="name@example.com"
                  required
                  :disabled="isLoading"
                  class="h-10"
                />
              </div>

              <div class="space-y-2">
                <Label for="password">Password</Label>
                <Input
                  id="password"
                  v-model="password"
                  type="password"
                  placeholder="••••••••"
                  required
                  :disabled="isLoading"
                  class="h-10"
                />
              </div>

              <Button type="submit" class="w-full h-10" :disabled="isLoading">
                {{ isLoading ? 'Signing in...' : 'Sign in' }}
              </Button>
            </form>
          </CardContent>
          <CardFooter class="flex flex-col space-y-4">
            <div class="relative w-full">
              <div class="absolute inset-0 flex items-center">
                <span class="w-full border-t" />
              </div>
              <div class="relative flex justify-center text-xs uppercase">
                <span class="bg-background px-2 text-muted-foreground">
                  New to xVault?
                </span>
              </div>
            </div>
            <router-link 
              to="/auth/register" 
              class="inline-flex h-10 w-full items-center justify-center rounded-md border border-input bg-background px-4 py-2 text-sm font-medium ring-offset-background transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            >
              Create an account
            </router-link>
          </CardFooter>
        </Card>
      </div>
    </div>
  </div>
</template>
