import { afterEach, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ElMessageBox } from 'element-plus'
import Delivery from './delivery.vue'
import { fmtTitle } from '@/utils/fmtRouterTitle'

const mocks = vi.hoisted(() => ({
  request: vi.fn(),
  allOptions: vi.fn(),
  showError: vi.fn(),
  push: vi.fn(),
  routeQuery: { whDelivery: '1', whId: '10' }
}))
vi.mock('element-plus', () => ({
  ElMessage: { error: vi.fn(), success: vi.fn(), warning: vi.fn() },
  ElMessageBox: { confirm: vi.fn(() => Promise.resolve()) }
}))
vi.mock('vue-router', () => ({
  useRoute: () => ({ name: 'warehouseDelivery', query: mocks.routeQuery }),
  useRouter: () => ({ push: mocks.push })
}))
vi.mock('../api/common', () => ({
  request: mocks.request,
  allOptions: mocks.allOptions,
  idOf: value => value?.ID ?? value?.id,
  pageSequenceNumber: (index, page, pageSize) => (page - 1) * pageSize + index + 1,
  showError: mocks.showError
}))
vi.mock('@/api/user', () => ({ getUserList: vi.fn(async () => ({ code: 0, data: { list: [] } })) }))
vi.mock('../components/delivery-material-picker.vue', () => ({ default: { props: ['materials'], template: '<div class="picker-materials">{{ materials.map(material => material.code).join(",") }}</div>' } }))
vi.mock('../api/table-widths', () => ({ getTableWidths: vi.fn(async () => ({})), saveTableWidths: vi.fn(async () => ({})) }))

let wrapper
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  mocks.routeQuery = { whDelivery: '1', whId: '10' }
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

it('quick-adds a material found by mnemonic to the delivery rows', async () => {
  mocks.allOptions.mockImplementation(async kind => {
    if (kind === 'materials') return [{ id: 7, code: 'M-07', name: '青菜', mnemonic: 'qingcai' }]
    if (kind === 'units') return [{ id: 9, name: '千克' }]
    return []
  })
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/materials/detail')) return { id: 7, code: 'M-07', name: '青菜', units: [{ id: 30, unitId: 9, name: '千克', enabled: true, isBase: true }] }
    if (url.endsWith('/detail')) return { note: { id: 10, number: 'SK260101000001', status: 'draft', brandId: 2, deliveredAt: '2026-01-01T00:00:00Z' }, details: [] }
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data'], template: '<div><span v-for="row in data" :key="row.rowKey">{{ row.materialName }}</span><slot /></div>' },
    'el-table-column': { data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test' } }), template: '<div><slot :row="fakeRow" /></div>' }
  } } })
  await flushPromises()
  let suggestions
  const state = wrapper.vm.$.setupState
  state.suggestQuickMaterials('qing', items => { suggestions = items })
  expect(suggestions).toEqual([expect.objectContaining({ id: 7, value: 'M-07 青菜' })])
  await state.quickAddMaterial(suggestions[0])
  await flushPromises()

  expect(mocks.request).toHaveBeenCalledWith({ url: '/warehouse/master/materials/detail', method: 'get', params: { id: 7 } })
  expect(wrapper.text()).toContain('青菜')
})

it('creates a missing material with a base unit and adds it to the delivery', async () => {
  mocks.allOptions.mockImplementation(async kind => kind === 'units' ? [{ id: 9, name: '千克' }] : [])
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/materials/create')) return {
      ID: 22,
      code: 'M-22',
      name: '新物料',
      mnemonic: 'xinwuliao',
      units: [{ ID: 35, unitId: 9, factor: '1', isBase: true, enabled: true }]
    }
    if (url.endsWith('/detail')) return { note: { id: 10, number: 'SK260101000001', status: 'draft', brandId: 2, deliveredAt: '2026-01-01T00:00:00Z' }, details: [] }
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data'], template: '<div><span v-for="row in data" :key="row.rowKey">{{ row.materialName }}</span><slot /></div>' },
    'el-table-column': { data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test' } }), template: '<div><slot :row="fakeRow" /></div>' }
  } } })
  await flushPromises()
  const state = wrapper.vm.$.setupState
  await wrapper.findAll('el-button').find(button => button.text().includes('新建物料并添加')).trigger('click')
  await flushPromises()
  expect(state.quickMaterialDialogOpen).toBe(true)
  Object.assign(state.quickMaterialForm, { name: '新物料', code: 'M-22', mnemonic: 'xinwuliao', unitId: 9 })
  await state.createQuickMaterial()
  await flushPromises()

  expect(mocks.request).toHaveBeenCalledWith({
    url: '/warehouse/master/materials/create',
    method: 'post',
    data: { name: '新物料', code: 'M-22', mnemonic: 'xinwuliao', units: [{ unitId: 9, factor: '1', isBase: true, enabled: true }] }
  })
  expect(wrapper.text()).toContain('新物料')
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

