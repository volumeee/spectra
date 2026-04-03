import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Layout from '@/components/layout/Layout'
import Overview from '@/pages/Overview'
import RemoteBrowser from '@/pages/RemoteBrowser'
import Sessions from '@/pages/Sessions'
import Profiles from '@/pages/Profiles'
import Plugins from '@/pages/Plugins'
import AIAgent from '@/pages/AIAgent'
import SpectraQL from '@/pages/SpectraQL'
import Recordings from '@/pages/Recordings'
import SettingsPage from '@/pages/Settings'
import Webhooks from '@/pages/Webhooks'
import Schedules from '@/pages/Schedules'
import JobHistory from '@/pages/JobHistory'

const qc = new QueryClient({ defaultOptions: { queries: { retry: 1, staleTime: 2000 } } })

export default function App() {
  return (
    <QueryClientProvider client={qc}>
      <BrowserRouter>
        <Routes>
          {/* Remote browser opens full-page (no sidebar) */}
          <Route path="/remote" element={<RemoteBrowser />} />
          {/* Dashboard with sidebar */}
          <Route element={<Layout />}>
            <Route path="/" element={<Overview />} />
            <Route path="/sessions" element={<Sessions />} />
            <Route path="/profiles" element={<Profiles />} />
            <Route path="/plugins" element={<Plugins />} />
            <Route path="/ai" element={<AIAgent />} />
            <Route path="/query" element={<SpectraQL />} />
            <Route path="/recordings" element={<Recordings />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/webhooks" element={<Webhooks />} />
            <Route path="/schedules" element={<Schedules />} />
            <Route path="/jobs" element={<JobHistory />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
