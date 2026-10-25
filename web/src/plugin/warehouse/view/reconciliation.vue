<template>
  <div ref="reportRoot" class="warehouse-reconciliation">
    <aside v-if="!collapsed" class="report-sidebar gva-table-box">
      <div class="sidebar-heading">
        <strong>报表方案操作面板</strong>
        <el-button text circle aria-label="收起报表方案操作面板" title="收起报表方案操作面板" :icon="Fold" @click="collapsed = true" />
      </div>
      <div class="column-tools">
        <strong>显示列</strong>
        <el-button link @click="useDefaultColumns">恢复默认</el-button>
      </div>
      <el-input v-model="columnKeyword" clearable placeholder="搜索字段名称" class="sidebar-control" />
      <p class="report-hint">勾选后查询生效；拖动结果表头可调整显示、方案和导出顺序。</p>
      <el-checkbox-group :model-value="columns" class="column-picker" @update:model-value="selectColumns">
        <el-checkbox v-for="column in filteredAvailable" :key="column.id" :label="column.id" :title="column.label">{{ column.label }}</el-checkbox>
      </el-checkbox-group>
      <div class="column-tools"><span>已选 {{ columns.length }} 列</span></div>
      <div class="scheme-editor">
        <div class="column-tools"><strong>方案操作</strong><el-button link :loading="schemesLoading" @click="refreshSchemes">刷新</el-button></div>
        <el-select v-model="schemeScope" clearable placeholder="个人和共享方案" class="sidebar-control" @change="refreshSchemes">
          <el-option label="个人方案" value="personal" />
          <el-option label="共享方案" value="shared" />
        </el-select>
        <p class="scheme-selection"><span class="field-label">当前方案</span><strong>{{ selectedScheme?.name || '默认列（未命名方案）' }}</strong></p>
        <div class="scheme-toolbar">
          <el-button size="small" type="primary" :disabled="!permissions.create" @click="openScheme('create')">新建方案</el-button>
          <el-button size="small" :disabled="!permissions.create" @click="openScheme('copy')">另存方案</el-button>
          <el-button size="small" :disabled="!canEditScheme" @click="openScheme('edit')">保存编辑</el-button>
          <el-button size="small" type="danger" plain :disabled="!canDeleteScheme" @click="removeScheme">删除方案</el-button>
          <el-button v-if="selectedScheme?.scope === 'personal'" size="small" :disabled="!permissions.shared || !canEditScheme" @click="publishScheme">发布共享</el-button>
        </div>
        <p class="report-hint">方案只保存显示列及顺序，不改变筛选条件。</p>
      </div>
    </aside>
    <main class="report-main">
      <section class="gva-search-box report-search">
        <div class="report-heading">
          <div class="report-title">
            <el-button v-if="collapsed" text circle aria-label="展开报表方案操作面板" title="展开报表方案操作面板" :icon="Expand" @click="collapsed = false" />
            <div>
              <h2>对账表</h2>
              <p>按所选单据状态统计未删除的送货单历史快照</p>
            </div>
          </div>
          <div class="heading-actions">
            <el-tag v-if="selectedScheme" effect="plain">{{ selectedScheme.name }}</el-tag>
          </div>
        </div>
        <el-form label-position="top" @submit.prevent="search">
          <div class="report-filters">
            <el-form-item label="选择方案">
              <el-select :model-value="selectedScheme ? idOf(selectedScheme) : ''" clearable filterable :loading="schemesLoading" placeholder="默认列（未命名方案）" @change="selectScheme">
                <el-option v-for="scheme in schemeOptions" :key="idOf(scheme)" :label="scheme.name" :value="idOf(scheme)">
                  <span>{{ scheme.name }}</span><small class="scheme-scope">{{ scheme.scope === 'shared' ? '共享' : '个人' }}</small>
                </el-option>
                <template #footer>
                  <el-pagination v-if="schemeTotal > 20" small layout="prev, pager, next" :total="schemeTotal" :page-size="20" :current-page="schemePage" @current-change="changeSchemePage" />
                  <span v-else class="report-hint">方案只应用列设置</span>
                </template>
              </el-select>
            </el-form-item>
            <el-form-item label="送货日期">
              <el-date-picker v-model="filters.deliveredRange" type="daterange" :shortcuts="dateShortcuts" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期（含）" />
            </el-form-item>
            <el-form-item label="审核日期">
              <el-date-picker v-model="filters.reviewedRange" type="daterange" :shortcuts="dateShortcuts" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期（含）" />
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
              <el-select v-model="filters.department" clearable filterable placeholder="全部历史部门">
                <el-option v-for="department in options.departments" :key="department" :value="department" :label="department" />
              </el-select>
            </el-form-item>
            <el-form-item label="送货单号">
              <el-input v-model="filters.number" clearable placeholder="输入单号关键字" @keyup.enter="search" />
            </el-form-item>
            <el-form-item label="物料关键字">
              <el-input v-model="filters.materialKeyword" clearable placeholder="快照编号或名称" @keyup.enter="search" />
            </el-form-item>
            <el-form-item label="单据状态">
              <el-select v-model="filters.statuses" multiple collapse-tags collapse-tags-tooltip placeholder="已审核、已复审">
                <el-option v-for="status in statusOptions" :key="status.value" :value="status.value" :label="status.label" />
              </el-select>
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
      <Teleport to="body" :disabled="!fullscreen">
      <div class="gva-table-box report-results" :class="{ 'report-results-fullscreen': fullscreen }" :style="fullscreen ? fullscreenStyle : undefined">
        <el-alert v-if="pendingChanges" title="筛选条件或显示列已修改，表格仍显示上次查询结果；导出和打印也使用上次查询条件。点击“查询”应用修改。" type="warning" show-icon :closable="false" class="result-alert" />
        <div class="results-heading">
          <div>
            <h3>{{ selectedScheme?.name ? `送货对账表 · ${selectedScheme.name}` : '送货对账表' }}</h3>
            <p v-if="lastQuery">{{ resultFilterDescription }}</p>
            <p v-else>设置筛选条件和显示列后查询</p>
          </div>
          <div class="result-actions">
            <div class="results-meta"><strong>{{ projection.total.toLocaleString() }}</strong><span>条记录</span></div>
            <el-button :disabled="!lastQuery || loading" :loading="exporting" @click="downloadExcel">导出全部</el-button>
            <el-button :disabled="!lastQuery || loading" :loading="printing" @click="printAll">打印全部</el-button>
            <el-button text circle :icon="fullscreen ? Close : FullScreen" :aria-label="fullscreen ? '退出结果全屏' : '结果全屏'" :title="fullscreen ? '退出结果全屏（Esc）' : '结果全屏'" @click="toggleFullscreen" />
          </div>
        </div>
        <el-table
          ref="resultTable"
          v-loading="loading"
          :data="projection.list"
          border
          stripe
          size="small"
          :height="fullscreen ? '100%' : undefined"
          :max-height="fullscreen ? undefined : 'calc(100vh - 430px)'"
          :show-summary="Boolean(lastQuery)"
          :summary-method="tableSummary"
          :header-cell-class-name="headerCellClass"
          @sort-change="changeReportSort"
          @header-dragend="onHeaderDragEnd"
          empty-text="无符合所选条件的数据"
          class="report-table"
        >
          <el-table-column type="index" column-key="__index" label="序号" :width="columnWidths.__index || 65" fixed="left" align="center" :index="pageIndex" />
          <el-table-column v-for="column in resultColumns" :key="column.id" :prop="column.id" :column-key="column.id" :label="column.label" :width="columnWidths[column.id] || columnWidth(column)" min-width="48" show-overflow-tooltip sortable="custom" :align="column.kind === 'metric' ? 'center' : 'left'">
            <template #header>
              <span class="draggable-report-header" :class="{ 'is-dragging': draggedColumn === column.id }" tabindex="0" :data-column-id="column.id" :aria-label="`${column.label}，按住拖动插入调整列顺序，或按Alt加左右方向键`" :aria-grabbed="draggedColumn === column.id" @pointerdown="startHeaderPointer($event, column.id)" @keydown.alt.left.prevent.stop="moveResultColumn(column.id, -1)" @keydown.alt.right.prevent.stop="moveResultColumn(column.id, 1)">{{ column.label }}</span>
            </template>
            <template #default="{ row }">
              <el-tooltip v-if="row.values[column.id] == null && row.unavailable?.[column.id]" :content="row.unavailable[column.id]" placement="top"><span class="unavailable">—</span></el-tooltip>
              <el-button v-else-if="column.id === 'noteNumber' && row.noteId" link type="primary" @click="openDelivery(row.noteId)">{{ row.values[column.id] ?? '—' }}</el-button>
              <span v-else>{{ row.values[column.id] ?? '—' }}</span>
              <small v-if="column.id === 'baseQuantity' && row.baseUnits?.length" class="base-unit">{{ row.baseUnits.map((item) => `${item.unitName} #${item.unitId}`).join('、') }}</small>
            </template>
          </el-table-column>
          <template #empty><el-empty :description="queryError ? '查询失败，请检查条件后重试' : '无符合所选条件的数据'" /></template>
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
      </Teleport>
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
    <Teleport to="body">
      <div v-if="headerGhost" class="report-column-drag-ghost" :style="headerGhost.style" aria-hidden="true">{{ headerGhost.label }}</div>
      <div v-if="headerInsertion" class="report-column-insertion" :style="headerInsertion" aria-hidden="true" />
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onDeactivated, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Close, Expand, Fold, FullScreen } from '@element-plus/icons-vue'
import { useUserStore } from '@/pinia/modules/user'
import { idOf, pageSequenceNumber, showError } from '../api/common'
import { loadReportColumns, loadReportOptions, queryReport, exportReport, printReport, listReportSchemes, getReportScheme, createReportScheme, updateReportScheme, deleteReportScheme, publishReportScheme } from '../api/reconciliation'

