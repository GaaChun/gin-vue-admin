import { afterEach, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import Delivery from './delivery.vue'
import MaterialFilter from '../components/material-filter.vue'
import { DISPLAY_SETTING_KEY } from '../delivery-display'

const mocks = vi.hoisted(() => ({
  user: null,
  route: null,
  request: vi.fn(),
  materials: [
    { id: 3, code: 'M3', name: '红油', mnemonic: 'hy' },
    { id: 7, code: 'M7', name: '白油', mnemonic: 'by' }
  ]
}))
vi.mock('@/pinia/modules/user', () => ({ useUserStore: () => mocks.user }))
vi.mock('vue-router', () => ({ useRoute: () => mocks.route, useRouter: () => ({ push: vi.fn() }) }))
vi.mock('@/api/user', () => ({ getUserList: async () => ({ code: 0, data: { list: [] } }), setSelfSetting: vi.fn() }))
vi.mock('../api/common', () => ({
  request: mocks.request,
  allOptions: async resource => resource === 'materials' ? mocks.materials : [],
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
  expect(headers('.detail-table')).toEqual(['', '行', '单价', '数量', '物料编号', '单位', '物料名称', '操作'])
  const frozen = wrapper.findAll('.detail-table .el-table__header-wrapper th.el-table-fixed-column--left .cell').map(cell => cell.text())
  expect(frozen).toEqual(expect.arrayContaining(['单价', '数量']))
  expect(frozen).not.toContain('物料编号')
  expect(wrapper.findAll('.detail-table input[inputmode="decimal"]')).toHaveLength(0)
  await wrapper.find('.detail-table .el-table__body-wrapper tr').trigger('dblclick')
  await flushPromises()
  const inputs = wrapper.findAll('.detail-table input[inputmode="decimal"]')
  expect(inputs.map(input => input.element.value)).toEqual(['3', '2'])
  await inputs[0].trigger('keydown', { key: 'Tab' })
  await flushPromises()
  expect(document.activeElement).toBe(inputs[1].element)
  expect(wrapper.find('.detail-table').text()).not.toContain('规格型号')
})

const listRequests = () => mocks.request.mock.calls.map(([request]) => request).filter(request => request.url === '/warehouse/delivery/list')

it('uses the configured inline material filter and only submits selected IDs when querying, retaining them across pages', async () => {
  setup({ filters: { basic: ['materialIds'], more: [] } })
  await flushPromises()
  const field = wrapper.find('[data-filter-group="basic"] [data-filter-key="materialIds"]')
  const materialFilter = field.findComponent(MaterialFilter)
  expect(materialFilter.exists()).toBe(true)
  expect(materialFilter.props('materials')).toEqual(mocks.materials)
  expect(materialFilter.props('modelValue')).toEqual([])
  const initialRequests = listRequests().length

  materialFilter.vm.$emit('update:modelValue', [3, 7])
  await flushPromises()
  expect(materialFilter.props('modelValue')).toEqual([3, 7])
  expect(listRequests()).toHaveLength(initialRequests)

  await wrapper.find('form.delivery-filters').trigger('submit')
  await flushPromises()
  expect(listRequests().at(-1).params).toMatchObject({ materialIds: '3,7', page: 1 })

  const pagination = wrapper.findComponent({ name: 'ElPagination' })
  pagination.vm.$emit('update:current-page', 2)
  pagination.vm.$emit('current-change', 2)
  await flushPromises()
  expect(listRequests().at(-1).params).toMatchObject({ materialIds: '3,7', page: 2 })
  expect(materialFilter.props('modelValue')).toEqual([3, 7])
})

it('clears selected material IDs on reset and queries without a material restriction', async () => {
  setup({ filters: { basic: ['materialIds'], more: [] } })
  await flushPromises()
  const materialFilter = wrapper.findComponent(MaterialFilter)
  materialFilter.vm.$emit('update:modelValue', [3, 7])
  await wrapper.find('form.delivery-filters').trigger('submit')
  await flushPromises()
  expect(listRequests().at(-1).params.materialIds).toBe('3,7')

  await wrapper.findAll('form.delivery-filters button').find(button => button.text() === '重置').trigger('click')
  await flushPromises()
  expect(materialFilter.props('modelValue')).toEqual([])
  expect(listRequests().at(-1).params.materialIds).toBeUndefined()
  expect(listRequests().at(-1).params.page).toBe(1)
})

it('clears hidden material filter values when display settings remove the field', async () => {
  setup({ filters: { basic: ['number'], more: ['materialIds'] } })
  await flushPromises()
  expect(wrapper.find('[data-filter-group="more"] [data-filter-key="materialIds"]').exists()).toBe(true)
  wrapper.findComponent(MaterialFilter).vm.$emit('update:modelValue', [7])
  await wrapper.find('form.delivery-filters').trigger('submit')
  await flushPromises()
  expect(listRequests().at(-1).params.materialIds).toBe('7')

  mocks.user.userInfo.originSetting[DISPLAY_SETTING_KEY] = { filters: { basic: ['number'], more: [] } }
  await flushPromises()
  expect(wrapper.findComponent(MaterialFilter).exists()).toBe(false)
  await wrapper.find('form.delivery-filters').trigger('submit')
  await flushPromises()
  expect(listRequests().at(-1).params.materialIds).toBeUndefined()

  mocks.user.userInfo.originSetting[DISPLAY_SETTING_KEY] = { filters: { basic: ['materialIds'], more: [] } }
  await flushPromises()
  expect(wrapper.findComponent(MaterialFilter).props('modelValue')).toEqual([])
})

it('mounts editing controls for only one row and unmounts them on outside pointerdown', async () => {
  setup({}, { whDelivery: '1', whId: '10' })
  await flushPromises()
  const state = wrapper.vm.$.setupState
  const source = state.rows[0]
  state.rows = Array.from({ length: 3 }, (_, index) => ({ ...source, rowKey: `row-${index}`, id: index + 1, errors: {}, units: [...source.units] }))
  await flushPromises()
  expect(wrapper.findAll('.detail-table input[inputmode="decimal"]')).toHaveLength(0)
  await state.focusCell(state.rows[0], 'quantity')
  await flushPromises()
  expect(wrapper.findAll('.detail-table input[inputmode="decimal"]')).toHaveLength(3)
  await state.focusCell(state.rows[1], 'quantity')
  await flushPromises()
  expect(wrapper.findAll('.detail-table .delivery-editing-row')).toHaveLength(1)
  expect(wrapper.findAll('.detail-table input[inputmode="decimal"]')).toHaveLength(3)
  document.body.dispatchEvent(new Event('pointerdown', { bubbles: true }))
  await flushPromises()
  expect(wrapper.findAll('.detail-table input[inputmode="decimal"]')).toHaveLength(0)
})

it('closes the real material select after Enter confirmation and focuses quantity', async () => {
  setup({}, { whDelivery: '1', whId: '10' })
  await flushPromises()
  const state = wrapper.vm.$.setupState
  mocks.request.mockResolvedValue({ id: 3, code: 'M3', name: '红油', units: [{ ID: 4, unitId: 5, name: '件', enabled: true, isBase: true }] })
  await state.focusCell(state.rows[0], 'material')
  await flushPromises()
  const material = wrapper.find('.detail-table .el-select input')
  expect(material.attributes('aria-expanded')).toBe('true')
  await material.trigger('keydown', { key: 'Enter' })
  await flushPromises()
  expect(material.attributes('aria-expanded')).toBe('false')
  expect(state.rows[0].materialCode).toBe('M3')
  expect(document.activeElement).toBe(wrapper.find('.detail-table input[inputmode="decimal"]').element)
})

it('confirms the keyboard-highlighted material rather than always choosing the first', async () => {
  setup({}, { whDelivery: '1', whId: '10' })
  await flushPromises()
  const state = wrapper.vm.$.setupState
  mocks.request.mockImplementation(async ({ params }) => ({ id: params.id, code: params.id === 7 ? 'M7' : 'M3', name: '物料', units: [{ ID: 4, unitId: 5, name: '件', enabled: true, isBase: true }] }))
  await state.focusCell(state.rows[0], 'material')
  await flushPromises()
  const material = wrapper.find('.detail-table .el-select input')
  await material.trigger('keydown', { key: 'ArrowDown' })
  await flushPromises()
  await material.trigger('keydown', { key: 'Enter' })
  await flushPromises()
  expect(state.rows[0].materialId).toBe(7)
  expect(material.attributes('aria-expanded')).toBe('false')
})

it('marks selected detail rows independently from the active editor row', async () => {
  setup({}, { whDelivery: '1', whId: '10' })
  await flushPromises()
  const row = wrapper.find('.detail-table .el-table__body-wrapper tr')
  await row.findAll('td')[2].trigger('click')
  await flushPromises()
  expect(wrapper.vm.$.setupState.selectedDetailRows).toHaveLength(1)
  expect(row.classes()).toContain('delivery-selected-row')
  await row.findAll('td')[2].trigger('click')
  await flushPromises()
  expect(row.classes()).not.toContain('delivery-selected-row')
})
