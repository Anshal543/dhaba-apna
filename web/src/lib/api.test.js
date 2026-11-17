import test from 'node:test'
import assert from 'node:assert/strict'
import { createApi, normalizeMenu, ApiError } from './api.js'

function fakeFetch(status, body, calls = []) {
  return async (path, options) => {
    calls.push({ path, options })
    return {
      ok: status >= 200 && status < 300,
      status,
      json: async () => {
        if (body === undefined) throw new Error('no body')
        return body
      },
    }
  }
}

test('getMenu accepts a bare array', async () => {
  const api = createApi(fakeFetch(200, [{ id: 1, name: 'Dal', price: '5.5' }]))
  assert.deepEqual(await api.getMenu(), [
    { id: 1, name: 'Dal', price: 5.5, description: '', category: '' },
  ])
})

test('normalizeMenu accepts { items }, skips nameless rows, tolerates junk', () => {
  assert.equal(normalizeMenu({ items: [{ name: 'Naan', price: 1 }, { price: 2 }] }).length, 1)
  assert.deepEqual(normalizeMenu(null), [])
  assert.deepEqual(normalizeMenu({}), [])
  assert.equal(normalizeMenu([{ name: 'Naan' }])[0].id, 'Naan')
  assert.equal(normalizeMenu([{ name: 'Naan', price: 'x' }])[0].price, 0)
})

test('createOrder POSTs JSON to /api/orders', async () => {
  const calls = []
  const api = createApi(fakeFetch(201, { id: 9, status: 'new' }, calls))
  const order = await api.createOrder({ customer: 'Ali', items: [] })
  assert.equal(order.id, 9)
  assert.equal(calls[0].path, '/api/orders')
  assert.equal(calls[0].options.method, 'POST')
  assert.equal(calls[0].options.headers['Content-Type'], 'application/json')
  assert.deepEqual(JSON.parse(calls[0].options.body), { customer: 'Ali', items: [] })
})

test('API error message is taken from { error }', async () => {
  const api = createApi(fakeFetch(400, { error: 'customer is required' }))
  await assert.rejects(api.createOrder({}), (e) => {
    assert.ok(e instanceof ApiError)
    assert.equal(e.message, 'customer is required')
    assert.equal(e.status, 400)
    return true
  })
})

test('non-JSON error body falls back to a status message', async () => {
  const api = createApi(fakeFetch(502, undefined))
  await assert.rejects(api.getStats(), /Request failed \(502\)/)
})

test('network failure becomes a friendly ApiError', async () => {
  const api = createApi(async () => {
    throw new TypeError('Failed to fetch')
  })
  await assert.rejects(api.getOrders(), (e) => e instanceof ApiError && e.status === 0)
})

test('getOrders passes the limit, getStats and getInfo hit the right paths', async () => {
  const calls = []
  const api = createApi(fakeFetch(200, [], calls))
  await api.getOrders(10)
  await api.getStats()
  await api.getInfo()
  assert.deepEqual(calls.map((c) => c.path), ['/api/orders?limit=10', '/api/stats', '/api/info'])
})
