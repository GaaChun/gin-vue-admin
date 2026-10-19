import { afterEach, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({ getTableWidths: vi.fn(), saveTableWidths: vi.fn(), showError: vi.fn() }))
vi.mock('./api/table-widths', () => ({ getTableWidths: mocks.getTableWidths, saveTableWidths: mocks.saveTableWidths }))
vi.mock('./api/common', () => ({ showError: mocks.showError }))

import { useTableWidths } from './use-table-widths'

afterEach(() => vi.clearAllMocks())

it('loads saved widths and persists a resized column under the table key', async () => {
  mocks.getTableWidths.mockResolvedValue({ number: 180 })
  mocks.saveTableWidths.mockResolvedValue({})
  const state = useTableWidths()
  await state.load('warehouse.delivery.list')
  expect(state.widths.value).toEqual({ number: 180 })

  await state.headerDragEnd('warehouse.delivery.list', 205.6, 160, { property: 'number' })
  expect(mocks.getTableWidths).toHaveBeenCalledWith('warehouse.delivery.list')
  expect(mocks.saveTableWidths).toHaveBeenCalledWith({
    tableKey: 'warehouse.delivery.list',
    widths: { number: 206 }
  })
})
