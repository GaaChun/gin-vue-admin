<template>
  <div class="warehouse-reconciliation">
    <aside v-show="!collapsed" class="report-sidebar gva-table-box">
      <h3>报表方案</h3>
      <div class="scheme-toolbar">
        <el-button size="small" :disabled="!permissions.create" @click="openScheme('create')">新建</el-button>
        <el-button size="small" :disabled="!permissions.create" @click="openScheme('copy')">另存为</el-button>
        <el-button size="small" @click="refreshSchemes">刷新</el-button>
      </div>
      <el-select v-model="schemeScope" clearable placeholder="个人和共享" @change="refreshSchemes">
        <el-option label="个人方案" value="personal" /><el-option label="共享方案" value="shared" />
      </el-select>
      <div class="scheme-list">
        <button v-for="scheme in schemes" :key="idOf(scheme)" type="button" class="scheme-item" :class="{ selected: idOf(selectedScheme) === idOf(scheme) }" @click="loadScheme(scheme)">
          {{ scheme.name }} <small>{{ scheme.scope === 'shared' ? '共享' : '个人' }}</small>
        </button>
        <el-empty v-if="!schemes.length" description="暂无可用方案" :image-size="50" />
      </div>
      <el-pagination small layout="prev, pager, next" :total="schemeTotal" :page-size="20" :current-page="schemePage" @current-change="changeSchemePage" />
      <div v-if="selectedScheme" class="scheme-toolbar">
        <el-button size="small" :disabled="!canEditScheme" @click="openScheme('edit')">编辑</el-button>
        <el-button size="small" type="danger" :disabled="!canDeleteScheme" @click="removeScheme">删除</el-button>
        <el-button v-if="selectedScheme.scope === 'personal'" size="small" :disabled="!permissions.shared || !canEditScheme" @click="publishScheme">发布共享</el-button>
      </div>
      <h3>有序列配置</h3>
      <p class="report-hint">加载方案仅改变列，不改变筛选。修改列后需重新查询。</p>
      <el-checkbox-group :model-value="columns" @update:model-value="selectColumns">
        <div v-for="column in available" :key="column.id"><el-checkbox :label="column.id">{{ column.label }}（{{ column.kind === 'dimension' ? '分组' : '数值' }}）</el-checkbox></div>
      </el-checkbox-group>
      <ol class="ordered-columns">
        <li v-for="(id, index) in columns" :key="id">
          <span>{{ columnLabel(id) }}</span>
          <el-button size="small" text :disabled="index === 0" :aria-label="`${columnLabel(id)}上移`" @click="moveColumn(index, -1)">↑</el-button>
          <el-button size="small" text :disabled="index === columns.length - 1" :aria-label="`${columnLabel(id)}下移`" @click="moveColumn(index, 1)">↓</el-button>
        </li>
      </ol>
      <el-button size="small" @click="useDefaultColumns">恢复默认列</el-button>
    </aside>
    <main class="report-main">
      <div class="gva-search-box">
        <div class="report-toolbar">
          <el-button @click="collapsed = !collapsed">{{ collapsed ? '展开方案与列' : '收起方案与列' }}</el-button>
          <strong>送货对账表</strong>
        </div>
        <el-form label-width="100px" @submit.prevent="search">
          <div class="report-filters">
            <el-form-item label="送货日期"><el-date-picker v-model="filters.deliveredRange" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期（含）" /></el-form-item>
            <el-form-item label="审核日期"><el-date-picker v-model="filters.reviewedRange" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期（含）" /></el-form-item>
            <el-form-item label="品牌快照"><el-select v-model="filters.brandId" clearable filterable placeholder="全部历史品牌"><el-option v-for="(item, index) in options.brands" :key="index" :value="item.id" :label="`${item.code} ${item.name} #${item.id}`" /></el-select></el-form-item>
            <el-form-item label="类别快照"><el-select v-model="filters.categoryId" clearable filterable placeholder="精确历史类别"><el-option v-for="(item, index) in options.categories" :key="index" :value="item.id" :label="`${item.code} ${item.name} #${item.id}`" /></el-select></el-form-item>
            <el-form-item label="送货单号"><el-input v-model="filters.number" clearable placeholder="单号包含" @keyup.enter="search" /></el-form-item>
            <el-form-item label="物料关键字"><el-input v-model="filters.materialKeyword" clearable placeholder="快照编号或名称包含" @keyup.enter="search" /></el-form-item>
          </div>
          <div class="report-toolbar">
            <el-button type="primary" :loading="loading" :disabled="!available.length || !columns.length" @click="search">查询</el-button>
            <el-button @click="reset">重置</el-button>
            <el-button :disabled="!lastQuery || loading" :loading="exporting" @click="downloadExcel">全量Excel</el-button>
            <el-button :disabled="!lastQuery || loading" :loading="printing" @click="printAll">打印全量</el-button>
          </div>
        </el-form>
        <p class="report-hint">仅已审核且未删除单据；日期按北京时间转换为 [开始日00:00,结束次日00:00)。类别按历史快照精确匹配。</p>
      </div>
      <div class="gva-table-box">
        <el-alert v-if="pendingChanges" title="列或筛选已修改，当前结果仍对应上次查询；导出和打印使用上次查询条件。请点击查询以应用修改。" type="warning" :closable="false" />
        <el-alert v-if="queryError" :title="queryError" type="error" :closable="false" />
        <p v-if="lastQuery" class="report-hint">{{ resultFilterDescription }}</p>
        <el-table v-loading="loading" :data="projection.list" border empty-text="无符合条件的已审核数据">
          <el-table-column v-for="column in resultColumns" :key="column.id" :label="column.label" min-width="140" :align="column.kind === 'metric' ? 'right' : 'left'">
            <template #default="{ row }">
              <el-tooltip v-if="row.values[column.id] == null && row.unavailable?.[column.id]" :content="row.unavailable[column.id]" placement="top"><span class="unavailable">—</span></el-tooltip>
              <span v-else>{{ row.values[column.id] ?? '—' }}</span>
              <small v-if="column.id === 'baseQuantity' && row.baseUnits?.length" class="base-unit">{{ row.baseUnits.map((item) => `${item.unitName} #${item.unitId}`).join('、') }}</small>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination class="report-pagination" v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[20, 50, 100]" :total="projection.total" layout="total, sizes, prev, pager, next, jumper" @current-change="loadPage" @size-change="changePageSize" />
        <ReportSummary :summary="projection.summary" :columns="projection.columns" />
        <p v-for="warning in projection.warnings" :key="warning" class="report-hint">{{ warning }}</p>
      </div>
    </main>
    <el-dialog v-model="schemeDialog" :title="schemeMode === 'edit' ? '编辑方案（保存当前有序列）' : '另存当前列为方案'" width="480px" destroy-on-close>
      <el-form label-width="80px" @submit.prevent="saveScheme">
        <el-form-item label="方案名称"><el-input v-model="schemeForm.name" maxlength="100" show-word-limit /></el-form-item>
        <el-form-item label="范围"><el-radio-group v-model="schemeForm.scope"><el-radio label="personal">个人</el-radio><el-radio label="shared" :disabled="!permissions.shared">共享</el-radio></el-radio-group></el-form-item>
        <el-form-item label="保存列">{{ columns.map(columnLabel).join('、') || '尚未选择列' }}</el-form-item>
      </el-form>
      <template #footer><el-button @click="schemeDialog = false">取消</el-button><el-button type="primary" :loading="savingScheme" @click="saveScheme">保存</el-button></template>
    </el-dialog>
    <ReportPrintView :data="printData" :registry="available" :generated-at="generatedAt" />
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '@/pinia/modules/user'
import { idOf, showError } from '../api/common'
import { loadReportColumns, loadReportOptions, queryReport, exportReport, printReport, listReportSchemes, getReportScheme, createReportScheme, updateReportScheme, deleteReportScheme, publishReportScheme } from '../api/reconciliation'
import ReportSummary from '../components/report-summary.vue'
import ReportPrintView from '../components/report-print-view.vue'

