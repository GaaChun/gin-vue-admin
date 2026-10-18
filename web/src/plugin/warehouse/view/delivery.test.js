import { afterEach, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ElMessageBox } from 'element-plus'
import Delivery from './delivery.vue'
import { fmtTitle } from '@/utils/fmtRouterTitle'

const mocks = vi.hoisted(() => ({
  request: vi.fn(),
  allOptions: vi.fn(),
  showError: vi.fn(),
  push: vi.fn()
}))

vi.mock('element-plus', () => ({
  ElMessage: { error: vi.fn(), success: vi.fn(), warning: vi.fn() },
  ElMessageBox: { confirm: vi.fn(() => Promise.resolve()) }
}))
vi.mock('vue-router', () => ({
  useRoute: () => ({ name: 'warehouseDelivery', query: { whDelivery: '1', whId: '10' } }),
  useRouter: () => ({ push: mocks.push })
}))
vi.mock('../api/common', () => ({
  request: mocks.request,
  allOptions: mocks.allOptions,
  idOf: value => value?.ID ?? value?.id,
  showError: mocks.showError
}))
vi.mock('../components/delivery-material-picker.vue', () => ({ default: { props: ['materials'], template: '<div class="picker-materials">{{ materials.map(material => material.code).join(",") }}</div>' } }))

let wrapper
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.clearAllMocks()
})

it('bulk reviews only selected drafts using each note version', async () => {
  mocks.allOptions.mockResolvedValue([])
  const rows = [
    { id: 11, version: 2, number: 'SK260101000011', status: 'draft' },
    { id: 12, version: 4, number: 'SK260101000012', status: 'reviewed' }
  ]
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/list')) return { list: rows, total: 2, page: 1, pageSize: 20 }
    if (url.endsWith('/review')) return {}
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data'], emits: ['selection-change'], template: '<div><button class="select-rows" @click="$emit(\'selection-change\', data)">select</button><slot /></div>' },
    'el-table-column': { props: ['type','selectable'], data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test' } }), template: '<div><slot :row="fakeRow" /></div>' }
  } } })
  await flushPromises()
  await wrapper.find('.select-rows').trigger('click')
  await flushPromises()
  const bulk = wrapper.findAll('el-button').find(button => button.text().trim() === '批量审核')
  expect(bulk.attributes('disabled')).toBe('false')
  await bulk.trigger('click')
  await flushPromises()
  expect(mocks.request).toHaveBeenCalledWith({ url: '/warehouse/delivery/review', method: 'post', data: { id: 11, version: 2 } })
  expect(mocks.request.mock.calls.filter(([arg]) => arg.url.endsWith('/review'))).toHaveLength(1)
})

it('confirms and sends delete for an editable delivery note', async () => {
  mocks.allOptions.mockResolvedValue([])
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/detail')) return { note: { id: 10, number: 'SK260101000001', status: 'draft', brandId: 2, deliveredAt: '2026-01-01T00:00:00Z' }, details: [] }
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    if (url.endsWith('/delete')) return { success: true }
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, {
    global: {
      stubs: {
        'el-table': { props: ['data'], template: '<div v-if="data?.length"><slot /></div>' },
        'el-table-column': { data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test' } }), template: '<div><slot :row="fakeRow" /></div>' }
      }
    }
  })
  await flushPromises()
  const deleteButton = wrapper.findAll('el-button').find(button => button.text().trim() === '删除')
  expect(deleteButton, 'editable delivery delete button').toBeTruthy()
  await deleteButton.trigger('click')
  await flushPromises()

  expect(ElMessageBox.confirm).toHaveBeenCalledWith('确定删除送货单“SK260101000001”？', '删除确认', expect.objectContaining({ type: 'warning' }))
  expect(mocks.request).toHaveBeenCalledWith({ url: '/warehouse/delivery/delete', method: 'delete', data: { id: 10, version: undefined } })
})

it('passes materials from every customer to the picker', async () => {
  mocks.allOptions.mockImplementation(async kind => kind === 'materials' ? [{ id: 7, code: 'M-OTHER', brandId: 99 }] : [])
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/detail')) return { note: { id: 10, number: 'SK260101000001', status: 'draft', brandId: 2, deliveredAt: '2026-01-01T00:00:00Z' }, details: [] }
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, {
    global: {
      stubs: {
        'el-table': { props: ['data'], template: '<div v-if="data?.length"><slot /></div>' },
        'el-table-column': { data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test' } }), template: '<div><slot :row="fakeRow" /></div>' }
      }
    }
  })
  await flushPromises()
  expect(wrapper.find('.picker-materials').text()).toContain('M-OTHER')
})

