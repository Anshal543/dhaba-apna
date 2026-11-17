import { api } from '../api.js'
import { usePolling } from '../hooks/usePolling.js'
import { minuteLabel } from '../lib/format.js'

export default function StatsPanel() {
  const { data, error, loading } = usePolling(() => api.getStats(), 5000)
  const history = data?.history ?? []
  const peak = Math.max(1, ...history.map((h) => h.orders))

  return (
    <section>
      <h2>Live stats</h2>
      {loading && <p className="muted">Loading stats…</p>}
      {error && <p className="error">{error.message}{data ? ' (showing last known numbers)' : ''}</p>}

      {data && (
        <div className="stats-grid">
          <div className="card big-number">
            <span className="muted">Orders per minute</span>
            <strong>{data.orders_per_minute}</strong>
          </div>

          <div className="card">
            <h3>Last 10 minutes</h3>
            <div className="bars" role="img" aria-label="Orders per minute, last 10 minutes">
              {history.map((h) => (
                <div key={h.minute} className="bar-col" title={`${minuteLabel(h.minute)} UTC: ${h.orders}`}>
                  <div className="bar" style={{ height: `${(h.orders / peak) * 100}%` }} />
                  <span>{h.orders}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="card">
            <h3>Top dishes</h3>
            {data.top_dishes.length === 0 ? (
              <p className="muted">No orders yet.</p>
            ) : (
              <ol className="top-dishes">
                {data.top_dishes.map((d) => (
                  <li key={d.name}>
                    <span>{d.name}</span>
                    <strong>{d.count}</strong>
                  </li>
                ))}
              </ol>
            )}
          </div>
        </div>
      )}
    </section>
  )
}
