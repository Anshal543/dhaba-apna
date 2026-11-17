import test from 'node:test'
import assert from 'node:assert/strict'
import { addItem, setQty, total, itemCount, toOrderPayload, MAX_QTY } from './cart.js'

const karahi = { id: 1, name: 'Chicken Karahi', price: 12.5 }
const naan = { id: 2, name: 'Naan', price: 0.5 }

test('addItem adds a new line with qty 1', () => {
  assert.deepEqual(addItem([], karahi), [{ ...karahi, qty: 1 }])
})

test('addItem on an existing item increases qty and does not mutate', () => {
  const cart = [{ ...karahi, qty: 1 }]
  const next = addItem(cart, karahi)
  assert.equal(next[0].qty, 2)
  assert.equal(cart[0].qty, 1)
})

test('setQty updates, removes at 0, and caps at MAX_QTY', () => {
  const cart = [{ ...karahi, qty: 1 }, { ...naan, qty: 2 }]
  assert.equal(setQty(cart, 2, 5)[1].qty, 5)
  assert.deepEqual(setQty(cart, 2, 0).map((c) => c.id), [1])
  assert.equal(setQty(cart, 1, 1000)[0].qty, MAX_QTY)
})

test('total and itemCount', () => {
  const cart = [{ ...karahi, qty: 2 }, { ...naan, qty: 4 }]
  assert.equal(total(cart), 27)
  assert.equal(itemCount(cart), 6)
  assert.equal(total([]), 0)
})

test('toOrderPayload matches the orders-api contract', () => {
  const cart = [{ ...karahi, qty: 2 }]
  assert.deepEqual(toOrderPayload('  Ali ', cart), {
    customer: 'Ali',
    items: [{ name: 'Chicken Karahi', qty: 2, price: 12.5 }],
  })
})
