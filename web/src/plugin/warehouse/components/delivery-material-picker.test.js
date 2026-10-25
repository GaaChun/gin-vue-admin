import { afterEach, expect, it } from 'vitest'
import { DOMWrapper, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ElementPlus from 'element-plus'
import DeliveryMaterialPicker from './delivery-material-picker.vue'

const materials = [
  { id: 1, code: 'MAT-RED', name: '红油', specification: '500ml', mnemonic: 'HongYou', categoryId: 10 },
  { ID: 2, code: 'MAT-WHITE', name: '白油', specification: '1L', mnemonic: 'BaiYou', categoryId: 20 },
  { id: 3, code: 'BOX-01', name: '包装箱', specification: '大号', mnemonic: 'BaoZhuangXiang', categoryId: 11 },
  { id: 4, code: 'VEG-01', name: '青菜', specification: '散装', mnemonic: 'QingCai', categoryId: 30 }
]
const categories = [
  { id: 10, label: '食品', children: [{ id: 11, label: '食品包装' }] },
  { id: 20, label: '油料' },
  { id: 30, label: '蔬菜' }
]
let wrapper

async function setup(props = {}) {
  wrapper = mount(DeliveryMaterialPicker, {
    attachTo: document.body,
    props: {
      modelValue: true,
      materials,
      categories,
      ...props,
      'onUpdate:modelValue': value => wrapper.setProps({ modelValue: value })
    },
    global: {
      plugins: [ElementPlus],
      stubs: {
        'el-dialog': {
          props: ['modelValue'],
          template: '<section v-if="modelValue"><slot /><slot name="footer" /></section>'
        }
      }
    }
  })
  await nextTick()
  return wrapper
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

const control = name => wrapper.get(`[data-testid="${name}"]`)
const tableRows = side => control(`${side}-table`).findAll('.el-table__body tbody tr')
const search = (side, value) => control(`${side}-search`).setValue(value)
const click = name => control(name).trigger('click')

function expectRows(side, names) {
  const rows = tableRows(side)
  expect(rows).toHaveLength(names.length)
  names.forEach((name, index) => expect(rows[index].text()).toContain(name))
}

async function checkRow(side, name) {
  const row = tableRows(side).find(item => item.text().includes(name))
  expect(row, `visible ${side} row ${name}`).toBeDefined()
  await row.get('input[type="checkbox"]').setValue(true)
}

async function chooseCategory(label) {
  await control('category-filter').get('.el-select__wrapper').trigger('click')
  await nextTick()
  const body = new DOMWrapper(document.body)
  const node = body.findAll('.el-tree-node__content').find(item => item.text() === label)
  expect(node, `category option ${label}`).toBeDefined()
  await node.trigger('click')
  await nextTick()
}

it('checks candidates before moving them right, then checks selected rows before moving them left', async () => {
  await setup()
  expectRows('candidate', ['红油', '白油', '包装箱', '青菜'])
  expectRows('selected', [])
  expect(control('move-right').attributes('disabled')).toBeDefined()
  expect(control('move-left').attributes('disabled')).toBeDefined()

  await checkRow('candidate', '白油')
  expectRows('selected', [])
  expectRows('candidate', ['红油', '白油', '包装箱', '青菜'])
  expect(control('move-right').attributes('disabled')).toBeUndefined()
  await click('move-right')
  expectRows('candidate', ['红油', '包装箱', '青菜'])
  expectRows('selected', ['白油'])
  expect(control('move-right').attributes('disabled')).toBeDefined()

  await checkRow('selected', '白油')
  expectRows('selected', ['白油'])
  await click('move-left')
  expectRows('candidate', ['红油', '白油', '包装箱', '青菜'])
  expectRows('selected', [])
  expect(control('move-left').attributes('disabled')).toBeDefined()
  expect(wrapper.emitted('confirm')).toBeUndefined()
})

it('clears hidden checkbox selections when filtering rather than transferring stale rows', async () => {
  await setup()
  await checkRow('candidate', '红油')
  await search('candidate', '白油')
  expectRows('candidate', ['白油'])
  expect(control('move-right').attributes('disabled')).toBeDefined()
  await search('candidate', '')
  expect(tableRows('candidate')[0].get('input[type="checkbox"]').element.checked).toBe(false)
  await search('candidate', 'MAT-')
  await click('move-all-right')
  await checkRow('selected', '红油')
  await search('selected', '白油')
  expectRows('selected', ['白油'])
  expect(control('move-left').attributes('disabled')).toBeDefined()
  await search('selected', '')
  expectRows('selected', ['红油', '白油'])
  expect(tableRows('selected')[0].get('input[type="checkbox"]').element.checked).toBe(false)
  expect(wrapper.emitted('confirm')).toBeUndefined()
})

it('moves all and only currently visible rows in either direction', async () => {
  await setup()
  await search('candidate', 'MAT-')
  await click('move-all-right')
  expectRows('candidate', [])
  expectRows('selected', ['红油', '白油'])
  expect(control('move-all-right').attributes('disabled')).toBeDefined()

  await search('candidate', '')
  await click('move-all-right')
  expectRows('selected', ['红油', '白油', '包装箱', '青菜'])
  await search('selected', '油')
  await click('move-all-left')
  expectRows('selected', [])
  expect(control('move-all-left').attributes('disabled')).toBeDefined()
  expectRows('candidate', ['红油', '白油'])
  await search('selected', '')
  expectRows('selected', ['包装箱', '青菜'])
  expect(wrapper.emitted('confirm')).toBeUndefined()
})

it.each([
  [' mat-white ', ['白油']],
  ['红油', ['红油']],
  ['HONGYOU', ['红油']],
  ['500ml', ['红油']],
  ['not-a-material', []]
])('filters visible candidates by code, name, specification or mnemonic: %s', async (keyword, names) => {
  await setup()
  await search('candidate', keyword)
  expectRows('candidate', names)
  expectRows('selected', [])
  expect(wrapper.emitted('confirm')).toBeUndefined()
})

it('keeps left and right searches independent without discarding hidden selected rows', async () => {
  await setup()
  await search('candidate', 'MAT-')
  await click('move-all-right')
  await search('selected', ' BAIYOU ')
  expectRows('selected', ['白油'])
  await search('candidate', '大号')
  expectRows('candidate', ['包装箱'])
  expectRows('selected', ['白油'])
  await click('move-all-right')
  expectRows('candidate', [])
  expectRows('selected', ['白油'])
  expect(control('selected-search').element.value).toBe(' BAIYOU ')
  await search('selected', '')
  expectRows('selected', ['红油', '白油', '包装箱'])
  await search('candidate', '')
  expectRows('candidate', ['青菜'])
})

it('includes category descendants, combines category with search and leaves selected rows unaffected', async () => {
  await setup()
  await checkRow('candidate', '白油')
  await click('move-right')
  await chooseCategory('食品')
  expectRows('candidate', ['红油', '包装箱'])
  expectRows('selected', ['白油'])
  await search('candidate', 'BOX-')
  expectRows('candidate', ['包装箱'])
  await click('move-all-right')
  expectRows('candidate', [])
  expectRows('selected', ['白油', '包装箱'])
  await search('candidate', '')
  expectRows('candidate', ['红油'])
  await search('selected', '白油')
  await click('move-all-left')
  expectRows('candidate', ['红油'])
  await search('selected', '')
  expectRows('selected', ['包装箱'])
})

it('deduplicates id/ID records, preserves first records and appends batches in visible order', async () => {
  await setup({ materials: [...materials, { ID: 1, code: 'DUPLICATE', name: '重复红油', categoryId: 10 }] })
  await checkRow('candidate', '白油')
  await click('move-right')
  // Checkbox click order does not reorder the visible candidate batch.
  await checkRow('candidate', '包装箱')
  await checkRow('candidate', '红油')
  await click('move-right')
  expectRows('selected', ['白油', '红油', '包装箱'])
  await click('move-all-right')
  await click('move-all-right')
  expectRows('selected', ['白油', '红油', '包装箱', '青菜'])
  await checkRow('selected', '红油')
  await click('move-left')
  await checkRow('candidate', '红油')
  await click('move-right')
  expectRows('selected', ['白油', '包装箱', '青菜', '红油'])
  expect(wrapper.emitted('confirm')).toBeUndefined()
  await click('picker-confirm')
  expect(wrapper.emitted('confirm')).toEqual([[[materials[1], materials[2], materials[3], materials[0]]]])
})

it('emits rows only on final confirmation, not while checking, filtering or transferring', async () => {
  await setup()
  expect(control('picker-confirm').attributes('disabled')).toBeDefined()
  await checkRow('candidate', '白油')
  await checkRow('candidate', '红油')
  expect(wrapper.emitted('confirm')).toBeUndefined()
  await click('move-right')
  await search('selected', '白油')
  expect(wrapper.emitted('confirm')).toBeUndefined()
  await click('picker-confirm')
  // Confirmation includes hidden selected rows in transfer order, not checkbox click order.
  expect(wrapper.emitted('confirm')).toEqual([[[materials[0], materials[1]]]])
  expect(wrapper.emitted('update:modelValue')).toBeUndefined()
})

it('cancels without emitting selected rows', async () => {
  await setup()
  await click('move-all-right')
  await click('picker-cancel')
  expect(wrapper.emitted('update:modelValue')).toEqual([[false]])
  expect(wrapper.emitted('confirm')).toBeUndefined()
  expect(wrapper.find('[data-testid="candidate-table"]').exists()).toBe(false)
})

it('resets both searches, category, selected rows and checkbox state when reopened', async () => {
  await setup()
  await checkRow('candidate', '白油')
  await click('move-right')
  await chooseCategory('食品')
  await search('candidate', '红油')
  await search('selected', '白油')
  await checkRow('candidate', '红油')
  await checkRow('selected', '白油')
  await click('picker-cancel')
  await wrapper.setProps({ modelValue: true })
  await nextTick()
  expect(control('candidate-search').element.value).toBe('')
  expect(control('selected-search').element.value).toBe('')
  expectRows('candidate', ['红油', '白油', '包装箱', '青菜'])
  expectRows('selected', [])
  expect(tableRows('candidate').every(row => !row.get('input[type="checkbox"]').element.checked)).toBe(true)
  for (const name of ['move-right', 'move-left', 'move-all-left', 'picker-confirm']) {
    expect(control(name).attributes('disabled')).toBeDefined()
  }
  expect(wrapper.emitted('confirm')).toBeUndefined()
})

it('locks transfers, confirmation, cancellation, searches and checkboxes while loading', async () => {
  await setup()
  await checkRow('candidate', '红油')
  await click('move-right')
  await checkRow('candidate', '白油')
  await checkRow('selected', '红油')
  await wrapper.setProps({ loading: true })
  for (const name of ['move-all-right', 'move-right', 'move-left', 'move-all-left', 'picker-confirm', 'picker-cancel']) {
    expect(control(name).attributes('disabled'), name).toBeDefined()
    await click(name)
  }
  for (const side of ['candidate', 'selected']) {
    expect(control(`${side}-search`).element.disabled).toBe(true)
    expect(tableRows(side).every(row => row.get('input[type="checkbox"]').element.disabled)).toBe(true)
  }
  expect(control('category-filter').find('.el-select__wrapper.is-disabled').exists()).toBe(true)
  expectRows('candidate', ['白油', '包装箱', '青菜'])
  expectRows('selected', ['红油'])
  expect(wrapper.emitted('confirm')).toBeUndefined()
  expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  await wrapper.setProps({ loading: false })
  await click('picker-confirm')
  expect(wrapper.emitted('confirm')).toEqual([[[materials[0]]]])
})
