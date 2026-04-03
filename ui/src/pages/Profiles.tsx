import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useState } from 'react'
import { UserCircle, Plus, Trash2 } from 'lucide-react'

export default function Profiles() {
  const qc = useQueryClient()
  const { data } = useQuery({ queryKey: ['profiles'], queryFn: api.profiles.list, refetchInterval: 10000 })
  const [form, setForm] = useState({ name: '', locale: 'en-US', timezone: 'America/New_York', stealth_level: 'basic' })
  const create = useMutation({ mutationFn: () => api.profiles.create(form), onSuccess: () => { qc.invalidateQueries({ queryKey: ['profiles'] }); setForm({ name: '', locale: 'en-US', timezone: 'America/New_York', stealth_level: 'basic' }) } })
  const del = useMutation({ mutationFn: (id: string) => api.profiles.delete(id), onSuccess: () => qc.invalidateQueries({ queryKey: ['profiles'] }) })

  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold flex items-center gap-2"><UserCircle className="h-6 w-6" />Browser Profiles</h2>
      <Card>
        <CardHeader className="pb-3"><CardTitle className="text-base">Create Profile</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <Input placeholder="Profile name" value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
            <Input placeholder="Locale (en-US)" value={form.locale} onChange={e => setForm(f => ({ ...f, locale: e.target.value }))} />
            <Input placeholder="Timezone" value={form.timezone} onChange={e => setForm(f => ({ ...f, timezone: e.target.value }))} />
            <Input placeholder="Stealth level (basic/advanced)" value={form.stealth_level} onChange={e => setForm(f => ({ ...f, stealth_level: e.target.value }))} />
          </div>
          <Button onClick={() => create.mutate()} disabled={!form.name || create.isPending} size="sm"><Plus className="h-3 w-3 mr-1" />Create</Button>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-3"><CardTitle className="text-base">Profiles ({data?.count || 0})</CardTitle></CardHeader>
        <CardContent>
          {(data?.profiles || []).length === 0 && <p className="text-sm text-muted-foreground">No profiles. Requires SQLite storage.</p>}
          <div className="space-y-2">
            {(data?.profiles || []).map((p: any) => (
              <div key={p.id} className="flex items-center justify-between border rounded-md p-3">
                <div>
                  <span className="font-medium">{p.name}</span>
                  <p className="text-xs text-muted-foreground">{p.locale} · {p.timezone} · {p.stealth_level}</p>
                </div>
                <Button variant="ghost" size="icon" onClick={() => del.mutate(p.id)}><Trash2 className="h-4 w-4 text-destructive" /></Button>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
