import { useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Play, Puzzle } from 'lucide-react'

export default function Plugins() {
  const { data: plugins } = useQuery({ queryKey: ['plugins'], queryFn: api.plugins, refetchInterval: 5000 })
  const [selected, setSelected] = useState<string | null>(null)

  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold flex items-center gap-2"><Puzzle className="h-6 w-6" />Plugins</h2>

      <div className="grid grid-cols-3 gap-3">
        {(plugins?.plugins || []).map((p: any) => (
          <Card key={p.manifest?.name} className={`cursor-pointer transition-colors ${selected === p.manifest?.name ? 'ring-2 ring-primary' : 'hover:bg-accent/50'}`}
            onClick={() => setSelected(p.manifest?.name)}>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <span className="font-mono font-medium">{p.manifest?.name}</span>
                <Badge variant={p.status === 'running' ? 'success' : 'secondary'}>{p.status}</Badge>
              </div>
              <p className="text-xs text-muted-foreground mt-1">v{p.manifest?.version} · {(p.manifest?.methods || []).join(', ') || 'auto'}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      {selected && <Playground plugin={selected} />}
    </div>
  )
}

function Playground({ plugin }: { plugin: string }) {
  const [method, setMethod] = useState('')
  const [params, setParams] = useState('{\n  "url": "https://example.com"\n}')
  const [result, setResult] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: () => api.execute(plugin, method, JSON.parse(params)),
    onSuccess: (data) => setResult(JSON.stringify(data, null, 2)),
    onError: (err: any) => setResult(`Error: ${err.message}`),
  })

  return (
    <Card>
      <CardHeader className="pb-3"><CardTitle className="text-base">Playground — {plugin}</CardTitle></CardHeader>
      <CardContent className="space-y-3">
        <div className="flex gap-2">
          <Input placeholder="method (e.g. capture)" value={method} onChange={e => setMethod(e.target.value)} className="w-48" />
          <Button onClick={() => mutation.mutate()} disabled={!method || mutation.isPending} size="sm">
            <Play className="h-3 w-3 mr-1" />{mutation.isPending ? 'Running...' : 'Execute'}
          </Button>
        </div>
        <textarea value={params} onChange={e => setParams(e.target.value)}
          className="w-full h-32 rounded-md border bg-background p-3 font-mono text-sm" />
        {result && (
          <pre className="w-full max-h-64 overflow-auto rounded-md bg-muted p-3 font-mono text-xs">{result}</pre>
        )}
      </CardContent>
    </Card>
  )
}
