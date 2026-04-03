import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Settings as SettingsIcon, Globe, Monitor, Puzzle, Database, Shield, Activity } from 'lucide-react'

export default function SettingsPage() {
  const { data: ready } = useQuery({ queryKey: ['ready'], queryFn: api.ready })
  const { data: pressure } = useQuery({ queryKey: ['pressure'], queryFn: api.pressure })

  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold flex items-center gap-2"><SettingsIcon className="h-6 w-6" />Settings</h2>
      <p className="text-sm text-muted-foreground">Server configuration is read from <code className="bg-muted px-1 rounded">spectra.yaml</code>. Changes require server restart.</p>
      <Tabs defaultValue="status">
        <TabsList>
          <TabsTrigger value="status"><Activity className="h-3 w-3 mr-1" />Status</TabsTrigger>
          <TabsTrigger value="browser"><Monitor className="h-3 w-3 mr-1" />Browser</TabsTrigger>
          <TabsTrigger value="plugins"><Puzzle className="h-3 w-3 mr-1" />Plugins</TabsTrigger>
          <TabsTrigger value="storage"><Database className="h-3 w-3 mr-1" />Storage</TabsTrigger>
        </TabsList>
        <TabsContent value="status">
          <Card><CardContent className="p-4 space-y-2">
            <Row label="Server" value="Running" />
            <Row label="CPU" value={`${pressure?.cpu_percent?.toFixed(1) || 0}%`} />
            <Row label="Memory" value={`${pressure?.memory_percent?.toFixed(1) || 0}%`} />
            <Row label="Overloaded" value={pressure?.overloaded ? '⚠️ Yes' : '✅ No'} />
            <Row label="Plugins" value={`${ready?.plugins || 0} loaded`} />
            <Row label="Browser Pool" value={`${ready?.browser_pool?.active || 0} active / ${ready?.browser_pool?.max || 0} max`} />
            <Row label="Queue" value={`${ready?.queue?.running || 0} running, ${ready?.queue?.pending || 0} pending`} />
          </CardContent></Card>
        </TabsContent>
        <TabsContent value="browser">
          <Card><CardContent className="p-4 space-y-2 text-sm">
            <p className="text-muted-foreground">Configure in <code>spectra.yaml</code> → <code>browser:</code></p>
            <Row label="max_instances" value={`${ready?.browser_pool?.max || 5}`} />
            <Row label="share_pool" value="true (recommended)" />
            <Row label="warm_pool_size" value="2" />
            <Row label="recycle_after" value="50 uses" />
            <Row label="Default viewport" value="1920 × 1080" />
          </CardContent></Card>
        </TabsContent>
        <TabsContent value="plugins">
          <Card><CardContent className="p-4 space-y-2 text-sm">
            <p className="text-muted-foreground">Configure in <code>spectra.yaml</code> → <code>plugins:</code></p>
            <Row label="dir" value="./bin/plugins" />
            <Row label="pool_size" value="3 processes per plugin" />
            <Row label="call_timeout" value="60s" />
          </CardContent></Card>
        </TabsContent>
        <TabsContent value="storage">
          <Card><CardContent className="p-4 space-y-2 text-sm">
            <p className="text-muted-foreground">Configure in <code>spectra.yaml</code> → <code>storage:</code></p>
            <Row label="driver" value="sqlite (recommended) or memory" />
            <Row label="sqlite_path" value="./spectra.db" />
            <p className="text-xs text-muted-foreground mt-2">SQLite enables: sessions, profiles, job history, webhook/schedule persistence</p>
          </CardContent></Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return <div className="flex justify-between py-1 border-b last:border-0"><span className="text-muted-foreground">{label}</span><span className="font-mono">{value}</span></div>
}
