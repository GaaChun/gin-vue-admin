import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import Master from './master.vue'

const mocks = vi.hoisted(() => ({
  route: null,
  push: vi.fn(),
  replace: vi.fn(),
  reserveCode: vi.fn(),
  detail: vi.fn(),
  save: vi.fn(),
  showError: vi.fn()
}))
vi.mock('vue-router', () => ({ useRoute: () => mocks.route, useRouter: () => ({ push: mocks.push, replace: mocks.replace }) }))
vi.mock('@/api/user', () => ({ getUserList: async () => ({ code: 0, data: { list: [] } }) }))
vi.mock('@/pinia/modules/user', () => ({ useUserStore: () => ({ userInfo: { ID: 1 } }) }))
vi.mock('@/utils/format', () => ({ formatDate: value => String(value) }))
vi.mock('../api/table-widths', () => ({ getTableWidths: async () => ({}), saveTableWidths: vi.fn() }))
vi.mock('../api/common', () => ({
  allOptions: async kind => kind === 'brands' ? [{ id: 2, name: '客户甲', code: 'KH000002' }] : [],
  idOf: item => item?.ID ?? item?.id,
  pageSequenceNumber: (index, page, size) => (page - 1) * size + index + 1,
  showError: mocks.showError
}))
vi.mock('../api/master', () => ({
  listMaster: async () => ({ list: [], total: 0 }),
  getCategoryTree: async () => [],
  reserveMasterCode: mocks.reserveCode,
  detailMaster: mocks.detail,
  saveMaster: mocks.save,
  suggestMasterMnemonic: async () => ({ mnemonic: 'KH' }),
  deleteMaster: vi.fn(),
  downloadMasterTemplate: vi.fn(),
  exportMaster: vi.fn(),
  importMaster: vi.fn(),
  uploadMasterImage: vi.fn()
}))

let wrapper
beforeEach(() => {
  mocks.route = reactive({ name: 'warehouseBrand', query: { whDetail: 'brands', whNew: '1' } })
  mocks.reserveCode.mockResolvedValue({ code: 'KH000001' })
  mocks.detail.mockResolvedValue({ id: 9, code: 'EXISTING-9', name: '原客户', mnemonic: 'YKH', version: 4 })
  mocks.save.mockImplementation(async (resource, data) => ({ ...data, id: data.id || 9, version: (data.version || 0) + 1 }))
})
afterEach(() => { wrapper?.unmount(); wrapper = undefined; vi.clearAllMocks() })
function render() {
  wrapper = mount(Master, { attachTo: document.body, global: { plugins: [ElementPlus] } })
}
function field(label) {
  return wrapper.findAll('.master-editor-grid .el-form-item').find(item => {
    const fieldLabel = item.find('label')
    return fieldLabel.exists() && fieldLabel.text() === label
  })
}

it.each([
  ['warehouseBrand', 'brands', 'KH000001'],
  ['warehouseCategory', 'categories', 'LB000001'],
  ['warehouseMaterial', 'materials', 'WL000001']
])('prefills a reserved editable code for %s without changing it on save', async (routeName, resource, code) => {
  mocks.route.name = routeName
  mocks.route.query.whDetail = resource
  mocks.reserveCode.mockResolvedValue({ code })
  render()
  await flushPromises()
  expect(mocks.reserveCode).toHaveBeenCalledExactlyOnceWith(resource)
  const codeInput = field('编号').find('input')
  expect(codeInput.element.value).toBe(code)
  expect(codeInput.attributes('readonly')).toBeUndefined()
  await codeInput.setValue('MANUAL-42')
  await field('名称').find('input').setValue('手工资料')
  await field('助记符').find('input').setValue('SGZL')
  await field('助记符').find('input').trigger('input')
  if (resource === 'materials') {
    // Units have their own existing validation; the code is still in the payload unchanged.
    wrapper.vm.$.setupState.form.units = [{ unitId: 1, factor: '1', isBase: true, enabled: true }]
  }
  const save = wrapper.findAll('button').find(button => button.text() === '保存')
  await save.trigger('click')
  await flushPromises()
  expect(mocks.save).toHaveBeenCalledWith(resource, expect.objectContaining({ code: 'MANUAL-42', name: '手工资料' }))
})

it('never allocates codes for units or overwrites an existing master code', async () => {
  mocks.route.name = 'warehouseUnit'
  mocks.route.query = { whDetail: 'units', whNew: '1' }
  render()
  await flushPromises()
  expect(mocks.reserveCode).not.toHaveBeenCalled()
  expect(field('编号')).toBeUndefined()
  mocks.route.name = 'warehouseBrand'
  mocks.route.query = { whDetail: 'brands', whId: '9' }
  await flushPromises()
  expect(mocks.detail).toHaveBeenCalledWith('brands', 9)
  expect(field('编号').find('input').element.value).toBe('EXISTING-9')
  expect(mocks.reserveCode).not.toHaveBeenCalled()
})

it('discards a stale code response after changing master resources', async () => {
  let resolveOld
  mocks.reserveCode.mockImplementation(resource => resource === 'brands' ? new Promise(resolve => { resolveOld = resolve }) : Promise.resolve({ code: 'LB000001' }))
  render()
  await flushPromises()
  mocks.route.name = 'warehouseCategory'
  mocks.route.query = { whDetail: 'categories', whNew: '1' }
  await flushPromises()
  resolveOld({ code: 'KH000999' })
  await flushPromises()
  expect(field('编号').find('input').element.value).toBe('LB000001')
})

it('shows the selected customer name only and renders name/code in separate candidate columns', async () => {
  mocks.route.name = 'warehouseMaterial'
  mocks.route.query = { whDetail: 'materials', whNew: '1' }
  render()
  await flushPromises()
  const customer = field('客户').findComponent({ name: 'ElSelect' })
  customer.vm.$emit('update:modelValue', 2)
  await flushPromises()
  expect(customer.find('.el-select__selected-item.el-select__placeholder').text()).toBe('客户甲')
  const options = wrapper.findAllComponents({ name: 'ElOption' }).filter(option => option.props('value') === 2)
  expect(options.length).toBeGreaterThan(0)
  for (const option of options) {
    expect(option.props('label')).toBe('客户甲')
    expect(option.find('.master-option > span').text()).toBe('客户甲')
    expect(option.find('.master-option-code').text()).toBe('KH000002')
  }
})

it('surfaces number allocation failure instead of inventing a local code', async () => {
  mocks.reserveCode.mockRejectedValue(new Error('编号服务不可用'))
  render()
  await flushPromises()
  expect(wrapper.find('.master-detail-page').text()).toContain('编号服务不可用')
  expect(field('编号').find('input').element.value).toBe('')
  expect(mocks.showError).toHaveBeenCalled()
})
