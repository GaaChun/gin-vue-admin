import { describe, expect, it } from 'vitest'
import { defaultMenuEntries, sortRecentVisits } from './recentVisits'

describe('dashboard recent menu entries', () => {
  it('uses visible leaf routes ordered by ID when visit history is empty', () => {
    const menus = [
      { ID: 12, name: 'last', path: 'last', meta: { title: 'Last' } },
      { ID: 3, name: 'first', path: 'first', meta: { title: 'First' } },
      { ID: 1, name: 'group', path: 'group', children: [{ ID: 4, name: 'nested', path: 'nested' }] },
      { ID: 2, name: 'hidden', path: 'hidden', hidden: true },
      { ID: 5, name: 'external', path: 'https://example.test' }
    ]

    expect(defaultMenuEntries(menus).map(({ id, name }) => [id, name])).toEqual([
      [3, 'first'],
      [4, 'nested'],
      [12, 'last']
    ])
  })

  it('sorts existing visits by count, then latest time, then ID and caps at six', () => {
    const visits = Array.from({ length: 8 }, (_, index) => ({
      id: index + 1,
      visitCount: index < 2 ? 5 : 1,
      lastVisitedAt: index === 0 ? '2026-10-18T10:00:00Z' : '2026-10-18T11:00:00Z'
    }))

    expect(sortRecentVisits(visits).map(({ id }) => id)).toEqual([2, 1, 3, 4, 5, 6])
  })
})
