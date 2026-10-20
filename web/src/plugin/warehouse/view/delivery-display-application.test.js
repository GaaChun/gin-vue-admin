import { afterEach, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import Delivery from './delivery.vue'
import { DISPLAY_SETTING_KEY } from '../delivery-display'

const mocks = vi.hoisted(() => ({ user: null, route: null, request: vi.fn() }))
vi.mock('@/pinia/modules/user', () => ({ useUserStore: () => mocks.user }))
vi.mock('vue-router', () => ({ useRoute: () => mocks.route, useRouter: () => ({ push: vi.fn() }) }))
vi.mock('@/api/user', () => ({ getUserList: async () => ({ code: 0, data: { list: [] } }), setSelfSetting: vi.fn() }))
vi.mock('../api/common', () => ({
  request: mocks.request,
  allOptions: async () => [],
  idOf: value => value?.ID ?? value?.id,
  pageSequenceNumber: (index, page, size) => (page - 1) * size + index + 1,
  showError: vi.fn()
}))
vi.mock('../api/table-widths', () => ({ getTableWidths: async () => ({}), saveTableWidths: async () => ({}) }))
vi.mock('../components/delivery-material-picker.vue', () => ({ default: { template: '<div />' } }))
let wrapper
function setup(settings, query = {}) {
  mocks.user = reactive({ userInfo: { ID: 1, originSetting: { [DISPLAY_SETTING_KEY]: settings } }, ResetUserInfo: vi.fn() })
  mocks.route = reactive({ name: 'warehouseDelivery', query })
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    if (url.endsWith('/departments')) return []
    if (url.endsWith('/detail')) return {
      note: { id: 10, number: 'SK261001000001', status: 'draft', brandId: 2, deliveredAt: '2026-10-01T00:00:00Z' },
      details: [{ id: 1, materialId: 3, materialUnitId: 4, unitId: 5, unitNameSnapshot: '件', materialCodeSnapshot: 'M3', materialNameSnapshot: '物料', quantity: '2', unitPrice: '3', amount: '6' }]
    }
    throw new Error(`Unexpected URL ${url}`)
  })
  wrapper = mount(Delivery, { attachTo: document.body, global: { plugins: [ElementPlus] } })
}
afterEach(() => { wrapper?.unmount(); vi.clearAllMocks() })
const headers = selector => wrapper.findAll(`${selector} .el-table__header-wrapper th .cell`).map(cell => cell.text())

it('applies saved column order and filter placement, including changes made while the list is cached', async () => {
  setup({ filters: { basic: ['dates', 'department'], more: ['number'] }, list: { columns: ['totalAmount', 'number'], widths: {} } })
  await flushPromises()
  expect(headers('.delivery-page .el-table')).toEqual(['', '序号', '金额', '单号', '操作'])
  expect(wrapper.findAll('[data-filter-group="basic"] label').map(label => label.text())).toEqual(['送货日期', '部门'])
  expect(wrapper.find('[data-filter-group="more"]').isVisible()).toBe(false)
  const moreButton = wrapper.findAll('button').find(button => button.text() === '更多条件')
  await moreButton.trigger('click')
  expect(wrapper.find('[data-filter-group="more"]').isVisible()).toBe(true)
  mocks.user.userInfo.originSetting[DISPLAY_SETTING_KEY] = { filters: { basic: ['number', 'dates'], more: [] }, list: { columns: ['number', 'brandNameSnapshot'], widths: {} } }
  await flushPromises()
  expect(headers('.delivery-page .el-table')).toEqual(['', '序号', '单号', '客户', '操作'])
  expect(wrapper.find('[data-filter-key="department"]').exists()).toBe(false)
})

it('renders reordered detail inputs with frozen columns and follows the displayed editing order on Tab', async () => {
  setup({ detail: { columns: ['unitPrice', 'quantity', 'materialCode', 'unitName', 'materialName'], widths: { unitPrice: 190 }, frozen: 2 } }, { whDelivery: '1', whId: '10' })
  await flushPromises()
  expect(headers('.detail-table')).toEqual(['行', '单价', '数量', '物料编号', '单位', '物料名称', '操作'])
  const frozen = wrapper.findAll('.detail-table .el-table__header-wrapper th.el-table-fixed-column--left .cell').map(cell => cell.text())
  expect(frozen).toEqual(['行', '单价', '数量'])
  const inputs = wrapper.findAll('.detail-table input[inputmode="decimal"]')
  expect(inputs.map(input => input.element.value)).toEqual(['3', '2'])
  await inputs[0].trigger('keydown', { key: 'Tab' })
  await flushPromises()
  expect(document.activeElement).toBe(inputs[1].element)
  expect(wrapper.find('.detail-table').text()).not.toContain('规格型号')
})
