import { Film } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'

export default function Recordings() {
  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold flex items-center gap-2"><Film className="h-6 w-6" />Recordings</h2>
      <Card><CardContent className="p-8 text-center text-muted-foreground">
        <Film className="h-12 w-12 mx-auto mb-3 opacity-30" />
        <p>Session recordings will appear here when <code>recording.enabled: true</code> is set in config.</p>
        <p className="text-xs mt-2">Recordings are captured via CDP screencast and saved to the recordings directory.</p>
      </CardContent></Card>
    </div>
  )
}
