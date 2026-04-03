import { NavLink, Outlet } from 'react-router-dom'
import { cn } from '@/lib/utils'
import {
  LayoutDashboard, Monitor, Users, Puzzle, Bot, Link2, Film,
  Settings, Bell, Clock, ClipboardList, UserCircle
} from 'lucide-react'

const nav = [
  { label: 'Overview', to: '/', icon: LayoutDashboard },
  { group: 'BROWSER' },
  { label: 'Remote', to: '/remote', icon: Monitor, external: true },
  { label: 'Sessions', to: '/sessions', icon: Users },
  { label: 'Profiles', to: '/profiles', icon: UserCircle },
  { group: 'AUTOMATION' },
  { label: 'Plugins', to: '/plugins', icon: Puzzle },
  { label: 'AI Agent', to: '/ai', icon: Bot },
  { label: 'SpectraQL', to: '/query', icon: Link2 },
  { label: 'Recordings', to: '/recordings', icon: Film },
  { group: 'SYSTEM' },
  { label: 'Settings', to: '/settings', icon: Settings },
  { label: 'Webhooks', to: '/webhooks', icon: Bell },
  { label: 'Schedules', to: '/schedules', icon: Clock },
  { label: 'Job History', to: '/jobs', icon: ClipboardList },
] as const

type NavItem = { label: string; to: string; icon: any; external?: boolean } | { group: string }

export default function Layout() {
  return (
    <div className="flex h-screen">
      <aside className="w-56 border-r bg-card flex flex-col">
        <div className="p-4 border-b">
          <h1 className="text-lg font-bold flex items-center gap-2">
            <img src="/logo-transparent.png" alt="Spectra" className="h-7 w-7" />
            Spectra
          </h1>
          <p className="text-xs text-muted-foreground">Dashboard</p>
        </div>
        <nav className="flex-1 overflow-y-auto p-2 space-y-0.5">
          {(nav as readonly NavItem[]).map((item, i) =>
            'group' in item ? (
              <p key={i} className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider px-3 pt-4 pb-1">{item.group}</p>
            ) : item.external ? (
              <a key={item.to} href={item.to} target="_blank" rel="noopener noreferrer"
                className="flex items-center gap-3 rounded-md px-3 py-2 text-sm text-muted-foreground hover:bg-accent/50 hover:text-foreground transition-colors">
                <item.icon className="h-4 w-4" />
                {item.label}
                <span className="ml-auto text-[10px]">↗</span>
              </a>
            ) : (
              <NavLink key={item.to} to={item.to} end={item.to === '/'} className={({ isActive }) => cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors',
                isActive ? 'bg-accent text-accent-foreground font-medium' : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground'
              )}>
                <item.icon className="h-4 w-4" />
                {item.label}
              </NavLink>
            )
          )}
        </nav>
        <div className="p-3 border-t text-[10px] text-muted-foreground">v0.3.0 · MIT</div>
      </aside>
      <main className="flex-1 overflow-y-auto">
        <Outlet />
      </main>
    </div>
  )
}
