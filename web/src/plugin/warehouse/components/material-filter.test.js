import { afterEach, expect, it, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import MaterialFilter from './material-filter.vue'

const materials = [
  { id: 3, code: 'MAT-RED', name: '红油', mnemonic: 'HongYou' },
  { ID: 7, code: 'MAT-WHITE', name: '白油', mnemonic: 'BaiYou' },
  { id: 11, code: 'BOX-01', name: '包装箱', mnemonic: 'BaoZhuangXiang' }
]
let wrapper
function setup(modelValue = [], realPopover = false) {
  wrapper = mount(MaterialFilter, {
    attachTo: document.body,
    props: {
      materials,
      modelValue,
      'onUpdate:modelValue': value => wrapper.setProps({ modelValue: value })
    },
    global: {
      plugins: [ElementPlus],
      stubs: realPopover ? { transition: false } : {
        'el-popover': { template: '<div><slot name="reference" /><slot /></div>' }
      }
    }
  })
  return wrapper
}
afterEach(async () => {
  const element = wrapper?.element
  wrapper?.unmount()
  element?.remove()
  vi.useRealTimers()
  await flushPromises()
})
const options = () => wrapper.findAll('.material-filter-option')
const visibleCodes = () => options().map(option => option.find('small').text())
const search = keyword => wrapper.find('.material-filter-search input').setValue(keyword)
const pickerVisible = () => {
  const panel = document.querySelector('.material-filter-panel')
  return panel ? new DOMWrapper(panel).isVisible() : false
}
const waitForPicker = (visible, stage) => vi.waitFor(() => {
  expect(pickerVisible(), stage).toBe(visible)
}, { interval: 10, timeout: 1000 })

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

it.each(['focus', 'click', 'Enter', ' '])('opens and focuses the mounted search on reference %s', async (activation) => {
  setup([], true)
  await flushPromises()
  const reference = wrapper.find('input[readonly]')
  if (activation === 'focus') reference.element.focus()
  else if (activation === 'click') {
    reference.element.focus()
    await reference.trigger('click')
  } else await reference.trigger('keydown', { key: activation, code: activation === ' ' ? 'Space' : activation })
  await waitForPicker(true, 'initial open')

  const input = document.querySelector('.material-filter-search input')
  await vi.waitFor(() => expect(document.activeElement).toBe(input))
  input.value = 'WHITE'
  input.dispatchEvent(new Event('input', { bubbles: true }))
  await flushPromises()
  expect(Array.from(document.querySelectorAll('.material-filter-option small'), element => element.textContent)).toEqual(['MAT-WHITE'])

  input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', code: 'Escape', bubbles: true }))
  await waitForPicker(false, 'Escape close')
  await reference.trigger('keydown', { key: 'Enter', code: 'Enter' })
  await waitForPicker(true, 'reopen')
  await vi.waitFor(() => expect(document.activeElement).toBe(document.querySelector('.material-filter-search input')))
  expect(document.activeElement.value).toBe('')
})

it('shows every selected material name and code on hover without opening the picker', async () => {
  vi.useFakeTimers()
  setup([3, 7, 11], true)
  await flushPromises()
  await wrapper.find('.el-input').trigger('mouseenter')
  await vi.advanceTimersByTimeAsync(250)
  await flushPromises()

  const tooltip = document.querySelector('.material-filter-selection-tooltip')
  expect(tooltip).not.toBeNull()
  expect(new DOMWrapper(tooltip).isVisible()).toBe(true)
  for (const material of materials) {
    expect(tooltip.textContent).toContain(material.name)
    expect(tooltip.textContent).toContain(material.code)
  }
  expect(pickerVisible()).toBe(false)
})

it('clears selected materials without opening the closed picker or swallowing keyboard activation', async () => {
  setup([3, 7], true)
  await flushPromises()
  await wrapper.find('.material-filter-clear').trigger('keydown', { key: 'Enter', code: 'Enter' })
  expect(pickerVisible()).toBe(false)
  await wrapper.find('.material-filter-clear').trigger('click')
  expect(wrapper.props('modelValue')).toEqual([])
  expect(pickerVisible()).toBe(false)
})

it('keeps the teleported panel interactive and closes on an outside click', async () => {
  setup([], true)
  await flushPromises()
  wrapper.find('input[readonly]').element.focus()
  await waitForPicker(true, 'initial option open')
  const option = document.querySelector('.material-filter-option')
  option.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))
  option.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }))
  option.click()
  await flushPromises()
  expect(wrapper.props('modelValue')).toEqual([3])
  await waitForPicker(true, 'after selection')

  document.body.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))
  document.body.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }))
  await waitForPicker(false, 'outside close')
})

it('shows selected details on a fresh hover after Escape without reopening on restored focus', async () => {
  setup([], true)
  await flushPromises()
  wrapper.find('input[readonly]').element.focus()
  await waitForPicker(true, 'open before selecting')
  document.querySelector('.material-filter-option').click()
  await flushPromises()
  document.querySelector('.material-filter-search input').dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', code: 'Escape', bubbles: true }))
  await waitForPicker(false, 'Escape after selecting')
  await wrapper.find('.el-input').trigger('mouseleave')
  await wrapper.find('.el-input').trigger('mouseenter')
  await vi.waitFor(() => {
    const tooltip = document.querySelector('.material-filter-selection-tooltip')
    expect(tooltip).not.toBeNull()
    expect(new DOMWrapper(tooltip).isVisible()).toBe(true)
    expect(tooltip.textContent).toContain('红油')
    expect(tooltip.textContent).toContain('MAT-RED')
    expect(pickerVisible()).toBe(false)
  }, { interval: 10, timeout: 1000 })
})
