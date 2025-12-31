// User types
export interface User {
  id: string
  tenant_id: string
  email: string
  role: 'owner' | 'admin' | 'member'
  created_at: string
  updated_at: string
}

// Admin User Management types
export interface AdminCreateUserRequest {
  email: string
  password: string
  name: string
  role?: 'owner' | 'admin' | 'member'
}

export interface AdminUpdateUserRequest {
  email?: string
  role?: 'owner' | 'admin' | 'member'
}

// Tenant types
export interface Tenant {
  id: string
  name: string
  plan?: string
  created_at: string
  updated_at: string
}

// Auth types
export interface RegisterRequest {
  name: string
  email: string
  password: string
}

export interface RegisterResponse {
  user: User
  tenant: Tenant
  access_token: string
  refresh_token: string
  expires_at: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  user: User
  tenant: Tenant
  access_token: string
  refresh_token: string
  expires_at: string
}

export interface RefreshRequest {
  refresh_token: string
}

export interface RefreshResponse {
  access_token: string
  refresh_token: string
  expires_at: string
}

export interface AuthResponse {
  user_id: string
  tenant_id: string
  email: string
  role: string
}

// Source types
export type SourceType = 'ssh' | 'sftp' | 'ftp' | 'mysql' | 'postgresql'

export interface SourceConfig {
  // SSH/SFTP/FTP config
  host?: string
  port?: number
  username?: string
  paths?: string[]
  use_password?: boolean
  // Database config
  database?: string
  tables?: string[]  // MySQL
  schemas?: string[] // PostgreSQL
}

export interface Source {
  id: string
  tenant_id: string
  name: string
  type: SourceType
  status: 'active' | 'disabled'
  config: SourceConfig
  credential_id: string
  created_at: string
  updated_at: string
}

export interface CreateSourceRequest {
  name: string
  type: SourceType
  config: SourceConfig
}

// Admin Source Management types
export interface AdminCreateSourceRequest {
  tenant_id: string
  type: SourceType
  name: string
  config: SourceConfig
  credential: string // Base64-encoded password or private key
}

export interface AdminUpdateSourceRequest {
  name?: string
  status?: 'active' | 'disabled'
  config?: SourceConfig
  credential?: string // Base64-encoded new credential (for rotation)
}

// Test Connection types
export interface TestConnectionRequest {
  type: SourceType
  host: string
  port: number
  username: string
  credential: string // Base64-encoded password or private key
  use_private_key: boolean
  database?: string // For mysql/postgresql
}

export interface TestConnectionResult {
  success: boolean
  message: string
  details?: string
}

// Snapshot types
export interface Snapshot {
  id: string
  tenant_id: string
  source_id: string
  status: 'pending' | 'completed' | 'failed'
  size_bytes: number
  file_count: number
  storage_backend: string
  worker_id: string
  location: string
  created_at: string
  completed_at: string | null
  manifest: Record<string, unknown> | null
}

// Admin Snapshot type (includes tenant/source info)
export interface AdminSnapshot {
  id: string
  tenant_id: string
  tenant_name: string
  source_id: string
  source_name: string
  source_type: string
  job_id?: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  size_bytes: number
  started_at?: string
  finished_at?: string
  duration_ms?: number
  storage_backend: string
  worker_id?: string
  download_token?: string
  download_expires_at?: string
  download_url?: string
  created_at: string
  updated_at: string
}

// Job types
export type JobStatus = 'pending' | 'claimed' | 'running' | 'completed' | 'failed'
export type JobType = 'backup' | 'restore' | 'delete' | 'retention_eval'

export interface Job {
  id: string
  tenant_id: string
  source_id: string | null
  type: JobType
  status: JobStatus
  target_worker_id: string | null
  lease_expires_at: string | null
  created_at: string
  started_at: string | null
  completed_at: string | null
  error: string | null
  result: Record<string, unknown> | null
}

export interface EnqueueJobRequest {
  source_id: string
}

// Schedule types
export interface Schedule {
  id: string
  tenant_id: string
  source_id: string
  cron?: string | null
  interval_minutes?: number | null
  timezone: string
  status: 'enabled' | 'disabled'
  retention_policy: {
    mode: 'all' | 'latest_n' | 'within_duration'
    keep_last_n?: number
    keep_within_duration?: string
  }
  last_run_at?: string | null
  next_run_at?: string | null
  created_at: string
  updated_at: string
}

export interface CreateScheduleRequest {
  source_id: string
  schedule: string
  enabled?: boolean
  retention_policy_id?: string
}

export interface UpdateScheduleRequest {
  schedule?: string
  enabled?: boolean
  retention_policy_id?: string
}

// Admin Schedule Management types
export interface AdminCreateScheduleRequest {
  source_id: string
  cron?: string
  interval_minutes?: number
  timezone?: string
  retention_policy?: {
    mode: 'all' | 'latest_n' | 'within_duration'
    keep_last_n?: number
    keep_within_duration?: string
  }
}

export interface AdminUpdateScheduleRequest {
  cron?: string
  interval_minutes?: number
  timezone?: string
  status?: 'enabled' | 'disabled'
  retention_policy?: {
    mode: 'all' | 'latest_n' | 'within_duration'
    keep_last_n?: number
    keep_within_duration?: string
  }
}

// Retention Policy types
export type RetentionMode = 'all' | 'latest_n' | 'within_duration'

export interface RetentionPolicy {
  id: string
  tenant_id: string
  name: string
  mode: RetentionMode
  keep_last_n: number | null
  keep_within_duration: string | null
  created_at: string
  updated_at: string
}

export interface CreateRetentionPolicyRequest {
  name: string
  mode: RetentionMode
  keep_last_n?: number
  keep_within_duration?: string
}

export interface UpdateRetentionPolicyRequest {
  name?: string
  mode?: RetentionMode
  keep_last_n?: number
  keep_within_duration?: string
}

// Settings types
export interface Setting {
  key: string
  value: string
  description: string
}

export interface UpdateSettingRequest {
  value: string
}

// Worker types
export interface Worker {
  id: string
  status: 'active' | 'inactive'
  last_heartbeat_at: string | null
  capabilities: string[]
}

// Restore Job types
export interface RestoreJob {
  id: string
  snapshot_id: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  download_token: string | null
  download_expires_at: string | null
  download_url: string | null
  created_at: string
  completed_at: string | null
  error: string | null
}