import ReportPrintView from '../components/report-print-view.vue'
import { useTableWidths } from '../use-table-widths'

defineOptions({ name: 'WarehouseReconciliation' })
const defaults = ['deliveryDay', 'noteNumber', 'brand', 'department', 'materialCode', 'materialName', 'specification', 'unit', 'quantity', 'unitPrice', 'amount']
const user = useUserStore()
const router = useRouter()
function openDelivery(id) {
  router.push({ name: 'warehouseDelivery', query: { whDelivery: '1', whId: String(id) } })
}
const available = ref([])
const columns = ref([...defaults])
const collapsed = ref(true)
const reportRoot = ref(null)
const fullscreen = ref(false)
const fullscreenStyle = ref({})
let fullscreenPanel
let panelObserver
function updateFullscreenBounds() {
  if (!fullscreenPanel) return
  const bounds = fullscreenPanel.getBoundingClientRect()
  const top = Math.max(0, bounds.top)
  const left = Math.max(0, bounds.left)
  fullscreenStyle.value = {
    top: `${top}px`, left: `${left}px`,
    width: `${Math.max(0, Math.min(bounds.right, window.innerWidth) - left)}px`,
    height: `${Math.max(0, Math.min(bounds.bottom, window.innerHeight) - top)}px`
  }
}
function exitFullscreen() {
  fullscreen.value = false
  panelObserver?.disconnect()
  panelObserver = undefined
  fullscreenPanel = undefined
  window.removeEventListener('resize', updateFullscreenBounds)
  window.removeEventListener('scroll', updateFullscreenBounds, true)
}
function toggleFullscreen() {
  if (fullscreen.value) { exitFullscreen(); return }
  fullscreenPanel = reportRoot.value?.closest('[role="tabpanel"], .gva-container2') || reportRoot.value?.closest('#gva-base-load-dom')?.parentElement || reportRoot.value?.parentElement
  if (!fullscreenPanel) return
  updateFullscreenBounds()
  fullscreen.value = true
  if (typeof ResizeObserver !== 'undefined') {
    panelObserver = new ResizeObserver(updateFullscreenBounds)
    panelObserver.observe(fullscreenPanel)
  }
  window.addEventListener('resize', updateFullscreenBounds)
  window.addEventListener('scroll', updateFullscreenBounds, true)
}
function handleEscape(event) {
  if (event.key !== 'Escape') return
  if (headerGesture) { event.preventDefault(); cancelHeaderPointer(); return }
  exitFullscreen()
}
const columnKeyword = ref('')
const filteredAvailable = computed(() => {
  const keyword = columnKeyword.value.trim().toLocaleLowerCase()
  return available.value.filter((column) => !keyword || column.label.toLocaleLowerCase().includes(keyword))
})
const options = ref({ brands: [], categories: [], departments: [] })
const statusOptions = [{ value: 'draft', label: '草稿' }, { value: 'reviewed', label: '已审核' }, { value: 'reReviewed', label: '已复审' }]
const defaultStatuses = ['reviewed', 'reReviewed']
const blankFilters = () => ({ deliveredRange: null, reviewedRange: null, brandId: undefined, categoryId: undefined, department: '', number: '', materialKeyword: '', statuses: [...defaultStatuses] })
const filters = reactive(blankFilters())
const page = ref(1)
const pageSize = ref(20)
const sortBy = ref('')
const sortOrder = ref('')
function pageIndex(index) { return pageSequenceNumber(index, page.value, pageSize.value) }
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
const resultTable = ref(null)
const headerPreview = ref(null)
const resultColumns = computed(() => (headerPreview.value || projection.value.columns).map((id) => available.value.find((column) => column.id === id)).filter(Boolean))
const tableWidthKey = 'warehouse.reconciliation.results'
const { widths: columnWidths, load: loadColumnWidths, headerDragEnd } = useTableWidths()
function onHeaderDragEnd(newWidth, oldWidth, column) {
  headerDragEnd(tableWidthKey, newWidth, oldWidth, column)
}
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

