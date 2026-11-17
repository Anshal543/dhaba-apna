// Thin client for the backend. Same-origin /api/... paths: Nginx (prod) or Vite (dev) forwards them.
//   /api/menu    -> menu-api    /menu
//   /api/orders  -> orders-api  /orders
//   /api/info    -> orders-api  /info
//   /api/stats   -> stats-api   /api/stats

export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export function createApi(fetchImpl = (...args) => fetch(...args)) {
  async function request(path, options) {
    let res
    try {
      res = await fetchImpl(path, options)
    } catch {
      throw new ApiError('Cannot reach the server', 0)
    }
    let body = null
    try {
      body = await res.json()
    } catch {
      // empty or non-JSON body
    }
    if (!res.ok) {
      throw new ApiError((body && body.error) || `Request failed (${res.status})`, res.status)
    }
    return body
  }

  return {
    async getMenu() {
      const body = await request('/api/menu')
      return normalizeMenu(body)
    },
    createOrder(payload) {
      return request('/api/orders', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
    },
    getOrders(limit = 50) {
      return request(`/api/orders?limit=${limit}`)
    },
    getStats() {
      return request('/api/stats')
    },
    getInfo() {
      return request('/api/info')
    },
  }
}

// menu-api may return a bare array or { items: [...] }. Keep only what the UI needs.
export function normalizeMenu(body) {
  const list = Array.isArray(body) ? body : body && Array.isArray(body.items) ? body.items : []
  return list
    .filter((m) => m && m.name)
    .map((m) => ({
      id: m.id ?? m.name,
      name: m.name,
      price: Number(m.price ?? m.price_pkr) || 0,
      description: m.description || '',
      category: m.category || '',
    }))
}
