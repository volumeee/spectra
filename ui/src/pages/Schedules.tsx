import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Clock, Plus, Trash2 } from 'lucide-react'

export default function Schedules() {
  const qc = useQueryClient()
  const { data } = useQuery({ queryKey: ['schedules'], queryFn: api.schedules.list, refetchInterval: 10000 })
  const [form, setForm] = useState({ cron: '0 * * * *', plugin: 'screenshot', method: 'capture', params: '{"url":"https://example.com"}' })
  const create = useMutation({ mutationFn: () => api.schedules.create({ ...form, params: JSON.parse(form.params) }), onSuccess: () => qc.invalidateQueries({ queryKey: ['schedules'] }) })
  const del = useMutation({ mutationFn: (id: string) => api.schedules.delete(id), onSuccess: () => qc.invalidateQueries({ queryKey: ['schedules'] }) })

  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold flex items-center gap-2"><Clock className="h-6 w-6" />Schedules</h2>
      <Card>
        <CardHeader className="pb-3"><CardTitle className="text-base">Create Schedule</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <div className="grid grid-cols-3 gap-2">
            <Input placeholder="Cron (0 * * * *)" value={form.cron} onChange={e => setForm(f => ({ ...f, cron: e.target.value }))} />
            <Input placeholder="Plugin" value={form.plugin} onChange={e => setForm(f => ({ ...f, plugin: e.target.value }))} />
            <Input placeholder="Method" value={form.method} onChange={e => setForm(f => ({ ...f, method: e.target.value }))} />
          </div>
          <textarea value={form.params} onChange={e => setForm(f => ({ ...f, params: e.target.value }))} className="w-full h-20 rounded-md border bg-background p-2 font-mono text-sm" />
          <Button onClick={() => create.mutate()} size="sm"><Plus className="h-3 w-3 mr-1" />Create</Button>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4 space-y-2">
          {(Array.isArray(data) ? data : []).length === 0 && <p className="text-sm text-muted-foreground">No schedules. Enable scheduler in config first.</p>}
          {(Array.isArray(data) ? data : []).map((s: any) => (
            <div key={s.id} className="flex items-center justify-between border rounded-md p-3 text-sm">
              <div>
                <span className="font-mono">{s.cron}</span> → <span>{s.plugin}/{s.method}</span>
                {s.next_run && <p className="text-xs text-muted-foreground">Next: {s.next_run}</p>}
              </div>
              <Button variant="ghost" size="icon" onClick={() => del.mutate(s.id)}><Trash2 className="h-4 w-4 text-destructive" /></Button>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
