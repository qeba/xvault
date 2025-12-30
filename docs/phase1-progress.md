# Phase 1: Frontend Dashboard & JWT Authentication

**Last Updated:** 2025-12-30

**Status:** 🚧 **In Progress - Admin-First Approach**

**IMPORTANT:** This phase prioritizes the **admin dashboard** with full CRUD control before building user-facing features.

## Admin-First Priority

The frontend is being built with an **admin-first approach** to ensure full platform control is working before adding user-facing features:

1. ✅ **Backend JWT** - Complete authentication system
2. 🔴 **Admin Dashboard** (CURRENT FOCUS)
   - Add users to the platform
   - Add sources for backups
   - Create schedules for users
   - Set retention policies
   - Generate download links for backups
3. ⏳ **User Dashboard** - Simplified view for end users
4. ⏳ **Home/Marketing** - Placeholder only for now

---

## Technology Stack

| Component | Technology | Version | Notes |
|-----------|------------|---------|-------|
| **Framework** | Vue 3 | Latest | Composition API, `<script setup>` |
| **Build Tool** | Vite | Latest | Fast dev server, optimized builds |
| **UI Library** | shadcn-vue | Latest | Vue port of shadcn/ui (Radix Vue + Tailwind) |
| **Styling** | Tailwind CSS | Latest | Utility-first CSS with shadcn themes |
| **Icons** | Lucide Vue | Latest | Consistent icon set |
| **State Management** | Pinia | Latest | Official Vue state library |
| **Routing** | Vue Router | Latest | Official Vue router |
| **HTTP Client** | Axios | Latest | API communication with interceptors |
| **Forms** | VeeValidate + Yup | Latest | Form validation |
| **Components** | Reka UI | Latest | Headless UI primitives (via shadcn-vue) |
| **Dark Mode** | VueUse | `useDark` | Reactive dark mode with persistence |

**Alternative Considered**: Nuxt 4 (decided against - simpler with Vue 3 + Vite for monorepo integration)

---

## Reference Templates

Research conducted for UI/UX inspiration:

