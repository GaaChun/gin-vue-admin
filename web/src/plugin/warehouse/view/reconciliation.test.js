import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const mocks = vi.hoisted(() => ({
  loadReportColumns: vi.fn(), loadReportOptions: vi.fn(), queryReport: vi.fn(),
  exportReport: vi.fn(), printReport: vi.fn(), listReportSchemes: vi.fn(),
  getReportScheme: vi.fn(), createReportScheme: vi.fn(), updateReportScheme: vi.fn(),
  deleteReportScheme: vi.fn(), publishReportScheme: vi.fn(),
  showError: vi.fn(), warning: vi.fn(), success: vi.fn(), push: vi.fn()
}))
vi.mock('@/pinia/modules/user', () => ({ useUserStore: () => ({ userInfo: { id: 12 } }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }))
vi.mock('element-plus', () => ({
  ElMessage: { warning: mocks.warning, success: mocks.success },
  ElMessageBox: { confirm: vi.fn(async () => {}) }
}))
vi.mock('../api/common', () => ({
  idOf: value => value?.id,
  pageSequenceNumber: (index, page, pageSize) => (page - 1) * pageSize + index + 1,
  showError: mocks.showError
}))
vi.mock('../api/reconciliation', () => ({
  loadReportColumns: mocks.loadReportColumns, loadReportOptions: mocks.loadReportOptions,
  queryReport: mocks.queryReport, exportReport: mocks.exportReport, printReport: mocks.printReport,
  listReportSchemes: mocks.listReportSchemes, getReportScheme: mocks.getReportScheme,
  createReportScheme: mocks.createReportScheme, updateReportScheme: mocks.updateReportScheme,
  deleteReportScheme: mocks.deleteReportScheme, publishReportScheme: mocks.publishReportScheme
}))
vi.mock('../api/table-widths', () => ({ getTableWidths: vi.fn(async () => ({})), saveTableWidths: vi.fn(async () => ({})) }))

import Reconciliation from './reconciliation.vue'

const registry = [
  { id: 'deliveryDay', label: '送货日', kind: 'dimension' },
  { id: 'noteNumber', label: '送货单号', kind: 'dimension' },
  { id: 'quantity', label: '数量', kind: 'metric' },
  { id: 'unitPrice', label: '单价', kind: 'metric' },
  { id: 'amount', label: '金额', kind: 'metric' }
]
const ids = registry.map(column => column.id)
const defaults = ['reviewed', 'reReviewed']
const Select = {
  name: 'TestSelect', props: ['modelValue', 'placeholder', 'multiple'],
  emits: ['update:modelValue', 'change'],
  template: '<div class="select-stub" :data-placeholder="placeholder"><slot /><slot name="footer" /></div>'
}
const DatePicker = {
  name: 'TestDatePicker', props: ['shortcuts', 'modelValue'], emits: ['update:modelValue'],
  template: '<div class="date-stub" />'
}
const Pagination = {
  name: 'TestPagination', props: ['currentPage', 'pageSize', 'total'],
  emits: ['update:currentPage', 'update:pageSize', 'current-change', 'size-change'],
  template: '<div class="pagination-stub" />'
}
const Table = {
  name: 'TestTable', props: ['summaryMethod', 'showSummary', 'data'], emits: ['sort-change', 'header-dragend'],
  methods: {
    sortHeader(event) {
      const property = event.target.closest('.column-stub')?.dataset.property
      if (property) this.$emit('sort-change', { prop: property, order: 'ascending' })
    }
  },
  template: '<div class="table-stub" @click="sortHeader"><table><thead><tr><slot /></tr></thead></table><div v-if="showSummary" class="table-summary">{{ summaryMethod({ columns: [{}, { property: "deliveryDay" }, { property: "amount" }, { property: "quantity" }, { property: "unitPrice" }] }).join("|") }}</div></div>'
}
const stubs = {
  'el-button': { props: ['disabled', 'icon'], template: '<button :disabled="disabled"><slot /></button>' },
  'el-form': { template: '<form><slot /></form>' },
  'el-form-item': { props: ['label'], template: '<div><label>{{ label }}</label><slot /></div>' },
  'el-select': Select,
  'el-option': { props: ['label', 'value'], template: '<div class="option-stub" :data-value="value">{{ label }}<slot /></div>' },
  'el-date-picker': DatePicker,
  'el-input': { props: ['modelValue'], emits: ['update:modelValue'], template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />' },
  'el-checkbox-group': { props: ['modelValue'], emits: ['update:modelValue'], template: '<div><slot /></div>' },
  'el-checkbox': { template: '<span><slot /></span>' },
  'el-table': Table,
  'el-table-column': { props: ['prop', 'align'], template: '<th class="column-stub" :data-property="prop" :data-align="align"><slot name="header" /></th>' },
  'el-pagination': Pagination,
  'el-dialog': { props: ['modelValue'], template: '<div v-if="modelValue" class="dialog-stub"><slot /><slot name="footer" /></div>' },
  'el-alert': { props: ['title'], template: '<div>{{ title }}</div>' },
  ReportPrintView: true
}
let wrapper
let host
const button = text => wrapper.findAll('button').find(item => item.text() === text)
const select = placeholder => wrapper.findAllComponents(Select).find(item => item.props('placeholder') === placeholder)
async function start() {
  host = document.createElement('div')
  host.id = 'gva-base-load-dom'
  host.setAttribute('role', 'tabpanel')
  host.getBoundingClientRect = () => ({ top: 110, left: 210, right: 1000, bottom: 800 })
  document.body.append(host)
  wrapper = mount(Reconciliation, { attachTo: host, global: { stubs, directives: { loading: () => {} } } })
  await flushPromises()
}
async function setSelect(placeholder, value) {
  const component = select(placeholder)
  component.vm.$emit('update:modelValue', value)
  await flushPromises()
}
const headerOrder = () => wrapper.findAll('.column-stub [data-column-id]').map(item => item.attributes('data-column-id'))
const header = id => wrapper.find(`.column-stub [data-column-id="${id}"]`)
function mockHeaderGeometry() {
  const bounds = (left, width = 100) => ({ x: left, y: 120, left, right: left + width, top: 120, bottom: 160, width, height: 40 })
  vi.spyOn(wrapper.find('.table-stub').element, 'getBoundingClientRect').mockImplementation(() => bounds(100, 600))
  for (const item of wrapper.findAll('.column-stub')) {
    vi.spyOn(item.element, 'getBoundingClientRect').mockImplementation(() => {
      const index = Array.from(item.element.parentElement.children).indexOf(item.element)
      return bounds(100 + index * 100)
    })
  }
  for (const item of wrapper.findAll('.column-stub [data-column-id]')) {
    vi.spyOn(item.element, 'getBoundingClientRect').mockImplementation(() => item.element.closest('th').getBoundingClientRect())
  }
}
async function pointer(target, type, x, y = 140) {
  const event = new MouseEvent(type, { bubbles: true, cancelable: true, clientX: x, clientY: y, button: 0, buttons: type === 'pointerup' ? 0 : 1 })
  Object.defineProperties(event, { pointerId: { value: 1 }, pointerType: { value: 'mouse' }, isPrimary: { value: true } })
  target.dispatchEvent(event)
  await flushPromises()
  return event
}
async function beginHeaderDrag(source, x) {
  const center = header(source).element.getBoundingClientRect().left + 50
  await pointer(header(source).element, 'pointerdown', center)
  await pointer(window, 'pointermove', x)
}
beforeEach(() => {
  mocks.loadReportColumns.mockResolvedValue(registry)
  mocks.loadReportOptions.mockResolvedValue({ brands: [], categories: [], departments: ['历史停用部门', '销售部'] })
  mocks.listReportSchemes.mockResolvedValue({ list: [], total: 0, permissions: { create: true, update: true, delete: true, shared: true } })
  mocks.queryReport.mockImplementation(async request => ({
    list: [], columns: [...request.columns], total: 101, page: request.page, pageSize: request.pageSize,
    summary: { amount: '24.00', quantity: '3.0000', baseQuantity: '8.0000000000' }, warnings: []
  }))
  mocks.createReportScheme.mockImplementation(async data => ({ ...data, id: 8, ownerId: 12, version: 1, columnsJson: JSON.stringify(data.columns) }))
  mocks.printReport.mockResolvedValue({ title: '送货对账表', projection: { columns: ids, list: [], summary: {}, total: 0, warnings: [] } })
  vi.stubGlobal('print', vi.fn())
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  host?.remove()
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

it('starts collapsed, keeps one scheme/column panel, defaults both audited states and preserves global summaries', async () => {
  await start()
  expect(wrapper.find('.report-sidebar').exists()).toBe(false)
  expect(wrapper.find('[aria-label="展开报表方案操作面板"]').exists()).toBe(true)
  expect(mocks.queryReport).toHaveBeenCalledWith({ filters: { statuses: defaults }, columns: ids, page: 1, pageSize: 20 })
  expect(wrapper.find('.table-summary').text()).toBe('合计||24.00|3.0000|—')
  expect(wrapper.find('.result-alert').exists()).toBe(false)
  expect(select('已审核、已复审').props('multiple')).not.toBeUndefined()
  expect(select('全部历史部门').text()).toContain('历史停用部门')
  for (const field of ['quantity', 'unitPrice', 'amount']) {
    expect(wrapper.find(`[data-property="${field}"]`).attributes('data-align')).toBe('center')
  }
  await wrapper.find('[aria-label="展开报表方案操作面板"]').trigger('click')
  expect(wrapper.find('.report-sidebar').text()).toContain('报表方案操作面板')
  expect(wrapper.find('.report-sidebar').text()).toContain('显示列')
  expect(wrapper.find('.report-sidebar').text()).toContain('保存编辑')
  expect(wrapper.find('el-tabs').exists()).toBe(false)
  await wrapper.find('[aria-label="收起报表方案操作面板"]').trigger('click')
  expect(wrapper.find('.report-sidebar').exists()).toBe(false)
})

it('applies status/department only on query and reuses their snapshot for paging, export and print', async () => {
  await start()
  await setSelect('已审核、已复审', ['draft', 'reReviewed'])
  await setSelect('全部历史部门', '历史停用部门')
  expect(mocks.queryReport).toHaveBeenCalledTimes(1)
  expect(wrapper.find('.result-alert').exists()).toBe(true)
  await button('导出全部').trigger('click')
  expect(mocks.exportReport.mock.lastCall[0].filters).toEqual({ statuses: defaults })
  await button('查询').trigger('click')
  await flushPromises()
  const filters = { statuses: ['draft', 'reReviewed'], department: '历史停用部门' }
  expect(mocks.queryReport.mock.lastCall[0].filters).toEqual(filters)
  const pagination = wrapper.findComponent(Pagination)
  pagination.vm.$emit('update:currentPage', 2)
  pagination.vm.$emit('current-change', 2)
  await flushPromises()
  expect(mocks.queryReport.mock.lastCall[0]).toMatchObject({ filters, page: 2 })
  await button('导出全部').trigger('click')
  await button('打印全部').trigger('click')
  await flushPromises()
  expect(mocks.exportReport.mock.lastCall[0]).toMatchObject({ filters, page: 1 })
  expect(mocks.printReport.mock.lastCall[0]).toMatchObject({ filters, page: 1 })
  await button('重置').trigger('click')
  await flushPromises()
  expect(mocks.queryReport.mock.lastCall[0].filters).toEqual({ statuses: defaults })
})

it('previews insertion while held and commits its order only on release for results, export, print, queries and schemes', async () => {
  await start()
  mockHeaderGeometry()
  const ordered = ['amount', 'deliveryDay', 'noteNumber', 'quantity', 'unitPrice']
  await beginHeaderDrag('amount', 205)
  expect(headerOrder()).toEqual(ordered)
  expect(document.body.querySelector('.report-column-drag-ghost')).not.toBeNull()
  expect(document.body.querySelector('.report-column-insertion')).not.toBeNull()
  const ghostStyle = document.body.querySelector('.report-column-drag-ghost').getAttribute('style')
  await pointer(window, 'pointermove', 215)
  expect(document.body.querySelector('.report-column-drag-ghost').getAttribute('style')).not.toBe(ghostStyle)
  expect(mocks.queryReport).toHaveBeenCalledTimes(1)
  expect(wrapper.find('.result-alert').exists()).toBe(false)
  expect(wrapper.find('.table-summary').text()).toBe('合计||24.00|3.0000|—')
  await button('导出全部').trigger('click')
  await button('打印全部').trigger('click')
  await flushPromises()
  expect(mocks.exportReport.mock.lastCall[0].columns).toEqual(ids)
  expect(mocks.printReport.mock.lastCall[0].columns).toEqual(ids)
  await pointer(window, 'pointerup', 215)
  expect(headerOrder()).toEqual(ordered)
  expect(document.body.querySelector('.report-column-drag-ghost')).toBeNull()
  expect(document.body.querySelector('.report-column-insertion')).toBeNull()
  await header('amount').trigger('click')
  expect(mocks.queryReport).toHaveBeenCalledTimes(1)
  expect(wrapper.find('.result-alert').exists()).toBe(false)
  await button('导出全部').trigger('click')
  await button('打印全部').trigger('click')
  await flushPromises()
  expect(mocks.exportReport.mock.lastCall[0].columns).toEqual(ordered)
  expect(mocks.printReport.mock.lastCall[0].columns).toEqual(ordered)
  await button('查询').trigger('click')
  await flushPromises()
  expect(mocks.queryReport.mock.lastCall[0].columns).toEqual(ordered)
  await wrapper.find('[aria-label="展开报表方案操作面板"]').trigger('click')
  await button('新建方案').trigger('click')
  await wrapper.find('.dialog-stub input').setValue('拖拽方案')
  await button('保存').trigger('click')
  await flushPromises()
  expect(mocks.createReportScheme).toHaveBeenCalledWith({ name: '拖拽方案', scope: 'personal', columns: ordered })
})

it('inserts a column to the right without swapping the intervening columns', async () => {
  await start()
  mockHeaderGeometry()
  const ordered = ['noteNumber', 'quantity', 'unitPrice', 'deliveryDay', 'amount']
  await beginHeaderDrag('deliveryDay', 595)
  expect(headerOrder()).toEqual(ordered)
  await pointer(window, 'pointerup', 595)
  expect(headerOrder()).toEqual(ordered)
  await button('导出全部').trigger('click')
  await flushPromises()
  expect(mocks.exportReport.mock.lastCall[0].columns).toEqual(ordered)
  expect(mocks.queryReport).toHaveBeenCalledTimes(1)
})

it.each(['Escape', 'pointercancel', 'blur', 'outside release'])('cancels a live preview on %s without committing it', async cancellation => {
  await start()
  mockHeaderGeometry()
  await beginHeaderDrag('amount', 205)
  expect(headerOrder()).not.toEqual(ids)
  if (cancellation === 'Escape') window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
  else if (cancellation === 'blur') window.dispatchEvent(new Event('blur'))
  else await pointer(window, cancellation === 'pointercancel' ? 'pointercancel' : 'pointerup', 20, 300)
  await flushPromises()
  expect(headerOrder()).toEqual(ids)
  expect(document.body.querySelector('.report-column-drag-ghost')).toBeNull()
  expect(document.body.querySelector('.report-column-insertion')).toBeNull()
  await pointer(window, 'pointerup', 205)
  await button('导出全部').trigger('click')
  await button('打印全部').trigger('click')
  await flushPromises()
  expect(mocks.exportReport.mock.lastCall[0].columns).toEqual(ids)
  expect(mocks.printReport.mock.lastCall[0].columns).toEqual(ids)
  expect(mocks.queryReport).toHaveBeenCalledTimes(1)
  expect(wrapper.find('.result-alert').exists()).toBe(false)
})

it('keeps sub-threshold mouse motion as an ordinary sortable header click', async () => {
  await start()
  mockHeaderGeometry()
  await pointer(header('quantity').element, 'pointerdown', 450)
  await pointer(window, 'pointermove', 453, 141)
  expect(headerOrder()).toEqual(ids)
  expect(document.body.querySelector('.report-column-drag-ghost')).toBeNull()
  await pointer(window, 'pointerup', 453, 141)
  await header('quantity').trigger('click')
  await flushPromises()
  expect(mocks.queryReport.mock.lastCall[0]).toMatchObject({ columns: ids, sortBy: 'quantity', sortOrder: 'asc' })
  expect(mocks.queryReport).toHaveBeenCalledTimes(2)
  expect(headerOrder()).toEqual(ids)
})

it('supports keyboard header reordering without changing global summary or selecting the fixed index column', async () => {
  await start()
  await wrapper.find('[data-column-id="deliveryDay"]').trigger('keydown', { key: 'ArrowLeft', altKey: true })
  expect(wrapper.findAll('[data-column-id]').map(item => item.attributes('data-column-id'))).toEqual(ids)
  await wrapper.find('[data-column-id="quantity"]').trigger('keydown', { key: 'ArrowLeft', altKey: true })
  expect(wrapper.findAll('[data-column-id]').map(item => item.attributes('data-column-id'))).toEqual(['deliveryDay', 'quantity', 'noteNumber', 'unitPrice', 'amount'])
  expect(wrapper.find('.table-summary').text()).toBe('合计||24.00|3.0000|—')
})

it('loads a scheme from search without changing filters and respects mutation permissions', async () => {
  const scheme = { id: 3, name: '共享方案', scope: 'shared', ownerId: 7, version: 2, columnsJson: '["amount","quantity"]' }
  mocks.listReportSchemes.mockResolvedValue({ list: [scheme], total: 0, permissions: { create: false, update: true, delete: true, shared: false } })
  mocks.getReportScheme.mockResolvedValue(scheme)
  await start()
  await setSelect('全部历史部门', '销售部')
  select('默认列（未命名方案）').vm.$emit('change', 3)
  await flushPromises()
  expect(mocks.getReportScheme).toHaveBeenCalledWith(3)
  expect(mocks.queryReport).toHaveBeenCalledTimes(1)
  await wrapper.find('[aria-label="展开报表方案操作面板"]').trigger('click')
  expect(button('新建方案').attributes('disabled')).toBeDefined()
  expect(button('保存编辑').attributes('disabled')).toBeDefined()
  expect(button('删除方案').attributes('disabled')).toBeDefined()
  await button('查询').trigger('click')
  await flushPromises()
  expect(mocks.queryReport.mock.lastCall[0]).toMatchObject({ columns: ['amount', 'quantity'], filters: { department: '销售部', statuses: defaults } })
})

it('fills only the active tab panel with results, retains paging and exits on Escape', async () => {
  await start()
  await wrapper.find('[aria-label="结果全屏"]').trigger('click')
  const result = document.body.querySelector('.report-results-fullscreen')
  expect(result).not.toBeNull()
  expect(result.parentElement).toBe(document.body)
  expect(result.style.top).toBe('110px')
  expect(result.style.left).toBe('210px')
  expect(result.style.width).toBe(`${Math.min(1000, window.innerWidth) - 210}px`)
  expect(result.style.height).toBe(`${Math.min(800, window.innerHeight) - 110}px`)
  expect(result.querySelector('.pagination-stub')).not.toBeNull()
  window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
  await flushPromises()
  expect(document.body.querySelector('.report-results-fullscreen')).toBeNull()
  expect(wrapper.find('.report-results').exists()).toBe(true)
  expect(mocks.queryReport).toHaveBeenCalledTimes(1)
})

it('offers nine Shanghai-calendar date shortcuts across year, week and leap-month boundaries', async () => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(new Date('2025-12-31T16:30:00Z'))
  await start()
  const pickers = wrapper.findAllComponents(DatePicker)
  const shortcuts = pickers[0].props('shortcuts')
  expect(pickers[1].props('shortcuts')).toBe(shortcuts)
  const format = date => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  const ranges = Object.fromEntries(shortcuts.map(item => [item.text, item.value().map(format)]))
  expect(ranges).toEqual({
    今天: ['2026-01-01', '2026-01-01'], 昨天: ['2025-12-31', '2025-12-31'],
    今年: ['2026-01-01', '2026-12-31'], 去年: ['2025-01-01', '2025-12-31'],
    上周: ['2025-12-22', '2025-12-28'], 本周: ['2025-12-29', '2026-01-04'],
    本月: ['2026-01-01', '2026-01-31'], 上月: ['2025-12-01', '2025-12-31'], 前月: ['2025-11-01', '2025-11-30']
  })
  vi.setSystemTime(new Date('2024-03-01T04:00:00Z'))
  expect(shortcuts.find(item => item.text === '上月').value().map(format)).toEqual(['2024-02-01', '2024-02-29'])
  pickers[0].vm.$emit('update:modelValue', ['2024-02-01', '2024-02-29'])
  await button('查询').trigger('click')
  await flushPromises()
  expect(mocks.queryReport.mock.lastCall[0].filters).toMatchObject({ deliveredFrom: '2024-02-01T00:00:00+08:00', deliveredTo: '2024-03-01T00:00:00+08:00' })
})