it('returns to the delivery list after normal save of a new note', async () => {
  sessionStorage.removeItem('warehouse.delivery.createDraft')
  mocks.routeQuery = { whDelivery: '1', whNew: '1' }
  mocks.allOptions.mockResolvedValue([])
  const saved = { note: { id: 20, version: 1, number: 'SK260101000020', status: 'draft', brandId: 2, deliveredAt: '2026-01-01T00:00:00Z' }, details: [] }
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/reserveNumber')) return { number: 'SK260101000020', token: 'reservation-20' }
    if (url.endsWith('/create')) return saved
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data'], template: '<div><slot /></div>' },
    'el-table-column': { data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test' } }), template: '<div><slot :row="fakeRow" /></div>' }
  } } })
  await flushPromises()
  const state = wrapper.vm.$.setupState
  state.form.brandId = 2
  state.form.deliveredAt = '2026-01-01'
  Object.assign(state.rows[0], { materialId: 7, materialUnitId: 8, quantity: '1', unitPrice: '2' })
  await wrapper.findAll('el-button').find(button => button.text().trim() === '保存').trigger('click')
  await flushPromises()

  expect(mocks.push).toHaveBeenCalledWith(expect.objectContaining({
    name: 'warehouseDelivery',
    query: {}
  }))
})

it('clears and reserves a new note when continuing from the existing new-note route', async () => {
  sessionStorage.removeItem('warehouse.delivery.createDraft')
  mocks.routeQuery = { whDelivery: '1', whNew: '1' }
  mocks.allOptions.mockResolvedValue([])
  const saved = { note: { id: 21, version: 1, number: 'SK260101000021', status: 'draft', brandId: 2, deliveredAt: '2026-01-01T00:00:00Z' }, details: [] }
  let reserveCount = 0
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/reserveNumber')) {
      reserveCount++
      return { number: `SK26010100002${reserveCount}`, token: `reservation-${reserveCount}` }
    }
    if (url.endsWith('/create')) return saved
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data'], template: '<div><slot /></div>' },
    'el-table-column': { data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test' } }), template: '<div><slot :row="fakeRow" /></div>' }
  } } })
  await flushPromises()
  const state = wrapper.vm.$.setupState
  state.form.brandId = 2
  state.form.department = '测试部门'
  state.form.deliveredAt = '2026-01-01'
  Object.assign(state.rows[0], { materialId: 7, materialUnitId: 8, quantity: '1', unitPrice: '2' })
  await wrapper.findAll('el-button').find(button => button.text().trim() === '保存并继续新增').trigger('click')
  await flushPromises()

  expect(reserveCount).toBe(2)
  expect(state.form.department).toBe('')
  expect(state.note.id).toBeUndefined()
  expect(state.rows).toHaveLength(1)
  expect(state.rows[0].materialId).toBeUndefined()
})

it('shows department immediately after customer in the delivery list', async () => {
  mocks.routeQuery = {}
  mocks.allOptions.mockResolvedValue([])
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/list')) return { list: [{ id: 10, department: '采购部', brandNameSnapshot: '客户A' }], total: 1, page: 1, pageSize: 20 }
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data'], template: '<div><slot /></div>' },
    'el-table-column': {
      props: ['label', 'prop', 'type'],
      data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test' } }),
      template: '<div class="column" :data-label="label" :data-prop="prop"><slot :row="fakeRow" /></div>'
    }
  } } })
  await flushPromises()

  const labels = wrapper.findAll('.column').map(column => column.attributes('data-label'))
  expect(labels.indexOf('部门')).toBe(labels.indexOf('客户') + 1)
  expect(wrapper.findAll('.column')[labels.indexOf('部门')].attributes('data-prop')).toBe('department')
})

