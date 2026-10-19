import { ref } from 'vue'
import { getTableWidths, saveTableWidths } from './api/table-widths'
import { showError } from './api/common'

export function useTableWidths() {
  const widths = ref({})
  let loadSequence = 0
  let writeQueue = Promise.resolve()

  async function load(tableKey) {
    const sequence = ++loadSequence
    widths.value = {}
    try {
      const saved = await getTableWidths(tableKey)
      if (sequence === loadSequence) widths.value = saved || {}
    } catch (error) {
      if (sequence === loadSequence) showError(error)
    }
  }

  function headerDragEnd(tableKey, newWidth, _oldWidth, column) {
    const key = column.property || column.columnKey
    const width = Math.round(newWidth)
    if (!key || !Number.isFinite(width)) return
    const next = { ...widths.value, [key]: width }
    widths.value = next
    writeQueue = writeQueue.catch(() => {}).then(async () => {
      try {
        await saveTableWidths({ tableKey, widths: next })
      } catch (error) {
        showError(error)
      }
    })
    return writeQueue
  }

  return { widths, load, headerDragEnd }
}
