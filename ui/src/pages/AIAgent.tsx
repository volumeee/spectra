import { useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Bot, Send, Settings, History } from 'lucide-react'

export default function AIAgent() {
  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold flex items-center gap-2"><Bot className="h-6 w-6" />AI Agent</h2>
      <Tabs defaultValue="chat">
        <TabsList>
          <TabsTrigger value="chat"><Send className="h-3 w-3 mr-1" />Chat</TabsTrigger>
          <TabsTrigger value="settings"><Settings className="h-3 w-3 mr-1" />Settings</TabsTrigger>
          <TabsTrigger value="history"><History className="h-3 w-3 mr-1" />History</TabsTrigger>
        </TabsList>
        <TabsContent value="chat"><AIChat /></TabsContent>
        <TabsContent value="settings"><AISettings /></TabsContent>
        <TabsContent value="history"><AIHistory /></TabsContent>
      </Tabs>
    </div>
  )
}

function AIChat() {
  const [task, setTask] = useState('')
  const [messages, setMessages] = useState<{ role: string; content: string }[]>([])
  const [aiConfig, setAiConfig] = useState(() => {
    try { return JSON.parse(localStorage.getItem('spectra-ai-config') || '{}') } catch { return {} }
  })

  const mutation = useMutation({
    mutationFn: (t: string) => api.execute('ai', 'execute', {
      task: t,
      openai_api_key: aiConfig.apiKey || '',
      base_url: aiConfig.baseUrl || undefined,
      model: aiConfig.model || 'gpt-4o',
      max_steps: aiConfig.maxSteps || 20,
      config: { planning: true, self_correction: true, memory: true },
    }),
    onSuccess: (data) => {
      const d = data?.data || data
      setMessages(prev => [...prev, { role: 'assistant', content: JSON.stringify(d, null, 2) }])
    },
    onError: (err: any) => {
      setMessages(prev => [...prev, { role: 'assistant', content: `Error: ${err.message}` }])
    },
  })

  const send = () => {
    if (!task.trim()) return
    setMessages(prev => [...prev, { role: 'user', content: task }])
    mutation.mutate(task)
    setTask('')
  }

  return (
    <Card>
      <CardContent className="p-4 space-y-3">
        {!aiConfig.apiKey && (
          <div className="rounded-md bg-amber-500/10 border border-amber-500/20 p-3 text-sm text-amber-500">
            ⚠️ Set your API key in the Settings tab first
          </div>
        )}
        <div className="h-96 overflow-y-auto space-y-3 border rounded-md p-3">
          {messages.length === 0 && <p className="text-sm text-muted-foreground">Ask the AI agent to do something...</p>}
          {messages.map((m, i) => (
            <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div className={`max-w-[80%] rounded-lg px-3 py-2 text-sm ${m.role === 'user' ? 'bg-primary text-primary-foreground' : 'bg-muted'}`}>
                <pre className="whitespace-pre-wrap font-mono text-xs">{m.content}</pre>
              </div>
            </div>
          ))}
          {mutation.isPending && <div className="flex justify-start"><div className="bg-muted rounded-lg px-3 py-2 text-sm animate-pulse">🤖 Working...</div></div>}
        </div>
        <div className="flex gap-2">
          <Input placeholder="e.g. Go to HN and find top 3 stories..." value={task} onChange={e => setTask(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && send()} disabled={mutation.isPending} />
          <Button onClick={send} disabled={mutation.isPending || !task.trim()}><Send className="h-4 w-4" /></Button>
        </div>
      </CardContent>
    </Card>
  )
}

function AISettings() {
  const [config, setConfig] = useState(() => {
    try { return JSON.parse(localStorage.getItem('spectra-ai-config') || '{}') } catch { return {} }
  })
  const save = () => { localStorage.setItem('spectra-ai-config', JSON.stringify(config)) }
  const update = (k: string, v: any) => setConfig((c: any) => ({ ...c, [k]: v }))

  return (
    <Card>
      <CardContent className="p-4 space-y-4">
        <div className="space-y-2">
          <label className="text-sm font-medium">API Key</label>
          <Input type="password" placeholder="sk-..." value={config.apiKey || ''} onChange={e => update('apiKey', e.target.value)} />
        </div>
        <div className="space-y-2">
          <label className="text-sm font-medium">Base URL (optional, for Ollama/custom)</label>
          <Input placeholder="https://api.openai.com/v1" value={config.baseUrl || ''} onChange={e => update('baseUrl', e.target.value)} />
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Model</label>
            <Input placeholder="gpt-4o" value={config.model || ''} onChange={e => update('model', e.target.value)} />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">Max Steps</label>
            <Input type="number" placeholder="20" value={config.maxSteps || ''} onChange={e => update('maxSteps', parseInt(e.target.value))} />
          </div>
        </div>
        <Button onClick={save}>Save Settings</Button>
        <p className="text-xs text-muted-foreground">Settings saved in browser localStorage</p>
      </CardContent>
    </Card>
  )
}

function AIHistory() {
  const { data: jobs } = useQuery({ queryKey: ['ai-jobs'], queryFn: () => api.jobs(50), refetchInterval: 5000 })
  const aiJobs = (jobs?.jobs || []).filter((j: any) => j.plugin === 'ai')

  return (
    <Card>
      <CardContent className="p-4">
        <div className="space-y-2">
          {aiJobs.length === 0 && <p className="text-sm text-muted-foreground">No AI jobs yet</p>}
          {aiJobs.map((j: any) => (
            <div key={j.id} className="flex items-center justify-between text-sm border-b pb-2">
              <span className="font-mono text-xs">{j.id?.slice(0, 8)}</span>
              <span>{j.method}</span>
              <Badge variant={j.status === 'completed' ? 'success' : 'destructive'}>{j.status}</Badge>
              <span className="text-xs text-muted-foreground">{j.result?.duration_ms}ms</span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