defineOptions({ name: 'WarehouseReconciliation' })
const defaults = ['deliveryDay', 'noteNumber', 'brand', 'materialCode', 'materialName', 'specification', 'unit', 'quantity', 'unitPrice', 'amount']
const user = useUserStore()
const available = ref([])
const columns = ref([...defaults])
const collapsed = ref(false)
const options = ref({ brands: [], categories: [] })
const blankFilters = () => ({ deliveredRange: null, reviewedRange: null, brandId: undefined, categoryId: undefined, number: '', materialKeyword: '' })
const filters = reactive(blankFilters())
const page = ref(1)
const pageSize = ref(20)
const projection = ref({ list: [], columns: [], total: 0, summary: {}, warnings: [] })
const lastQuery = ref(null)
const loading = ref(false)
const exporting = ref(false)
const printing = ref(false)
const queryError = ref('')
const printData = ref(null)
const generatedAt = ref('')
let requestSequence = 0
const columnLabel = (id) => available.value.find((column) => column.id === id)?.label || id
const resultColumns = computed(() => projection.value.columns.map((id) => available.value.find((column) => column.id === id)).filter(Boolean))

function dateRange(range) {
  if (!range?.length) return [undefined, undefined]
  const [year, month, day] = range[1].split('-').map(Number)
  const nextDay = new Date(Date.UTC(year, month - 1, day + 1)).toISOString().slice(0, 10)
  return [`${range[0]}T00:00:00+08:00`, `${nextDay}T00:00:00+08:00`]
}
function currentFilters() {
  const [deliveredFrom, deliveredTo] = dateRange(filters.deliveredRange)
  const [reviewedFrom, reviewedTo] = dateRange(filters.reviewedRange)
  return Object.fromEntries(Object.entries({ deliveredFrom, deliveredTo, reviewedFrom, reviewedTo, brandId: filters.brandId || undefined, categoryId: filters.categoryId || undefined, number: filters.number.trim() || undefined, materialKeyword: filters.materialKeyword.trim() || undefined }).filter(([, value]) => value !== undefined))
}
const pendingChanges = computed(() => lastQuery.value && (JSON.stringify(columns.value) !== JSON.stringify(lastQuery.value.columns) || JSON.stringify(currentFilters()) !== JSON.stringify(lastQuery.value.filters)))
const resultFilterDescription = computed(() => {
  const f = lastQuery.value?.filters || {}
  const parts = ['当前结果：已审核快照']
  if (f.deliveredFrom) parts.push(`送货 [${f.deliveredFrom}, ${f.deliveredTo})`)
  if (f.reviewedFrom) parts.push(`审核 [${f.reviewedFrom}, ${f.reviewedTo})`)
  if (f.brandId) parts.push(`品牌 #${f.brandId}`)
  if (f.categoryId) parts.push(`类别 #${f.categoryId}`)
  if (f.number) parts.push(`单号包含 ${f.number}`)
  if (f.materialKeyword) parts.push(`物料包含 ${f.materialKeyword}`)
  return parts.join('；')
})
function selectColumns(values) {
  columns.value = [...columns.value.filter((id) => values.includes(id)), ...values.filter((id) => !columns.value.includes(id))]
}
function moveColumn(index, offset) {
  const next = [...columns.value]
  ;[next[index], next[index + offset]] = [next[index + offset], next[index]]
  columns.value = next
}
function useDefaultColumns() { columns.value = defaults.filter((id) => available.value.some((column) => column.id === id)) }
async function executeQuery(request) {
  const sequence = ++requestSequence
  loading.value = true
  queryError.value = ''
  try {
    const result = await queryReport(request)
    if (sequence !== requestSequence) return
    projection.value = result
    lastQuery.value = { filters: { ...request.filters }, columns: [...request.columns] }
    page.value = result.page
    pageSize.value = result.pageSize
  } catch (error) {
    if (sequence === requestSequence) { queryError.value = error.message; showError(error) }
  } finally { if (sequence === requestSequence) loading.value = false }
}
async function search() {
  if (!columns.value.length) { ElMessage.warning('至少选择一列'); return }
  await executeQuery({ filters: currentFilters(), columns: [...columns.value], page: 1, pageSize: pageSize.value })
}
async function reset() { Object.assign(filters, blankFilters()); await search() }
async function loadPage() {
  if (!lastQuery.value) return
  await executeQuery({ filters: { ...lastQuery.value.filters }, columns: [...lastQuery.value.columns], page: page.value, pageSize: pageSize.value })
}
async function changePageSize() { page.value = 1; await loadPage() }
async function downloadExcel() {
  exporting.value = true
  try { await exportReport({ ...lastQuery.value, page: 1, pageSize: pageSize.value }) } catch (error) { showError(error) } finally { exporting.value = false }
}
function finishPrint() { document.body.classList.remove('warehouse-report-printing') }
async function printAll() {
  printing.value = true
  try {
    printData.value = await printReport({ ...lastQuery.value, page: 1, pageSize: pageSize.value })
    generatedAt.value = new Date().toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })
    await nextTick()
    document.body.classList.add('warehouse-report-printing')
    window.print()
  } catch (error) { showError(error) } finally { finishPrint(); printing.value = false }
}

