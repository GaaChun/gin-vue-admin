import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { reactive, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { ElMessage, ElMessageBox } from 'element-plus'
import Delivery from './delivery.vue'
import { fmtTitle } from '@/utils/fmtRouterTitle'

const mocks = vi.hoisted(() => ({
  request: vi.fn(),
  allOptions: vi.fn(),
  showError: vi.fn(),
  push: vi.fn(),
  routeQuery: { whDelivery: '1', whId: '10' },
  userStore: null
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
vi.mock('@/api/user', () => ({ getUserList: vi.fn(async () => ({ code: 0, data: { list: [] } })), setSelfSetting: vi.fn(async () => ({ code: 0 })) }))
vi.mock('../components/delivery-material-picker.vue', () => ({ default: { props: ['materials'], template: '<div class="picker-materials">{{ materials.map(material => material.code).join(",") }}</div>' } }))
vi.mock('../components/material-filter.vue', () => ({ default: { props: ['modelValue', 'materials'], emits: ['update:modelValue'], template: '<div class="material-filter-stub" />' } }))
vi.mock('../api/table-widths', () => ({ getTableWidths: vi.fn(async () => ({})), saveTableWidths: vi.fn(async () => ({})) }))
vi.mock('@/pinia/modules/user', () => ({ useUserStore: () => mocks.userStore }))

let wrapper
beforeEach(() => {
  sessionStorage.clear()
  mocks.userStore = reactive({ userInfo: { ID: 1, originSetting: {} }, ResetUserInfo: vi.fn() })
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  mocks.routeQuery = { whDelivery: '1', whId: '10' }
  sessionStorage.clear()
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
  const bulk = wrapper.find('[data-action="review-selected"]')
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
  const deleteButton = wrapper.findAll('.delivery-detail-footer el-button').find(button => button.text().trim() === '删除')
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
    if (url.endsWith('/reserveCode')) return { code: 'M-GENERATED' }
    if (url.endsWith('/mnemonic')) return { mnemonic: 'xinwuliao' }
    if (url.endsWith('/materials/create')) return {
      ID: 22,
      code: 'M-GENERATED',
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
  state.quickMaterialSearch = '新物料'
  await wrapper.findAll('el-button').find(button => button.text().includes('新建物料并添加')).trigger('click')
  await flushPromises()
  expect(state.quickMaterialDialogOpen).toBe(true)
  expect(state.quickMaterialForm).toMatchObject({ name: '新物料', code: 'M-GENERATED', mnemonic: 'xinwuliao' })
  expect(mocks.request).toHaveBeenCalledWith({ url: '/warehouse/master/reserveCode', method: 'post', data: { resource: 'materials' }, donNotShowLoading: true })
  expect(mocks.request).toHaveBeenCalledWith({ url: '/warehouse/master/mnemonic', method: 'post', data: { name: '新物料' }, donNotShowLoading: true })
  Object.assign(state.quickMaterialForm, { unitId: 9 })
  await state.createQuickMaterial()
  await flushPromises()

  expect(mocks.request).toHaveBeenCalledWith({
    url: '/warehouse/master/materials/create',
    method: 'post',
    data: { name: '新物料', code: 'M-GENERATED', mnemonic: 'xinwuliao', units: [{ unitId: 9, factor: '1', isBase: true, enabled: true }] }
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

it('keeps a newly saved note open with its persisted identity for subsequent updates', async () => {
  sessionStorage.removeItem('warehouse.delivery.1.draft.new')
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
    query: { whDelivery: '1', whId: '20' }
  }))
  expect(state.note).toMatchObject({ id: 20, version: 1 })
  expect(state.editorOpen).toBe(true)
  expect(state.dirty).toBe(false)
})

it('clears and reserves a new note when continuing from the existing new-note route', async () => {
  sessionStorage.removeItem('warehouse.delivery.1.draft.new')
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
  const secondMaterialId = state.rows[1].materialId
  state.rows[1].materialId = state.rows[0].materialId
  await Promise.all([state.loadRowUnits(state.rows[0]), state.loadRowUnits(state.rows[1])])
  state.rows[1].materialId = secondMaterialId
  expect(mocks.request.mock.calls.map(([request]) => request.url).filter(url => url === '/warehouse/master/materials/detail')).toHaveLength(1)
  expect(state.rows.slice(0, 2).every(row => row.units.some(binding => binding.ID === 8))).toBe(true)
  expect(wrapper.find('.delivery-detail-heading').text()).toContain('送货单详情')
  expect(wrapper.find('.delivery-detail-heading').text()).toContain('未审核')
  expect(wrapper.find('.delivery-detail-scroll').exists()).toBe(true)
  expect(wrapper.find('.delivery-detail-footer').text()).toContain('审核')
  expect(wrapper.find('.delivery-detail-footer').text()).toContain('删除')
  expect(wrapper.find('.delivery-detail-footer').text()).toContain('刷新')
  expect(wrapper.find('.delivery-detail-footer').text()).toContain('新增')
  expect(wrapper.find('.delivery-detail-footer').text()).toContain('上一张')
  expect(wrapper.find('.delivery-detail-footer').text()).toContain('下一张')
  expect(wrapper.find('.delivery-detail-footer').text()).not.toContain('返回列表')
  expect(wrapper.find('.delivery-detail-footer').text()).not.toContain('复制单据')
  state.notes = [
    { id: 9, version: 1, number: 'A' },
    { id: 10, version: 1, number: 'B' },
    { id: 11, version: 1, number: 'C' }
  ]
  await state.navigateDocument(1)
  expect(mocks.push).toHaveBeenCalledWith({ name: 'warehouseDelivery', query: { whDelivery: '1', whId: '11' } })
  await state.openNew()
  expect(mocks.push).toHaveBeenLastCalledWith({ name: 'warehouseDelivery', query: { whDelivery: '1', whNew: '1' } })
  await state.reloadEditor()
  expect(mocks.request.mock.calls.filter(([request]) => request.url === '/warehouse/delivery/detail')).toHaveLength(2)
  state.changeDetailSort({ prop: 'quantity', order: 'ascending' })
  expect(state.displayRows.map(row => row.materialCode)).toEqual(['M1', 'M2'])
  expect(state.rows.map(row => row.materialCode)).toEqual(['M2', 'M1'])

  await state.save()
  await flushPromises()
  const update = mocks.request.mock.calls.map(([request]) => request).find(request => request.url.endsWith('/update'))
  expect(update.data.details.map(detail => detail.id)).toEqual([4, 5])
  expect(update.data.details.map(detail => detail.materialId)).toEqual([7, 6])
})
it('defaults new material values and calculates editor amount with decimal precision', async () => {
  mocks.allOptions.mockResolvedValue([])
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    if (url.endsWith('/departments')) return {}
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data'], template: '<div><slot /></div>' },
    'el-table-column': {
      props: ['label', 'prop', 'sortable', 'type'],
      data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test', quantity: '', unitPrice: '', materialCode: '', materialName: '', specification: '', unitName: '' } }),
      template: '<div><slot :row="fakeRow" /></div>'
    }
  } } })
  await flushPromises()
  const state = wrapper.vm.$.setupState
  const row = state.emptyRow()
  state.applyMaterial(row, {
    material: { id: 7, code: 'M7', name: '物料七' },
    bindings: [{ ID: 8, unitId: 9, name: '件', enabled: true, isBase: true }],
    baseBinding: { ID: 8, unitId: 9, name: '件', enabled: true, isBase: true }
  })
  expect(row).toMatchObject({ quantity: '0', unitPrice: '0', amount: '0' })
  expect(state.bindingLabel({ name: '件', isBase: true, enabled: true })).toBe('件')
  row.quantity = '1.25'
  row.unitPrice = '2.4'
  state.clearFieldError(row, 'quantity')
  expect(row.amount).toBe('3')
})
it('copies reviewed delivery data into a new draft without copying identities', async () => {
  sessionStorage.removeItem('warehouse.delivery.1.draft.new')
  mocks.allOptions.mockResolvedValue([])
  const aggregate = {
    note: { id: 22, version: 5, number: 'SK260101000022', status: 'reviewed', brandId: 3, department: '仓库', deliveredAt: '2026-02-03T00:00:00Z' },
    details: [
      { id: 99, materialId: 7, materialUnitId: 8, unitId: 9, baseUnitId: 9, factor: '1', quantity: '2.5', unitPrice: '12.25', amount: '30.63', remark: '轻放，送至东门', materialCodeSnapshot: 'M-7', materialNameSnapshot: '物料七', specificationSnapshot: '规格A', unitNameSnapshot: '件' }
    ]
  }
  mocks.request.mockImplementation(async ({ url }) => {
    if (url === '/warehouse/delivery/detail') return aggregate
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    if (url.endsWith('/departments')) return {}
    if (url.endsWith('/reserveNumber')) return { number: 'SK260101000023', token: 'new-token' }
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data'], template: '<div><slot /></div>' },
    'el-table-column': {
      props: ['label', 'prop', 'sortable', 'type'],
      data: () => ({ fakeRow: { keyword: '', units: [], errors: {}, status: 'draft', rowKey: 'test', quantity: '', unitPrice: '', materialCode: '', materialName: '', specification: '', unitName: '' } }),
      template: '<div><slot :row="fakeRow" /></div>'
    }
  } } })
  await flushPromises()

  const state = wrapper.vm.$.setupState
  expect(wrapper.find('.delivery-detail-footer').text()).toContain('复制单据')
  await state.copyDocument()

  const draft = JSON.parse(sessionStorage.getItem('warehouse.delivery.1.draft.new'))
  expect(draft.form).toMatchObject({ brandId: 3, department: '仓库' })
  expect(draft.reservation).toEqual({})
  expect(draft.rows).toHaveLength(1)
  expect(draft.rows[0]).toMatchObject({
    materialId: 7, materialUnitId: 8, materialCode: 'M-7',
    quantity: '2.5', unitPrice: '12.25', remark: '轻放，送至东门', unitsLoaded: false
  })
  expect(draft.rows[0].id).toBeUndefined()
  expect(draft.rows[0].amount).toBe('30.63')
  expect(state.note.id).toBe(22)
  expect(mocks.push).toHaveBeenCalledWith({ name: 'warehouseDelivery', query: { whDelivery: '1', whNew: '1' } })
  await state.initializeNew()
  expect(state.note.id).toBeUndefined()
  expect(state.form).toMatchObject({ brandId: 3, department: '仓库', deliveredAt: '2026-02-03' })
  expect(state.reservation).toMatchObject({ number: 'SK260101000023', token: 'new-token' })
  expect(state.rows).toHaveLength(1)
  expect(state.rows[0]).toMatchObject({ materialId: 7, materialUnitId: 8, quantity: '2.5', unitPrice: '12.25', remark: '轻放，送至东门' })
  expect(state.rows[0].id).toBeUndefined()
  expect(mocks.request).toHaveBeenCalledWith({ url: '/warehouse/delivery/reserveNumber', method: 'post' })
})

