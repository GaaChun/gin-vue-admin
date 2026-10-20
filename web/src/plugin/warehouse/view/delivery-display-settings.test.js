import { afterEach, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import Settings from './delivery-display-settings.vue'
import { DISPLAY_SETTING_KEY } from '../delivery-display'

const mocks = vi.hoisted(() => ({ user: null, setSelfSetting: vi.fn(), confirm: vi.fn(), success: vi.fn(), guard: null }))
vi.mock('@/api/user', () => ({ setSelfSetting: mocks.setSelfSetting }))
vi.mock('@/pinia/modules/user', () => ({ useUserStore: () => mocks.user }))
vi.mock('vue-router', () => ({ onBeforeRouteLeave: guard => { mocks.guard = guard } }))
vi.mock('element-plus', () => ({ ElMessage: { success: mocks.success }, ElMessageBox: { confirm: mocks.confirm } }))
let wrapper
function mountSettings() {
  mocks.user = reactive({ userInfo: { ID: 1, originSetting: { primaryColor: '#123456' } }, ResetUserInfo: vi.fn(function (value) { this.userInfo = { ...this.userInfo, ...value } }) })
  wrapper = shallowMount(Settings, { global: { directives: { loading: () => {} } } })
  return wrapper.vm.$.setupState
}
afterEach(() => { wrapper?.unmount(); vi.clearAllMocks() })
it('keeps drafts local until successful self-setting persistence and preserves unrelated settings', async () => {
  const state = mountSettings()
  state.draft.list.columns = []
  expect(mocks.user.userInfo.originSetting[DISPLAY_SETTING_KEY]).toBeUndefined()
  mocks.setSelfSetting.mockResolvedValue({ code: 0 })
  await state.save()
  expect(mocks.setSelfSetting).toHaveBeenCalledWith({ [DISPLAY_SETTING_KEY]: expect.objectContaining({ list: { columns: [], widths: {} } }) })
  expect(mocks.user.userInfo.originSetting.primaryColor).toBe('#123456')
  expect(mocks.user.userInfo.originSetting[DISPLAY_SETTING_KEY].list.columns).toEqual([])
  expect(state.dirty).toBe(false)
})
it('retains the draft and persisted settings when save fails', async () => {
  const state = mountSettings()
  state.draft.list.columns = []
  mocks.setSelfSetting.mockResolvedValue({ code: 7, msg: '保存失败' })
  await state.save()
  expect(mocks.user.ResetUserInfo).not.toHaveBeenCalled()
  expect(state.saveError).toBe('保存失败')
  expect(state.dirty).toBe(true)
})
it('discards old-account drafts and ignores an old-account save completing after account change', async () => {
  const state = mountSettings()
  let resolve
  mocks.setSelfSetting.mockReturnValue(new Promise(done => { resolve = done }))
  state.draft.list.columns = []
  const pending = state.save()
  mocks.user.userInfo = { ID: 2, originSetting: { [DISPLAY_SETTING_KEY]: ['status'] } }
  await flushPromises()
  expect(state.draft.list.columns).toEqual(['status'])
  resolve({ code: 0 })
  await pending
  expect(mocks.user.ResetUserInfo).not.toHaveBeenCalled()
  expect(mocks.success).not.toHaveBeenCalled()
  expect(mocks.user.userInfo.ID).toBe(2)
})
it('confirms unsaved route navigation and cancels leaving when dismissed', async () => {
  const state = mountSettings()
  expect(await mocks.guard()).toBe(true)
  state.draft.list.columns = []
  mocks.confirm.mockRejectedValue('cancel')
  expect(await mocks.guard()).toBe(false)
  mocks.confirm.mockResolvedValue('confirm')
  expect(await mocks.guard()).toBe(true)
})
it('refreshes external current-account settings only while the draft is clean', async () => {
  const state = mountSettings()
  mocks.user.userInfo.originSetting[DISPLAY_SETTING_KEY] = ['status']
  await flushPromises()
  expect(state.draft.list.columns).toEqual(['status'])
  expect(state.dirty).toBe(false)
  state.draft.list.columns = ['number']
  mocks.user.userInfo.originSetting[DISPLAY_SETTING_KEY] = ['department']
  await flushPromises()
  expect(state.draft.list.columns).toEqual(['number'])
  expect(state.dirty).toBe(true)
})
