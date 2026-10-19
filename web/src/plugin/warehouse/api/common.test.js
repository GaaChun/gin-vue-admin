import { expect, it } from 'vitest'
import { pageSequenceNumber } from './common'

it('continues visible row numbering across pages and page-size changes', () => {
  expect(pageSequenceNumber(0, 1, 20)).toBe(1)
  expect(pageSequenceNumber(19, 1, 20)).toBe(20)
  expect(pageSequenceNumber(0, 2, 20)).toBe(21)
  expect(pageSequenceNumber(0, 2, 50)).toBe(51)
})