async function mountEditableDelivery(status = 'draft') {
  mocks.allOptions.mockImplementation(async kind => kind === 'brands' ? [{ id: 2, name: '客户甲', code: 'KH2' }] : [])
  mocks.request.mockImplementation(async ({ url, data }) => {
    if (url === '/warehouse/delivery/detail') return {
      note: { id: 10, version: 3, number: 'SK10', status, brandId: 2, deliveredAt: '2026-01-01T00:00:00Z' },
      details: [
        { id: 1, materialId: 7, materialUnitId: 8, unitId: 9, quantity: '3', unitPrice: '0.3333', amount: '1.00', materialCodeSnapshot: 'M7', unitNameSnapshot: '件' },
        { id: 2, materialId: 7, materialUnitId: 8, unitId: 9, quantity: '1', unitPrice: '0.1', amount: '0.10', materialCodeSnapshot: 'M7', unitNameSnapshot: '件' }
      ]
    }
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    if (url.endsWith('/departments')) return ['采购部', '仓库']
    if (url.endsWith('/reReview')) return { id: 10, status: 'reReviewed', reReviewOpinion: data.opinion }
    if (url.endsWith('/update') || url.endsWith('/create')) return {
      Note: { id: data.id || 20, version: (data.version || 0) + 1, number: 'SK10', status: data.review ? 'reviewed' : 'draft', brandId: data.brandId, deliveredAt: data.deliveredAt },
      Details: data.details.map((detail, index) => ({ ...detail, id: detail.id || index + 1, materialCodeSnapshot: 'M7', unitNameSnapshot: '件' }))
    }
    throw new Error(`Unexpected request: ${url}`)
  })
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data'], template: '<div><slot /></div>' },
    'el-table-column': { template: '<div />' },
    'el-dialog': { props: ['modelValue'], template: '<div v-if="modelValue"><slot /><slot name="footer" /></div>' }
  } } })
  await flushPromises()
  return wrapper.vm.$.setupState
}

