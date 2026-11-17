export function formatPrice(amount) {
  return `Rs ${Number(amount).toFixed(2)}`
}

export const STATUSES = ['new', 'cooking', 'ready']

export const STATUS_LABELS = { new: 'New', cooking: 'Cooking', ready: 'Ready' }

// Splits a list of orders into the three kitchen columns.
// Orders with an unknown status are ignored. Input order is kept.
export function groupByStatus(orders) {
  const groups = { new: [], cooking: [], ready: [] }
  for (const o of orders) {
    if (groups[o.status]) groups[o.status].push(o)
  }
  return groups
}

export function summarizeItems(items) {
  return items.map((i) => `${i.qty} × ${i.name}`).join(', ')
}

// "2026-11-17T11:59:00Z" -> "11:59" (UTC, so every screen shows the same thing)
export function minuteLabel(iso) {
  return iso.slice(11, 16)
}
