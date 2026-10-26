import { expect, it } from 'vitest'
import { matchesMasterKeyword, materialSelectionSummary } from './master-search'

it('matches names, codes and every delimited mnemonic without joining aliases', () => {
  const record = { name: '食品采购', code: 'KH-01', mnemonic: 'spcg，SHI PIN;采购；cai\tgou\nextra' }
  for (const query of ['食品', 'kh-01', 'spcg', 'shi', 'pin', '采购', 'cai', 'gou', 'EXTRA']) {
    expect(matchesMasterKeyword(record, query), query).toBe(true)
  }
  expect(matchesMasterKeyword(record, 'missing')).toBe(false)
  expect(matchesMasterKeyword(record, 'shi pin')).toBe(false)
  expect(matchesMasterKeyword(record, '')).toBe(true)
})

it('matches departments by name or any mnemonic while ignoring unrelated fields', () => {
  const record = { name: '行政部', code: 'not-a-department-code', mnemonic: 'XZB,xingzheng' }
  expect(matchesMasterKeyword(record, '行政', ['name'])).toBe(true)
  expect(matchesMasterKeyword(record, 'XINGZHENG', ['name'])).toBe(true)
  expect(matchesMasterKeyword(record, 'not-a-department-code', ['name'])).toBe(false)
})

it('shows the material name for one selection and the count for multiple selections', () => {
  const materials = [{ ID: 1, name: '青菜' }, { id: 2, name: '白菜' }]
  expect(materialSelectionSummary([], materials)).toBe('')
  expect(materialSelectionSummary([1], materials)).toBe('青菜')
  expect(materialSelectionSummary([2], materials)).toBe('白菜')
  expect(materialSelectionSummary([1, 2], materials)).toBe('已选 2 种物料')
  expect(materialSelectionSummary([3], materials)).toBe('已选 1 种物料')
})
