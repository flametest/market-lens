import { BrowserRouter, Routes, Route } from 'react-router-dom'
import MainLayout from './components/layout/MainLayout'
import Dashboard from './pages/Dashboard'
import MarketOverview from './pages/MarketOverview'
import ChartAnalysis from './pages/ChartAnalysis'
import Strategy from './pages/Strategy'
import Backtest from './pages/Backtest'
import Trading from './pages/Trading'
import AIAnalysis from './pages/AIAnalysis'
import Settings from './pages/Settings'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<MainLayout />}>
          <Route index element={<Dashboard />} />
          <Route path="market" element={<MarketOverview />} />
          <Route path="chart" element={<ChartAnalysis />} />
          <Route path="strategy" element={<Strategy />} />
          <Route path="backtest" element={<Backtest />} />
          <Route path="trading" element={<Trading />} />
          <Route path="ai" element={<AIAnalysis />} />
          <Route path="settings" element={<Settings />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
