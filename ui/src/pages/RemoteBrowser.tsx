import { useState, useRef, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Eye, MousePointer, Bot, Send, Camera, FileText, ArrowRight } from 'lucide-react'

export default function RemoteBrowser() {
  const [url, setUrl] = useState('https://example.com')
  const [mode, setMode] = useState<'view' | 'interactive' | 'ai'>('interactive')
  const [connected, setConnected] = useState(false)
  const [aiTask, setAiTask] = useState('')
  const [aiLog, setAiLog] = useState<string[]>([])
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const wsRef = useRef<WebSocket | null>(null)

  // Connect to live view WebSocket
  const connect = useCallback((sessionId: string) => {
    const ws = new WebSocket(`ws://${location.host}/api/sessions/${sessionId}/live`)
    ws.onopen = () => setConnected(true)
    ws.onclose = () => setConnected(false)
    ws.onmessage = (e) => {
      const data = JSON.parse(e.data)
      if (data.type === 'frame' && canvasRef.current) {
        const img = new Image()
        img.onload = () => {
          const ctx = canvasRef.current?.getContext('2d')
          if (ctx) { ctx.drawImage(img, 0, 0, canvasRef.current!.width, canvasRef.current!.height) }
        }
        img.src = `data:image/png;base64,${data.data}`
      }
    }
    wsRef.current = ws
  }, [])

  // Navigate via API
  const navigate = async () => {
    try {
      await fetch('/api/recorder/record', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url, steps: [], output_mode: 'frames' }),
      })
    } catch {}
  }

  // Mouse click handler for interactive mode
  const handleCanvasClick = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (mode !== 'interactive' || !canvasRef.current) return
    const rect = canvasRef.current.getBoundingClientRect()
    const x = Math.round((e.clientX - rect.left) / rect.width * 1920)
    const y = Math.round((e.clientY - rect.top) / rect.height * 1080)
    // Send click coordinates to backend (future: CDP Input.dispatchMouseEvent)
    console.log(`Click at ${x}, ${y}`)
  }

  // AI agent
  const runAI = async () => {
    if (!aiTask.trim()) return
    setAiLog(l => [...l, `> ${aiTask}`])
    const config = JSON.parse(localStorage.getItem('spectra-ai-config') || '{}')
    try {
      const res = await fetch('/api/ai/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          task: aiTask, url, openai_api_key: config.apiKey || '',
          model: config.model || 'gpt-4o', max_steps: 10,
          config: { planning: true, self_correction: true, memory: true },
        }),
      })
      const data = await res.json()
      const d = data?.data || data
      if (d?.action_log) {
        d.action_log.forEach((a: any) => setAiLog(l => [...l, `${a.success ? '✅' : '❌'} ${a.type}: ${a.description || ''}`]))
      }
      if (d?.result) setAiLog(l => [...l, `🎯 Result: ${d.result}`])
    } catch (e: any) { setAiLog(l => [...l, `❌ Error: ${e.message}`]) }
    setAiTask('')
  }

  return (
    <div className="h-screen flex flex-col bg-background">
      {/* Top bar */}
      <div className="border-b p-3 flex items-center gap-3">
        <span className="text-lg font-bold">🔮</span>
        <div className="flex-1 flex gap-2">
          <Input value={url} onChange={e => setUrl(e.target.value)} onKeyDown={e => e.key === 'Enter' && navigate()}
            className="max-w-lg" placeholder="https://..." />
          <Button size="sm" onClick={navigate}><ArrowRight className="h-4 w-4" /></Button>
        </div>
        <div className="flex gap-1">
          <Button variant={mode === 'view' ? 'default' : 'ghost'} size="sm" onClick={() => setMode('view')}><Eye className="h-3 w-3 mr-1" />View</Button>
          <Button variant={mode === 'interactive' ? 'default' : 'ghost'} size="sm" onClick={() => setMode('interactive')}><MousePointer className="h-3 w-3 mr-1" />Interactive</Button>
          <Button variant={mode === 'ai' ? 'default' : 'ghost'} size="sm" onClick={() => setMode('ai')}><Bot className="h-3 w-3 mr-1" />AI</Button>
        </div>
        <Badge variant={connected ? 'success' : 'secondary'}>{connected ? 'Connected' : 'Disconnected'}</Badge>
      </div>

      {/* Main area */}
      <div className="flex-1 flex">
        {/* Browser canvas */}
        <div className="flex-1 bg-black flex items-center justify-center p-2">
          <canvas ref={canvasRef} width={1920} height={1080} onClick={handleCanvasClick}
            className={`max-w-full max-h-full bg-zinc-900 rounded ${mode === 'interactive' ? 'cursor-crosshair' : 'cursor-default'}`}
            style={{ aspectRatio: '16/9' }} />
        </div>

        {/* AI panel (collapsible) */}
        {mode === 'ai' && (
          <div className="w-80 border-l flex flex-col">
            <div className="p-3 border-b font-medium text-sm flex items-center gap-2"><Bot className="h-4 w-4" />AI Agent</div>
            <div className="flex-1 overflow-y-auto p-3 space-y-1">
              {aiLog.length === 0 && <p className="text-xs text-muted-foreground">Tell the AI what to do...</p>}
              {aiLog.map((l, i) => <p key={i} className="text-xs font-mono">{l}</p>)}
            </div>
            <div className="p-3 border-t flex gap-2">
              <Input value={aiTask} onChange={e => setAiTask(e.target.value)} onKeyDown={e => e.key === 'Enter' && runAI()}
                placeholder="e.g. click login..." className="text-sm" />
              <Button size="icon" onClick={runAI}><Send className="h-4 w-4" /></Button>
            </div>
          </div>
        )}
      </div>

      {/* Bottom bar */}
      <div className="border-t p-2 flex gap-2">
        <Button variant="outline" size="sm" onClick={() => fetch('/api/screenshot/capture', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ url }) })}>
          <Camera className="h-3 w-3 mr-1" />Screenshot
        </Button>
        <Button variant="outline" size="sm" onClick={() => fetch('/api/pdf/generate', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ url }) })}>
          <FileText className="h-3 w-3 mr-1" />PDF
        </Button>
        <div className="flex-1" />
        <span className="text-xs text-muted-foreground self-center">1920×1080 · {mode} mode</span>
      </div>
    </div>
  )
}
