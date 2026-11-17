// Cart = array of { id, name, price, qty }. All functions are pure and return a new array.

export const MAX_QTY = 99

export function addItem(cart, item) {
  const existing = cart.find((c) => c.id === item.id)
  if (existing) {
    return setQty(cart, item.id, existing.qty + 1)
  }
  return [...cart, { id: item.id, name: item.name, price: item.price, qty: 1 }]
}

// qty <= 0 removes the line; qty is capped at MAX_QTY (the API rejects more).
export function setQty(cart, id, qty) {
  if (qty <= 0) return cart.filter((c) => c.id !== id)
  return cart.map((c) => (c.id === id ? { ...c, qty: Math.min(qty, MAX_QTY) } : c))
}

export function total(cart) {
  return cart.reduce((sum, c) => sum + c.price * c.qty, 0)
}

export function itemCount(cart) {
  return cart.reduce((sum, c) => sum + c.qty, 0)
}

// Body for POST /orders on orders-api.
export function toOrderPayload(customer, cart) {
  return {
    customer: customer.trim(),
    items: cart.map((c) => ({ name: c.name, qty: c.qty, price: c.price })),
  }
}