| Template | Framework | Notes | Used For |
|----------|-----------|-------|----------|
| [shadcn-admin](https://github.com/satnaing/shadcn-admin) | React | Reference for layout patterns, component structure | Layout inspiration |
| [nuxt-shadcn-dashboard](https://github.com/dianprata/nuxt-shadcn-dashboard) | Nuxt 4 | Vue implementation reference | Component patterns |
| [shadcn-vue](https://www.shadcn-vue.com/) | Vue 3 | Official Vue port | Component library |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      FRONTEND (Vue 3 + Vite)                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐          │
│  │   Admin     │    │    User     │    │   Public    │          │
│  │  Dashboard  │    │  Dashboard  │    │    Pages    │          │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘          │
│         │                  │                  │                  │
│         └──────────────────┼──────────────────┘                  │
│                            │                                     │
│                   ┌────────▼────────┐                           │
│                   │  Shared Layout  │                           │
│                   │  + Components   │                           │
│                   └────────┬────────┘                           │
│                            │                                     │
│                   ┌────────▼────────┐                           │
│                   │  Auth Guard     │                           │
│                   │  (JWT Middleware)│                           │
│                   └────────┬────────┘                           │
│                            │                                     │
│                   ┌────────▼────────┐                           │
│                   │  API Client     │                           │
│                   │  (Axios + Token)│                           │
│                   └────────┬────────┘                           │
└────────────────────────────┼─────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      BACKEND (Go + Fiber)                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                   JWT Endpoints                          │   │
│  │  POST /api/v1/auth/register                              │   │
│  │  POST /api/v1/auth/login                                 │   │
│  │  POST /api/v1/auth/refresh                               │   │
│  │  POST /api/v1/auth/logout                                │   │
│  │  GET  /api/v1/auth/me                                    │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Protected API Endpoints                      │   │
│  │  /api/v1/sources/*        (JWT required)                  │   │
│  │  /api/v1/snapshots/*      (JWT required)                  │   │
│  │  /api/v1/jobs/*           (JWT required)                  │   │
│  │  /api/v1/schedules/*      (JWT required)                  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
```

---

## Phase 1 Implementation Plan

### Milestone 1: Backend JWT Authentication

**Goal**: Implement JWT-based authentication in the Hub API

**Priority**: 🔴 **HIGH** - Must complete before frontend

| Task | Status | Notes |
|------|--------|-------|
| 1.1 Install JWT dependencies | ⏳ | `github.com/golang-jwt/jwt/v5` |
| 1.2 Add password hashing utility | ⏳ | `golang.org/x/crypto/bcrypt` |
| 1.3 Create auth service layer | ⏳ | Token generation, validation, refresh |
| 1.4 Implement register endpoint | ⏳ | `POST /api/v1/auth/register` |
| 1.5 Implement login endpoint | ⏳ | `POST /api/v1/auth/login` |
| 1.6 Implement refresh endpoint | ⏳ | `POST /api/v1/auth/refresh` |
| 1.7 Implement logout endpoint | ⏳ | `POST /api/v1/auth/logout` (token blacklist) |
| 1.8 Implement /me endpoint | ⏳ | `GET /api/v1/auth/me` |
| 1.9 Create JWT middleware | ⏳ | Fiber middleware for protected routes |
| 1.10 Add refresh token storage | ⏳ | Database table for refresh tokens |
| 1.11 Update users table if needed | ⏳ | Ensure password_hash exists |
| 1.12 Write auth tests | ⏳ | Unit tests for auth service |

**Database Changes**:

```sql
-- Refresh tokens table
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    revoked_at TIMESTAMP,
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at) WHERE revoked_at IS NULL;
```

**JWT Token Structure**:

```go
// Access Token (15 minutes)
type AccessTokenClaims struct {
    UserID    string `json:"user_id"`
    TenantID  string `json:"tenant_id"`
    Email     string `json:"email"`
    Role      string `json:"role"`
    TokenID   string `json:"jti"` // Unique token identifier
    jwt.RegisteredClaims
}

// Refresh Token (7 days)
type RefreshTokenClaims struct {
    UserID    string `json:"user_id"`
    TokenID   string `json:"jti"`
    jwt.RegisteredClaims
}
```

---

### Milestone 2: Frontend Project Setup

**Goal**: Initialize Vue 3 + Vite project with shadcn-vue

| Task | Status | Notes |
|------|--------|-------|
| 2.1 Create frontend directory | ⏳ | `/web` or `/frontend` in monorepo |
| 2.2 Initialize Vite + Vue 3 project | ⏳ | `npm create vite@latest` |
| 2.3 Install TypeScript | ⏳ | Strict mode enabled |
| 2.4 Install Tailwind CSS | ⏳ | With shadcn-vue theme |
| 2.5 Install shadcn-vue CLI | ⏳ | `npx shadcn-vue@latest init` |
| 2.6 Configure shadcn-vue | ⏳ | `components.json` setup |
| 2.7 Install core shadcn components | ⏳ | button, card, input, form, etc. |
| 2.8 Install additional dependencies | ⏳ | Pinia, Vue Router, Axios, VeeValidate |
| 2.9 Configure Vue Router | ⏳ | Route structure setup |
| 2.10 Configure Pinia stores | ⏳ | Auth store, app store |
| 2.11 Setup ESLint + Prettier | ⏳ | Code formatting |
| 2.12 Setup environment variables | ⏳ | `.env` files |

**Project Structure**:

```
/web
  /src
    /assets         # Static assets, images
    /components     # Vue components
      /ui           # shadcn-vue components
      /layout       # Layout components (Header, Sidebar, etc.)
      /auth         # Auth components (LoginForm, RegisterForm)
      /dashboard    # Dashboard components
    /composables    # Vue composables (useAuth, useApi)
    /router        # Vue Router configuration
    /stores         # Pinia stores (auth, sources, snapshots)
    /lib            # Utilities (api client, helpers)
    /types          # TypeScript type definitions
    /views          # Page components
      /auth         # Auth pages (login, register)
      /dashboard    # Dashboard pages
      /admin        # Admin pages
  /public           # Public assets
  index.html
  vite.config.ts
  tailwind.config.js
  tsconfig.json
  package.json
```

---

### Milestone 3: Authentication UI

**Goal**: Build login, registration, and auth flow

| Task | Status | Notes |
|------|--------|-------|
| 3.1 Create auth store (Pinia) | ⏳ | State: user, token, isAuthenticated |
| 3.2 Create API client with Axios | ⏳ | Interceptor for token injection |
| 3.3 Build LoginForm component | ⏳ | Email + password validation |
| 3.4 Build RegisterForm component | ⏳ | Name, email, password, tenant creation |
| 3.5 Create login page | ⏳ | `/auth/login` route |
| 3.6 Create register page | ⏳ | `/auth/register` route |
| 3.7 Implement auth guard | ⏳ | Router guard for protected routes |
| 3.8 Token refresh logic | ⏳ | Auto-refresh on 401 responses |
| 3.9 Logout functionality | ⏳ | Clear state, redirect to login |
| 3.10 Loading states | ⏳ | During API calls |
| 3.11 Error handling | ⏳ | Display errors to user |
| 3.12 Form validation | ⏳ | VeeValidate + Yup schemas |

**Components to Build**:

```vue
<!-- LoginForm.vue -->
<template>
  <Card class="w-full max-w-sm">
    <CardHeader>
      <CardTitle>Sign in to xVault</CardTitle>
      <CardDescription>Enter your credentials to access your account</CardDescription>
    </CardHeader>
    <CardContent>
      <form @submit="handleSubmit">
        <!-- Email input -->
        <!-- Password input -->
        <!-- Remember me checkbox -->
        <!-- Forgot password link -->
      </form>
    </CardContent>
    <CardFooter>
      <!-- Login button -->
      <!-- Sign up link -->
    </CardFooter>
  </Card>
</template>
```

---

### Milestone 4: Admin Dashboard 🔴 **PRIMARY FOCUS**

**Goal**: Build admin dashboard for platform management with full CRUD control

**Access**: Users with `role: admin` only

**Priority**: 🔴 **HIGHEST** - This is the primary focus of Phase 1

**Admin Capabilities Required**:
- ✅ Add new users to the platform
- ✅ Add sources for backups on behalf of users
- ✅ Create schedules for users
- ✅ Set retention policies per source
- ✅ Generate download links for backups
- ✅ View all snapshots across all tenants
- ✅ Monitor workers and system health
- ✅ Manage system settings

| Task | Status | Notes |
|------|--------|-------|
| 4.1 Create admin layout | ⏳ | Sidebar navigation + header |
| 4.2 Build admin dashboard home | ⏳ | Platform stats (tenants, backups, storage) |
| 4.3 Tenants management page | ⏳ | List, create, view, delete tenants |
| 4.4 Users management page | ⏳ | List, create, view, delete users |
| 4.5 System settings page | ⏳ | Download expiration, max sizes, etc. |
| 4.6 Worker monitoring page | ⏳ | List workers, status, last seen |
| 4.7 Audit log page | ⏳ | View audit events |
| 4.8 Data tables | ⏳ | Reusable data table component |
| 4.9 Search/filter | ⏳ | For all list views |
| 4.10 Pagination | ⏳ | For all list views |

**Admin Routes**:

```
/admin
  /dashboard          # Admin home with stats
  /tenants            # Tenant management
    /:id              # Tenant details
  /users              # User management
    /:id              # User details
  /workers            # Worker monitoring
  /settings           # System settings
  /audit              # Audit logs
```

---

### Milestone 5: User Dashboard ⏳ **DEFERRED**

**Goal**: Build simplified user dashboard for backup management

**Access**: All authenticated users

**Status**: ⏳ **DEFERRED** - Will be completed after admin dashboard is fully functional

**Note**: This milestone will be simplified initially. Admin needs to work perfectly before building user-facing features.

| Task | Status | Notes |
|------|--------|-------|
| 5.1 Create user layout | ⏳ | Simplified sidebar + header |
| 5.2 Build user dashboard home | ⏳ | Recent backups, storage used |
| 5.3 Sources list page | ⏳ | All user sources |
| 5.4 Source create page | ⏳ | Form for SSH/SFTP/FTP/DB |
| 5.5 Source detail page | ⏳ | View source, snapshots, jobs |
| 5.6 Snapshots list page | ⏳ | All snapshots across sources |
| 5.7 Snapshot detail page | ⏳ | View manifest, restore button |
| 5.8 Jobs list page | ⏳ | Job history with status |
| 5.9 Schedules management | ⏳ | CRUD schedules with retention |
| 5.10 Retention policy editor | ⏳ | UI for retention rules |
| 5.11 Manual backup trigger | ⏳ | Button to enqueue job |
| 5.12 Restore flow | ⏳ | Trigger restore, show download link |

**User Routes**:

```
/dashboard          # User home
/sources            # All sources
  /new              # Create source
  /:id              # Source details
/snapshots          # All snapshots
  /:id              # Snapshot details
/jobs               # Job history
/schedules          # Backup schedules
  /new              # Create schedule
  /:id              # Edit schedule
/settings           # User settings (profile)
```

---

### Milestone 6: Dark Mode & Theming

**Goal**: Implement dark mode with system preference detection

| Task | Status | Notes |
|------|--------|-------|
| 6.1 Setup theme provider | ⏳ | VueUse `useDark` composable |
| 6.2 Create theme toggle | ⏳ | Button component in header |
| 6.3 Persist theme preference | ⏳ | localStorage |
| 6.4 System preference detection | ⏳ | Respect `prefers-color-scheme` |
| 6.5 Theme transition | ⏳ | Smooth transitions between themes |
| 6.6 Test all components | ⏳ | Ensure all shadcn components work |

**Implementation**:

```vue
<!-- ThemeToggle.vue -->
<script setup lang="ts">
import { useDark, useToggle } from '@vueuse/core'

const isDark = useDark({
  storageKey: 'xvault-theme',
  valueDark: 'dark',
  valueLight: 'light',
})
const toggleTheme = useToggle(isDark)
</script>

<template>
  <Button variant="ghost" size="icon" @click="toggleTheme()">
    <Sun v-if="isDark" class="h-5 w-5" />
    <Moon v-else class="h-5 w-5" />
  </Button>
</template>
```

---

### Milestone 7: Deployment & Integration

**Goal**: Deploy frontend with backend integration

| Task | Status | Notes |
|------|--------|-------|
| 7.1 Build for production | ⏳ | `vite build` |
| 7.2 Configure base URL | ⏳ | API proxy or CORS |
| 7.3 Docker containerization | ⏳ | Multi-stage Dockerfile |
| 7.4 Update docker-compose | ⏳ | Add frontend service |
| 7.5 Nginx config | ⏳ | Serve SPA, proxy API |
| 7.6 Environment variables | ⏳ | Production config |
| 7.7 Testing end-to-end | ⏳ | Full user flows |

**Dockerfile**:

```dockerfile
# Build stage
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Production stage
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

**nginx.conf**:

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    # SPA fallback
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API proxy
    location /api/ {
        proxy_pass http://hub:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

## API Endpoints Summary

### Auth Endpoints (New)

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/v1/auth/register` | Register new user + tenant | No |
| POST | `/api/v1/auth/login` | Login, return access + refresh token | No |
| POST | `/api/v1/auth/refresh` | Refresh access token | No (but requires valid refresh token) |
| POST | `/api/v1/auth/logout` | Logout, blacklist tokens | Yes |
| GET | `/api/v1/auth/me` | Get current user info | Yes |

### Protected Endpoints (Existing)

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/sources` | List sources | Yes (JWT) |
| POST | `/api/v1/sources` | Create source | Yes (JWT) |
| GET | `/api/v1/sources/:id` | Get source | Yes (JWT) |
| PUT | `/api/v1/sources/:id` | Update source | Yes (JWT) |
| DELETE | `/api/v1/sources/:id` | Delete source | Yes (JWT) |
| GET | `/api/v1/snapshots` | List snapshots | Yes (JWT) |
| GET | `/api/v1/snapshots/:id` | Get snapshot | Yes (JWT) |
| POST | `/api/v1/snapshots/:id/restore` | Trigger restore | Yes (JWT) |
| POST | `/api/v1/jobs` | Enqueue job | Yes (JWT) |
| GET | `/api/v1/jobs` | List jobs | Yes (JWT) |
| GET | `/api/v1/schedules` | List schedules | Yes (JWT) |
| POST | `/api/v1/schedules` | Create schedule | Yes (JWT) |
| PUT | `/api/v1/schedules/:id` | Update schedule | Yes (JWT) |

### Admin Endpoints (Admin Role Required)

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/admin/users` | List all users | Yes (JWT + admin role) |
| GET | `/api/v1/admin/users/:id` | Get user by ID | Yes (JWT + admin role) |
| POST | `/api/v1/admin/users` | Create new user + tenant | Yes (JWT + admin role) |
| PUT | `/api/v1/admin/users/:id` | Update user | Yes (JWT + admin role) |
| DELETE | `/api/v1/admin/users/:id` | Delete user | Yes (JWT + admin role) |
| GET | `/api/v1/admin/tenants` | List all tenants | Yes (JWT + admin role) |
| GET | `/api/v1/admin/tenants/:id` | Get tenant by ID | Yes (JWT + admin role) |
| DELETE | `/api/v1/admin/tenants/:id` | Delete tenant with cleanup | Yes (JWT + admin role) |
| GET | `/api/v1/admin/settings` | List settings | Yes (JWT + admin role) |
| PUT | `/api/v1/admin/settings/:key` | Update setting | Yes (JWT + admin role) |
| POST | `/api/v1/admin/retention/run` | Run retention for all sources | Yes (JWT + admin role) |
| POST | `/api/v1/admin/retention/run/:sourceId` | Run retention for source | Yes (JWT + admin role) |

---

## Progress Tracking

### Backend JWT: ✅ 12/12 Complete (Milestone 1 DONE)

### Frontend Setup: ✅ 12/12 Complete (Milestone 2 DONE)

### Auth UI: ✅ 12/12 Complete (Milestone 3 DONE)

### Admin Dashboard: 🚧 7/10 Complete (Milestone 4 IN PROGRESS)

### User Dashboard: ⏸️ 0/12 Complete (DEFERRED)

### Dark Mode: ⏳ 0/6 Complete

### Deployment: ✅ 7/7 Complete (Milestone 7 DONE)

**Overall Progress**: 50/71 tasks (70%)
**Admin Dashboard Progress**: 7/10 tasks (70%) - **IN PROGRESS**

---

## Implementation Summary

### ✅ Completed (Milestones 1-3, 7)

**Backend JWT Authentication (Milestone 1)**:
- Complete auth endpoints: register, login, refresh, logout, /me
- JWT middleware for protected routes
- Refresh token storage and management
- Admin-only endpoints for user/tenant management

**Frontend Project Setup (Milestone 2)**:
- Vue 3 + Vite + TypeScript project in `/web`
- Tailwind CSS v4 configured
- shadcn-vue components (Button, Card, Input, Label, Dialog)
- Pinia stores (auth, admin, sources, snapshots, schedules)
- Vue Router with auth guards
- Axios API client with token interceptors and auto-refresh

**Authentication UI (Milestone 3)**:
- Login and Register pages
- Auth store with login/register/logout/fetchMe
- API client with JWT token injection
- Auto token refresh on 401 responses
- Router guards for protected routes
- Admin role checking

**Deployment (Milestone 7)**:
- Multi-stage Dockerfile ([`deploy/docker/web/Dockerfile`](deploy/docker/web/Dockerfile))
- Nginx config with SPA fallback and API proxy ([`deploy/docker/web/nginx.conf`](deploy/docker/web/nginx.conf))
- Docker Compose updated with frontend service
- Development server with API proxy configured

### 🚧 In Progress (Milestone 4)

**Admin Dashboard - Completed**:
- ✅ Admin layout with sidebar navigation ([`AdminLayout.vue`](web/src/components/layout/AdminLayout.vue))
- ✅ Admin dashboard home with stats ([`DashboardView.vue`](web/src/views/admin/DashboardView.vue))
- ✅ Admin API store with CRUD operations ([`admin.ts`](web/src/stores/admin.ts))
- ✅ Backend admin endpoints for user/tenant management
- ✅ Type definitions for all entities ([`types/index.ts`](web/src/types/index.ts))
- ✅ **Full CRUD forms in admin views** - Create, Edit, Delete dialogs for Users, Sources, Schedules

**Admin Dashboard - Completed (Updated)**:
- ✅ **Tenant deletion with worker cleanup** - Delete tenant cascades to users, sources, schedules; enqueues delete_snapshot jobs for worker storage cleanup
- ✅ **TenantsView with View/Delete dialogs** - View button shows tenant info, Delete button with confirmation
- ✅ **Backend DELETE /admin/tenants/:id endpoint** - Service layer enqueues cleanup jobs before deletion

**Admin Dashboard - Remaining Tasks**:
- ⏳ Worker monitoring page
- ⏳ Audit log page
- ⏳ Data tables with pagination
- ⏳ Enhanced search/filter functionality

### ⏸️ Deferred (Milestone 5)

User dashboard is intentionally deferred until admin dashboard is fully functional.

---

## Quick Reference Commands

### Backend Development
```bash
# Run Hub with auth
go run ./cmd/hub

# Run tests
go test ./internal/hub/...
```

### Frontend Development
```bash
cd web

# Install dependencies
npm install

# Run dev server (with API proxy to Hub on port 8080)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

---

## Notes & Decisions

### Technology Choices

1. **Vue 3 + Vite over Nuxt 4**: Simpler integration with existing monorepo, less opinionated, faster builds

2. **shadcn-vue over component libraries**: Copy-paste components = full control, easier customization, no vendor lock-in

3. **Pinia over Vuex**: Official Vue state management, better TypeScript support

4. **VeeValidate over Vuelidate**: Better Composition API support, more flexible

5. **Axios over Fetch**: Interceptors for auth tokens, better error handling, request cancellation

### Security Considerations

- JWT access tokens: 15 minute expiration
- JWT refresh tokens: 7 day expiration
- Refresh tokens stored in httpOnly cookies (production)
- Access tokens stored in memory (Pinia store)
- CSRF protection via SameSite cookies
- Passwords hashed with bcrypt (cost 12)
- Token blacklist on logout

### UI/UX Principles

- Mobile-first responsive design
- Accessible (WCAG AA compliant)
- Dark mode by default, system preference detection
- Fast loading (code splitting, lazy routes)
- Clear error messages with actionable next steps

---

## References

- [Vue 3 Documentation](https://vuejs.org/)
- [shadcn-vue Documentation](https://www.shadcn-vue.com/)
- [Vite Documentation](https://vitejs.dev/)
- [Pinia Documentation](https://pinia.vuejs.org/)
- [Vue Router Documentation](https://router.vuejs.org/)
- [VeeValidate Documentation](https://vee-validate.logaretm.com/)
- [xVault API Reference](/docs/api-reference.md)
- [xVault Architecture](/docs/architecture.md)
- [xVault Data Model](/docs/data-model.md)
