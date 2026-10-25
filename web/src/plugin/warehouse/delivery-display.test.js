import { expect, it } from 'vitest'
import { DETAIL_FIELDS, LIST_FIELDS, getDefaultDateRange, normalizeDisplaySettings } from './delivery-display'

it('defaults to supported fields and preserves legacy user column order including an empty list', () => {
  expect(normalizeDisplaySettings().list.columns).toEqual(LIST_FIELDS.map(field => field.key))
  expect(normalizeDisplaySettings(['status', 'number', 'status', 'stock'])).toEqual(expect.objectContaining({ list: { columns: ['status', 'number'], widths: {} } }))
  expect(normalizeDisplaySettings([]).list.columns).toEqual([])
})
it('deduplicates groups, retains required date and detail fields, and bounds widths and freezing', () => {
  const result = normalizeDisplaySettings({
    dateDefault: 'invalid', filters: { basic: ['status', 'status', 'unknown'], more: ['status', 'number'] },
    list: { columns: [], widths: { number: 1, status: 9999, unknown: 100, deliveredAt: NaN } },
    detail: { columns: ['amount', 'amount', 'unknown'], widths: { amount: 100.7 }, frozen: 99 }
  })
  expect(result.dateDefault).toBe('system')
  expect(result.filters).toEqual({ basic: ['status', 'dates'], more: ['number'] })
  expect(result.list).toEqual({ columns: [], widths: { number: 80, status: 800 } })
  expect(result.detail.columns).toEqual(['amount', ...DETAIL_FIELDS.filter(field => field.required).map(field => field.key)])
  expect(result.detail.widths).toEqual({ amount: 101 })
  expect(result.detail.frozen).toBe(result.detail.columns.length)
  expect(normalizeDisplaySettings({ detail: { frozen: -4 } }).detail.frozen).toBe(0)
})
it('does not mutate persisted settings or share default arrays', () => {
  const raw = { filters: { basic: ['number'], more: [] }, detail: { columns: ['amount'] } }
  const original = JSON.stringify(raw)
  const result = normalizeDisplaySettings(raw)
  result.detail.columns.reverse()
  expect(JSON.stringify(raw)).toBe(original)
  result.list.columns.pop()
  expect(normalizeDisplaySettings().list.columns).toHaveLength(LIST_FIELDS.length)
})
it('uses Shanghai midnight and inclusive rolling ranges across year boundaries', () => {
  const now = new Date('2025-12-31T16:00:00Z')
  expect(getDefaultDateRange('today', now)).toEqual(['2026-01-01', '2026-01-01'])
  expect(getDefaultDateRange('yesterday', now)).toEqual(['2025-12-31', '2025-12-31'])
  expect(getDefaultDateRange('last7', now)).toEqual(['2025-12-26', '2026-01-01'])
  expect(getDefaultDateRange('last30', now)).toEqual(['2025-12-03', '2026-01-01'])
  expect(getDefaultDateRange('today', new Date('2025-12-31T15:59:59Z'))).toEqual(['2025-12-31', '2025-12-31'])
})
it('handles leap days, month/year starts and unconfigured defaults', () => {
  const now = new Date('2024-02-29T04:00:00Z')
  expect(getDefaultDateRange('thisMonth', now)).toEqual(['2024-02-01', '2024-02-29'])
  expect(getDefaultDateRange('thisYear', now)).toEqual(['2024-01-01', '2024-02-29'])
  expect(getDefaultDateRange('last7', new Date('2024-03-01T04:00:00Z'))).toEqual(['2024-02-24', '2024-03-01'])
  expect(getDefaultDateRange('system', now)).toEqual([])
  expect(getDefaultDateRange('unknown', now)).toEqual([])
})

it('makes modification user and time available to document display settings', () => {
  expect(LIST_FIELDS).toEqual(expect.arrayContaining([
    expect.objectContaining({ key: 'updatedByName', label: '修改人' }),
    expect.objectContaining({ key: 'updatedAt', label: '修改时间' })
  ]))
  expect(normalizeDisplaySettings({ list: { columns: ['updatedByName', 'updatedAt'] } }).list.columns).toEqual(['updatedByName', 'updatedAt'])
})
