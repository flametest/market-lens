import { NavLink } from 'react-router-dom'
import { useStore } from '../../store/useStore'
import {
  LayoutDashboard,
  BarChart3,
  LineChart,
  Cpu,
  FlaskConical,
  ArrowLeftRight,
  Brain,
  Settings,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/market', icon: BarChart3, label: 'Market' },
  { to: '/chart', icon: LineChart, label: 'Chart' },
  { to: '/strategy', icon: Cpu, label: 'Strategy' },
  { to: '/backtest', icon: FlaskConical, label: 'Backtest' },
  { to: '/trading', icon: ArrowLeftRight, label: 'Trading' },
  { to: '/ai', icon: Brain, label: 'AI Analysis' },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

export default function Sidebar() {
  const { sidebarCollapsed, toggleSidebar } = useStore()

  return (
    <aside
      className="fixed left-0 top-0 bottom-0 flex flex-col border-r border-[var(--color-border)] bg-[var(--color-surface)] z-40 transition-all duration-300"
      style={{ width: sidebarCollapsed ? 64 : 220 }}
    >
      <div className="flex items-center gap-3 px-4 h-16 border-b border-[var(--color-border)]">
        <div className="w-8 h-8 rounded-lg bg-[var(--color-accent)] flex items-center justify-center flex-shrink-0">
          <LineChart size={18} className="text-white" />
        </div>
        {!sidebarCollapsed && (
          <span className="font-heading text-base font-bold tracking-tight text-[var(--color-text-primary)] animate-fade-in">
            Market Lens
          </span>
        )}
      </div>

      <nav className="flex-1 py-4 px-2 space-y-1 overflow-y-auto">
        {navItems.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-150 group ${
                isActive
                  ? 'bg-[var(--color-accent)]/10 text-[var(--color-accent-glow)]'
                  : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-elevated)] hover:text-[var(--color-text-primary)]'
              }`
            }
          >
            <Icon size={18} className="flex-shrink-0" />
            {!sidebarCollapsed && <span className="animate-fade-in">{label}</span>}
          </NavLink>
        ))}
      </nav>

      <button
        onClick={toggleSidebar}
        className="flex items-center justify-center h-12 border-t border-[var(--color-border)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors cursor-pointer bg-transparent"
      >
        {sidebarCollapsed ? <ChevronRight size={16} /> : <ChevronLeft size={16} />}
      </button>
    </aside>
  )
}