it('saves edited content and reviews atomically through update', async () => {
  const state = await mountEditableDelivery()
  state.rows[0].quantity = '4'
  state.clearFieldError(state.rows[0], 'quantity')
  await wrapper.findAll('.delivery-detail-footer el-button').find(button => button.text().trim() === '审核').trigger('click')
  await flushPromises()
  const update = mocks.request.mock.calls.map(([arg]) => arg).find(arg => arg.url.endsWith('/update'))
  expect(update.data.review).toBe(true)
  expect(update.data.details[0].quantity).toBe('4')
  expect(mocks.request.mock.calls.some(([arg]) => arg.url.endsWith('/review'))).toBe(false)
})

it('rounds line money before summing and derives unit price exactly from editable amount', async () => {
  const state = await mountEditableDelivery()
  const row = state.rows[0]
  row.quantity = '3'
  row.amount = '10.00'
  state.clearFieldError(row, 'amount')
  expect(row.unitPrice).toBe('3.3333')
  expect(state.validatedDetails()[0].amount).toBe('10')
  expect(state.tableSummary({ columns: [{}, { property: 'amount' }] })).toEqual(['合计', '10.10'])
  row.unitPrice = '0.335'
  state.clearFieldError(row, 'unitPrice')
  expect(row.amount).toBe('1.01')
  expect(state.validatedDetails()[0].amount).toBeUndefined()
  expect(state.calculateAmount('99999999999999.9999', '0.0001')).toBe('10000000000.00')
})

it('keeps exactly one active row and removes selected lines only locally after confirmation', async () => {
  const state = await mountEditableDelivery()
  const [first, second] = state.rows
  await state.activateRow(first)
  expect(state.activeRowKey).toBe(first.rowKey)
  await state.focusCell(second, 'quantity')
  expect(state.activeRowKey).toBe(second.rowKey)
  state.handleEditorPointerDown({ target: document.body })
  expect(state.activeRowKey).toBe('')
  state.detailSelectionChanged([first])
  ElMessageBox.confirm.mockRejectedValueOnce('cancel')
  await state.removeSelectedRows()
  expect(state.rows).toHaveLength(2)
  await state.removeSelectedRows()
  expect(state.rows.map(row => row.id)).toEqual([2])
  expect(mocks.request.mock.calls.some(([arg]) => /\/(delete|update)$/.test(arg.url))).toBe(false)
})

it('collects optional review opinion and blocks unreview until rereview is cancelled', async () => {
  const state = await mountEditableDelivery('reviewed')
  await wrapper.findAll('.delivery-detail-footer el-button').find(button => button.text().trim() === '复审').trigger('click')
  expect(state.reReviewDialog.open).toBe(true)
  state.reReviewDialog.opinion = '核对无误'
  await state.confirmReReview()
  expect(mocks.request).toHaveBeenCalledWith({
    url: '/warehouse/delivery/reReview', method: 'post', data: { id: 10, version: 3, opinion: '核对无误' }
  })
  await flushPromises()
  expect(wrapper.find('.delivery-detail-footer').text()).toContain('取消复审')
  expect(wrapper.find('.delivery-detail-footer').text()).not.toContain('反审核')
  expect(wrapper.find('.editor-header el-input[type="textarea"]').attributes('model-value')).toBe('核对无误')
})

it('saves and reviews a new delivery in one create request', async () => {
  const state = await mountEditableDelivery()
  state.note = {}
  state.reservation.token = 'new-token'
  await state.save(false, true)
  expect(mocks.request).toHaveBeenCalledWith(expect.objectContaining({
    url: '/warehouse/delivery/create', method: 'post',
    data: expect.objectContaining({ reservationToken: 'new-token', review: true })
  }))
})

it('batch rereviews only reviewed notes with one shared opinion and their versions', async () => {
  const state = await mountEditableDelivery()
  const reviewed = { id: 20, version: 8, status: 'reviewed' }
  const draft = { id: 21, version: 9, status: 'draft' }
  state.openReReview([reviewed, draft], true)
  expect(state.reReviewDialog.targets).toEqual([reviewed])
  state.reReviewDialog.opinion = '批量核对'
  await state.confirmReReview()
  expect(mocks.request).toHaveBeenCalledWith({
    url: '/warehouse/delivery/reReview', method: 'post', data: { id: 20, version: 8, opinion: '批量核对' }
  })
})

it('does not submit batch deletion when cancelled and explains mixed selection', async () => {
  const state = await mountEditableDelivery()
  state.selectionChanged([{ id: 20, version: 8, status: 'draft' }, { id: 21, version: 9, status: 'reviewed' }, { id: 22, version: 1, status: 'reReviewed' }])
  ElMessageBox.confirm.mockRejectedValueOnce('cancel')
  await state.batchTransition('delete')
  expect(ElMessageBox.confirm).toHaveBeenCalledWith(
    '确定删除选中的 1 张未审核送货单？另有 2 张已审核或已复审单据不会删除。此操作不可恢复。',
    '删除确认', expect.any(Object)
  )
  expect(mocks.request.mock.calls.some(([arg]) => arg.url.endsWith('/delete'))).toBe(false)
})

