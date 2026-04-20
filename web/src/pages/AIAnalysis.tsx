import { useState } from 'react'
import { Brain, Send, AlertTriangle, ShieldAlert, ThermometerSun } from 'lucide-react'
import { mockSentiments, mockRiskEvents } from '../mock/data'
import type { SentimentResult } from '../types'

export default function AIAnalysis() {
  const [inputText, setInputText] = useState('')
  const [sentiments] = useState<SentimentResult[]>(mockSentiments)

  const riskIcons = {
    panic: <ShieldAlert size={14} className="text-loss" />,
    overheat: <ThermometerSun size={14} className="text-warn" />,
    pause: <AlertTriangle size={14} className="text-warn" />,
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Brain size={20} className="text-[var(--color-accent-glow)]" />
          <h1 className="font-heading text-2xl font-bold tracking-tight">AI Analysis</h1>
        </div>
      </div>

      <div className="card">
        <h2 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-3">
          Sentiment Analysis
        </h2>
        <div className="flex gap-3">
          <textarea
            value={inputText}
            onChange={(e) => setInputText(e.target.value)}
            placeholder="Paste news, earnings report, or announcement text here to analyze sentiment..."
            className="input-field min-h-[80px] resize-none flex-1 text-sm"
          />
          <button className="btn-primary flex items-center gap-2 self-end" disabled={!inputText.trim()}>
            <Send size={14} /> Analyze
          </button>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="col-span-2 space-y-3">
          <h2 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
            Analysis Results
          </h2>
          {sentiments.map(s => (
            <div key={s.id} className="card animate-slide-up">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <span className="font-data text-sm font-semibold text-[var(--color-text-primary)]">
                    {s.symbols.join(', ')}
                  </span>
                  <span className={`badge ${s.label === 'POSITIVE' ? 'badge-positive' : s.label === 'NEGATIVE' ? 'badge-negative' : 'badge-neutral'}`}>
                    {s.label}
                  </span>
                </div>
                <div className="flex items-center gap-3">
                  <div className="flex items-center gap-1.5">
                    <span className="text-xs text-[var(--color-text-muted)]">Score</span>
                    <span className={`font-data text-sm font-bold ${s.score >= 0 ? 'text-profit' : 'text-loss'}`}>
                      {s.score >= 0 ? '+' : ''}{s.score.toFixed(2)}
                    </span>
                  </div>
                  <span className="text-xs text-[var(--color-text-muted)]">
                    {new Date(s.timestamp).toLocaleString()}
                  </span>
                </div>
              </div>

              <p className="text-xs text-[var(--color-text-muted)] mb-3 line-clamp-2">{s.text}</p>
              <p className="text-sm text-[var(--color-text-secondary)] mb-3">{s.summary}</p>

              <div className="flex items-center gap-2 flex-wrap">
                {s.keywords.map(kw => (
                  <span
                    key={kw}
                    className="px-2 py-0.5 rounded text-[10px] font-medium bg-[var(--color-accent)]/10 text-[var(--color-accent-glow)]"
                  >
                    {kw}
                  </span>
                ))}
              </div>

              <div className="mt-3 flex items-center gap-3">
                <span className="text-xs text-[var(--color-text-muted)]">Sentiment</span>
                <div className="flex-1 h-2 bg-[var(--color-border)] rounded-full overflow-hidden">
                  <div
                    className={`h-full rounded-full transition-all ${
                      s.score >= 0 ? 'bg-[var(--color-profit)]' : 'bg-[var(--color-loss)]'
                    }`}
                    style={{ width: `${Math.abs(s.score) * 100}%` }}
                  />
                </div>
              </div>
            </div>
          ))}
        </div>

        <div className="space-y-4">
          <div>
            <h2 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-3">
              AI Risk Events
            </h2>
            <div className="space-y-2">
              {mockRiskEvents.map(e => (
                <div key={e.id} className="card-elevated">
                  <div className="flex items-center gap-2 mb-2">
                    {riskIcons[e.type]}
                    <span className={`badge ${e.type === 'panic' ? 'badge-negative' : 'badge-running'}`}>
                      {e.type.toUpperCase()}
                    </span>
                  </div>
                  <p className="text-xs text-[var(--color-text-secondary)]">{e.reason}</p>
                  <span className="text-[10px] text-[var(--color-text-muted)] mt-2 block">
                    {new Date(e.timestamp).toLocaleString()}
                  </span>
                </div>
              ))}
            </div>
          </div>

          <div>
            <h2 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-3">
              AI Configuration
            </h2>
            <div className="space-y-2">
              {[
                { label: 'Signal Participation', enabled: true },
                { label: 'Confidence Adjustment', enabled: true },
                { label: 'Risk Enhancement', enabled: true },
                { label: 'Signal Weight', enabled: true, value: '30%' },
              ].map((cfg, i) => (
                <div key={i} className="card-elevated flex items-center justify-between">
                  <span className="text-xs text-[var(--color-text-secondary)]">{cfg.label}</span>
                  {cfg.value ? (
                    <span className="font-data text-xs font-medium text-[var(--color-accent-glow)]">{cfg.value}</span>
                  ) : (
                    <div className={`w-8 h-4 rounded-full transition-colors ${cfg.enabled ? 'bg-[var(--color-accent)]' : 'bg-[var(--color-border-light)]'} relative cursor-pointer`}>
                      <div className={`absolute top-0.5 w-3 h-3 rounded-full bg-white transition-transform ${cfg.enabled ? 'left-[18px]' : 'left-0.5'}`} />
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
