<template>
  <div class="warehouse-reconciliation">
    <aside v-if="!collapsed" class="report-sidebar gva-table-box">
      <div class="sidebar-heading">
        <div>
          <strong>报表设置</strong>
          <p>管理个人/共享方案与报表列</p>
        </div>
        <el-button link @click="collapsed = true">收起</el-button>
      </div>
      <el-tabs v-model="sidebarTab" class="sidebar-tabs">
        <el-tab-pane label="方案" name="schemes">
          <div class="scheme-toolbar">
            <el-button size="small" type="primary" :disabled="!permissions.create" @click="openScheme('create')">新建方案</el-button>
            <el-button size="small" :disabled="!permissions.create" @click="openScheme('copy')">另存为</el-button>
            <el-button size="small" :loading="schemesLoading" @click="refreshSchemes">刷新</el-button>
          </div>
          <el-select v-model="schemeScope" clearable placeholder="个人和共享方案" class="sidebar-control" @change="refreshSchemes">
            <el-option label="个人方案" value="personal" />
            <el-option label="共享方案" value="shared" />
          </el-select>
          <div class="scheme-list">
            <button
              v-for="scheme in schemes"
              :key="idOf(scheme)"
              type="button"
              class="scheme-item"
              :class="{ selected: idOf(selectedScheme) === idOf(scheme) }"
              @click="loadScheme(scheme)"
            >
              <span>{{ scheme.name }}</span>
              <el-tag size="small" :type="scheme.scope === 'shared' ? 'success' : 'info'">{{ scheme.scope === 'shared' ? '共享' : '个人' }}</el-tag>
            </button>
            <el-empty v-if="!schemes.length && !schemesLoading" description="暂无可用方案" :image-size="60" />
          </div>
          <el-pagination
            v-if="schemeTotal > 20"
            small
            layout="prev, pager, next"
            :total="schemeTotal"
            :page-size="20"
            :current-page="schemePage"
            @current-change="changeSchemePage"
          />
          <div class="scheme-selection">
            <span class="field-label">当前方案</span>
            <strong>{{ selectedScheme?.name || '默认列（未命名方案）' }}</strong>
          </div>
          <div v-if="selectedScheme" class="scheme-toolbar scheme-actions">
            <el-button size="small" :disabled="!canEditScheme" @click="openScheme('edit')">编辑</el-button>
            <el-button size="small" type="danger" plain :disabled="!canDeleteScheme" @click="removeScheme">删除</el-button>
            <el-button v-if="selectedScheme.scope === 'personal'" size="small" :disabled="!permissions.shared || !canEditScheme" @click="publishScheme">发布共享</el-button>
          </div>
        </el-tab-pane>
        <el-tab-pane label="显示列" name="columns">
          <div class="column-tools">
            <span>已选 {{ columns.length }} 列</span>
            <el-button link @click="useDefaultColumns">恢复默认</el-button>
          </div>
          <el-input v-model="columnKeyword" clearable placeholder="搜索字段名称" class="sidebar-control" />
          <p class="report-hint">勾选控制结果、统计、导出和打印；顺序可在下方调整。修改后点击查询生效。</p>
          <el-checkbox-group :model-value="columns" @update:model-value="selectColumns">
            <div v-for="column in filteredAvailable" :key="column.id" class="column-option">
              <el-checkbox :label="column.id">{{ column.label }}</el-checkbox>
              <el-tag size="small" effect="plain">{{ column.kind === 'dimension' ? '维度' : '指标' }}</el-tag>
            </div>
          </el-checkbox-group>
          <el-divider />
          <div class="column-tools">
            <strong>显示顺序</strong>
            <span class="report-hint">共 {{ columns.length }} 列</span>
          </div>
          <ol class="ordered-columns">
            <li v-for="(id, index) in columns" :key="id">
              <span>{{ columnLabel(id) }}</span>
              <div>
                <el-button size="small" text :disabled="index === 0" :aria-label="`${columnLabel(id)}上移`" @click="moveColumn(index, -1)">上移</el-button>
                <el-button size="small" text :disabled="index === columns.length - 1" :aria-label="`${columnLabel(id)}下移`" @click="moveColumn(index, 1)">下移</el-button>
              </div>
            </li>
          </ol>
        </el-tab-pane>
      </el-tabs>
    </aside>
    <main class="report-main">
      <section class="gva-search-box report-search">
        <div class="report-heading">
          <div class="report-title">
            <el-button v-if="collapsed" @click="collapsed = false">展开方案与列</el-button>
            <div>
              <h2>对账表</h2>
              <p>仅统计已审核且未删除的送货单历史快照</p>
            </div>
          </div>
          <div class="heading-actions">
            <el-tag v-if="selectedScheme" effect="plain">{{ selectedScheme.name }}</el-tag>
            <el-button :disabled="!lastQuery || loading" :loading="exporting" @click="downloadExcel">导出全部</el-button>
            <el-button :disabled="!lastQuery || loading" :loading="printing" @click="printAll">打印全部</el-button>
          </div>
        </div>
        <el-form label-position="top" @submit.prevent="search">
          <div class="report-filters">
            <el-form-item label="送货日期">
              <el-date-picker v-model="filters.deliveredRange" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期（含）" />
            </el-form-item>
            <el-form-item label="审核日期">
              <el-date-picker v-model="filters.reviewedRange" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期（含）" />
            </el-form-item>
            <el-form-item label="客户快照">
              <el-select v-model="filters.brandId" clearable filterable placeholder="全部历史客户">
                <el-option v-for="(item, index) in options.brands" :key="index" :value="item.id" :label="`${item.code} · ${item.name}`" />
              </el-select>
            </el-form-item>
            <el-form-item label="类别快照">
              <el-select v-model="filters.categoryId" clearable filterable placeholder="全部历史类别（精确匹配）">
                <el-option v-for="(item, index) in options.categories" :key="index" :value="item.id" :label="`${item.code} · ${item.name}`" />
              </el-select>
            </el-form-item>
            <el-form-item label="部门">
              <el-input v-model="filters.department" clearable placeholder="精确匹配部门" @keyup.enter="search" />
            </el-form-item>
            <el-form-item label="送货单号">
              <el-input v-model="filters.number" clearable placeholder="输入单号关键字" @keyup.enter="search" />
            </el-form-item>
            <el-form-item label="物料关键字">
              <el-input v-model="filters.materialKeyword" clearable placeholder="快照编号或名称" @keyup.enter="search" />
            </el-form-item>
          </div>
          <div class="filter-actions">
            <span class="filter-note">日期范围按北京时间；结束日期包含当天。</span>
            <div>
              <el-button @click="reset">重置</el-button>
              <el-button type="primary" :loading="loading" :disabled="!available.length || !columns.length" @click="search">查询</el-button>
            </div>
          </div>
        </el-form>
      </section>
      <el-alert v-if="queryError" :title="queryError" type="error" show-icon :closable="false" class="report-alert" />
      <div class="gva-table-box report-results">
        <el-alert v-if="pendingChanges" title="筛选条件或显示列已修改，表格仍显示上次查询结果；导出和打印也使用上次查询条件。点击“查询”应用修改。" type="warning" show-icon :closable="false" class="result-alert" />
        <div class="results-heading">
          <div>
            <h3>{{ selectedScheme?.name ? `送货对账表 · ${selectedScheme.name}` : '送货对账表' }}</h3>
            <p v-if="lastQuery">{{ resultFilterDescription }}</p>
            <p v-else>设置筛选条件和显示列后查询</p>
          </div>
          <div class="results-meta">
            <strong>{{ projection.total.toLocaleString() }}</strong>
            <span>条记录</span>
          </div>
        </div>
        <el-table
          v-loading="loading"
          :data="projection.list"
          border
          stripe
          size="small"
          :max-height="'calc(100vh - 470px)'"
          :show-summary="Boolean(lastQuery)"
          :summary-method="tableSummary"
          empty-text="无符合条件的已审核数据"
          class="report-table"
        >
          <el-table-column type="index" label="#" width="58" fixed="left" align="center" />
          <el-table-column v-for="column in resultColumns" :key="column.id" :prop="column.id" :label="column.label" :min-width="columnWidth(column)" show-overflow-tooltip :align="column.kind === 'metric' ? 'right' : 'left'">
            <template #default="{ row }">
              <el-tooltip v-if="row.values[column.id] == null && row.unavailable?.[column.id]" :content="row.unavailable[column.id]" placement="top"><span class="unavailable">—</span></el-tooltip>
              <span v-else>{{ row.values[column.id] ?? '—' }}</span>
              <small v-if="column.id === 'baseQuantity' && row.baseUnits?.length" class="base-unit">{{ row.baseUnits.map((item) => `${item.unitName} #${item.unitId}`).join('、') }}</small>
            </template>
          </el-table-column>
          <template #empty><el-empty :description="queryError ? '查询失败，请检查条件后重试' : '无符合条件的已审核数据'" /></template>
        </el-table>
        <div class="results-footer">
          <span v-if="lastQuery">汇总行统计全部查询结果，不受分页影响；数量与基本数量按服务端精度累计</span>
          <el-pagination
            v-model:current-page="page"
            v-model:page-size="pageSize"
            :page-sizes="[20, 50, 100]"
            :total="projection.total"
            layout="total, sizes, prev, pager, next, jumper"
            @current-change="loadPage"
            @size-change="changePageSize"
          />
        </div>
        
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