it('batch deletes only drafts with versions and refreshes after partial success', async () => {
  const state = await mountEditableDelivery()
  state.selectionChanged([
    { id: 20, version: 8, status: 'draft' }, { id: 21, version: 9, status: 'reviewed' },
    { id: 22, version: 10, status: 'draft' }, { id: 23, version: 11, status: 'reReviewed' }
  ])
  mocks.request.mockImplementation(async ({ url, data }) => {
    if (url.endsWith('/delete')) {
      if (data.id === 22) throw new Error('版本冲突')
      return {}
    }
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    throw new Error(`Unexpected request: ${url}`)
  })
  await state.batchTransition('delete')
  const deletions = mocks.request.mock.calls.map(([arg]) => arg).filter(arg => arg.url.endsWith('/delete'))
  expect(deletions).toEqual([
    { url: '/warehouse/delivery/delete', method: 'delete', data: { id: 20, version: 8 } },
    { url: '/warehouse/delivery/delete', method: 'delete', data: { id: 22, version: 10 } }
  ])
  expect(ElMessage.warning).toHaveBeenCalledWith('已删除 1 张，其余未完成；请刷新后重试')
  expect(state.selectedNotes).toEqual([])
  expect(mocks.request.mock.calls.at(-1)[0].url).toBe('/warehouse/delivery/list')
})


it('preserves normal rounded prices and rehydrates only unmatched saved amount overrides', async () => {
  const state = await mountEditableDelivery()
  await state.readAggregate({
    note: { id: 10, status: 'draft', brandId: 2, deliveredAt: '2026-01-01T00:00:00Z' },
    details: [
      { id: 1, materialId: 7, materialUnitId: 8, quantity: '0.3333', unitPrice: '0.05', amount: '0.02' },
      { id: 2, materialId: 7, materialUnitId: 8, quantity: '999999', unitPrice: '0.0000', amount: '0.01' }
    ]
  })
  const details = state.validatedDetails()
  expect(details[0]).toMatchObject({ unitPrice: '0.05' })
  expect(details[0].amount).toBeUndefined()
  expect(details[1].amount).toBe('0.01')
})

it('displays detail amounts without insignificant fractional zeros', async () => {
  const state = await mountEditableDelivery()
  await state.readAggregate({
    note: { id: 10, status: 'draft', brandId: 2, deliveredAt: '2026-01-01T00:00:00Z' },
    details: [
      { id: 1, materialId: 7, materialUnitId: 8, quantity: '4', unitPrice: '11', amount: '44.00' },
      { id: 2, materialId: 7, materialUnitId: 8, quantity: '4', unitPrice: '11.1', amount: '44.40' },
      { id: 3, materialId: 7, materialUnitId: 8, quantity: '3', unitPrice: '0.335', amount: '1.01' },
      { id: 4, materialId: 7, materialUnitId: 8, quantity: '1', unitPrice: '1', amount: '1.50' }
    ]
  })
  const amounts = state.rows.map(row => state.rowAmount(row))
  expect(amounts).toEqual(['44', '44.4', '1.01', '1.5'])
  expect(state.rows.map(row => row.amount)).toEqual(['44', '44.4', '1.01', '1.5'])
  expect(state.rows[3].amountEdited).toBe(true)
  expect(state.validatedDetails()[3].amount).toBe('1.5')
  expect(state.calculateAmount('4', '11')).toBe('44.00')
  expect(state.calculateAmount('4', '11.1')).toBe('44.40')
  expect(state.tableSummary({ columns: [{}, { property: 'amount' }] })).toEqual(['合计', '90.91'])
})

it('activates the next row on line completion', async () => {
  const state = await mountEditableDelivery()
  const row = state.rows[0]
  const enter = { key: 'Enter', preventDefault: vi.fn(), stopImmediatePropagation: vi.fn() }
  row.amount = '0.00'
  state.cellKeydown(enter, row, 'remark')
  await flushPromises()
  expect(state.activeRowKey).toBe(state.rows[1].rowKey)
  expect(state.bindingLabel({ name: '件', enabled: false })).toBe('件')
})

it('cancels rereview with the current version before offering unreview', async () => {
  const state = await mountEditableDelivery('reReviewed')
  const requestsBefore = mocks.request.mock.calls.length
  await state.transition(state.note, 'unreview')
  expect(mocks.request.mock.calls).toHaveLength(requestsBefore)
  mocks.request.mockImplementation(async ({ url }) => {
    if (url.endsWith('/cancelReReview')) return { id: 10, version: 4, status: 'reviewed', number: 'SK10' }
    if (url.endsWith('/list')) return { list: [], total: 0, page: 1, pageSize: 20 }
    throw new Error(`Unexpected request: ${url}`)
  })
  await state.transition(state.note, 'cancelReReview')
  expect(mocks.request).toHaveBeenCalledWith({ url: '/warehouse/delivery/cancelReReview', method: 'post', data: { id: 10, version: 3 } })
  expect(state.note.status).toBe('reviewed')
  await flushPromises()
  expect(wrapper.find('.delivery-detail-footer').text()).toContain('反审核')
})

it('restores draft rows with unique keys so newly added rows cannot share editing state', async () => {
  const state = await mountEditableDelivery()
  sessionStorage.setItem('warehouse.delivery.1.draft.new', JSON.stringify({
    form: { brandId: 2, department: '', deliveredAt: '2026-01-01' },
    reservation: { number: 'SKNEW', token: 'reserved' },
    rows: [{ ...state.emptyRow(), rowKey: 'delivery-row-1', materialId: 7 }]
  }))
  await state.initializeNew()
  state.addBlankRow()
  await flushPromises()
  expect(new Set(state.rows.map(row => row.rowKey)).size).toBe(state.rows.length)
  expect(state.activeRowKey).toBe(state.rows.at(-1).rowKey)
  sessionStorage.removeItem('warehouse.delivery.1.draft.new')
  state.editorOpen = false
})