const schemes = ref([])
const schemePage = ref(1)
const schemeTotal = ref(0)
const schemeScope = ref('')
const selectedScheme = ref(null)
const permissions = ref({ create: false, update: false, delete: false, shared: false })
const owned = computed(() => selectedScheme.value?.ownerId === idOf(user.userInfo))
const canEditScheme = computed(() => selectedScheme.value && permissions.value.update && (selectedScheme.value.scope === 'shared' ? permissions.value.shared : owned.value))
const canDeleteScheme = computed(() => selectedScheme.value && permissions.value.delete && (selectedScheme.value.scope === 'shared' ? permissions.value.shared : owned.value))
const schemeDialog = ref(false)
const schemeMode = ref('create')
const schemeForm = reactive({ name: '', scope: 'personal' })
const savingScheme = ref(false)
async function fetchSchemes() {
  const data = await listReportSchemes({ page: schemePage.value, pageSize: 20, scope: schemeScope.value || undefined })
  schemes.value = data.list
  schemeTotal.value = data.total
  permissions.value = data.permissions
}
async function refreshSchemes() { schemePage.value = 1; try { await fetchSchemes() } catch (error) { showError(error) } }
async function changeSchemePage(value) { schemePage.value = value; try { await fetchSchemes() } catch (error) { showError(error) } }
async function loadScheme(scheme) {
  try {
    const data = await getReportScheme(idOf(scheme))
    const ids = JSON.parse(data.columnsJson)
    if (!Array.isArray(ids) || !ids.length || ids.some((id) => !available.value.some((column) => column.id === id))) throw new Error('方案列配置无效')
    selectedScheme.value = data
    columns.value = [...ids]
  } catch (error) { showError(error) }
}
function openScheme(mode) {
  schemeMode.value = mode
  schemeForm.name = mode === 'edit' ? selectedScheme.value.name : mode === 'copy' && selectedScheme.value ? `${selectedScheme.value.name}副本` : ''
  schemeForm.scope = mode === 'edit' ? selectedScheme.value.scope : 'personal'
  schemeDialog.value = true
}
async function saveScheme() {
  if (!schemeForm.name.trim()) { ElMessage.warning('请输入方案名称'); return }
  if (!columns.value.length) { ElMessage.warning('至少选择一列'); return }
  savingScheme.value = true
  try {
    const data = { name: schemeForm.name.trim(), scope: schemeForm.scope, columns: [...columns.value] }
    selectedScheme.value = schemeMode.value === 'edit' ? await updateReportScheme({ ...data, id: idOf(selectedScheme.value), version: selectedScheme.value.version }) : await createReportScheme(data)
    schemeDialog.value = false
    await refreshSchemes()
    ElMessage.success('方案已保存，筛选条件未改变')
  } catch (error) { showError(error) } finally { savingScheme.value = false }
}
async function removeScheme() {
  try {
    await ElMessageBox.confirm(`删除方案“${selectedScheme.value.name}”？当前本地列仍可继续使用。`, '删除方案', { type: 'warning' })
    await deleteReportScheme({ id: idOf(selectedScheme.value), version: selectedScheme.value.version })
    selectedScheme.value = null
    await refreshSchemes()
    ElMessage.success('方案已删除')
  } catch (error) { showError(error) }
}
async function publishScheme() {
  try {
    await ElMessageBox.confirm('将当前方案发布为共享，并保存当前有序列？', '发布共享', { type: 'warning' })
    selectedScheme.value = await publishReportScheme({ id: idOf(selectedScheme.value), version: selectedScheme.value.version, name: selectedScheme.value.name, scope: 'shared', columns: [...columns.value] })
    await refreshSchemes()
    ElMessage.success('已发布共享方案')
  } catch (error) { showError(error) }
}
onMounted(async () => {
  window.addEventListener('afterprint', finishPrint)
  await Promise.all([
    loadReportColumns().then((data) => { available.value = data; useDefaultColumns() }).catch(showError),
    loadReportOptions().then((data) => { options.value = data }).catch(showError),
    refreshSchemes()
  ])
  if (available.value.length) await search()
})
onBeforeUnmount(() => { requestSequence++; finishPrint(); window.removeEventListener('afterprint', finishPrint) })
</script>