it('searches the delivery customer by name, mnemonic, and code', async () => {
  mocks.allOptions.mockImplementation(async kind => kind === 'brands' ? [
    { id: 2, name: '食品集团', mnemonic: 'shipin', code: 'KH-01' },
    { id: 3, name: '清洁用品', mnemonic: 'qingjie', code: 'KH-02' }
  ] : [])
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/detail')) return { note: { id: 10, number: 'SK260101000001', status: 'draft', brandId: 2, deliveredAt: '2026-01-01T00:00:00Z' }, details: [] }
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, {
    global: {
      stubs: {
        'el-table': { props: ['data'], template: '<div v-if="data?.length"><slot /></div>' },
        'el-table-column': { data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test' } }), template: '<div><slot :row="fakeRow" /></div>' },
        'el-select': { props: ['filterMethod'], template: '<div><input v-if="filterMethod" class="select-search" @input="filterMethod($event.target.value)"><slot /></div>' },
        'el-option': { props: ['label'], template: '<div class="option">{{ label }}<slot /></div>' }
      }
    }
  })
  await flushPromises()
  expect(wrapper.find('.delivery-detail-page el-date-picker').attributes('type')).toBe('date')
  const input = wrapper.find('.brand-select .select-search')
  for (const query of ['食品', 'shipin', 'KH-01']) {
    await input.setValue(query)
    await flushPromises()
    const options = wrapper.find('.delivery-detail-page .brand-select').findAll('.option').map(option => option.text())
    const matchingOptions = options.filter(option => option.includes('KH-'))
    expect(matchingOptions).toHaveLength(1)
    expect(matchingOptions[0]).toContain('食品集团')
  }
})

it('saves delivery date at midnight Shanghai time without seconds input', async () => {
  mocks.allOptions.mockResolvedValue([])
  const aggregate = {
    note: { id: 10, version: 1, number: 'SK260101000001', status: 'draft', brandId: 2, deliveredAt: '2026-05-09T13:15:45Z' },
    details: [{ id: 4, materialId: 7, materialUnitId: 8, unitId: 9, baseUnitId: 9, factor: '1', quantity: '1', unitPrice: '2', materialCodeSnapshot: 'M1', materialNameSnapshot: '物料', unitNameSnapshot: '件' }]
  }
  mocks.request.mockImplementation(async ({ url }) => {
    if (url === '/warehouse/master/materials/detail') return { units: [{ ID: 8, unitId: 9, factor: '1', enabled: true, isBase: true, name: '件' }] }
    if (url.endsWith('/detail')) return aggregate
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    if (url.endsWith('/update')) return aggregate
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, {
    global: {
      stubs: {
        'el-table': { props: ['data'], template: '<div v-if="data?.length"><slot /></div>' },
        'el-table-column': { data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test' } }), template: '<div><slot :row="fakeRow" /></div>' }
      }
    }
  })
  await flushPromises()
  expect(wrapper.find('.delivery-detail-page el-date-picker').attributes('type')).toBe('date')
  await wrapper.findAll('el-button').find(button => button.text().trim() === '保存').trigger('click')
  await flushPromises()
  expect(mocks.request).toHaveBeenCalledWith(expect.objectContaining({
    url: '/warehouse/delivery/update',
    data: expect.objectContaining({ deliveredAt: '2026-05-09T00:00:00+08:00' })
  }))
})

it('saves the edited note and routes to a cleared new-note form', async () => {
  mocks.allOptions.mockResolvedValue([])
  const aggregate = {
    note: { id: 10, version: 2, number: 'SK260101000010', status: 'draft', brandId: 2, deliveredAt: '2026-01-01T00:00:00Z' },
    details: [{ id: 4, materialId: 7, materialUnitId: 8, unitId: 9, baseUnitId: 9, factor: '1', quantity: '1', unitPrice: '2', materialCodeSnapshot: 'M1', materialNameSnapshot: '物料', unitNameSnapshot: '件' }]
  }
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/detail')) return aggregate
    if (url.endsWith('/update')) return aggregate
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    if (url === '/warehouse/master/materials/detail') return { units: [{ ID: 8, unitId: 9, factor: '1', enabled: true, isBase: true, name: '件' }] }
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data'], template: '<div><slot /></div>' },
    'el-table-column': { data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test' } }), template: '<div><slot :row="fakeRow" /></div>' }
  } } })
  await flushPromises()
  await wrapper.findAll('el-button').find(button => button.text().trim() === '保存并继续新增').trigger('click')
  await flushPromises()
  expect(mocks.request).toHaveBeenCalledWith(expect.objectContaining({
    url: '/warehouse/delivery/update',
    data: expect.objectContaining({ id: 10, version: 2 })
  }))
  expect(mocks.push).toHaveBeenCalledWith(expect.objectContaining({
    query: expect.objectContaining({ whDelivery: '1', whNew: '1' })
  }))
})

it('labels create and edit route tabs with the operation', () => {
  expect(fmtTitle('客户', { params: {}, query: { whDetail: 'brands', whNew: '1' } })).toBe('客户-新建')
  expect(fmtTitle('客户', { params: {}, query: { whDetail: 'brands', whId: '4' } })).toBe('客户-编辑')
  expect(fmtTitle('送货单', { params: {}, query: { whDelivery: '1', whNew: '1' } })).toBe('送货单-新建')
})
