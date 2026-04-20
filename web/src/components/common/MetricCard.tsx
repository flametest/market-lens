interface MetricCardProps {
  label: string
  value: string
  change?: number
  icon?: React.ReactNode
  className?: string
}

export default function MetricCard({ label, value, change, icon, className = '' }: MetricCardProps) {
  return (
    <div className={`card ${className}`}>
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
          {label}
        </span>
        {icon && <span className="text-[var(--color-text-muted)]">{icon}</span>}
      </div>
      <div className="flex items-end gap-2">
        <span className="font-data text-xl font-semibold text-[var(--color-text-primary)]">
          {value}
        </span>
        {change !== undefined && (
          <span className={`font-data text-xs font-medium pb-0.5 ${change >= 0 ? 'text-profit' : 'text-loss'}`}>
            {change >= 0 ? '+' : ''}{change.toFixed(2)}%
          </span>
        )}
      </div>
    </div>
  )
}
