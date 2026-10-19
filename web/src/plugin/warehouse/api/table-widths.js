import { setSelfSetting } from '@/api/user'
import { useUserStore } from '@/pinia/modules/user'

const settingKey = (tableKey) => `warehouseTableWidths.${tableKey}`

export const getTableWidths = (tableKey) => {
  const setting = useUserStore().userInfo.originSetting || {}
  return setting[settingKey(tableKey)] || {}
}

export const saveTableWidths = async ({ tableKey, widths }) => {
  const user = useUserStore()
  const settings = user.userInfo.originSetting || {}
  const key = settingKey(tableKey)
  const response = await setSelfSetting({ [key]: widths })
  if (response?.code !== 0) throw new Error(response?.msg || '保存表格列宽失败')
  user.ResetUserInfo({ originSetting: { ...settings, [key]: widths } })
}
