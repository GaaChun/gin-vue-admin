<template>
  <div class="display-settings gva-table-box">
    <header class="page-heading"><div><h2>单据展示设置</h2><p>仅对当前登录账号生效，保存后应用于送货单与对账表。</p></div><el-tag :type="dirty ? 'warning' : 'success'">{{ dirty ? '有未保存的修改' : '已保存' }}</el-tag></header>
    <div class="settings-layout">
      <aside class="document-tree"><div class="tree-title">单据管理</div><div class="tree-item">送货单与对账表</div></aside>
      <main class="settings-main" v-loading="saving">
        <el-alert v-if="saveError" :title="saveError" type="error" show-icon :closable="false" class="save-error" />
        <el-tabs v-model="tab"><el-tab-pane label="列表展示" name="list" /><el-tab-pane label="详情展示" name="detail" /><el-tab-pane label="对账表查询" name="reconciliation" /></el-tabs>
        <template v-if="tab === 'list'">
          <section class="setting-section"><div class="section-heading"><h3>日期筛选默认值</h3><span>按上海时区计算，近 7 / 30 天含当天</span></div><el-radio-group v-model="draft.dateDefault"><el-radio v-for="preset in datePresets" :key="preset.key" :value="preset.key">{{ preset.label }}</el-radio></el-radio-group></section>
          <section class="setting-section"><div class="section-heading"><h3>筛选区域</h3><span>拖动调整顺序或移入其他分组，送货日期不可移除</span></div>
            <div v-for="group in filterGroups" :key="group.key" class="filter-row"><div class="group-label">{{ group.label }}</div><draggable v-model="filterLists[group.key]" :group="{ name: 'delivery-filters' }" item-key="key" class="filter-drop" :move="canMoveFilter" :disabled="saving" :animation="180" @change="syncFilters"><template #item="{ element }"><div class="filter-preview"><span class="drag-handle">⋮⋮</span><span>{{ element.label }}<b v-if="element.key === 'dates'" class="required-mark"> *</b></span><div v-if="group.key !== 'unused'" class="input-preview">{{ element.key === 'dates' ? '开始日期 — 结束日期' : '请选择或输入' }}</div><el-button v-if="group.key !== 'unused' && element.key !== 'dates'" link type="info" @click="removeFilter(element.key)">移除</el-button><el-button v-if="group.key === 'unused'" link type="primary" @click="addFilter(element.key)">添加</el-button></div></template><template #footer><span v-if="!filterLists[group.key].length" class="empty-tip">拖入字段</span></template></draggable></div>
          </section>
        </template>
        <section v-if="tab === 'reconciliation'" class="setting-section">
          <div class="section-heading"><h3>对账表查询条件</h3><span>拖动配置默认筛选、更多筛选及未使用字段；隐藏条件不参与新查询。</span></div>
          <div v-for="group in filterGroups" :key="group.key" class="filter-row">
            <div class="group-label">{{ group.label }}</div>
            <draggable v-model="reportFilterLists[group.key]" :group="{ name: 'reconciliation-filters' }" item-key="key" class="filter-drop" :disabled="saving" :animation="180" @change="syncReportFilters">
              <template #item="{ element }">
                <div class="filter-preview"><span class="drag-handle">⋮⋮</span><span>{{ element.label }}</span>
                  <div v-if="group.key !== 'unused'" class="input-preview">{{ ['deliveredRange', 'reviewedRange'].includes(element.key) ? '开始日期 — 结束日期' : '请选择或输入' }}</div>
                  <el-button v-if="group.key !== 'unused'" link type="info" @click="removeReportFilter(element.key)">移除</el-button>
                  <el-button v-else link type="primary" @click="addReportFilter(element.key)">添加</el-button>
                </div>
              </template>
              <template #footer><span v-if="!reportFilterLists[group.key].length" class="empty-tip">拖入字段</span></template>
            </draggable>
          </div>
        </section>
        <section v-if="tab !== 'reconciliation'" class="setting-section"><div class="section-heading"><h3>{{ tab === 'list' ? '列表区域' : '明细区域' }}</h3><span>拖动列头排序；列宽 80–800 px{{ tab === 'detail' ? '；带 * 的字段不可移除' : '' }}</span></div>
          <el-checkbox v-if="tab === 'list'" v-model="draft.list.showActions" :disabled="saving">显示操作列</el-checkbox>
          <div v-if="tab === 'detail'" class="freeze-control"><span>冻结至第</span><el-input-number v-model="draft.detail.frozen" :min="0" :max="draft.detail.columns.length" :precision="0" size="small" /><span>列（0 表示不冻结）</span></div>
          <div class="column-scroll"><draggable :key="`${tab}-selected`" v-model="columnLists[tab]" :group="{ name: `delivery-${tab}-columns` }" item-key="key" class="column-preview" :move="canMoveColumn" :animation="180" :disabled="saving" @change="syncColumns"><template #item="{ element, index }"><div class="preview-column" :class="{ frozen: tab === 'detail' && index < draft.detail.frozen }" :style="{ width: `${draft[tab].widths[element.key] || element.width}px` }"><div class="column-heading"><span class="drag-handle">⋮⋮</span>{{ element.label }}<b v-if="element.required" class="required-mark"> *</b><el-button v-if="!element.required" link type="info" @click="removeColumn(element.key)">移除</el-button></div><div class="column-sample">{{ sampleValue(element.key) }}</div><div class="column-width"><span>列宽</span><el-input-number :model-value="draft[tab].widths[element.key] || element.width" :min="80" :max="800" :precision="0" size="small" controls-position="right" @update:model-value="setWidth(element.key, $event)" /></div></div></template><template #footer><div v-if="!columnLists[tab].length" class="empty-tip">拖入可用列以显示；也可保留空列表</div></template></draggable></div>
          <div class="spare-heading">可用字段 <span>拖入上方展示区域，或点击添加</span></div><draggable :key="`${tab}-spare`" v-model="spareLists[tab]" :group="{ name: `delivery-${tab}-columns` }" item-key="key" class="spare-columns" :move="canMoveColumn" :animation="180" :disabled="saving" @change="syncColumns"><template #item="{ element }"><el-tag class="spare-tag" effect="plain" @click="addColumn(element.key)"><span class="drag-handle">⋮⋮</span>{{ element.label }} ＋</el-tag></template><template #footer><span v-if="!spareLists[tab].length" class="empty-tip">所有字段均已展示</span></template></draggable>
        </section>
        <footer class="settings-footer"><span>设置仅保存在当前账号，不影响其他员工。</span><div><el-button :disabled="saving" @click="restore">恢复默认</el-button><el-button type="primary" :loading="saving" :disabled="!dirty || !accountKey" @click="save">保存设置</el-button></div></footer>
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, onActivated, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import draggable from 'vuedraggable'
import { ElMessage, ElMessageBox } from 'element-plus'
import { setSelfSetting } from '@/api/user'
import { useUserStore } from '@/pinia/modules/user'
import { DISPLAY_SETTING_KEY, FILTER_FIELDS, RECONCILIATION_FILTER_FIELDS, LIST_FIELDS, DETAIL_FIELDS, normalizeDisplaySettings } from '../delivery-display'

