import { create } from 'zustand'

interface AppState {
  sidebarCollapsed: boolean
  toggleSidebar: () => void
  activeSymbol: string
  setActiveSymbol: (symbol: string) => void
}

export const useStore = create<AppState>((set) => ({
  sidebarCollapsed: false,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
  activeSymbol: 'AAPL',
  setActiveSymbol: (symbol) => set({ activeSymbol: symbol }),
}))
