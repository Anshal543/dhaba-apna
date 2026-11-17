import test from 'node:test'
import assert from 'node:assert/strict'
import { formatPrice, groupByStatus, summarizeItems, minuteLabel } from './format.js'

test('formatPrice', () => {
  assert.equal(formatPrice(12.5), 'Rs 12.50')
  assert.equal(formatPrice('3'), 'Rs 3.00')
})

test('groupByStatus splits into columns, keeps order, ignores unknown', () => {
  const orders = [
    { id: 3, status: 'ready' },
    { id: 2, status: 'cooking' },
    { id: 1, status: 'new' },
    { id: 4, status: 'new' },
    { id: 5, status: 'weird' },
  ]
  const g = groupByStatus(orders)
  assert.deepEqual(g.new.map((o) => o.id), [1, 4])
  assert.deepEqual(g.cooking.map((o) => o.id), [2])
  assert.deepEqual(g.ready.map((o) => o.id), [3])
})

test('groupByStatus on empty input gives three empty columns', () => {
  assert.deepEqual(groupByStatus([]), { new: [], cooking: [], ready: [] })
})

test('summarizeItems', () => {
  assert.equal(summarizeItems([{ qty: 2, name: 'Karahi' }, { qty: 1, name: 'Naan' }]), '2 × Karahi, 1 × Naan')
})

test('minuteLabel', () => {
  assert.equal(minuteLabel('2026-11-17T11:59:00Z'), '11:59')
})
