import { useEffect, useState } from 'react'
import { api } from '../api.js'
import { addItem, setQty, total, itemCount, toOrderPayload } from '../lib/cart.js'
import { formatPrice } from '../lib/format.js'

export default function OrderPage() {
  const [menu, setMenu] = useState([])
  const [menuError, setMenuError] = useState(null)
  const [loading, setLoading] = useState(true)
  const [cart, setCart] = useState([])
  const [customer, setCustomer] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState(null)
  const [placed, setPlaced] = useState(null)

  useEffect(() => {
    let cancelled = false
    api
      .getMenu()
      .then((items) => !cancelled && setMenu(items))
      .catch((e) => !cancelled && setMenuError(e.message))
      .finally(() => !cancelled && setLoading(false))
    return () => {
      cancelled = true
    }
  }, [])

  async function placeOrder(e) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      const order = await api.createOrder(toOrderPayload(customer, cart))
      setPlaced(order)
      setCart([])
    } catch (err) {
      setError(err.message)
    } finally {
      setSubmitting(false)
    }
  }

  const canSubmit = customer.trim() !== '' && cart.length > 0 && !submitting

  return (
    <div className="order-layout">
      <section>
        <h2>Menu</h2>
        {loading && <p className="muted">Loading menu…</p>}
        {menuError && <p className="error">Could not load the menu: {menuError}</p>}
        {!loading && !menuError && menu.length === 0 && <p className="muted">The menu is empty.</p>}
        <ul className="menu">
          {menu.map((item) => (
            <li key={item.id} className="card">
              <div>
                <strong>{item.name}</strong>
                {item.description && <p className="muted">{item.description}</p>}
              </div>
              <div className="menu-actions">
                <span>{formatPrice(item.price)}</span>
                <button onClick={() => setCart((c) => addItem(c, item))}>Add</button>
              </div>
            </li>
          ))}
        </ul>
      </section>

      <aside>
        <h2>Your order {itemCount(cart) > 0 && <span className="badge">{itemCount(cart)}</span>}</h2>

        {placed && (
          <div className="success">
            Order #{placed.id} placed. <a href="#kitchen">Watch it in the kitchen</a>
          </div>
        )}

        <form onSubmit={placeOrder} className="card cart">
          <label>
            Your name
            <input value={customer} onChange={(e) => setCustomer(e.target.value)} maxLength={100} />
          </label>

          {cart.length === 0 ? (
            <p className="muted">Nothing added yet.</p>
          ) : (
            <ul className="cart-lines">
              {cart.map((line) => (
                <li key={line.id}>
                  <span>{line.name}</span>
                  <span className="qty">
                    <button type="button" aria-label={`Fewer ${line.name}`} onClick={() => setCart((c) => setQty(c, line.id, line.qty - 1))}>
                      −
                    </button>
                    {line.qty}
                    <button type="button" aria-label={`More ${line.name}`} onClick={() => setCart((c) => setQty(c, line.id, line.qty + 1))}>
                      +
                    </button>
                  </span>
                  <span>{formatPrice(line.price * line.qty)}</span>
                </li>
              ))}
            </ul>
          )}

          <div className="total">
            <span>Total</span>
            <strong>{formatPrice(total(cart))}</strong>
          </div>

          {error && <p className="error">{error}</p>}
          <button type="submit" className="primary" disabled={!canSubmit}>
            {submitting ? 'Placing…' : 'Place order'}
          </button>
        </form>
      </aside>
    </div>
  )
}
