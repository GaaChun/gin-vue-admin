export const DISPLAY_SETTING_KEY = 'warehouseDocumentDisplay.delivery'

export const FILTER_FIELDS = [
  { key: 'number', label: '单号' }, { key: 'dates', label: '送货日期' },
  { key: 'brandId', label: '客户' }, { key: 'materialIds', label: '物料' }, { key: 'department', label: '部门' }, { key: 'status', label: '状态' }
]
export const LIST_FIELDS = [
  { key: 'number', label: '单号', width: 170 }, { key: 'deliveredAt', label: '送货日期', width: 120 },
  { key: 'brandNameSnapshot', label: '客户', width: 180 }, { key: 'department', label: '部门', width: 120 },
  { key: 'status', label: '状态', width: 100 }, { key: 'totalAmount', label: '金额', width: 130 },
  { key: 'createdByNameSnapshot', label: '创建人', width: 120 }, { key: 'createdAt', label: '创建时间', width: 180 },
  { key: 'updatedByName', label: '修改人', width: 120 }, { key: 'updatedAt', label: '修改时间', width: 180 },
  { key: 'reviewedByNameSnapshot', label: '审核人', width: 120 }, { key: 'reviewedAt', label: '审核时间', width: 180 }
]
export const DETAIL_FIELDS = [
  { key: 'materialCode', label: '物料编号', width: 150, required: true },
  { key: 'materialName', label: '物料名称', width: 200, required: true },
  { key: 'specification', label: '规格型号', width: 150, required: false },
  { key: 'unitName', label: '单位', width: 100, required: true },
  { key: 'quantity', label: '数量', width: 130, required: true },
  { key: 'unitPrice', label: '单价', width: 130, required: true },
  { key: 'amount', label: '金额', width: 140, required: false }
]
const presets = ['system', 'thisMonth', 'last7', 'last30', 'today', 'yesterday', 'thisYear']
const keys = (fields) => fields.map((field) => field.key)
function cleanKeys(value, fields, fallback) {
  return Array.isArray(value) ? [...new Set(value.filter((key) => fields.some((field) => field.key === key)))] : [...fallback]
}
function cleanWidths(value, fields) {
  const widths = {}
  for (const { key } of fields) {
    const width = value?.[key]
    if (typeof width === 'number' && Number.isFinite(width)) widths[key] = Math.min(800, Math.max(80, Math.round(width)))
  }
  return widths
}
export function normalizeDisplaySettings(raw) {
  const source = raw && typeof raw === 'object' && !Array.isArray(raw) ? raw : {}
  const basic = cleanKeys(source.filters?.basic, FILTER_FIELDS, ['number', 'dates', 'brandId', 'materialIds'])
  const more = cleanKeys(source.filters?.more, FILTER_FIELDS, ['department', 'status']).filter((key) => !basic.includes(key))
  if (!basic.includes('dates') && !more.includes('dates')) basic.push('dates')
  const detailColumns = cleanKeys(source.detail?.columns, DETAIL_FIELDS, keys(DETAIL_FIELDS))
  for (const field of DETAIL_FIELDS) if (field.required && !detailColumns.includes(field.key)) detailColumns.push(field.key)
  const frozen = source.detail?.frozen
  return {
    dateDefault: presets.includes(source.dateDefault) ? source.dateDefault : 'system',
    filters: { basic, more },
    list: { columns: cleanKeys(Array.isArray(raw) ? raw : source.list?.columns, LIST_FIELDS, keys(LIST_FIELDS)), widths: cleanWidths(source.list?.widths, LIST_FIELDS) },
    detail: { columns: detailColumns, widths: cleanWidths(source.detail?.widths, DETAIL_FIELDS), frozen: typeof frozen === 'number' && Number.isFinite(frozen) ? Math.min(detailColumns.length, Math.max(0, Math.floor(frozen))) : 0 }
  }
}

// Calendar arithmetic is performed in UTC after extracting the Shanghai calendar day.
export function getDefaultDateRange(preset, now = new Date()) {
  if (!presets.includes(preset) || preset === 'system') return []
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(now)
  const part = (type) => Number(parts.find((item) => item.type === type).value)
  const year = part('year'); const month = part('month'); const day = part('day')
  const end = new Date(Date.UTC(year, month - 1, day))
  const start = new Date(end)
  if (preset === 'thisMonth') start.setUTCDate(1)
  if (preset === 'thisYear') start.setUTCMonth(0, 1)
  if (preset === 'last7') start.setUTCDate(day - 6)
  if (preset === 'last30') start.setUTCDate(day - 29)
  if (preset === 'yesterday') { start.setUTCDate(day - 1); end.setUTCDate(day - 1) }
  return [start, end].map((date) => date.toISOString().slice(0, 10))
}
