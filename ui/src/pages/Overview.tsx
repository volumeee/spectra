import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Activity, CheckCircle, Clock, Monitor, Cpu, MemoryStick, Layers } from 'lucide-react'

export default function Overview() {
  const { data: metrics } = useQuery({ queryKey: ['metrics'], queryFn: api.metrics, refetchInterval: 3000 })
  const { data: pressure } = useQuery({ queryKey: ['pressure'], queryFn: api.pressure, refetchInterval: 3000 })
  const { data: ready } = useQuery({ queryKey: ['ready'], queryFn: api.ready, refetchInterval: 5000 })
  const { data: plugins } = useQuery({ queryKey: ['plugins'], queryFn: api.plugins, refetchInterval: 10000 })
  const { data: jobs } = useQuery({ queryKey: ['jobs'], queryFn: () => api.jobs(10), refetchInterval: 5000 })

  const m = metrics || { total_requests: 0, total_success: 0, total_failed: 0, avg_duration_ms: 0, by_plugin: {} }
  const p = pressure || { cpu_percent: 0, memory_percent: 0, overloaded: false }
  const pool = ready?.browser_pool || { active: 0, idle: 0, total: 0, max: 0 }
  const queue = ready?.queue || { running: 0, pending: 0, completed: 0, failed: 0 }

  const successRate = m.total_requests > 0 ? ((m.total_success / m.total_requests) * 100).toFixed(1) : '0'

  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold">Overview</h2>

      {/* Stats Cards */}
      <div className="grid grid-cols-4 gap-4">
        <StatCard icon={Activity} label="Total Requests" value={m.total_requests} />
        <StatCard icon={CheckCircle} label="Success Rate" value={`${successRate}%`} />
        <StatCard icon={Clock} label="Avg Duration" value={`${Math.round(m.avg_duration_ms || 0)}ms`} />
        <StatCard icon={Monitor} label="Browsers" value={`${pool.active}/${pool.max}`} sub={`${pool.idle} idle`} />
      </div>

      {/* Health */}
      <Card>
        <CardHeader className="pb-3"><CardTitle className="text-base">System Health</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <HealthBar icon={Cpu} label="CPU" value={p.cpu_percent} />
          <HealthBar icon={MemoryStick} label="Memory" value={p.memory_percent} />
          <div className="flex gap-4 text-sm text-muted-foreground">
            <span>Queue: {queue.running} running, {queue.pending} pending</span>
            <span>Pool: {pool.active} active, {pool.idle} idle, {pool.total}/{pool.max} total</span>
          </div>
          {p.overloaded && <Badge variant="destructive">⚠️ System Overloaded</Badge>}
        </CardContent>
      </Card>

      <div className="grid grid-cols-2 gap-4">
        {/* Plugins */}
        <Card>
          <CardHeader className="pb-3"><CardTitle className="text-base flex items-center gap-2"><Layers className="h-4 w-4" />Plugins</CardTitle></CardHeader>
          <CardContent>
            <div className="space-y-2">
              {(plugins?.plugins || []).map((p: any) => (
                <div key={p.manifest?.name} className="flex items-center justify-between text-sm">
                  <span className="font-mono">{p.manifest?.name}</span>
                  <Badge variant={p.status === 'running' ? 'success' : 'secondary'}>{p.status}</Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Recent Jobs */}
        <Card>
          <CardHeader className="pb-3"><CardTitle className="text-base">Recent Jobs</CardTitle></CardHeader>
          <CardContent>
            <div className="space-y-2">
              {(jobs?.jobs || []).slice(0, 8).map((j: any) => (
                <div key={j.id} className="flex items-center justify-between text-sm">
                  <span className="font-mono text-xs truncate w-20">{j.id?.slice(0, 8)}</span>
                  <span className="text-muted-foreground">{j.plugin}/{j.method}</span>
                  <Badge variant={j.status === 'completed' ? 'success' : 'destructive'}>{j.status}</Badge>
                  <span className="text-xs text-muted-foreground">{j.result?.duration_ms}ms</span>
                </div>
              ))}
              {(!jobs?.jobs || jobs.jobs.length === 0) && <p className="text-sm text-muted-foreground">No jobs yet</p>}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function StatCard({ icon: Icon, label, value, sub }: { icon: any; label: string; value: any; sub?: string }) {
  return (
    <Card>
      <CardContent className="p-4">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-md bg-primary/10"><Icon className="h-4 w-4 text-primary" /></div>
          <div>
            <p className="text-2xl font-bold">{value}</p>
            <p className="text-xs text-muted-foreground">{label}{sub && ` · ${sub}`}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function HealthBar({ icon: Icon, label, value }: { icon: any; label: string; value: number }) {
  const pct = Math.min(100, Math.max(0, value))
  const color = pct > 90 ? 'bg-red-500' : pct > 70 ? 'bg-amber-500' : 'bg-emerald-500'
  return (
    <div className="flex items-center gap-3">
      <Icon className="h-4 w-4 text-muted-foreground" />
      <span className="text-sm w-16">{label}</span>
      <div className="flex-1 h-2 rounded-full bg-secondary">
        <div className={`h-full rounded-full transition-all ${color}`} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-sm font-mono w-12 text-right">{pct.toFixed(0)}%</span>
    </div>
  )
}