// Extract Shanghai's calendar day before doing date arithmetic, independent of the browser timezone.
function shortcutRange(preset) {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date())
  const part = (name) => Number(parts.find((item) => item.type === name).value)
  const year = part('year'); const month = part('month') - 1; const day = part('day')
  let start = new Date(Date.UTC(year, month, day))
  let end = new Date(start)
  if (preset === 'yesterday') { start.setUTCDate(day - 1); end = new Date(start) }
  if (preset === 'thisYear' || preset === 'lastYear') {
    const target = year - (preset === 'lastYear' ? 1 : 0)
    start = new Date(Date.UTC(target, 0, 1)); end = new Date(Date.UTC(target, 11, 31))
  }
  if (preset === 'thisWeek' || preset === 'lastWeek') {
    const monday = day - (start.getUTCDay() + 6) % 7 - (preset === 'lastWeek' ? 7 : 0)
    start = new Date(Date.UTC(year, month, monday)); end = new Date(Date.UTC(year, month, monday + 6))
  }
  if (['thisMonth', 'lastMonth', 'previousMonth'].includes(preset)) {
    const target = month - (preset === 'lastMonth' ? 1 : preset === 'previousMonth' ? 2 : 0)
    start = new Date(Date.UTC(year, target, 1)); end = new Date(Date.UTC(year, target + 1, 0))
  }
  // Element Plus expects Date values; local midnight preserves the selected calendar label.
  return [start, end].map((date) => new Date(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()))
}
const dateShortcuts = [
  ['今天', 'today'], ['昨天', 'yesterday'], ['今年', 'thisYear'], ['去年', 'lastYear'],
  ['上周', 'lastWeek'], ['本周', 'thisWeek'], ['本月', 'thisMonth'], ['上月', 'lastMonth'], ['前月', 'previousMonth']
].map(([text, preset]) => ({ text, value: () => shortcutRange(preset) }))
function dateRange(range) {
  if (!range?.length) return [undefined, undefined]
  const [year, month, day] = range[1].split('-').map(Number)
  const nextDay = new Date(Date.UTC(year, month - 1, day + 1)).toISOString().slice(0, 10)
  return [`${range[0]}T00:00:00+08:00`, `${nextDay}T00:00:00+08:00`]
}
function currentFilters() {
  const [deliveredFrom, deliveredTo] = dateRange(filters.deliveredRange)
  const [reviewedFrom, reviewedTo] = dateRange(filters.reviewedRange)
  return Object.fromEntries(Object.entries({ deliveredFrom, deliveredTo, reviewedFrom, reviewedTo, brandId: filters.brandId || undefined, categoryId: filters.categoryId || undefined, department: filters.department.trim() || undefined, number: filters.number.trim() || undefined, materialKeyword: filters.materialKeyword.trim() || undefined, statuses: [...(filters.statuses.length ? filters.statuses : defaultStatuses)] }).filter(([, value]) => value !== undefined))
}
const pendingChanges = computed(() => lastQuery.value && (JSON.stringify(columns.value) !== JSON.stringify(lastQuery.value.columns) || JSON.stringify(currentFilters()) !== JSON.stringify(lastQuery.value.filters) || sortBy.value !== lastQuery.value.sortBy || sortOrder.value !== lastQuery.value.sortOrder))
const resultFilterDescription = computed(() => {
  const f = lastQuery.value?.filters || {}
  const parts = [`当前结果：${(f.statuses || defaultStatuses).map((value) => statusOptions.find((status) => status.value === value)?.label || value).join('、')}快照`]
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
const draggedColumn = ref('')
const headerGhost = ref(null)
const headerInsertion = ref(null)
let headerGesture
let headerScrollFrame
let blockedClickTable
let blockedClickTimer
function headerCellClass({ column }) {
  return draggedColumn.value === column.property ? 'is-reorder-placeholder' : ''
}
function clearHeaderClickBlock() {
  window.removeEventListener('click', blockHeaderClick, true)
  window.removeEventListener('pointerdown', clearHeaderClickBlock, true)
  window.clearTimeout(blockedClickTimer)
  blockedClickTable = undefined
}
function blockHeaderClick(event) {
  if (!blockedClickTable?.contains(event.target)) return
  event.preventDefault()
  event.stopImmediatePropagation()
  clearHeaderClickBlock()
}
function startHeaderPointer(event, id) {
  if (event.button !== 0 || (event.pointerType && event.pointerType !== 'mouse') || loading.value || headerGesture) return
  const cell = event.currentTarget.closest('th')
  const table = resultTable.value?.$el
  if (!cell || !table || !projection.value.columns.includes(id)) return
  clearHeaderClickBlock()
  const bounds = cell.getBoundingClientRect()
  // The cell edge belongs to Element Plus's independent column-width resize gesture.
  if (event.clientX >= bounds.right - 8) return
  headerGesture = {
    id, pointerId: event.pointerId, table, cell,
    startX: event.clientX, startY: event.clientY, x: event.clientX, y: event.clientY,
    offsetX: event.clientX - bounds.left, offsetY: event.clientY - bounds.top,
    width: bounds.width, height: bounds.height, active: false,
    original: projection.value.columns
  }
  window.addEventListener('pointermove', moveHeaderPointer, true)
  window.addEventListener('pointerup', endHeaderPointer, true)
  window.addEventListener('pointercancel', cancelHeaderPointer, true)
  window.addEventListener('blur', cancelHeaderPointer)
  window.addEventListener('resize', cancelHeaderPointer)
  window.addEventListener('scroll', scrollHeaderPointer, true)
}
function headerCells() {
  if (!headerGesture) return []
  return [...headerGesture.table.querySelectorAll('.draggable-report-header[data-column-id]')]
    .map((label) => ({ id: label.dataset.columnId, cell: label.closest('th') }))
    .filter(({ cell }) => cell)
}
function headerDropBounds() {
  const gesture = headerGesture
  const cells = headerCells()
  const first = cells[0]?.cell
  if (!gesture || !first) return null
  const table = gesture.table.getBoundingClientRect()
  const row = first.getBoundingClientRect()
  const fixed = first.parentElement.querySelector('th.is-fixed-column--left')
  return { left: Math.max(table.left, fixed?.getBoundingClientRect().right || table.left), right: table.right, top: row.top, bottom: row.bottom }
}
function headerPointerInside() {
  const bounds = headerDropBounds()
  return bounds && headerGesture.x >= bounds.left && headerGesture.x <= bounds.right && headerGesture.y >= bounds.top && headerGesture.y <= bounds.bottom
}
function updateHeaderInsertion() {
  if (!headerGesture?.active) return
  const source = headerCells().find(({ id }) => id === headerGesture.id)?.cell
  const viewport = headerDropBounds()
  if (!source || !viewport || !headerPointerInside()) { headerInsertion.value = null; return }
  const bounds = source.getBoundingClientRect()
  headerInsertion.value = {
    left: `${Math.max(viewport.left, Math.min(bounds.left, viewport.right - 3))}px`,
    top: `${bounds.top}px`, height: `${bounds.height}px`
  }
}
function previewHeaderPosition() {
  const gesture = headerGesture
  if (!gesture?.active) return
  headerGhost.value = {
    label: columnLabel(gesture.id),
    style: { left: `${gesture.x - gesture.offsetX}px`, top: `${gesture.y - gesture.offsetY}px`, width: `${gesture.width}px`, height: `${gesture.height}px` }
  }
  if (!headerPointerInside()) { headerInsertion.value = null; return }
  const others = headerCells().filter(({ id }) => id !== gesture.id)
  const index = others.filter(({ cell }) => {
    const bounds = cell.getBoundingClientRect()
    return gesture.x > bounds.left + bounds.width / 2
  }).length
  const next = others.map(({ id }) => id)
  next.splice(index, 0, gesture.id)
  if (next.some((id, position) => id !== headerPreview.value[position])) {
    headerPreview.value = next
    // Element Plus observes the hidden column order before rendering its header cells.
    nextTick(() => nextTick(updateHeaderInsertion))
  } else updateHeaderInsertion()
}
function scheduleHeaderScroll() {
  if (headerScrollFrame || !headerGesture?.active) return
  const bounds = headerDropBounds()
  if (!bounds || !headerPointerInside()) return
  const direction = headerGesture.x < bounds.left + 36 ? -1 : headerGesture.x > bounds.right - 36 ? 1 : 0
  if (!direction) return
  const scroll = headerGesture.table.querySelector('.el-scrollbar__wrap')
  if (!scroll || (direction < 0 ? scroll.scrollLeft <= 0 : scroll.scrollLeft >= scroll.scrollWidth - scroll.clientWidth)) return
  headerScrollFrame = window.requestAnimationFrame(() => {
    headerScrollFrame = undefined
    if (!headerGesture?.active) return
    scroll.scrollLeft += direction * 14
    previewHeaderPosition()
    scheduleHeaderScroll()
  })
}
function moveHeaderPointer(event) {
  const gesture = headerGesture
  if (!gesture || event.pointerId !== gesture.pointerId) return
  if (event.buttons === 0) { cancelHeaderPointer(); return }
  gesture.x = event.clientX
  gesture.y = event.clientY
  if (!gesture.active) {
    if (Math.hypot(gesture.x - gesture.startX, gesture.y - gesture.startY) < 5) return
    gesture.active = true
    draggedColumn.value = gesture.id
    headerPreview.value = [...projection.value.columns]
    blockedClickTable = gesture.table
    window.addEventListener('click', blockHeaderClick, true)
  }
  event.preventDefault()
  previewHeaderPosition()
  scheduleHeaderScroll()
}
function scrollHeaderPointer() {
  if (!headerGesture?.active) return
  previewHeaderPosition()
  scheduleHeaderScroll()
}
function finishHeaderPointer() {
  const active = headerGesture?.active
  window.removeEventListener('pointermove', moveHeaderPointer, true)
  window.removeEventListener('pointerup', endHeaderPointer, true)
  window.removeEventListener('pointercancel', cancelHeaderPointer, true)
  window.removeEventListener('blur', cancelHeaderPointer)
  window.removeEventListener('resize', cancelHeaderPointer)
  window.removeEventListener('scroll', scrollHeaderPointer, true)
  if (headerScrollFrame) window.cancelAnimationFrame(headerScrollFrame)
  headerScrollFrame = undefined
  headerGesture = undefined
  draggedColumn.value = ''
  headerPreview.value = null
  headerGhost.value = null
  headerInsertion.value = null
  if (active) {
    window.addEventListener('pointerdown', clearHeaderClickBlock, true)
    blockedClickTimer = window.setTimeout(clearHeaderClickBlock, 400)
  }
}
function cancelHeaderPointer(event) {
  if (event?.pointerId != null && event.pointerId !== headerGesture?.pointerId) return
  finishHeaderPointer()
}
function commitResultColumns(next) {
  if (next.every((id, index) => id === projection.value.columns[index])) return
  projection.value = { ...projection.value, columns: [...next] }
  // Permuting visible columns cannot change grouping or totals, so exports can use the new order immediately.
  if (lastQuery.value) lastQuery.value = { ...lastQuery.value, columns: [...next] }
  const visible = new Set(next)
  const selected = new Set(columns.value)
  const ordered = next.filter((id) => selected.has(id))
  let index = 0
  columns.value = columns.value.map((id) => visible.has(id) ? ordered[index++] : id)
}
function endHeaderPointer(event) {
  const gesture = headerGesture
  if (!gesture || event.pointerId !== gesture.pointerId) return
  gesture.x = event.clientX
  gesture.y = event.clientY
  if (gesture.active && gesture.original === projection.value.columns && headerPointerInside()) {
    previewHeaderPosition()
    const next = [...headerPreview.value]
    finishHeaderPointer()
    commitResultColumns(next)
  } else finishHeaderPointer()
}
function moveResultColumn(id, offset) {
  cancelHeaderPointer()
  const next = [...projection.value.columns]
  const index = next.indexOf(id)
  const target = index + offset
  if (index < 0 || target < 0 || target >= next.length) return
  next.splice(index, 1)
  next.splice(target, 0, id)
  commitResultColumns(next)
}
function changeReportSort({ prop, order }) {
  sortBy.value = order ? prop : ''
  sortOrder.value = order === 'ascending' ? 'asc' : order === 'descending' ? 'desc' : ''
  page.value = 1
  search()
}
function useDefaultColumns() { columns.value = defaults.filter((id) => available.value.some((column) => column.id === id)) }
async function executeQuery(request) {
  cancelHeaderPointer()
  const sequence = ++requestSequence
  loading.value = true
  queryError.value = ''
  try {
    const result = await queryReport(request)
    if (sequence !== requestSequence) return
    projection.value = result
    lastQuery.value = {
      filters: { ...request.filters },
      columns: [...request.columns],
      sortBy: request.sortBy || '',
      sortOrder: request.sortOrder || ''
    }
    page.value = result.page
    pageSize.value = result.pageSize
  } catch (error) {
    if (sequence === requestSequence) { queryError.value = error.message; showError(error) }
  } finally { if (sequence === requestSequence) loading.value = false }
}
async function search() {
  if (!columns.value.length) { ElMessage.warning('至少选择一列'); return }
  const sort = sortBy.value ? { sortBy: sortBy.value, sortOrder: sortOrder.value } : {}
  await executeQuery({ filters: currentFilters(), columns: [...columns.value], page: 1, pageSize: pageSize.value, ...sort })
}
async function reset() { Object.assign(filters, blankFilters()); await search() }
async function loadPage() {
  if (!lastQuery.value) return
  const sort = lastQuery.value.sortBy ? { sortBy: lastQuery.value.sortBy, sortOrder: lastQuery.value.sortOrder } : {}
  await executeQuery({ filters: { ...lastQuery.value.filters }, columns: [...lastQuery.value.columns], page: page.value, pageSize: pageSize.value, ...sort })
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
const schemeOptions = computed(() => selectedScheme.value && !schemes.value.some((scheme) => idOf(scheme) === idOf(selectedScheme.value)) ? [selectedScheme.value, ...schemes.value] : schemes.value)
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
async function selectScheme(id) {
  if (!id) { selectedScheme.value = null; useDefaultColumns(); return }
  const scheme = schemeOptions.value.find((item) => idOf(item) === id)
  if (scheme) await loadScheme(scheme)
}
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
  window.addEventListener('keydown', handleEscape)
  await Promise.all([
    loadReportColumns().then((data) => { available.value = data; useDefaultColumns() }).catch(showError),
    loadReportOptions().then((data) => { options.value = data }).catch(showError),
    refreshSchemes(),
    loadColumnWidths(tableWidthKey)
  ])
  if (available.value.length) await search()
})
onDeactivated(() => {
  cancelHeaderPointer()
  clearHeaderClickBlock()
  exitFullscreen()
})
onBeforeUnmount(() => {
  requestSequence++
  cancelHeaderPointer()
  clearHeaderClickBlock()
  exitFullscreen()
  finishPrint()
  window.removeEventListener('afterprint', finishPrint)
  window.removeEventListener('keydown', handleEscape)
})
</script>

<style scoped>
.warehouse-reconciliation { display: flex; align-items: stretch; gap: 16px; }
.report-sidebar { width: 330px; flex: 0 0 330px; padding: 16px; height: calc(100vh - 120px); min-height: 560px; overflow: auto; box-sizing: border-box; }
.report-main { flex: 1; min-width: 0; }
.sidebar-heading, .report-heading, .results-heading, .filter-actions, .results-footer, .column-tools { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.sidebar-heading { margin-bottom: 8px; }
.sidebar-heading strong { font-size: 16px; }
.sidebar-heading p, .report-title p, .results-heading p { margin: 4px 0 0; color: var(--el-text-color-secondary); font-size: 12px; }
.sidebar-control { width: 100%; }
.scheme-toolbar { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; margin-bottom: 10px; }
.scheme-toolbar :deep(.el-button + .el-button) { margin-left: 0; }
.scheme-selection { display: grid; gap: 4px; padding: 12px; border-radius: 4px; background: var(--el-fill-color-light); overflow-wrap: anywhere; }
.field-label { color: var(--el-text-color-secondary); font-size: 12px; }
.column-tools { margin-bottom: 10px; }
.column-picker { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); margin: 10px 0; }
.column-picker :deep(.el-checkbox) { min-width: 0; margin-right: 8px; height: 30px; }
.column-picker :deep(.el-checkbox__label) { overflow: hidden; text-overflow: ellipsis; }
.scheme-editor { border-top: 1px solid var(--el-border-color-lighter); margin-top: 12px; padding-top: 14px; }
.scheme-scope { float: right; margin-left: 12px; color: var(--el-text-color-secondary); }
.report-hint { margin: 8px 0; color: var(--el-text-color-secondary); font-size: 12px; line-height: 1.5; overflow-wrap: anywhere; }
.report-search { margin-bottom: 16px; }
.report-heading { margin-bottom: 16px; }
.report-title, .heading-actions { display: flex; align-items: center; gap: 12px; }
.report-title h2, .results-heading h3 { margin: 0; font-size: 18px; }
.heading-actions { flex-wrap: wrap; justify-content: flex-end; }
.report-filters { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); column-gap: 12px; }
.report-filters :deep(.el-form-item) { margin-bottom: 12px; min-width: 0; }
.report-filters :deep(.el-date-editor), .report-filters :deep(.el-select), .report-filters :deep(.el-input) { width: 100%; }
.report-filters :deep(.el-range-input) { min-width: 0; }
.filter-actions { padding-top: 12px; border-top: 1px solid var(--el-border-color-lighter); }
.filter-actions > div { display: flex; gap: 8px; }
.filter-note { color: var(--el-text-color-secondary); font-size: 12px; }
.report-alert { margin-bottom: 12px; }
.report-results { min-width: 0; }
.report-results-fullscreen { position: fixed; z-index: 100; box-sizing: border-box; margin: 0; padding: 16px; background: var(--el-bg-color); display: flex; flex-direction: column; overflow: hidden; }
.report-results-fullscreen .report-table { flex: 1; min-height: 0; }
.report-results-fullscreen .results-heading, .report-results-fullscreen .results-footer, .report-results-fullscreen .result-alert { flex-shrink: 0; }
.report-results-fullscreen > .report-hint { max-height: 60px; overflow: auto; flex-shrink: 0; }
.result-actions { display: flex; align-items: center; gap: 12px; }
.result-alert { margin-bottom: 14px; }
.results-heading { margin-bottom: 14px; }
.results-meta { display: grid; min-width: 100px; justify-items: end; }
.results-meta strong { font-size: 20px; line-height: 1.2; color: var(--el-color-primary); }
.results-meta span { color: var(--el-text-color-secondary); font-size: 12px; }
.report-table { width: 100%; font-size: 14px; }
.report-table :deep(.el-table__cell) { padding-top: 3px; padding-bottom: 3px; }
.draggable-report-header { display: inline-block; max-width: calc(100% - 20px); padding: 2px 3px; cursor: grab; vertical-align: middle; user-select: none; touch-action: pan-y; }
.draggable-report-header.is-dragging { opacity: 0; }
.report-table :deep(.el-table__header-wrapper th.is-reorder-placeholder) { position: relative; background: var(--el-color-primary-light-9); outline: 1px dashed var(--el-color-primary); outline-offset: -2px; }
.report-table :deep(th.is-reorder-placeholder .caret-wrapper) { visibility: hidden; }
.report-column-drag-ghost { position: fixed; z-index: 4000; display: flex; align-items: center; box-sizing: border-box; padding: 3px 12px; pointer-events: none; cursor: grabbing; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; color: var(--el-text-color-primary); background: var(--el-bg-color); border: 1px solid var(--el-color-primary); border-radius: 4px; box-shadow: 0 6px 18px #0003; opacity: .92; font-size: 14px; font-weight: 600; }
.report-column-insertion { position: fixed; z-index: 4001; width: 3px; pointer-events: none; background: var(--el-color-primary); }
.report-table :deep(.cell) { overflow-wrap: anywhere; }
.report-table :deep(.el-table__header-wrapper th) { background: var(--el-fill-color-light); color: var(--el-text-color-primary); font-weight: 600; }
.report-table :deep(.el-table__footer-wrapper td) { background: var(--el-fill-color-light); color: var(--el-text-color-primary); font-weight: 600; }
.report-table :deep(.el-table__footer-wrapper .cell) { white-space: normal; }
.results-footer { flex-wrap: wrap; margin-top: 14px; }
.results-footer > span { color: var(--el-text-color-secondary); font-size: 12px; }
.results-footer :deep(.el-pagination) { margin-left: auto; }
.unavailable { cursor: help; border-bottom: 1px dashed var(--el-text-color-secondary); }
.base-unit { display: block; color: var(--el-text-color-secondary); }
@media (max-width: 1000px) {
  .warehouse-reconciliation { flex-direction: column; }
  .report-sidebar { width: 100%; flex-basis: auto; height: auto; min-height: 0; max-height: 560px; }
  .report-main { width: 100%; }
  .report-filters { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 680px) {
  .report-filters { grid-template-columns: minmax(0, 1fr); }
  .report-filters :deep(.el-form-item) { flex-direction: column; align-items: stretch; }
  .report-filters :deep(.el-form-item__label) { width: 100% !important; height: auto; justify-content: flex-start; padding: 0 0 4px; line-height: 20px; }
  .report-filters :deep(.el-form-item__content) { width: 100%; margin-left: 0 !important; }
  .report-heading, .results-heading, .filter-actions { align-items: flex-start; flex-direction: column; }
  .heading-actions { justify-content: flex-start; }
  .results-meta { justify-items: start; }
  .results-footer :deep(.el-pagination) { margin-left: 0; }
  .filter-note { line-height: 1.5; }
}
</style>