it.each(['Enter', 'Tab'])('moves backward on Shift+%s including across rows with open selectors', async key => {
  const state = await mountEditableDelivery()
  const [first, second] = state.rows
  const previousAmount = { focus: vi.fn() }
  const material = { focus: vi.fn(), blur: vi.fn(), expanded: true }
  const unit = { focus: vi.fn(), blur: vi.fn(), expanded: true }
  state.setCellRef(first, 'amount', previousAmount)
  state.setCellRef(second, 'material', material)
  state.setCellRef(second, 'unit', unit)
  const event = () => ({ key, shiftKey: true, preventDefault: vi.fn(), stopImmediatePropagation: vi.fn() })
  state.cellKeydown(event(), second, 'material')
  await flushPromises()
  expect(previousAmount.focus).toHaveBeenCalledOnce()
  expect(state.activeRowKey).toBe(first.rowKey)
  expect(material.expanded).toBe(false)
  second.unitOpen = true
  state.cellKeydown(event(), second, 'unit')
  await flushPromises()
  expect(material.focus).toHaveBeenCalledOnce()
  expect(state.activeRowKey).toBe(second.rowKey)
  expect(unit.expanded).toBe(false)
})

it('reorders sorted detail rows explicitly and submits that order while remaining in the editor', async () => {
  const state = await mountEditableDelivery()
  const [first, second] = state.rows
  state.changeDetailSort({ prop: 'quantity', order: 'ascending' })
  expect(state.displayRows.map(row => row.id)).toEqual([2, 1])
  state.draggedRowKey = first.rowKey
  state.dropRow(second)
  expect(state.rows.map(row => row.id)).toEqual([1, 2])
  state.draggedRowKey = first.rowKey
  state.dropRow(second, 'after')
  expect(state.rows.map(row => row.id)).toEqual([2, 1])
  await state.save()
  expect(state.rows.map(row => row.id)).toEqual([2, 1])
  expect(state.note.version).toBe(4)
  expect(state.editorOpen).toBe(true)
  expect(state.dirty).toBe(false)
  expect(mocks.push).not.toHaveBeenCalled()
  expect(state.validatedDetails().map(row => row.id)).toEqual([2, 1])
})

it('keeps derived decimals exact without trailing zeros and changes dates across month boundaries', async () => {
  const state = await mountEditableDelivery()
  const row = state.rows[0]
  row.quantity = '4'; row.amount = '44'
  state.clearFieldError(row, 'amount')
  expect(row.unitPrice).toBe('11')
  row.quantity = '3'; row.amount = '1'
  state.clearFieldError(row, 'amount')
  expect(row.unitPrice).toBe('0.3333')
  state.form.deliveredAt = '2024-03-01'
  state.adjustDeliveryDate({ key: 'ArrowLeft', preventDefault() {} })
  expect(state.form.deliveredAt).toBe('2024-02-29')
  state.adjustDeliveryDate({ key: 'ArrowRight', preventDefault() {} })
  expect(state.form.deliveredAt).toBe('2024-03-01')
})

async function remountDelivery(query = mocks.routeQuery) {
  wrapper?.unmount()
  mocks.routeQuery = reactive(query)
  wrapper = mount(Delivery, { global: { stubs: {
    'el-table': { props: ['data'], template: '<div><slot /></div>' },
    'el-table-column': { template: '<div />' },
    'el-dialog': { props: ['modelValue'], template: '<div v-if="modelValue"><slot /><slot name="footer" /></div>' }
  } } })
  await flushPromises()
  return wrapper.vm.$.setupState
}

it('retains edited headers and added/reordered rows through list and other-tab remounts without a discard prompt', async () => {
  let state = await mountEditableDelivery()
  state.form.department = '未保存的部门'
  state.rows[0].quantity = '17'
  state.addBlankRow()
  Object.assign(state.rows.at(-1), { materialId: 20, materialUnitId: 21, quantity: '5', unitPrice: '3' })
  state.draggedRowKey = state.rows.at(-1).rowKey
  state.dropRow(state.rows[0])
  const expectedRows = state.rows.map(({ id, materialId, quantity }) => ({ id, materialId, quantity }))
  await state.closeEditor()
  expect(ElMessageBox.confirm).not.toHaveBeenCalled()
  expect(mocks.push).toHaveBeenLastCalledWith({ name: 'warehouseDelivery', query: {} })
  state = await remountDelivery({})
  expect(state.editorOpen).toBe(false)
  state = await remountDelivery({ whDelivery: '1', whId: '10' })
  expect(state.form.department).toBe('未保存的部门')
  expect(state.rows.map(({ id, materialId, quantity }) => ({ id, materialId, quantity }))).toEqual(expectedRows)
  expect(state.dirty).toBe(true)
  state = await remountDelivery()
  expect(state.form.department).toBe('未保存的部门')
  expect(state.rows.map(row => row.materialId)).toEqual([20, 7, 7])
  expect(new Set(state.rows.map(row => row.rowKey)).size).toBe(3)
})

