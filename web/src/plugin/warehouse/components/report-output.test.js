import { afterEach, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ReportPrintView from './report-print-view.vue'
import ReportSummary from './report-summary.vue'

let wrapper
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = '' })

it('renders incomparable nulls as a dash and prints untrusted names as text', () => {
  const name = '<img src=x onerror=alert(1)>'
  wrapper = mount(ReportPrintView, { props: {
    registry: [{ id: 'materialName', label: '物料' }, { id: 'quantity', label: '数量' }],
    data: { title: '送货对账表', filterDescription: '已审核', projection: {
      total: 1, columns: ['materialName', 'quantity'], summary: {}, warnings: [],
      list: [{ values: { materialName: name, quantity: null }, unavailable: { quantity: '混合单位' } }]
    } }
  }, attachTo: document.body })
  const printed = document.querySelector('.warehouse-print-sheet')
  expect(printed.textContent).toContain(name)
  expect(printed.querySelector('img')).toBeNull()
  expect(printed.querySelectorAll('td')[1].textContent).toContain('—')
  expect(printed.querySelectorAll('td')[1].textContent).toContain('混合单位')
})

it('never renders quantity buckets when quantity is not part of the report projection', () => {
  wrapper = mount(ReportSummary, { props: { columns: ['brand', 'amount'], summary: {
    amount: '68.50', quantityBuckets: [{ materialId: 1, materialName: '物料', unitName: '箱', value: '2.0000' }]
  } } })
  expect(wrapper.text()).toContain('68.50')
  expect(wrapper.text()).not.toContain('2.0000')
  expect(wrapper.find('ul').exists()).toBe(false)
})