import ReportPrintView from '../components/report-print-view.vue'

defineOptions({ name: 'WarehouseReconciliation' })
const defaults = ['deliveryDay', 'noteNumber', 'brand', 'department', 'materialCode', 'materialName', 'specification', 'unit', 'quantity', 'unitPrice', 'amount']
const user = useUserStore()
const available = ref([])
const columns = ref([...defaults])
const collapsed = ref(false)
const sidebarTab = ref('schemes')
const columnKeyword = ref('')
const filteredAvailable = computed(() => {
  const keyword = columnKeyword.value.trim().toLocaleLowerCase()
  return available.value.filter((column) => !keyword || column.label.toLocaleLowerCase().includes(keyword))
})
const options = ref({ brands: [], categories: [] })
const blankFilters = () => ({ deliveredRange: null, reviewedRange: null, brandId: undefined, categoryId: undefined, department: '', number: '', materialKeyword: '' })
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
function columnWidth(column) {
  if (column.kind === 'metric') return column.id === 'baseQuantity' ? 150 : 125
  if (['deliveryDay', 'deliveryMonth', 'createdAt', 'reviewedAt'].includes(column.id)) return 155
  if (column.id === 'noteNumber') return 180
  if (column.id === 'materialName') return 210
  return 140
}
function tableSummary({ columns }) {
  return columns.map((column, index) => {
    if (index === 0) return '合计'
    const id = column.property
    if (id === 'amount') return projection.value.summary.amount ?? '—'
    if (id === 'quantity') return projection.value.summary.quantity ?? '—'
    if (id === 'baseQuantity') return projection.value.summary.baseQuantity ?? '—'
    if (id === 'unitPrice') return '—'
    return ''
  })
}

