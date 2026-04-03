import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { useState } from 'react'
import { Monitor, Plus, Trash2, ExternalLink } from 'lucide-react'

export default function Sessions() {
  const qc = useQueryClient()
  const { data } = useQuery({ queryKey: ['sessions'], queryFn: api.sessions.list, refetchInterval: 5000 })
  const [ttl, setTtl] = useState('3600')
  const create = useMutation({ mutationFn: () => api.sessions.create(parseInt(ttl)), onSuccess: () => qc.invalidateQueries({ queryKey: ['sessions'] }) })
  const del = useMutation({ mutationFn: (id: string) => api.sessions.delete(id), onSuccess: () => qc.invalidateQueries({ queryKey: ['sessions'] }) })

  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold flex items-center gap-2"><Monitor className="h-6 w-6" />Sessions</h2>
      <Card>
        <CardHeader className="pb-3"><CardTitle className="text-base">Create Session</CardTitle></CardHeader>
        <CardContent className="flex gap-2">
          <Input placeholder="TTL (seconds)" value={ttl} onChange={e => setTtl(e.target.value)} className="w-40" />
          <Button onClick={() => create.mutate()} disabled={create.isPending} size="sm"><Plus className="h-3 w-3 mr-1" />Create</Button>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-3"><CardTitle className="text-base">Active Sessions ({data?.count || 0})</CardTitle></CardHeader>
        <CardContent>
          {(data?.sessions || []).length === 0 && <p className="text-sm text-muted-foreground">No active sessions. Create one or enable SQLite storage.</p>}
          <div className="space-y-2">
            {(data?.sessions || []).map((s: any) => (
              <div key={s.id} className="flex items-center justify-between border rounded-md p-3">
                <div>
                  <span className="font-mono text-sm">{s.id?.slice(0, 12)}...</span>
                  <p className="text-xs text-muted-foreground">{s.url || 'No URL'} · {s.profile_id || 'No profile'}</p>
                </div>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" onClick={() => window.open(`/remote?session=${s.id}`, '_blank')}>
                    <ExternalLink className="h-3 w-3 mr-1" />Open Remote
                  </Button>
                  <Button variant="ghost" size="icon" onClick={() => del.mutate(s.id)}><Trash2 className="h-4 w-4 text-destructive" /></Button>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
