import { useEffect, useRef, useState } from 'react'

// Calls `fn` now and then every `ms` milliseconds. Keeps the last good data
// when a refresh fails, so the screen doesn't blank out on a network blip.
export function usePolling(fn, ms) {
  const [state, setState] = useState({ data: null, error: null, loading: true })
  const fnRef = useRef(fn)
  fnRef.current = fn

  useEffect(() => {
    let cancelled = false
    async function tick() {
      try {
        const data = await fnRef.current()
        if (!cancelled) setState({ data, error: null, loading: false })
      } catch (error) {
        if (!cancelled) setState((s) => ({ data: s.data, error, loading: false }))
      }
    }
    tick()
    const timer = setInterval(tick, ms)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [ms])

  return state
}