it('labels create and edit route tabs with the operation', () => {
  expect(fmtTitle('客户', { params: {}, query: { whDetail: 'brands', whNew: '1' } })).toBe('客户-新建')
  expect(fmtTitle('客户', { params: {}, query: { whDetail: 'brands', whId: '4' } })).toBe('客户-编辑')
  expect(fmtTitle('送货单', { params: {}, query: { whDelivery: '1', whNew: '1' } })).toBe('送货单-新建')
})
it('sorts the complete delivery list server-side when a sortable header changes', async () => {
  mocks.routeQuery = {}
  mocks.allOptions.mockResolvedValue([])
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    if (url.endsWith('/departments')) return []
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data'], template: '<div><slot /></div>' },
    'el-table-column': { props: ['label', 'prop', 'sortable'], template: '<div />' }
  } } })
  await flushPromises()

  wrapper.vm.$.setupState.changeSort({ prop: 'totalAmount', order: 'descending' })
  await flushPromises()

  const calls = mocks.request.mock.calls.map(([request]) => request).filter(request => request.url.endsWith('/list'))
  expect(calls.at(-1).params).toMatchObject({ page: 1, sortBy: 'totalAmount', sortOrder: 'desc' })
  expect(calls.at(-1).params.pageSize).toBe(20)
})
it('sorts displayed detail rows without changing their submitted order and uses compact rows', async () => {
  mocks.allOptions.mockResolvedValue([])
  const aggregate = {
    note: { id: 10, version: 1, number: 'SK260101000001', status: 'draft', brandId: 2, deliveredAt: '2026-01-01T00:00:00Z' },
    details: [
      { id: 4, materialId: 7, materialUnitId: 8, unitId: 9, baseUnitId: 9, factor: '1', quantity: '2', unitPrice: '20', amount: '40', materialCodeSnapshot: 'M2', materialNameSnapshot: '第二项', unitNameSnapshot: '件' },
      { id: 5, materialId: 6, materialUnitId: 8, unitId: 9, baseUnitId: 9, factor: '1', quantity: '1', unitPrice: '10', amount: '10', materialCodeSnapshot: 'M1', materialNameSnapshot: '第一项', unitNameSnapshot: '件' }
    ]
  }
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/detail')) return aggregate
    if (url === '/warehouse/master/materials/detail') return { units: [{ ID: 8, unitId: 9, factor: '1', enabled: true, isBase: true, name: '件' }] }
    if (url.endsWith('/update') || url.endsWith('/list') || url.endsWith('/departments')) return url.endsWith('/list') ? { list: [], total: 0, page: 1, pageSize: 20 } : {}
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data', 'size'], template: '<div class="detail-table-render" :data-size="size"><slot /></div>' },
    'el-table-column': {
      props: ['label', 'prop', 'sortable', 'type'],
      data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test', quantity: '', unitPrice: '', materialCode: '', materialName: '', specification: '', unitName: '' } }),
      template: '<div class="detail-column" :data-prop="prop" :data-sortable="sortable"><slot :row="fakeRow" /></div>'
    }
  } } })
  await flushPromises()

  const state = wrapper.vm.$.setupState
  expect(wrapper.find('.detail-table').attributes('data-size')).toBe('small')
  expect(wrapper.findAll('.detail-table .detail-column').filter(column => column.attributes('data-sortable') === 'custom')).toHaveLength(7)
  state.changeDetailSort({ prop: 'quantity', order: 'ascending' })
  expect(state.displayRows.map(row => row.materialCode)).toEqual(['M1', 'M2'])
  expect(state.rows.map(row => row.materialCode)).toEqual(['M2', 'M1'])

  await state.save()
  await flushPromises()
  const update = mocks.request.mock.calls.map(([request]) => request).find(request => request.url.endsWith('/update'))
  expect(update.data.details.map(detail => detail.id)).toEqual([4, 5])
  expect(update.data.details.map(detail => detail.materialId)).toEqual([7, 6])
})
