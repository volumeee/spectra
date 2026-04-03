import { useState } from 'react'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Link2, Play, Plus, Trash2 } from 'lucide-react'

export default function SpectraQL() {
  const [steps, setSteps] = useState([{ action: 'goto', url: 'https://example.com', selector: '', value: '' }])
  const [result, setResult] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const addStep = () => setSteps(s => [...s, { action: 'screenshot', url: '', selector: '', value: '' }])
  const removeStep = (i: number) => setSteps(s => s.filter((_, idx) => idx !== i))
  const updateStep = (i: number, k: string, v: string) => setSteps(s => s.map((step, idx) => idx === i ? { ...step, [k]: v } : step))

  const run = async () => {
    setLoading(true)
    try {
      const mapped = steps.map(s => {
        const step: any = { action: s.action }
        if (s.url) step.url = s.url
        if (s.selector) step.selector = s.selector
        if (s.value) step.value = s.value
        return step
      })
      const res = await api.query(mapped)
      setResult(JSON.stringify(res, null, 2))
    } catch (e: any) { setResult(`Error: ${e.message}`) }
    setLoading(false)
  }

  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold flex items-center gap-2"><Link2 className="h-6 w-6" />SpectraQL</h2>
      <Card>
        <CardHeader className="pb-3"><CardTitle className="text-base">Query Builder</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          {steps.map((s, i) => (
            <div key={i} className="flex gap-2 items-center">
              <span className="text-xs text-muted-foreground w-6">{i + 1}</span>
              <select value={s.action} onChange={e => updateStep(i, 'action', e.target.value)}
                className="h-9 rounded-md border bg-background px-2 text-sm">
                {['goto', 'click', 'type', 'scroll', 'wait_for', 'screenshot', 'evaluate_js', 'extract'].map(a => <option key={a}>{a}</option>)}
              </select>
              {s.action === 'goto' && <input placeholder="URL" value={s.url} onChange={e => updateStep(i, 'url', e.target.value)} className="flex-1 h-9 rounded-md border bg-background px-2 text-sm" />}
              {['click', 'wait_for', 'type'].includes(s.action) && <input placeholder="selector" value={s.selector} onChange={e => updateStep(i, 'selector', e.target.value)} className="flex-1 h-9 rounded-md border bg-background px-2 text-sm" />}
              {['type', 'evaluate_js'].includes(s.action) && <input placeholder="value" value={s.value} onChange={e => updateStep(i, 'value', e.target.value)} className="flex-1 h-9 rounded-md border bg-background px-2 text-sm" />}
              <Button variant="ghost" size="icon" onClick={() => removeStep(i)}><Trash2 className="h-3 w-3" /></Button>
            </div>
          ))}
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={addStep}><Plus className="h-3 w-3 mr-1" />Add Step</Button>
            <Button size="sm" onClick={run} disabled={loading}><Play className="h-3 w-3 mr-1" />{loading ? 'Running...' : 'Execute'}</Button>
          </div>
        </CardContent>
      </Card>
      {result && <Card><CardContent className="p-4"><pre className="max-h-96 overflow-auto font-mono text-xs">{result}</pre></CardContent></Card>}
    </div>
  )
}