<style scoped>
.warehouse-reconciliation { display: flex; align-items: flex-start; gap: 16px; }
.report-sidebar { width: 290px; flex: 0 0 290px; padding: 16px; max-height: calc(100vh - 120px); overflow: auto; box-sizing: border-box; }
.report-main { flex: 1; min-width: 0; }
.report-sidebar h3 { margin: 8px 0 12px; font-size: 16px; }
.report-toolbar, .scheme-toolbar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-bottom: 12px; }
.scheme-toolbar :deep(.el-button + .el-button) { margin-left: 0; }
.scheme-list { margin: 10px 0; max-height: 220px; overflow: auto; }
.scheme-item { display: block; width: 100%; padding: 8px; border: 0; background: transparent; color: var(--el-text-color-primary); cursor: pointer; text-align: left; overflow-wrap: anywhere; }
.scheme-item.selected { background: var(--el-color-primary-light-9); color: var(--el-color-primary); }
.scheme-item small { color: var(--el-text-color-secondary); }
.report-hint { font-size: 12px; color: var(--el-text-color-secondary); overflow-wrap: anywhere; }
.report-filters { display: grid; grid-template-columns: repeat(auto-fit, minmax(330px, 1fr)); gap: 0 12px; }
.report-filters :deep(.el-date-editor), .report-filters :deep(.el-select) { width: 100%; }
.ordered-columns { padding-left: 24px; }
.ordered-columns li { padding: 2px 0; }
.ordered-columns span { display: inline-block; width: 115px; }
.ordered-columns :deep(.el-button + .el-button) { margin-left: 0; }
.report-pagination { margin-top: 16px; justify-content: flex-end; flex-wrap: wrap; }
.unavailable { cursor: help; border-bottom: 1px dashed var(--el-text-color-secondary); }
.base-unit { display: block; color: var(--el-text-color-secondary); }
@media (max-width: 900px) {
  .warehouse-reconciliation { flex-direction: column; }
  .report-sidebar { width: 100%; flex-basis: auto; max-height: 400px; }
  .report-main { width: 100%; }
}
</style>
