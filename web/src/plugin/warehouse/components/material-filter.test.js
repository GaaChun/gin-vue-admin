import { afterEach, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import MaterialFilter from './material-filter.vue'

const materials = [
  { id: 3, code: 'MAT-RED', name: '红油', mnemonic: 'HongYou' },
  { ID: 7, code: 'MAT-WHITE', name: '白油', mnemonic: 'BaiYou' },
  { id: 11, code: 'BOX-01', name: '包装箱', mnemonic: 'BaoZhuangXiang' }
]
let wrapper
function setup(modelValue = []) {
  wrapper = mount(MaterialFilter, {
    props: {
      materials,
      modelValue,
      'onUpdate:modelValue': value => wrapper.setProps({ modelValue: value })
    },
    global: {
      plugins: [ElementPlus],
      stubs: {
        'el-popover': { template: '<div><slot name="reference" /><slot /></div>' }
      }
    }
  })
  return wrapper
}
afterEach(() => wrapper?.unmount())
const options = () => wrapper.findAll('.material-filter-option')
const visibleCodes = () => options().map(option => option.find('small').text())
const search = keyword => wrapper.find('.material-filter-search input').setValue(keyword)

it.each([
  [' mat-white ', ['MAT-WHITE']],
  ['红油', ['MAT-RED']],
  ['HONGYOU', ['MAT-RED']],
  ['not-a-material', []]
])('searches material codes, names and mnemonics with %s', async (keyword, expected) => {
  setup()
  await search(keyword)
  expect(visibleCodes()).toEqual(expected)
  expect(wrapper.emitted('update:modelValue')).toBeUndefined()
})

it('keeps selected numeric IDs while searching and restores selected state when candidates reappear', async () => {
  setup()
  expect(wrapper.find('.material-filter input[readonly]').exists()).toBe(true)
  await options()[0].trigger('click')
  expect(wrapper.props('modelValue')).toEqual([3])
  expect(options()[0].attributes('aria-pressed')).toBe('true')

  await search('BaiYou')
  expect(visibleCodes()).toEqual(['MAT-WHITE'])
  expect(wrapper.props('modelValue')).toEqual([3])
  await options()[0].trigger('click')
  expect(wrapper.props('modelValue')).toEqual([3, 7])
  expect(wrapper.props('modelValue').every(id => typeof id === 'number')).toBe(true)

  await search('')
  expect(options().map(option => option.attributes('aria-pressed'))).toEqual(['true', 'true', 'false'])
  await options()[0].trigger('click')
  expect(wrapper.props('modelValue')).toEqual([7])
  expect(options()[0].attributes('aria-pressed')).toBe('false')
})

it('selects only matching candidates, keeps prior selections and deduplicates repeated select-all actions', async () => {
  setup([11])
  await search('MAT-')
  await wrapper.find('.material-filter-select-all').trigger('click')
  expect(wrapper.props('modelValue')).toEqual([11, 3, 7])
  await wrapper.find('.material-filter-select-all').trigger('click')
  expect(wrapper.props('modelValue')).toEqual([11, 3, 7])
  expect(new Set(wrapper.props('modelValue')).size).toBe(3)

  await search('BOX-01')
  expect(options()[0].attributes('aria-pressed')).toBe('true')
  await wrapper.find('.material-filter-select-all').trigger('click')
  expect(wrapper.props('modelValue')).toEqual([11, 3, 7])
})

it('cancels all selections even when the search hides some selected materials', async () => {
  setup([3, 7, 11])
  await search('BOX-01')
  await wrapper.find('.material-filter-deselect-all').trigger('click')
  expect(wrapper.props('modelValue')).toEqual([])
  expect(options()[0].attributes('aria-pressed')).toBe('false')
  await search('')
  expect(options().every(option => option.attributes('aria-pressed') === 'false')).toBe(true)
})

it('clears the filter through its inline clear control and removes the selection summary', async () => {
  setup([3, 7])
  const input = wrapper.find('.material-filter input[readonly]')
  expect(input.element.value).not.toBe('')
  await wrapper.find('.material-filter-clear').trigger('click')
  expect(wrapper.props('modelValue')).toEqual([])
  expect(input.element.value).toBe('')
  expect(wrapper.find('.material-filter-clear').exists()).toBe(false)
})

it('accepts externally reset values and disables bulk selection when there are no matches', async () => {
  setup([3])
  await wrapper.setProps({ modelValue: [7] })
  expect(options().map(option => option.attributes('aria-pressed'))).toEqual(['false', 'true', 'false'])
  await search('missing')
  expect(wrapper.find('.material-filter-select-all').attributes('disabled')).toBeDefined()
  expect(wrapper.props('modelValue')).toEqual([7])
  await wrapper.setProps({ modelValue: [] })
  expect(wrapper.find('.material-filter-deselect-all').attributes('disabled')).toBeDefined()
  expect(wrapper.find('.material-filter input[readonly]').element.value).toBe('')
})