it('preserves list conditions, paging, sorting and expanded filters across remounts and display setting updates', async () => {
  await mountEditableDelivery()
  mocks.request.mockImplementation(async ({ url, params }) => {
    if (url.endsWith('/list')) return { list: [], total: 200, page: params.page, pageSize: params.pageSize }
    if (url.endsWith('/departments')) return []
    throw new Error(`Unexpected request: ${url}`)
  })
  let state = await remountDelivery({})
  Object.assign(state.filters, { number: 'SK42', department: '采购部', materialIds: [7, 8], brandId: 2, status: 'draft', detailRemark: '轻放', dates: ['2026-01-01', '2026-01-31'] })
  state.page = 3; state.pageSize = 50
  state.sortBy = 'department'; state.sortOrder = 'asc'; state.moreFiltersOpen = true
  await flushPromises()
  mocks.userStore.userInfo.originSetting = {
    'warehouseDocumentDisplay.delivery': { dateDefault: 'today', filters: { basic: ['dates', 'number', 'materialIds', 'brandId'], more: ['department', 'status', 'detailRemark'] }, list: { columns: ['number'] } }
  }
  await flushPromises()
  expect(state.filters.department).toBe('采购部')
  expect(state.filters.dates).toEqual(['2026-01-01', '2026-01-31'])
  state = await remountDelivery({})
  expect(state.filters).toMatchObject({ number: 'SK42', department: '采购部', materialIds: [7, 8], brandId: 2, status: 'draft', detailRemark: '轻放', dates: ['2026-01-01', '2026-01-31'] })
  expect([state.page, state.pageSize, state.sortBy, state.sortOrder, state.moreFiltersOpen]).toEqual([3, 50, 'department', 'asc', true])
  state.changeSort({ prop: 'department', order: 'ascending' })
  expect(state.page).toBe(3)
  expect(mocks.request.mock.calls.map(([request]) => request).filter(request => request.url.endsWith('/list')).at(-1).params).toMatchObject({
    page: 3, pageSize: 50, number: 'SK42', department: '采购部', materialIds: '7,8', detailRemark: '轻放', sortBy: 'department', sortOrder: 'asc'
  })
  state.resetSearch()
  await flushPromises()
  expect(state.filters.number).toBe('')
  expect(state.filters.detailRemark).toBe('')
  expect(state.filters.dates).not.toEqual(['2026-01-01', '2026-01-31'])
  expect(state.page).toBe(1)
})

it('isolates existing and new drafts and list conditions by account and document', async () => {
  let state = await mountEditableDelivery()
  const backend = mocks.request.getMockImplementation()
  mocks.request.mockImplementation(async request => {
    if (request.url.endsWith('/reserveNumber')) return { number: `SK-NEW-${mocks.userStore.userInfo.ID}`, token: `token-${mocks.userStore.userInfo.ID}` }
    const result = await backend(request)
    if (request.url === '/warehouse/delivery/detail') return { ...result, note: { ...result.note, id: request.params.id } }
    return result
  })
  state.form.department = '甲账号单据10'
  state.filters.number = '甲账号查询'
  state = await remountDelivery({ whDelivery: '1', whId: '11' })
  expect(state.form.department).toBe('')
  state.form.department = '甲账号单据11'
  state = await remountDelivery({ whDelivery: '1', whNew: '1' })
  expect(state.note.id).toBeUndefined()
  expect(state.rows.every(row => !row.id)).toBe(true)
  state.form.department = '甲账号新增'
  wrapper.unmount(); wrapper = undefined
  mocks.userStore.userInfo = { ID: 2, originSetting: {} }
  state = await remountDelivery({ whDelivery: '1', whId: '10' })
  expect(state.form.department).toBe('')
  expect(state.filters.number).toBe('')
  state.form.department = '乙账号单据10'
  state = await remountDelivery({ whDelivery: '1', whNew: '1' })
  expect(state.form.department).toBe('')
  expect(state.reservation.token).toBe('token-2')
  wrapper.unmount(); wrapper = undefined
  mocks.userStore.userInfo = { ID: 1, originSetting: {} }
  state = await remountDelivery({ whDelivery: '1', whId: '10' })
  expect(state.form.department).toBe('甲账号单据10')
  expect(state.filters.number).toBe('甲账号查询')
  state = await remountDelivery({ whDelivery: '1', whId: '11' })
  expect(state.form.department).toBe('甲账号单据11')
  state = await remountDelivery({ whDelivery: '1', whNew: '1' })
  expect(state.form.department).toBe('甲账号新增')
  expect(state.reservation.token).toBe('token-1')
})

it('does not overwrite a retained draft while loading and keeps its original version until explicit reload', async () => {
  let state = await mountEditableDelivery()
  state.form.department = '版本3的本地修改'
  await flushPromises()
  const backend = mocks.request.getMockImplementation()
  let resolveDetail
  mocks.request.mockImplementation(request => request.url === '/warehouse/delivery/detail'
    ? new Promise(resolve => { resolveDetail = resolve })
    : backend(request))
  const remount = remountDelivery()
  await flushPromises()
  const retained = JSON.parse(sessionStorage.getItem('warehouse.delivery.1.draft.10'))
  expect(retained.form.department).toBe('版本3的本地修改')
  const latest = {
    Note: { id: 10, version: 4, number: 'SK10', status: 'draft', brandId: 2, department: '服务端修改', deliveredAt: '2026-01-01T00:00:00Z' },
    Details: [{ id: 1, materialId: 7, materialUnitId: 8, quantity: '2', unitPrice: '3' }]
  }
  resolveDetail(latest)
  await flushPromises()
  state = await remount
  expect(state.form.department).toBe('版本3的本地修改')
  expect(state.note.version).toBe(3)
  mocks.request.mockImplementation(async request => request.url === '/warehouse/delivery/detail' ? latest : backend(request))
  await state.reloadEditor()
  expect(state.form.department).toBe('服务端修改')
  expect(state.note.version).toBe(4)
  expect(sessionStorage.getItem('warehouse.delivery.1.draft.10')).toBeNull()
  state.form.department = '读取版本4后修改'
  await state.save()
  expect(mocks.request).toHaveBeenCalledWith(expect.objectContaining({ url: '/warehouse/delivery/update', data: expect.objectContaining({ version: 4 }) }))
  expect(state.note.version).toBe(5)
  expect(sessionStorage.getItem('warehouse.delivery.1.draft.10')).toBeNull()
})

