import { afterEach, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const mocks = vi.hoisted(() => ({
  loadReportColumns: vi.fn(),
  loadReportOptions: vi.fn(),
  queryReport: vi.fn(),
  exportReport: vi.fn(),
  printReport: vi.fn(),
  listReportSchemes: vi.fn(),
  getReportScheme: vi.fn(),
  createReportScheme: vi.fn(),
  updateReportScheme: vi.fn(),
  deleteReportScheme: vi.fn(),
  publishReportScheme: vi.fn(),
  showError: vi.fn(),
  warning: vi.fn(),
  success: vi.fn()
}))

vi.mock('@/pinia/modules/user', () => ({ useUserStore: () => ({ userInfo: { id: 12 } }) }))
vi.mock('element-plus', () => ({
  ElMessage: { warning: mocks.warning, success: mocks.success },
  ElMessageBox: { confirm: vi.fn() }
}))
vi.mock('../api/common', () => ({
  idOf: value => value?.id,
  pageSequenceNumber: (index, page, pageSize) => (page - 1) * pageSize + index + 1,
  showError: mocks.showError
}))
vi.mock('../api/reconciliation', () => ({
  loadReportColumns: mocks.loadReportColumns,
  loadReportOptions: mocks.loadReportOptions,
  queryReport: mocks.queryReport,
  exportReport: mocks.exportReport,
  printReport: mocks.printReport,
  listReportSchemes: mocks.listReportSchemes,
  getReportScheme: mocks.getReportScheme,
  createReportScheme: mocks.createReportScheme,
  updateReportScheme: mocks.updateReportScheme,
  deleteReportScheme: mocks.deleteReportScheme,
  publishReportScheme: mocks.publishReportScheme
}))
vi.mock('../api/table-widths', () => ({ getTableWidths: vi.fn(async () => ({})), saveTableWidths: vi.fn(async () => ({})) }))

import Reconciliation from './reconciliation.vue'

let wrapper

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.clearAllMocks()
})

it('renders the report controls and queries the selected columns on initial load', async () => {
  const available = [
    { id: 'deliveryDay', label: '送货日', kind: 'dimension' },
    { id: 'noteNumber', label: '送货单号', kind: 'dimension' },
    { id: 'brand', label: '客户', kind: 'dimension' },
    { id: 'materialCode', label: '物料编号', kind: 'dimension' },
    { id: 'materialName', label: '物料名称', kind: 'dimension' },
    { id: 'specification', label: '规格', kind: 'dimension' },
    { id: 'unit', label: '单位', kind: 'dimension' },
    { id: 'quantity', label: '数量', kind: 'metric' },
    { id: 'unitPrice', label: '单价', kind: 'metric' },
    { id: 'amount', label: '金额', kind: 'metric' }
  ]
  mocks.loadReportColumns.mockResolvedValue(available)
  mocks.loadReportOptions.mockResolvedValue({ brands: [], categories: [] })
  mocks.listReportSchemes.mockResolvedValue({ list: [], total: 0, permissions: { create: false, update: false, delete: false, shared: false } })
  mocks.queryReport.mockResolvedValue({
    list: [],
    columns: ['deliveryDay', 'amount', 'quantity', 'unitPrice'],
    total: 0,
    page: 1,
    pageSize: 20,
    summary: {
      amount: '24.00',
      quantity: '3.0000',
      baseQuantity: '8.0000000000'
    },
    warnings: []
  })

  wrapper = mount(Reconciliation, {
    global: {
      stubs: {
        'el-form-item': {
          props: ['label'],
          template: '<div><label>{{ label }}</label><slot /></div>'
        },
        'el-table': {
          props: ['summaryMethod', 'showSummary'],
          template: '<div class="table-stub"><slot /><div v-if="showSummary" class="table-summary">{{ summaryMethod({ columns: [{}, { property: "deliveryDay" }, { property: "amount" }, { property: "quantity" }, { property: "unitPrice" }] }).join("|") }}</div></div>'
        },
        'el-table-column': { template: '<div />' }
      },
      directives: { loading: () => {} }
    }
  })
  await flushPromises()

  expect(wrapper.text()).toContain('对账表')
  expect(wrapper.text()).toContain('报表设置')
  expect(wrapper.text()).toContain('显示顺序')
  expect(wrapper.text()).toContain('送货日期')
  expect(wrapper.text()).toContain('客户快照')
  expect(wrapper.text()).toContain('物料关键字')
  expect(wrapper.text()).toContain('导出全部')
  expect(wrapper.text()).toContain('打印全部')
  expect(wrapper.find('.report-filters').exists()).toBe(true)
  expect(wrapper.find('.results-heading').exists()).toBe(true)
  expect(wrapper.find('.result-alert').exists()).toBe(false)
  expect(wrapper.find('.table-summary').text()).toBe('合计||24.00|3.0000|—')
  expect(wrapper.text()).not.toContain('全局金额合计：24.00')
  expect(mocks.queryReport).toHaveBeenCalledWith({ filters: {}, columns: available.slice(0, 10).map(column => column.id), page: 1, pageSize: 20 })
})
