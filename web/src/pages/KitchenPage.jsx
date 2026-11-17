import { api } from '../api.js'
import { usePolling } from '../hooks/usePolling.js'
import { STATUSES, STATUS_LABELS, groupByStatus, summarizeItems } from '../lib/format.js'

const READY_SHOWN = 10

export default function KitchenPage() {
  const { data, error, loading } = usePolling(() => api.getOrders(50), 3000)
  const groups = groupByStatus(data || [])

  return (
    <section>
      <h2>Kitchen</h2>
      {loading && <p className="muted">Loading orders…</p>}
      {error && <p className="error">{error.message}{data ? ' (showing last known orders)' : ''}</p>}

      <div className="kanban">
        {STATUSES.map((status) => {
          const orders = status === 'ready' ? groups.ready.slice(0, READY_SHOWN) : groups[status]
          return (
            <div key={status} className={`column ${status}`}>
              <h3>
                {STATUS_LABELS[status]} <span className="badge">{groups[status].length}</span>
              </h3>
              {orders.length === 0 && <p className="muted">No orders</p>}
              {orders.map((o) => (
                <article key={o.id} className="card order">
                  <strong>
                    #{o.id} · {o.customer}
                  </strong>
                  <p>{summarizeItems(o.items)}</p>
                </article>
              ))}
            </div>
          )
        })}
      </div>
    </section>
  )
}