const user = useUserStore()
const accountKey = computed(() => String(user.userInfo.ID || user.userInfo.id || user.userInfo.uuid || ''))
const tab = ref('list')
const draft = ref(normalizeDisplaySettings())
const savedSignature = ref('')
const dirty = computed(() => JSON.stringify(draft.value) !== savedSignature.value)
const saving = ref(false)
const saveError = ref('')
const filterLists = reactive({ basic: [], more: [], unused: [] })
const reportFilterLists = reactive({ basic: [], more: [], unused: [] })
const columnLists = reactive({ list: [], detail: [] })
const spareLists = reactive({ list: [], detail: [] })
let accountGeneration = 0
const datePresets = [
  { key: 'system', label: '系统默认' }, { key: 'thisMonth', label: '本月' }, { key: 'last7', label: '近7天' },
  { key: 'last30', label: '近30天' }, { key: 'today', label: '今天' }, { key: 'yesterday', label: '昨天' }, { key: 'thisYear', label: '本年' }
]
const filterGroups = [{ key: 'basic', label: '默认筛选' }, { key: 'more', label: '更多筛选' }, { key: 'unused', label: '未使用字段' }]
function rebuildLists() {
  for (const group of ['basic', 'more']) filterLists[group] = draft.value.filters[group].map((key) => FILTER_FIELDS.find((field) => field.key === key))
  filterLists.unused = FILTER_FIELDS.filter((field) => ![...draft.value.filters.basic, ...draft.value.filters.more].includes(field.key))
  for (const group of ['basic', 'more']) reportFilterLists[group] = draft.value.reconciliation.filters[group].map(key => RECONCILIATION_FILTER_FIELDS.find(field => field.key === key))
  reportFilterLists.unused = RECONCILIATION_FILTER_FIELDS.filter(field => ![...draft.value.reconciliation.filters.basic, ...draft.value.reconciliation.filters.more].includes(field.key))
  for (const [group, fields] of [['list', LIST_FIELDS], ['detail', DETAIL_FIELDS]]) {
    columnLists[group] = draft.value[group].columns.map((key) => fields.find((field) => field.key === key))
    spareLists[group] = fields.filter((field) => !draft.value[group].columns.includes(field.key))
  }
}
function reset(raw) {
  draft.value = normalizeDisplaySettings(raw)
  rebuildLists()
  saveError.value = ''
}
watch(accountKey, () => {
  accountGeneration++
  saving.value = false
  reset(user.userInfo.originSetting?.[DISPLAY_SETTING_KEY])
  savedSignature.value = JSON.stringify(draft.value)
}, { immediate: true, flush: 'sync' })
function refreshSavedSettings() {
  if (dirty.value || saving.value) return
  reset(user.userInfo.originSetting?.[DISPLAY_SETTING_KEY])
  savedSignature.value = JSON.stringify(draft.value)
}
watch(() => user.userInfo.originSetting?.[DISPLAY_SETTING_KEY], refreshSavedSettings, { deep: true })
onActivated(refreshSavedSettings)
function syncFilters() {
  draft.value.filters = { basic: filterLists.basic.map((field) => field.key), more: filterLists.more.map((field) => field.key) }
}
function canMoveFilter(event) { return event.draggedContext.element.key !== 'dates' || event.relatedContext.list !== filterLists.unused }
function removeFilter(key) {
  for (const group of ['basic', 'more']) filterLists[group] = filterLists[group].filter((field) => field.key !== key)
  syncFilters(); rebuildLists()
}
function addFilter(key) { filterLists.basic.push(FILTER_FIELDS.find((field) => field.key === key)); syncFilters(); rebuildLists() }
function syncReportFilters() {
  draft.value.reconciliation.filters = { basic: reportFilterLists.basic.map(field => field.key), more: reportFilterLists.more.map(field => field.key) }
}
function removeReportFilter(key) {
  for (const group of ['basic', 'more']) reportFilterLists[group] = reportFilterLists[group].filter(field => field.key !== key)
  syncReportFilters(); rebuildLists()
}
function addReportFilter(key) {
  reportFilterLists.basic.push(RECONCILIATION_FILTER_FIELDS.find(field => field.key === key))
  syncReportFilters(); rebuildLists()
}
function syncColumns() {
  for (const group of ['list', 'detail']) draft.value[group].columns = columnLists[group].map((field) => field.key)
  draft.value.detail.frozen = Math.min(draft.value.detail.frozen, draft.value.detail.columns.length)
}
function canMoveColumn(event) { return !event.draggedContext.element.required || event.relatedContext.list !== spareLists.detail }
function removeColumn(key) { columnLists[tab.value] = columnLists[tab.value].filter((field) => field.key !== key); syncColumns(); rebuildLists() }
function addColumn(key) { columnLists[tab.value].push(spareLists[tab.value].find((field) => field.key === key)); syncColumns(); rebuildLists() }
function setWidth(key, width) { if (Number.isFinite(width)) draft.value[tab.value].widths[key] = width }
function sampleValue(key) {
  if (['quantity', 'unitPrice', 'amount', 'totalAmount'].includes(key)) return '100.00'
  if (key === 'status') return '未审核'
  return '—'
}
async function restore() {
  try {
    await ElMessageBox.confirm('恢复送货单及对账表的默认展示设置？保存后生效。', '恢复默认', { type: 'warning', confirmButtonText: '恢复默认', cancelButtonText: '取消' })
    reset()
  } catch { /* User cancelled. */ }
}
async function save() {
  if (saving.value || !accountKey.value) return
  const owner = accountKey.value
  const generation = accountGeneration
  const settings = normalizeDisplaySettings(draft.value)
  saving.value = true
  saveError.value = ''
  try {
    const result = await setSelfSetting({ [DISPLAY_SETTING_KEY]: settings })
    if (owner !== accountKey.value || generation !== accountGeneration) return
    if (result?.code !== 0) throw new Error(result?.msg || '保存单据展示设置失败')
    user.ResetUserInfo({ originSetting: { ...(user.userInfo.originSetting || {}), [DISPLAY_SETTING_KEY]: settings } })
    reset(settings)
    savedSignature.value = JSON.stringify(settings)
    ElMessage.success('保存成功，仅对当前账号生效')
  } catch (error) {
    if (owner === accountKey.value && generation === accountGeneration) saveError.value = error?.message || '保存失败，请重试'
  } finally {
    if (owner === accountKey.value && generation === accountGeneration) saving.value = false
  }
}
onBeforeRouteLeave(async () => {
  if (!dirty.value) return true
  try {
    await ElMessageBox.confirm('有未保存的展示设置，离开将丢失这些修改。确定离开？', '未保存的修改', { type: 'warning', confirmButtonText: '离开', cancelButtonText: '继续编辑' })
    return true
  } catch { return false }
})
function beforeUnload(event) { if (dirty.value) { event.preventDefault(); event.returnValue = '' } }
window.addEventListener('beforeunload', beforeUnload)
onBeforeUnmount(() => { accountGeneration++; window.removeEventListener('beforeunload', beforeUnload) })
</script>