function dateRange(range) {
  if (!range?.length) return [undefined, undefined]
  const [year, month, day] = range[1].split('-').map(Number)
  const nextDay = new Date(Date.UTC(year, month - 1, day + 1)).toISOString().slice(0, 10)
  return [`${range[0]}T00:00:00+08:00`, `${nextDay}T00:00:00+08:00`]
}
function currentFilters() {
  const [deliveredFrom, deliveredTo] = dateRange(filters.deliveredRange)
  const [reviewedFrom, reviewedTo] = dateRange(filters.reviewedRange)
  return Object.fromEntries(Object.entries({ deliveredFrom, deliveredTo, reviewedFrom, reviewedTo, brandId: filters.brandId || undefined, categoryId: filters.categoryId || undefined, department: filters.department.trim() || undefined, number: filters.number.trim() || undefined, materialKeyword: filters.materialKeyword.trim() || undefined }).filter(([, value]) => value !== undefined))
}
const pendingChanges = computed(() => lastQuery.value && (JSON.stringify(columns.value) !== JSON.stringify(lastQuery.value.columns) || JSON.stringify(currentFilters()) !== JSON.stringify(lastQuery.value.filters)))
const resultFilterDescription = computed(() => {
  const f = lastQuery.value?.filters || {}
  const parts = ['当前结果：已审核快照']
  if (f.deliveredFrom) parts.push(`送货 [${f.deliveredFrom}, ${f.deliveredTo})`)
  if (f.reviewedFrom) parts.push(`审核 [${f.reviewedFrom}, ${f.reviewedTo})`)
  if (f.brandId) parts.push(`客户 #${f.brandId}`)
  if (f.categoryId) parts.push(`类别 #${f.categoryId}`)
  if (f.department) parts.push(`部门 ${f.department}`)
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
const schemesLoading = ref(false)
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
  schemesLoading.value = true
  try {
    const data = await listReportSchemes({ page: schemePage.value, pageSize: 20, scope: schemeScope.value || undefined })
    schemes.value = data.list
    schemeTotal.value = data.total
    permissions.value = data.permissions
  } finally {
    schemesLoading.value = false
  }
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
.warehouse-reconciliation { display: flex; align-items: stretch; gap: 16px; }
.report-sidebar { width: 330px; flex: 0 0 330px; padding: 16px; height: calc(100vh - 120px); min-height: 560px; overflow: auto; box-sizing: border-box; }
.report-main { flex: 1; min-width: 0; }
.sidebar-heading, .report-heading, .results-heading, .filter-actions, .results-footer, .column-tools { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.sidebar-heading { margin-bottom: 8px; }
.sidebar-heading strong { font-size: 16px; }
.sidebar-heading p, .report-title p, .results-heading p { margin: 4px 0 0; color: var(--el-text-color-secondary); font-size: 12px; }
.sidebar-tabs :deep(.el-tabs__content) { overflow: visible; }
.sidebar-control { width: 100%; }
.scheme-toolbar { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; margin-bottom: 10px; }
.scheme-toolbar :deep(.el-button + .el-button) { margin-left: 0; }
.scheme-list { margin: 10px 0; max-height: 280px; overflow: auto; }
.scheme-item { display: flex; align-items: center; justify-content: space-between; gap: 8px; width: 100%; padding: 10px 8px; border: 0; border-bottom: 1px solid var(--el-border-color-lighter); background: transparent; color: var(--el-text-color-primary); cursor: pointer; text-align: left; overflow-wrap: anywhere; }
.scheme-item:hover { background: var(--el-fill-color-light); }
.scheme-item.selected { background: var(--el-color-primary-light-9); color: var(--el-color-primary); }
.scheme-selection { display: grid; gap: 4px; padding: 12px; border-radius: 4px; background: var(--el-fill-color-light); overflow-wrap: anywhere; }
.field-label { color: var(--el-text-color-secondary); font-size: 12px; }
.scheme-actions { margin-top: 10px; }
.column-tools { margin-bottom: 10px; }
.column-option { display: flex; align-items: center; justify-content: space-between; gap: 8px; min-height: 30px; }
.column-option :deep(.el-checkbox) { min-width: 0; }
.column-option :deep(.el-checkbox__label) { overflow-wrap: anywhere; white-space: normal; }
.report-hint { margin: 8px 0; color: var(--el-text-color-secondary); font-size: 12px; line-height: 1.5; overflow-wrap: anywhere; }
.ordered-columns { margin: 0; padding-left: 22px; }
.ordered-columns li { display: flex; align-items: center; justify-content: space-between; gap: 4px; padding: 3px 0; }
.ordered-columns li > span { min-width: 0; overflow-wrap: anywhere; }
.ordered-columns :deep(.el-button + .el-button) { margin-left: 0; }
.report-search { margin-bottom: 16px; }
.report-heading { margin-bottom: 16px; }
.report-title, .heading-actions { display: flex; align-items: center; gap: 12px; }
.report-title h2, .results-heading h3 { margin: 0; font-size: 18px; }
.heading-actions { flex-wrap: wrap; justify-content: flex-end; }
.report-filters { display: grid; grid-template-columns: repeat(3, minmax(190px, 1fr)); column-gap: 16px; }
.report-filters :deep(.el-form-item) { margin-bottom: 12px; min-width: 0; }
.report-filters :deep(.el-date-editor), .report-filters :deep(.el-select), .report-filters :deep(.el-input) { width: 100%; }
.filter-actions { padding-top: 12px; border-top: 1px solid var(--el-border-color-lighter); }
.filter-actions > div { display: flex; gap: 8px; }
.filter-note { color: var(--el-text-color-secondary); font-size: 12px; }
.report-alert { margin-bottom: 12px; }
.report-results { min-width: 0; }
.result-alert { margin-bottom: 14px; }
.results-heading { margin-bottom: 14px; }
.results-meta { display: grid; min-width: 100px; justify-items: end; }
.results-meta strong { font-size: 20px; line-height: 1.2; color: var(--el-color-primary); }
.results-meta span { color: var(--el-text-color-secondary); font-size: 12px; }
.report-table { width: 100%; }
.report-table :deep(.cell) { overflow-wrap: anywhere; }
.report-table :deep(.el-table__header-wrapper th) { background: var(--el-fill-color-light); color: var(--el-text-color-primary); font-weight: 600; }
.report-table :deep(.el-table__footer-wrapper td) { background: var(--el-fill-color-light); color: var(--el-text-color-primary); font-weight: 600; }
.report-table :deep(.el-table__footer-wrapper .cell) { white-space: normal; }
.results-footer { flex-wrap: wrap; margin-top: 14px; }
.results-footer > span { color: var(--el-text-color-secondary); font-size: 12px; }
.results-footer :deep(.el-pagination) { margin-left: auto; }
.unavailable { cursor: help; border-bottom: 1px dashed var(--el-text-color-secondary); }
.base-unit { display: block; color: var(--el-text-color-secondary); }
@media (max-width: 1280px) {
  .report-filters { grid-template-columns: repeat(2, minmax(190px, 1fr)); }
}
@media (max-width: 1000px) {
  .warehouse-reconciliation { flex-direction: column; }
  .report-sidebar { width: 100%; flex-basis: auto; height: auto; min-height: 0; max-height: 560px; }
  .report-main { width: 100%; }
}
@media (max-width: 680px) {
  .report-filters { grid-template-columns: minmax(0, 1fr); }
  .report-heading, .results-heading, .filter-actions { align-items: flex-start; flex-direction: column; }
  .heading-actions { justify-content: flex-start; }
  .results-meta { justify-items: start; }
  .results-footer :deep(.el-pagination) { margin-left: 0; }
  .filter-note { line-height: 1.5; }
}
</style>
