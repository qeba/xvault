import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AdminLayout from '@/components/layout/AdminLayout.vue'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    component: () => import('@/views/HomeView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/auth/login',
    name: 'login',
    component: () => import('@/views/auth/LoginView.vue'),
    meta: { requiresAuth: false, hideForAuth: true },
  },
  {
    path: '/auth/register',
    name: 'register',
    component: () => import('@/views/auth/RegisterView.vue'),
    meta: { requiresAuth: false, hideForAuth: true },
  },
  {
    path: '/admin',
    component: AdminLayout,
    meta: { requiresAuth: true, requiresAdmin: true },
    children: [
      {
        path: '',
        redirect: '/admin/dashboard',
      },
      {
        path: 'dashboard',
        name: 'admin-dashboard',
        component: () => import('@/views/admin/DashboardView.vue'),
      },
      {
        path: 'tenants',
        name: 'admin-tenants',
        component: () => import('@/views/admin/TenantsView.vue'),
      },
      {
        path: 'users',
        name: 'admin-users',
        component: () => import('@/views/admin/UsersView.vue'),
      },
      {
        path: 'users/:id',
        name: 'admin-user-detail',
        component: () => import('@/views/admin/UserDetailView.vue'),
      },
      {
        path: 'sources',
        name: 'admin-sources',
        component: () => import('@/views/admin/SourcesView.vue'),
      },
      {
        path: 'schedules',
        name: 'admin-schedules',
        component: () => import('@/views/admin/SchedulesView.vue'),
      },
      {
        path: 'snapshots',
        name: 'admin-snapshots',
        component: () => import('@/views/admin/SnapshotsView.vue'),
      },
      {
        path: 'logs',
        name: 'admin-logs',
        component: () => import('@/views/admin/LogsView.vue'),
      },
      {
        path: 'workers',
        name: 'admin-workers',
        component: () => import('@/views/admin/WorkersView.vue'),
      },
      {
        path: 'settings',
        name: 'admin-settings',
        component: () => import('@/views/admin/SettingsView.vue'),
      },
    ],
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: () => import('@/views/dashboard/DashboardView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    redirect: '/',
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Navigation guard for auth
router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore()

  // Initialize auth state from storage if not loaded
  if (!authStore.isAuthenticated && authStore.accessToken) {
    try {
      await authStore.fetchMe()
    } catch {
      // Token invalid, clear state
      await authStore.logout()
    }
  }

  const requiresAuth = to.meta.requiresAuth !== false
  const requiresAdmin = to.meta.requiresAdmin === true
  const hideForAuth = to.meta.hideForAuth === true

  // Redirect authenticated users away from auth pages
  if (hideForAuth && authStore.isAuthenticated) {
    return next('/admin/dashboard')
  }

  // Check authentication
  if (requiresAuth && !authStore.isAuthenticated) {
    return next({ name: 'login', query: { redirect: to.fullPath } })
  }

  // Check admin role
  if (requiresAdmin && !authStore.isAdmin) {
    return next({ name: 'dashboard' })
  }

  next()
})

export default router