it('clears only the saved draft and resumes subsequent edits using the saved version', async () => {
  let state = await mountEditableDelivery()
  state.form.department = '待保存'
  await flushPromises()
  sessionStorage.setItem('warehouse.delivery.1.draft.new', JSON.stringify({ form: { department: '另外的新单' }, reservation: {}, rows: [] }))
  await state.save()
  expect(sessionStorage.getItem('warehouse.delivery.1.draft.10')).toBeNull()
  expect(JSON.parse(sessionStorage.getItem('warehouse.delivery.1.draft.new')).form.department).toBe('另外的新单')
  state.form.department = '保存后的继续修改'
  state = await remountDelivery()
  expect(state.note.version).toBe(4)
  expect(state.form.department).toBe('保存后的继续修改')
  await state.save()
  expect(mocks.request).toHaveBeenLastCalledWith(expect.objectContaining({ url: '/warehouse/delivery/list' }))
  expect(mocks.request.mock.calls.map(([request]) => request).filter(request => request.url.endsWith('/update')).at(-1).data.version).toBe(4)
})

it('preserves the same document while switching edit/list routes without remounting', async () => {
  mocks.routeQuery = reactive({ whDelivery: '1', whId: '10' })
  const state = await mountEditableDelivery()
  state.form.department = '当前编辑内容'
  state.filters.department = '当前查询部门'
  state.rows[1].quantity = '23'
  delete mocks.routeQuery.whDelivery
  delete mocks.routeQuery.whId
  await flushPromises()
  expect(state.editorOpen).toBe(false)
  Object.assign(mocks.routeQuery, { whDelivery: '1', whId: '10' })
  await flushPromises()
  expect(state.form.department).toBe('当前编辑内容')
  expect(state.rows[1].quantity).toBe('23')
  expect(state.filters.department).toBe('当前查询部门')
  expect(ElMessageBox.confirm).not.toHaveBeenCalled()
})

it('preserves each account when switching users without remounting and ignores an old account detail response', async () => {
  const state = await mountEditableDelivery()
  state.form.department = '账号甲编辑'
  state.filters.number = '账号甲查询'
  const backend = mocks.request.getMockImplementation()
  let finishOtherAccount
  mocks.request.mockImplementation(request => request.url === '/warehouse/delivery/detail'
    ? new Promise(resolve => { finishOtherAccount = resolve })
    : backend(request))
  mocks.userStore.userInfo = { ID: 2, originSetting: {} }
  await flushPromises()
  expect(state.filters.number).toBe('')
  expect(state.form.department).toBe('')
  const finishAccountTwo = finishOtherAccount
  mocks.request.mockImplementation(backend)
  mocks.userStore.userInfo = { ID: 1, originSetting: {} }
  await flushPromises()
  expect(state.filters.number).toBe('账号甲查询')
  expect(state.form.department).toBe('账号甲编辑')
  finishAccountTwo({ Note: { id: 10, version: 99, status: 'draft', department: '账号乙数据' }, Details: [] })
  await flushPromises()
  expect(state.form.department).toBe('账号甲编辑')
  expect(state.note.version).toBe(3)
})

it('does not restore a newly saved document or detail identities into the new-note draft', async () => {
  let state = await mountEditableDelivery()
  const backend = mocks.request.getMockImplementation()
  mocks.request.mockImplementation(request => request.url.endsWith('/reserveNumber')
    ? Promise.resolve({ number: 'SK-NEW', token: 'new-token' })
    : backend(request))
  state = await remountDelivery({ whDelivery: '1', whNew: '1' })
  state.form.brandId = 2
  Object.assign(state.rows[0], { materialId: 7, materialUnitId: 8, quantity: '1', unitPrice: '2' })
  await state.save()
  expect(state.note).toMatchObject({ id: 20, version: 1 })
  expect(sessionStorage.getItem('warehouse.delivery.1.draft.new')).toBeNull()
  state.form.department = '保存后编辑的是20号单据'
  await flushPromises()
  expect(JSON.parse(sessionStorage.getItem('warehouse.delivery.1.draft.20')).note.id).toBe(20)
  state = await remountDelivery({ whDelivery: '1', whNew: '1' })
  expect(state.note.id).toBeUndefined()
  expect(state.form.department).toBe('')
  expect(state.rows).toHaveLength(1)
  expect(state.rows[0].id).toBeUndefined()
  expect(state.rows[0].materialId).toBeUndefined()
})

it('allows saving zero quantity, unit price and amount but still rejects negatives and invalid precision', async () => {
  const state = await mountEditableDelivery()
  state.rows.splice(1)
  const row = state.rows[0]
  Object.assign(row, { quantity: '0', unitPrice: '0', amount: '0', amountEdited: true })
  expect(state.validatedDetails()).toEqual([expect.objectContaining({ quantity: '0', unitPrice: '0', amount: '0' })])
  await state.save()
  expect(mocks.request).toHaveBeenCalledWith(expect.objectContaining({
    url: '/warehouse/delivery/update',
    data: expect.objectContaining({ details: [expect.objectContaining({ quantity: '0', unitPrice: '0', amount: '0' })] })
  }))
  for (const field of ['quantity', 'unitPrice', 'amount']) {
    row[field] = '-1'
    expect(state.validateDecimal(row, field)).toBe(false)
    row[field] = field === 'amount' ? '0.001' : '0.00001'
    expect(state.validateDecimal(row, field)).toBe(false)
    row[field] = '0'
    expect(state.validateDecimal(row, field)).toBe(true)
  }
})

it('captures date arrow key events outside the date-picker component boundary', async () => {
  const state = await mountEditableDelivery()
  state.form.deliveredAt = '2024-03-01'
  await wrapper.find('.delivery-detail-page el-date-picker').trigger('keydown', { key: 'ArrowLeft' })
  expect(state.form.deliveredAt).toBe('2024-02-29')
})

