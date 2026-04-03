import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ClipboardList } from 'lucide-react'

export default function JobHistory() {
  const { data } = useQuery({ queryKey: ['jobs'], queryFn: () => api.jobs(100), refetchInterval: 3000 })
  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold flex items-center gap-2"><ClipboardList className="h-6 w-6" />Job History</h2>
      <Card>
        <CardContent className="p-4">
          <table className="w-full text-sm">
            <thead><tr className="text-left text-muted-foreground border-b">
              <th className="pb-2">ID</th><th className="pb-2">Plugin</th><th className="pb-2">Method</th><th className="pb-2">Status</th><th className="pb-2">Duration</th><th className="pb-2">Time</th>
            </tr></thead>
            <tbody>
              {(data?.jobs || []).map((j: any) => (
                <tr key={j.id} className="border-b last:border-0">
                  <td className="py-2 font-mono text-xs">{j.id?.slice(0, 8)}</td>
                  <td>{j.plugin}</td><td>{j.method}</td>
                  <td><Badge variant={j.status === 'completed' ? 'success' : 'destructive'} className="text-[10px]">{j.status}</Badge></td>
                  <td className="text-muted-foreground">{j.result?.duration_ms || 0}ms</td>
                  <td className="text-xs text-muted-foreground">{j.created_at?.slice(11, 19)}</td>
                </tr>
              ))}
              {(!data?.jobs || data.jobs.length === 0) && <tr><td colSpan={6} className="py-4 text-center text-muted-foreground">No jobs recorded</td></tr>}
            </tbody>
          </table>
        </CardContent>
      </Card>
    </div>
  )
}
