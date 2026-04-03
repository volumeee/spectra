const BASE = ''

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    ...init,
  })
  return res.json()
}

// Unwrap Spectra's {"success":true,"data":...} envelope
async function unwrap<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await request<{ success: boolean; data: T }>(path, init)
  return (res as any)?.data ?? res as T
}

export const api = {
  health: () => request<{ status: string }>('/health'),
  ready: () => request<any>('/ready'),
  pressure: () => request<{ cpu_percent: number; memory_percent: number; overloaded: boolean }>('/pressure'),
  plugins: () => unwrap<{ plugins: any[]; queue: any }>('/api/plugins'),
  metrics: () => request<any>('/api/metrics'),
  jobs: (limit = 50) => request<{ jobs: any[]; count: number }>(`/api/jobs?limit=${limit}`),
  sessions: {
    list: () => request<{ sessions: any[]; count: number }>('/api/sessions'),
    create: (ttl = 3600, profileId = '') => request<any>('/api/sessions', { method: 'POST', body: JSON.stringify({ ttl_seconds: ttl, profile_id: profileId }) }),
    delete: (id: string) => request<any>(`/api/sessions/${id}`, { method: 'DELETE' }),
  },
  profiles: {
    list: () => request<{ profiles: any[]; count: number }>('/api/profiles'),
    create: (data: any) => request<any>('/api/profiles', { method: 'POST', body: JSON.stringify(data) }),
    delete: (id: string) => request<any>(`/api/profiles/${id}`, { method: 'DELETE' }),
  },
  webhooks: {
    list: () => unwrap<any[]>('/api/webhooks'),
    create: (data: any) => unwrap<any>('/api/webhooks', { method: 'POST', body: JSON.stringify(data) }),
    delete: (id: string) => unwrap<any>(`/api/webhooks/${id}`, { method: 'DELETE' }),
  },
  schedules: {
    list: () => unwrap<any[]>('/api/schedules'),
    create: (data: any) => unwrap<any>('/api/schedules', { method: 'POST', body: JSON.stringify(data) }),
    delete: (id: string) => unwrap<any>(`/api/schedules/${id}`, { method: 'DELETE' }),
  },
  execute: (plugin: string, method: string, params: any) =>
    unwrap<any>(`/api/${plugin}/${method}`, { method: 'POST', body: JSON.stringify(params) }),
  query: (steps: any[]) =>
    unwrap<any>('/api/query', { method: 'POST', body: JSON.stringify({ steps }) }),
}