it('retains remark-only edits in existing drafts and saves and clears them verbatim', async () => {
  let state = await mountEditableDelivery()
  expect(state.dirty).toBe(false)
  state.rows[0].remark = '  送至东门  '
  await flushPromises()
  expect(state.dirty).toBe(true)
  expect(JSON.parse(sessionStorage.getItem('warehouse.delivery.1.draft.10')).rows[0].remark).toBe('  送至东门  ')
  state = await remountDelivery()
  expect(state.rows[0].remark).toBe('  送至东门  ')
  await state.save()
  expect(mocks.request).toHaveBeenCalledWith(expect.objectContaining({
    url: '/warehouse/delivery/update', data: expect.objectContaining({ details: expect.arrayContaining([expect.objectContaining({ id: 1, remark: '  送至东门  ' })]) })
  }))
  expect(state.rows[0].remark).toBe('  送至东门  ')
  expect(state.dirty).toBe(false)
  state.rows[0].remark = ''
  await state.save()
  const saved = mocks.request.mock.calls.map(([request]) => request).filter(request => request.url.endsWith('/update')).at(-1)
  expect(saved.data.details[0].remark).toBe('')
  expect(state.rows[0].remark).toBe('')
})

it('does not discard a remark-only trailing row as an empty row', async () => {
  const state = await mountEditableDelivery()
  const row = state.emptyRow()
  row.remark = '待补物料'
  state.rows.push(row)
  expect(state.validatedDetails()).toBeNull()
  expect(row.errors.materialId).toBeTruthy()
})

it.each(['Enter', 'Tab'])('skips remark on %s while allowing navigation out of a manually focused remark', async key => {
  const state = await mountEditableDelivery()
  const [first, second] = state.rows
  const remark = { focus: vi.fn() }
  const amount = { focus: vi.fn() }
  const material = { focus: vi.fn() }
  state.setCellRef(first, 'remark', remark)
  state.setCellRef(first, 'amount', amount)
  state.setCellRef(second, 'material', material)
  const event = shiftKey => ({ key, shiftKey, preventDefault: vi.fn(), stopImmediatePropagation: vi.fn() })
  state.cellKeydown(event(false), first, 'amount')
  await flushPromises()
  expect(remark.focus).not.toHaveBeenCalled()
  expect(material.focus).toHaveBeenCalledOnce()
  state.cellKeydown(event(true), first, 'remark')
  await flushPromises()
  expect(amount.focus).toHaveBeenCalledOnce()
  state.cellKeydown(event(false), first, 'remark')
  await flushPromises()
  expect(material.focus).toHaveBeenCalledTimes(2)
})

it.each([
  ['Enter', 'unitPrice'], ['Tab', 'unitPrice'], ['Enter', 'unit'], ['Tab', 'unit']
])('skips a reordered remark on %s and crosses rows from the last visible %s field', async (key, lastField) => {
  const state = await mountEditableDelivery()
  mocks.userStore.userInfo.originSetting['warehouseDocumentDisplay.delivery'] = {
    detail: { columns: lastField === 'unit'
      ? ['remark', 'materialCode', 'materialName', 'quantity', 'unitPrice', 'unitName']
      : ['remark', 'materialCode', 'materialName', 'unitName', 'quantity', 'unitPrice'] }
  }
  const [first, second] = state.rows
  const remark = { focus: vi.fn() }
  const material = { focus: vi.fn(), blur: vi.fn() }
  const last = { focus: vi.fn() }
  state.setCellRef(second, 'remark', remark)
  state.setCellRef(second, 'material', material)
  state.setCellRef(first, lastField, last)
  const event = shiftKey => ({ key, shiftKey, preventDefault: vi.fn(), stopImmediatePropagation: vi.fn() })
  state.cellKeydown(event(false), first, lastField)
  await flushPromises()
  expect(material.focus).toHaveBeenCalledOnce()
  state.cellKeydown(event(true), second, 'material')
  await flushPromises()
  expect(last.focus).toHaveBeenCalledOnce()
  expect(remark.focus).not.toHaveBeenCalled()
})

it('rehydrates a cached editor on activation without resurrecting an already-saved draft', async () => {
  await mountEditableDelivery()
  wrapper.unmount()
  const backend = mocks.request.getMockImplementation()
  let latest
  mocks.request.mockImplementation(request => request.url === '/warehouse/delivery/detail' && latest
    ? Promise.resolve(latest)
    : backend(request))
  const visible = ref(true)
  wrapper = mount({
    components: { Delivery },
    setup: () => ({ visible }),
    template: '<keep-alive><Delivery v-if="visible" /></keep-alive>'
  }, { global: { stubs: {
    'el-table': { props: ['data'], template: '<div><slot /></div>' },
    'el-table-column': { template: '<div />' },
    'el-dialog': { props: ['modelValue'], template: '<div v-if="modelValue"><slot /><slot name="footer" /></div>' }
  } } })
  await flushPromises()
  let state = wrapper.findComponent(Delivery).vm.$.setupState
  state.form.department = '缓存中的草稿'
  visible.value = false
  await flushPromises()
  expect(JSON.parse(sessionStorage.getItem('warehouse.delivery.1.draft.10')).form.department).toBe('缓存中的草稿')
  sessionStorage.removeItem('warehouse.delivery.1.draft.10')
  latest = { Note: { id: 10, version: 4, number: 'SK10', status: 'draft', brandId: 2, department: '已保存的新版本', deliveredAt: '2026-01-01T00:00:00Z' }, Details: [] }
  visible.value = true
  await flushPromises()
  state = wrapper.findComponent(Delivery).vm.$.setupState
  expect(state.form.department).toBe('已保存的新版本')
  expect(state.note.version).toBe(4)
  expect(state.dirty).toBe(false)
  expect(sessionStorage.getItem('warehouse.delivery.1.draft.10')).toBeNull()
})

it('removes an existing draft when all edits are intentionally reverted to the saved values', async () => {
  let state = await mountEditableDelivery()
  state.form.department = '临时修改'
  await flushPromises()
  expect(sessionStorage.getItem('warehouse.delivery.1.draft.10')).not.toBeNull()
  state.form.department = ''
  await flushPromises()
  expect(sessionStorage.getItem('warehouse.delivery.1.draft.10')).toBeNull()
  state = await remountDelivery()
  expect(state.form.department).toBe('')
  expect(state.dirty).toBe(false)
})