<style scoped>
.display-settings,.settings-layout,.settings-main,.setting-section{min-width:0;max-width:100%;box-sizing:border-box}
.settings-main{overflow:hidden}
.column-scroll{width:100%;max-width:100%;overflow-x:auto}
.display-settings{padding:24px}.page-heading{display:flex;align-items:center;justify-content:space-between;gap:16px;margin-bottom:24px}.page-heading h2{margin:0;font-size:20px}.page-heading p{margin:8px 0 0;color:var(--el-text-color-secondary);font-size:13px}.settings-layout{display:grid;grid-template-columns:180px minmax(0,1fr);gap:24px}.document-tree{border-right:1px solid var(--el-border-color-lighter);padding-right:16px}.tree-title{font-weight:600;padding:12px}.tree-item{margin-top:4px;padding:12px 12px 12px 28px;border-radius:6px;background:var(--el-color-primary-light-9);color:var(--el-color-primary);border-left:3px solid var(--el-color-primary)}.settings-main{min-width:0}.setting-section{margin-bottom:24px;padding:18px;border:1px solid var(--el-border-color-lighter);border-radius:8px}.section-heading{display:flex;align-items:baseline;gap:16px;flex-wrap:wrap;margin-bottom:18px}.section-heading h3{font-size:15px;margin:0}.section-heading>span,.spare-heading span{font-size:12px;color:var(--el-text-color-secondary)}.filter-row{display:flex;gap:16px;margin-top:14px}.group-label{flex:0 0 86px;padding-top:12px;font-size:13px;color:var(--el-text-color-regular)}.filter-drop{display:flex;flex:1;flex-wrap:wrap;align-items:center;gap:10px;min-height:62px;padding:12px;border:1px dashed var(--el-border-color);border-radius:6px;background:var(--el-fill-color-lighter)}.filter-preview{display:flex;align-items:center;flex-wrap:wrap;gap:8px;padding:9px 10px;background:var(--el-bg-color);border:1px solid var(--el-border-color-light);border-radius:4px;cursor:grab;font-size:13px}.input-preview{padding:6px 12px;border:1px solid var(--el-border-color);border-radius:4px;color:var(--el-text-color-placeholder);font-size:12px}.drag-handle{color:var(--el-text-color-placeholder);cursor:grab;margin-right:4px}.required-mark{color:var(--el-color-danger)}.column-scroll{overflow-x:auto;border:1px solid var(--el-border-color-light);border-radius:5px}.column-preview{display:flex;min-height:138px;min-width:100%;width:max-content}.preview-column{flex-shrink:0;border-right:1px solid var(--el-border-color-light)}.column-heading{display:flex;align-items:center;gap:3px;min-height:42px;padding:0 10px;background:var(--el-fill-color-light);cursor:grab;font-size:13px;white-space:nowrap}.column-heading .el-button{margin-left:auto;font-size:12px}.column-sample{padding:14px 12px;color:var(--el-text-color-placeholder);border-top:1px solid var(--el-border-color-lighter);border-bottom:1px solid var(--el-border-color-lighter)}.column-width{display:flex;flex-wrap:wrap;align-items:center;gap:5px;padding:8px;font-size:12px;color:var(--el-text-color-secondary)}.column-width .el-input-number{width:90px}.preview-column.frozen .column-heading{background:var(--el-color-primary-light-9);color:var(--el-color-primary)}.freeze-control{display:flex;gap:10px;align-items:center;margin-bottom:16px;font-size:13px;flex-wrap:wrap}.spare-heading{font-size:13px;margin:18px 0 10px}.spare-heading span{margin-left:10px}.spare-columns{min-height:48px;display:flex;flex-wrap:wrap;align-items:center;gap:10px;padding:12px;border:1px dashed var(--el-border-color);border-radius:5px}.spare-tag{cursor:grab}.empty-tip{padding:8px;color:var(--el-text-color-placeholder);font-size:12px}.settings-footer{display:flex;justify-content:space-between;align-items:center;gap:16px;padding:6px 0}.settings-footer>span{color:var(--el-text-color-secondary);font-size:12px}.save-error{margin-bottom:16px}@media(max-width:1000px){.settings-layout{grid-template-columns:140px minmax(0,1fr);gap:16px}.filter-row{flex-direction:column;gap:6px}.group-label{flex-basis:auto;padding-top:0}}@media(max-width:700px){.display-settings{padding:16px}.settings-layout{grid-template-columns:1fr}.document-tree{display:flex;align-items:center;border-right:0;border-bottom:1px solid var(--el-border-color-lighter);padding:0 0 12px}.tree-item{margin:0;padding:10px 18px}.page-heading{align-items:flex-start}.settings-footer{align-items:flex-start;flex-direction:column}.section-heading{gap:8px}.setting-section{padding:12px}}
</style>
