import { Outlet } from 'react-router-dom'
import { useStore } from '../../store/useStore'
import Sidebar from './Sidebar'
import Header from './Header'

export default function MainLayout() {
  const { sidebarCollapsed } = useStore()

  return (
    <div className="h-full flex">
      <Sidebar />
      <div
        className="flex-1 flex flex-col min-w-0 transition-all duration-300"
        style={{ marginLeft: sidebarCollapsed ? 64 : 220 }}
      >
        <Header />
        <main className="flex-1 overflow-auto p-6 bg-[var(--color-base)]">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
