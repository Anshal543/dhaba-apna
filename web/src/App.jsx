import { useEffect, useState } from 'react'
import { api } from './api.js'
import OrderPage from './pages/OrderPage.jsx'
import KitchenPage from './pages/KitchenPage.jsx'
import StatsPanel from './pages/StatsPanel.jsx'
import { usePolling } from './hooks/usePolling.js'

const TABS = [
  { id: 'order', label: 'Order' },
  { id: 'kitchen', label: 'Kitchen' },
  { id: 'stats', label: 'Stats' },
]

function tabFromHash() {
  const id = window.location.hash.replace('#', '')
  return TABS.some((t) => t.id === id) ? id : 'order'
}

export default function App() {
  const [tab, setTab] = useState(tabFromHash)

  useEffect(() => {
    const onHash = () => setTab(tabFromHash())
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  // Shows which orders-api version/pod answered: handy when we scale and roll out later.
  const info = usePolling(() => api.getInfo(), 5000)

  return (
    <div className="app">
      <header className="topbar">
        <h1>Dhaba Orders</h1>
        <nav>
          {TABS.map((t) => (
            <a key={t.id} href={`#${t.id}`} className={t.id === tab ? 'active' : ''}>
              {t.label}
            </a>
          ))}
        </nav>
      </header>

      <main>
        {tab === 'order' && <OrderPage />}
        {tab === 'kitchen' && <KitchenPage />}
        {tab === 'stats' && <StatsPanel />}
      </main>

      <footer className="footer">
        {info.data
          ? `orders-api ${info.data.version} · pod ${info.data.pod}`
          : info.error
            ? 'orders-api unreachable'
            : 'connecting…'}
      </footer>
    </div>
  )
}
