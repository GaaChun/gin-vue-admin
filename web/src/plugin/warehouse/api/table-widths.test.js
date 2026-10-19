import { afterEach, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({ setSelfSetting: vi.fn(), userInfo: { originSetting: { primaryColor: '#123456' } }, resetUserInfo: vi.fn() }))
vi.mock('@/api/user', () => ({ setSelfSetting: mocks.setSelfSetting }))
vi.mock('@/pinia/modules/user', () => ({ useUserStore: () => ({ userInfo: mocks.userInfo, ResetUserInfo: mocks.resetUserInfo }) }))

import { getTableWidths, saveTableWidths } from './table-widths'

afterEach(() => {
  mocks.userInfo.originSetting = { primaryColor: '#123456' }
  vi.clearAllMocks()
})

it('restores and saves a table width without replacing other user settings', async () => {
  const key = 'warehouseTableWidths.warehouse.delivery.list'
  mocks.userInfo.originSetting[key] = { number: 180 }
  mocks.setSelfSetting.mockResolvedValue({ code: 0 })
  expect(getTableWidths('warehouse.delivery.list')).toEqual({ number: 180 })

  await saveTableWidths({ tableKey: 'warehouse.delivery.list', widths: { number: 220 } })

  expect(mocks.setSelfSetting).toHaveBeenCalledWith({ [key]: { number: 220 } })
  expect(mocks.resetUserInfo).toHaveBeenCalledWith({ originSetting: { primaryColor: '#123456', [key]: { number: 220 } } })
})
