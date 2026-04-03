import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Bell, Plus, Trash2 } from 'lucide-react'

export default function Webhooks() {
  const qc = useQueryClient()
  const { data } = useQuery({ queryKey: ['webhooks'], queryFn: api.webhooks.list, refetchInterval: 10000 })
  const [form, setForm] = useState({ event: 'job.completed', target_url: '', secret: '' })
  const create = useMutation({ mutationFn: () => api.webhooks.create(form), onSuccess: () => { qc.invalidateQueries({ queryKey: ['webhooks'] }); setForm(f => ({ ...f, target_url: '', secret: '' })) } })
  const del = useMutation({ mutationFn: (id: string) => api.webhooks.delete(id), onSuccess: () => qc.invalidateQueries({ queryKey: ['webhooks'] }) })

  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold flex items-center gap-2"><Bell className="h-6 w-6" />Webhooks</h2>
      <Card>
        <CardHeader className="pb-3"><CardTitle className="text-base">Create Webhook</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <div className="flex gap-2">
            <select value={form.event} onChange={e => setForm(f => ({ ...f, event: e.target.value }))} className="h-10 rounded-md border bg-background px-2 text-sm">
              <option>job.completed</option><option>job.failed</option><option>plugin.crashed</option>
            </select>
            <Input placeholder="Target URL" value={form.target_url} onChange={e => setForm(f => ({ ...f, target_url: e.target.value }))} />
            <Input placeholder="Secret (optional)" value={form.secret} onChange={e => setForm(f => ({ ...f, secret: e.target.value }))} className="w-40" />
            <Button onClick={() => create.mutate()} disabled={!form.target_url} size="sm"><Plus className="h-3 w-3 mr-1" />Create</Button>
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4 space-y-2">
          {(Array.isArray(data) ? data : []).length === 0 && <p className="text-sm text-muted-foreground">No webhooks. Enable webhooks in config first.</p>}
          {(Array.isArray(data) ? data : []).map((w: any) => (
            <div key={w.id} className="flex items-center justify-between border rounded-md p-3 text-sm">
              <div><span className="font-mono">{w.event}</span> → <span className="text-muted-foreground">{w.target_url}</span></div>
              <Button variant="ghost" size="icon" onClick={() => del.mutate(w.id)}><Trash2 className="h-4 w-4 text-destructive" /></Button>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
