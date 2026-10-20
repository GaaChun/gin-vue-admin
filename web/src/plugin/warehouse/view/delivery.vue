<template>
  <div class="delivery-page">
    <div v-show="!detailRoute">
      <el-form inline class="delivery-filters" @submit.prevent="search">
        <template v-for="group in filterGroups" :key="group.key">
          <div v-show="group.key === 'basic' || moreFiltersOpen" :data-filter-group="group.key">
            <el-form-item v-for="field in group.fields" :key="field.key" :label="field.label" :data-filter-key="field.key">
              <el-input v-if="field.key === 'number'" v-model="filters.number" clearable placeholder="完整送货单号" @keyup.enter="search" />
              <el-date-picker v-else-if="field.key === 'dates'" v-model="filters.dates" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期" />
              <el-select v-else-if="field.key === 'brandId'" v-model="filters.brandId" clearable filterable class="brand-select">
                <el-option v-for="brand in brands" :key="idOf(brand)" :value="idOf(brand)" :label="`${brand.code} ${brand.name}`" />
              </el-select>
              <el-input v-else-if="field.key === 'department'" v-model="filters.department" clearable placeholder="输入部门" @keyup.enter="search" />
              <el-select v-else-if="field.key === 'status'" v-model="filters.status" clearable class="status-select"><el-option label="未审核" value="draft" /><el-option label="已审核" value="reviewed" /></el-select>
            </el-form-item>
          </div>
        </template>
        <el-form-item>
          <el-button type="primary" @click="search">查询</el-button><el-button @click="resetSearch">重置</el-button>
          <el-button v-if="displaySettings.filters.more.length" link type="primary" @click="moreFiltersOpen = !moreFiltersOpen">{{ moreFiltersOpen ? '收起更多条件' : '更多条件' }}</el-button>
        </el-form-item>
      </el-form>
    </div>
    <div v-show="!detailRoute" class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="primary" :disabled="!optionsReady || editorLoading" @click="openNew">新建送货单</el-button>
        <el-button :loading="listLoading" @click="loadList">刷新</el-button>
        <el-button v-if="!optionsReady" :loading="optionsLoading" @click="loadOptions">重新加载候选资料</el-button>
        <el-button type="success" :disabled="!selectedDrafts.length || actionLoading" :loading="batchLoading === 'review'" @click="batchTransition('review')">批量审核</el-button>
        <el-button type="warning" :disabled="!selectedReviewed.length || actionLoading" :loading="batchLoading === 'unreview'" @click="batchTransition('unreview')">批量反审核</el-button>
        <el-button @click="router.push({ name: 'warehouseDeliveryDisplaySettings' })">单据展示设置</el-button>
      </div>
      <el-alert v-if="listError" :title="listError" type="error" show-icon :closable="false" />
      <el-table :key="displaySettings.list.columns.join(',')" ref="listTable" v-loading="listLoading" :data="notes" :row-key="idOf" border size="small" :default-sort="{ prop: sortBy || 'deliveredAt', order: sortOrder === 'asc' ? 'ascending' : 'descending' }" @sort-change="changeSort" @header-dragend="onHeaderDragEnd" @selection-change="selectionChanged" @row-dblclick="openExisting($event)">
        <el-table-column type="selection" column-key="__selection" :width="columnWidths.__selection || 48" :selectable="selectableNote" />
        <el-table-column type="index" column-key="__index" label="序号" :width="columnWidths.__index || 65" align="center" :index="pageIndex" />
        <el-table-column v-for="column in orderedDisplayColumns" :key="column.key" :prop="column.key" :label="column.label" :width="displaySettings.list.widths[column.key] || column.width" min-width="80" :align="column.key === 'totalAmount' ? 'right' : 'left'" sortable="custom">
          <template #default="{ row }">
            <el-tag v-if="column.key === 'status'" :type="row.status === 'reviewed' ? 'success' : 'info'">{{ row.status === 'reviewed' ? '已审核' : '未审核' }}</el-tag>
            <span v-else-if="column.key === 'deliveredAt'">{{ displayDate(row.deliveredAt) }}</span>
            <span v-else-if="column.key === 'createdAt'">{{ displayTime(row.CreatedAt ?? row.createdAt) }}</span>
            <span v-else-if="column.key === 'reviewedAt'">{{ displayTime(row.reviewedAt) }}</span>
            <span v-else>{{ row[column.key] }}</span>
          </template>
        </el-table-column>
        <el-table-column column-key="__actions" label="操作" fixed="right" :width="columnWidths.__actions || 150"><template #default="{ row }">
          <div class="delivery-row-actions">
            <el-button link size="small" type="primary" :disabled="editorLoading || actionLoading" @click="openExisting(row)">{{ row.status === 'reviewed' ? '查看' : '编辑' }}</el-button>
            <el-button v-if="row.status === 'draft'" link size="small" type="success" :disabled="actionLoading" @click="transition(row, 'review')">审核</el-button>
            <el-button v-else link size="small" type="warning" :disabled="actionLoading" @click="transition(row, 'unreview')">反审核</el-button>
            <el-button v-if="row.status === 'draft'" link size="small" type="danger" :disabled="actionLoading" @click="transition(row, 'delete')">删除</el-button>
          </div>
        </template></el-table-column>
      </el-table>
      <div class="gva-pagination"><el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[20, 50, 100]" :total="total" layout="total, sizes, prev, pager, next, jumper" @current-change="loadList" @size-change="search" /></div>
    </div>
    

    <div v-if="detailRoute" v-loading="editorLoading" class="delivery-detail-page">
      <header class="delivery-detail-heading">
        <h2>{{ idOf(note) ? '送货单详情' : '新建送货单' }}</h2>
        <el-tag v-if="idOf(note)" :type="readonly ? 'success' : 'info'">{{ readonly ? '已审核' : '未审核' }}</el-tag>
        <el-tag v-else type="info">未保存</el-tag>
      </header>
      <div class="delivery-detail-scroll">
      <div class="delivery-editor">
        <el-alert v-if="reservationError" :title="reservationError" type="error" :closable="false" show-icon><template #default><el-button :loading="reserving" @click="reserveNumber">重新预占正式单号</el-button></template></el-alert>
        <el-alert v-if="conflictMessage" :title="conflictMessage" type="error" :closable="false" show-icon><template #default>当前表单未被覆盖。<el-button link type="primary" @click="reloadEditor">放弃本地修改并读取最新版本</el-button></template></el-alert>
        <el-alert v-if="readonly" title="已审核送货单只读。修改前请单独执行反审核。" type="info" :closable="false" />
        <el-form inline class="editor-header" :disabled="readonly || saving || actionLoading || editorLoading">
          <el-form-item label="正式单号"><strong>{{ note.number || reservation.number || (reserving ? '正在预占正式单号…' : '尚未取得正式单号') }}</strong></el-form-item>
          <el-form-item label="客户" required>
            <el-select :model-value="form.brandId" filterable :filter-method="filterBrands" :disabled="rowsLoading" class="brand-select" @change="changeBrand" @visible-change="handleBrandVisibility">
              <el-option v-if="form.brandId && !brands.some((brand) => idOf(brand) === form.brandId)" :value="form.brandId" :label="note.brandNameSnapshot || `历史客户 #${form.brandId}`" />
              <el-option v-for="brand in filteredBrands" :key="idOf(brand)" :value="idOf(brand)" :label="`${brand.code} ${brand.name}`" />
            </el-select>
          </el-form-item>
          <el-form-item label="部门"><el-autocomplete v-model="form.department" :fetch-suggestions="suggestDepartments" maxlength="100" placeholder="请输入部门" /></el-form-item>
          <el-form-item label="送货日期" required><el-date-picker v-model="form.deliveredAt" type="date" value-format="YYYY-MM-DD" placeholder="选择送货日期" /></el-form-item>
          <el-form-item v-if="idOf(note)" label="创建人"><el-input :model-value="note.createdByNameSnapshot || '—'" readonly /></el-form-item>
          <el-form-item v-if="idOf(note)" label="创建时间"><el-input :model-value="displayTime(note.createdAt ?? note.CreatedAt)" readonly /></el-form-item>
          <el-form-item v-if="idOf(note)" label="修改人"><el-input :model-value="userNickname(note.updatedBy)" readonly /></el-form-item>
          <el-form-item v-if="idOf(note)" label="修改时间"><el-input :model-value="displayTime(note.updatedAt ?? note.UpdatedAt)" readonly /></el-form-item>
          <el-form-item v-if="idOf(note)" label="审核人"><el-input :model-value="note.reviewedByNameSnapshot || '—'" readonly /></el-form-item>
          <el-form-item v-if="idOf(note)" label="审核时间"><el-input :model-value="displayTime(note.reviewedAt)" readonly /></el-form-item>
        </el-form>
        <div v-if="!readonly" class="editor-tools">
          <div class="editor-tool-actions">
            <el-button :disabled="!canEdit" @click="addBlankRow">新增空行</el-button>
            <el-button type="primary" :disabled="!canEdit || !optionsReady" @click="pickerOpen = true">批量选择物料</el-button>
            <el-button :disabled="!canEdit || !optionsReady" @click="openQuickMaterialCreate">新建物料并添加</el-button>
          </div>
          <span class="editor-hint">数量最多4位小数且大于0，单价最多4位小数且不小于0；Enter / Tab 前进，Shift+Tab 返回。</span>
        </div>
        <el-table :key="detailColumnSignature" :data="displayRows" row-key="rowKey" border size="small" class="detail-table" show-summary :summary-method="tableSummary" @sort-change="changeDetailSort" @header-dragend="onDetailHeaderDragEnd">
          <el-table-column type="index" label="行" width="50" :fixed="displaySettings.detail.frozen ? 'left' : false" />
          <el-table-column v-for="(column, index) in orderedDetailColumns" :key="column.key" :prop="column.key" :label="column.label" :width="displaySettings.detail.widths[column.key] || column.width" min-width="80" :fixed="index < displaySettings.detail.frozen ? 'left' : false" :align="column.key === 'amount' ? 'right' : 'left'" sortable="custom">
            <template #default="{ row }">
              <template v-if="column.key === 'materialCode'">
                <span v-if="readonly">{{ row.materialCode }}</span>
                <el-select v-else :ref="(element) => setCellRef(row, 'material', element)" :model-value="row.materialId" filterable :filter-method="(value) => row.keyword = value" :loading="row.loading" :disabled="!canEdit" placeholder="编号" @change="(value) => selectMaterial(row, value)" @visible-change="(open) => row.materialOpen = open" @keydown="cellKeydown($event, row, 'material')">
                  <el-option v-if="row.materialId && !inlineMaterials(row).some((item) => idOf(item) === row.materialId)" :value="row.materialId" :label="row.materialCode" disabled />
                  <el-option v-for="material in inlineMaterials(row)" :key="idOf(material)" :value="idOf(material)" :label="material.code"><span>{{ material.code }} {{ material.name }}</span><small class="option-spec">{{ material.specification }}</small></el-option>
                </el-select>
                <div v-if="row.errors.materialId" class="cell-error">{{ row.errors.materialId }}</div>
              </template>
              <template v-else-if="column.key === 'unitName'">
                <span v-if="readonly">{{ row.unitName }}</span>
                <el-select v-else :ref="(element) => setCellRef(row, 'unit', element)" :model-value="row.materialUnitId" :disabled="!canEdit || !row.materialId || row.loading" @change="(value) => selectUnit(row, value)" @visible-change="(open) => open ? loadRowUnits(row) : row.unitOpen = false" @keydown="cellKeydown($event, row, 'unit')">
                  <el-option v-for="binding in row.units" :key="idOf(binding)" :value="idOf(binding)" :label="bindingLabel(binding)" :disabled="!binding.enabled" />
                </el-select>
                <div v-if="row.errors.materialUnitId" class="cell-error">{{ row.errors.materialUnitId }}</div>
              </template>
              <template v-else-if="column.key === 'quantity' || column.key === 'unitPrice'">
                <span v-if="readonly">{{ row[column.key] }}</span>
                <el-input v-else :ref="(element) => setCellRef(row, column.key, element)" v-model="row[column.key]" :disabled="!canEdit" inputmode="decimal" @input="clearFieldError(row, column.key)" @compositionstart="row.composing = true" @compositionend="row.composing = false" @keydown="cellKeydown($event, row, column.key)" />
                <div v-if="row.errors[column.key]" class="cell-error">{{ row.errors[column.key] }}</div>
              </template>
              <span v-else-if="column.key === 'amount'">{{ row.amount ?? '保存后计算' }}</span>
              <span v-else>{{ row[column.key] }}</span>
            </template>
          </el-table-column>
          <el-table-column v-if="!readonly" label="操作" width="80" fixed="right"><template #default="{ row }"><el-button link type="danger" :disabled="!canEdit" @click="removeRow(row)">移除</el-button></template></el-table-column>
        </el-table>
        <p class="editor-hint">金额以服务端保存计算为准，不使用浮点数预估。完全空白的尾行不会提交，部分填写的行必须补齐。</p>
        
      </div>
      </div>
      <footer class="delivery-detail-footer">
        <div class="delivery-detail-navigation">
          <el-button :disabled="!idOf(note) || editorLoading || saving || actionLoading || rowsLoading" @click="reloadEditor">刷新</el-button>
          <el-button v-if="idOf(note) && readonly" :disabled="editorLoading || saving || actionLoading || rowsLoading || reserving" @click="copyDocument">复制单据</el-button>
          <el-button type="primary" :disabled="!optionsReady || editorLoading || saving || actionLoading || rowsLoading || reserving" @click="openNew">新增</el-button>
          <el-button :disabled="!canNavigatePrevious" @click="navigateDocument(-1)">上一张</el-button>
          <el-button :disabled="!canNavigateNext" @click="navigateDocument(1)">下一张</el-button>
        </div>
        <div class="delivery-detail-actions">
          <el-button v-if="idOf(note) && !readonly" type="danger" :disabled="!canEdit" @click="transition(note, 'delete')">删除</el-button>
          <el-button v-if="!readonly" :loading="saving" :disabled="!canEdit || (!idOf(note) && !reservation.token)" @click="save(true)">保存并继续新增</el-button>
          <el-button v-if="!readonly" type="primary" :loading="saving" :disabled="!canEdit || (!idOf(note) && !reservation.token)" @click="save()">保存</el-button>
          <el-button v-if="idOf(note) && !readonly" type="success" :loading="actionLoading" :disabled="dirty || !canEdit" @click="transition(note, 'review')">审核</el-button>
          <el-button v-if="idOf(note) && readonly" type="warning" :loading="actionLoading" :disabled="actionLoading || editorLoading" @click="transition(note, 'unreview')">反审核</el-button>
        </div>
      </footer>
    </div>
    <DeliveryMaterialPicker v-model="pickerOpen" :materials="materials" :categories="categoryTree" :loading="rowsLoading" @confirm="appendMaterials" />
    <el-dialog v-model="quickMaterialDialogOpen" title="新建物料并添加到送货单" width="480px" :close-on-click-modal="false">
      <el-form label-width="90px" @submit.prevent="createQuickMaterial">
        <el-form-item label="物料名称" required><el-input v-model="quickMaterialForm.name" maxlength="200" /></el-form-item>
        <el-form-item label="物料编号" required><el-input v-model="quickMaterialForm.code" maxlength="64" /></el-form-item>
        <el-form-item label="助记符" required><el-input v-model="quickMaterialForm.mnemonic" maxlength="100" /></el-form-item>
        <el-form-item label="基本单位" required>
          <el-select v-model="quickMaterialForm.unitId" filterable placeholder="选择基本单位" class="full-width">
            <el-option v-for="unit in units" :key="idOf(unit)" :value="idOf(unit)" :label="unit.name" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button :disabled="quickMaterialSaving" @click="quickMaterialDialogOpen = false">取消</el-button>
        <el-button type="primary" :loading="quickMaterialSaving" @click="createQuickMaterial">创建并添加</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, nextTick, onActivated, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import DeliveryMaterialPicker from '../components/delivery-material-picker.vue'
import { getUserList, setSelfSetting } from '@/api/user'
import { useUserStore } from '@/pinia/modules/user'
import { allOptions, idOf, pageSequenceNumber, request, showError } from '../api/common'
import { saveMaster } from '../api/master'
import { useTableWidths } from '../use-table-widths'
import { DISPLAY_SETTING_KEY, FILTER_FIELDS, LIST_FIELDS, DETAIL_FIELDS, normalizeDisplaySettings, getDefaultDateRange } from '../delivery-display'

defineOptions({ name: 'WarehouseDelivery' })
const base = '/warehouse/delivery'
const route = useRoute()
const router = useRouter()
const notes = ref([])
const brands = ref([])
const categories = ref([])
const materials = ref([])
const units = ref([])
const optionsReady = ref(false)
const optionsLoading = ref(false)
const listLoading = ref(false)
const listError = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filters = reactive({ number: '', dates: [], brandId: undefined, status: '', department: '' })
const tableWidthKey = 'warehouse.delivery.list'
const { widths: columnWidths, load: loadColumnWidths, headerDragEnd } = useTableWidths()
const userStore = useUserStore()
const displaySettings = computed(() => normalizeDisplaySettings(userStore.userInfo.originSetting?.[DISPLAY_SETTING_KEY]))
const listFieldMap = new Map(LIST_FIELDS.map((column) => [column.key, column]))
const detailFieldMap = new Map(DETAIL_FIELDS.map((column) => [column.key, column]))
const filterFieldMap = new Map(FILTER_FIELDS.map((field) => [field.key, field]))
const orderedDisplayColumns = computed(() => displaySettings.value.list.columns.map((key) => listFieldMap.get(key)))
const orderedDetailColumns = computed(() => displaySettings.value.detail.columns.map((key) => detailFieldMap.get(key)))
const detailColumnSignature = computed(() => `${displaySettings.value.detail.columns.join(',')}:${displaySettings.value.detail.frozen}`)
const moreFiltersOpen = ref(false)
const filterGroups = computed(() => ['basic', 'more'].map((key) => ({
  key, fields: displaySettings.value.filters[key].map((field) => filterFieldMap.get(field))
})))
const editingFields = computed(() => orderedDetailColumns.value.map(({ key }) =>
  ({ materialCode: 'material', unitName: 'unit', quantity: 'quantity', unitPrice: 'unitPrice' })[key]
).filter(Boolean))
let widthSaveQueue = Promise.resolve()
const sortBy = ref('')
const sortOrder = ref('')
const usersById = ref(new Map())
const listTable = ref()
function pageIndex(index) { return pageSequenceNumber(index, page.value, pageSize.value) }
const selectedNotes = ref([])
const batchLoading = ref('')
const selectedDrafts = computed(() => selectedNotes.value.filter((row) => row.status === 'draft'))
const selectedReviewed = computed(() => selectedNotes.value.filter((row) => row.status === 'reviewed'))
function selectionChanged(rows) { selectedNotes.value = rows }
function selectableNote(row) { return row.status === 'draft' || row.status === 'reviewed' }
async function batchTransition(action) {
  if (batchLoading.value) return
  const targets = action === 'review' ? selectedDrafts.value : selectedReviewed.value
  if (!targets.length) return
  const label = action === 'review' ? '审核' : '反审核'
  if (!(await confirm(`确定${label}选中的 ${targets.length} 张送货单？`, `${label}确认`))) return
  batchLoading.value = action
  let completed = 0
  try {
    for (const row of targets) {
      await request({ url: `${base}/${action}`, method: 'post', data: { id: idOf(row), version: row.version } })
      completed++
    }
    ElMessage.success(`${label}成功：${completed} 张`)
  } catch (error) {
    showError(error)
    if (completed) ElMessage.warning(`已${label} ${completed} 张，其余未完成；请刷新后重试`)
  } finally {
    batchLoading.value = ''
    selectedNotes.value = []
    listTable.value?.clearSelection?.()
    await loadList()
  }
}
const editorOpen = ref(false)
const editorLoading = ref(false)
const saving = ref(false)
const actionLoading = ref(false)
const rowsLoading = ref(false)
const reserving = ref(false)
const reservationError = ref('')
const conflictMessage = ref('')
const reservation = reactive({ number: '', token: '' })
const note = ref({})
const customerSearch = ref('')
const form = reactive({ brandId: undefined, department: '', deliveredAt: '' })
const departments = ref([])
const quickMaterialSearch = ref('')
const quickMaterialDialogOpen = ref(false)
const quickMaterialSaving = ref(false)
const quickMaterialForm = reactive({ name: '', code: '', mnemonic: '', unitId: undefined })
const rows = ref([])
const lineCategoryId = ref()
const pickerOpen = ref(false)
const savedSignature = ref('')
const cellRefs = new Map()
const materialUnitsCache = new Map()
let nextRowKey = 0
let listSequence = 0
const detailSortBy = ref('')
const detailSortOrder = ref('')
const displayRows = computed(() => {
  if (!detailSortBy.value || !detailSortOrder.value) return rows.value
  const direction = detailSortOrder.value === 'asc' ? 1 : -1
  return rows.value.map((row, index) => ({ row, index }))
    .sort((a, b) => {
      const left = a.row[detailSortBy.value]
      const right = b.row[detailSortBy.value]
      const leftMissing = left == null || left === ''
      const rightMissing = right == null || right === ''
      if (leftMissing || rightMissing) {
        if (leftMissing !== rightMissing) return leftMissing ? 1 : -1
        return a.index - b.index
      }
      return compareDetailValues(left, right) * direction || a.index - b.index
    })
    .map(({ row }) => row)
})
const readonly = computed(() => note.value.status === 'reviewed')
const canEdit = computed(() => !readonly.value && !saving.value && !actionLoading.value && !editorLoading.value && !rowsLoading.value)
const dirty = computed(() => editorOpen.value && signature() !== savedSignature.value)
const detailListIndex = computed(() => notes.value.findIndex((row) => idOf(row) === idOf(note.value)))
const canNavigatePrevious = computed(() => detailListIndex.value > 0 && !editorLoading.value && !saving.value && !actionLoading.value && !rowsLoading.value)
const canNavigateNext = computed(() => detailListIndex.value >= 0 && detailListIndex.value < notes.value.length - 1 && !editorLoading.value && !saving.value && !actionLoading.value && !rowsLoading.value)
const filteredBrands = computed(() => {
  const keyword = customerSearch.value.trim().toLocaleLowerCase()
  if (!keyword) return brands.value
  return brands.value.filter((brand) => [brand.name, brand.mnemonic, brand.code]
    .some((value) => String(value || '').toLocaleLowerCase().includes(keyword)))
})
function userNickname(id) {
  return usersById.value.get(Number(id)) || '—'
}
async function loadUsers() {
  try {
    const response = await getUserList({ page: 1, pageSize: 999 })
    if (response?.code === 0) {
      usersById.value = new Map((response.data?.list || []).map((user) => [Number(idOf(user)), user.nickName || '—']))
    }
  } catch (error) {
    showError(error)
  }
}
async function loadDepartments() {
  try {
    const result = await request({ url: `${base}/departments`, method: 'get' })
    departments.value = Array.isArray(result) ? result.filter((department) => typeof department === 'string' && department.trim()) : []
  } catch (error) {
    showError(error)
  }
}
function suggestDepartments(queryString, callback) {
  const keyword = queryString.trim().toLocaleLowerCase()
  callback(departments.value
    .filter((department) => !keyword || department.toLocaleLowerCase().includes(keyword))
    .map((value) => ({ value })))
}
function suggestQuickMaterials(queryString, callback) {
  const keyword = queryString.trim().toLocaleLowerCase()
  callback(materials.value
    .filter((item) => [item.code, item.name, item.mnemonic].some((value) => String(value || '').toLocaleLowerCase().includes(keyword)))
    .slice(0, 20)
    .map((item) => ({ ...item, value: `${item.code} ${item.name}` })))
}
async function quickAddMaterial(item) {
  quickMaterialSearch.value = ''
  await appendMaterials([item])
}
function openQuickMaterialCreate() {
  Object.assign(quickMaterialForm, { name: quickMaterialSearch.value.trim(), code: '', mnemonic: '', unitId: undefined })
  quickMaterialDialogOpen.value = true
}
async function createQuickMaterial() {
  if (quickMaterialSaving.value || !canEdit.value) return
  const payload = {
    name: quickMaterialForm.name.trim(),
    code: quickMaterialForm.code.trim(),
    mnemonic: quickMaterialForm.mnemonic.trim(),
    units: [{ unitId: Number(quickMaterialForm.unitId), factor: '1', isBase: true, enabled: true }]
  }
  if (!payload.name || !payload.code || !payload.mnemonic || !payload.units[0].unitId) {
    ElMessage.error('请填写物料名称、编号、助记符并选择基本单位')
    return
  }
  quickMaterialSaving.value = true
  try {
    const created = await saveMaster('materials', payload)
    const bindings = (created.units || []).filter((binding) => binding.enabled)
    const baseBinding = bindings.find((binding) => binding.isBase)
    if (!idOf(created) || !baseBinding) throw new Error('物料已创建，但服务端未返回有效的基本单位')
    const loaded = { material: created, bindings, baseBinding }
    materials.value = [created, ...materials.value.filter((item) => idOf(item) !== idOf(created))]
    rowsLoading.value = true
    const firstAppended = appendLoadedMaterials([loaded])
    quickMaterialDialogOpen.value = false
    quickMaterialSearch.value = ''
    ElMessage.success('物料创建并添加成功')
    if (firstAppended) await focusCell(firstAppended, 'unit')
  } catch (error) {
    showError(error)
  } finally {
    rowsLoading.value = false
    quickMaterialSaving.value = false
  }
}
const detailRoute = computed(() => route.query.whDelivery === '1' && (route.query.whId || route.query.whNew === '1'))
function listRouteQuery() {
  const query = { ...route.query }
  delete query.whDelivery; delete query.whId; delete query.whNew
  return query
}
function enterDetail(id) {
  const query = { ...route.query, whDelivery: '1' }
  if (id) {
    query.whId = String(id)
    delete query.whNew
  } else {
    delete query.whId
    query.whNew = '1'
  }
  return router.push({ name: route.name, query })
}
const unitNames = computed(() => new Map(units.value.map((item) => [idOf(item), item.name])))
const categoryTree = computed(() => {
  const nodes = new Map(categories.value.map((item) => [idOf(item), { id: idOf(item), label: `${item.code} ${item.name}`, children: [] }]))
  const roots = []
  for (const item of categories.value) {
    const node = nodes.get(idOf(item))
    const parent = nodes.get(item.parentId)
    if (parent && parent !== node) parent.children.push(node)
    else roots.push(node)
  }
  return roots
})
const lineCategoryIds = computed(() => {
  if (!lineCategoryId.value) return null
  const ids = new Set([lineCategoryId.value])
  let changed = true
  while (changed) {
    changed = false
    for (const category of categories.value) {
      if (ids.has(category.parentId) && !ids.has(idOf(category))) { ids.add(idOf(category)); changed = true }
    }
  }
  return ids
})
function changeDetailSort({ prop, order }) {
  detailSortBy.value = order ? prop : ''
  detailSortOrder.value = order === 'ascending' ? 'asc' : order === 'descending' ? 'desc' : ''
}
function decimalSortValue(value) {
  const match = /^([+-]?)(\d+)(?:\.(\d+))?$/.exec(String(value).trim())
  if (!match) return null
  const integer = match[2].replace(/^0+/, '') || '0'
  const fraction = (match[3] || '').replace(/0+$/, '')
  return { negative: match[1] === '-' && (integer !== '0' || fraction !== ''), integer, fraction }
}
function compareDecimalSortValues(left, right) {
  if (left.negative !== right.negative) return left.negative ? -1 : 1
  let order = left.integer.length === right.integer.length
    ? left.integer < right.integer ? -1 : left.integer > right.integer ? 1 : 0
    : left.integer.length < right.integer.length ? -1 : 1
  if (!order) {
    const scale = Math.max(left.fraction.length, right.fraction.length)
    const a = left.fraction.padEnd(scale, '0')
    const b = right.fraction.padEnd(scale, '0')
    order = a < b ? -1 : a > b ? 1 : 0
  }
  return left.negative ? -order : order
}
function compareDetailValues(left, right) {
  const leftDecimal = decimalSortValue(left)
  const rightDecimal = decimalSortValue(right)
  if (leftDecimal && rightDecimal) return compareDecimalSortValues(leftDecimal, rightDecimal)
  return String(left).localeCompare(String(right), 'zh-CN', { numeric: true, sensitivity: 'base' })
}
function tableSummary({ columns }) {
  return columns.map((column, index) => {
    if (index === 0) return '合计'
    if (column.property === 'amount') return note.value.totalAmount || '保存后计算'
    return ''
  })
}

function shanghaiInput(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23' }).format(date).replace(' ', 'T')
}
function displayTime(value) { return shanghaiInput(value).replace('T', ' ') || '—' }
function displayDate(value) { return shanghaiInput(value).slice(0, 10) || '—' }
function signature() {
  return JSON.stringify({ brandId: form.brandId, department: form.department, deliveredAt: form.deliveredAt, details: rows.value.map((row) => [row.id, row.materialId, row.materialUnitId, row.quantity, row.unitPrice]) })
}
function emptyRow() {
  return { rowKey: `delivery-row-${++nextRowKey}`, id: undefined, materialId: undefined, materialUnitId: undefined, materialCode: '', materialName: '', specification: '', unitName: '', quantity: '', unitPrice: '', units: [], unitsLoaded: false, keyword: '', errors: {}, loading: false, composing: false, materialOpen: false, unitOpen: false }
}
function isEmptyRow(row) { return !row.materialId && !row.materialUnitId && !row.quantity.trim() && !row.unitPrice.trim() }
function inlineMaterials(row) {
  const keyword = row.keyword.trim().toLocaleLowerCase()
  return materials.value.filter((item) => (!lineCategoryIds.value || lineCategoryIds.value.has(item.categoryId)) &&
    (!keyword || [item.code, item.name, item.mnemonic].some((value) => String(value || '').toLocaleLowerCase().includes(keyword))))
}
function bindingLabel(binding) {
  return `${binding.name || unitNames.value.get(binding.unitId) || `单位 #${binding.unitId}`}${binding.isBase ? '（基本）' : ''}${binding.enabled ? '' : '（历史/停用）'}`
}
async function loadOptions() {
  if (optionsLoading.value) return
  optionsLoading.value = true
  try {
    const [brandRows, categoryRows, materialRows, unitRows] = await Promise.all(['brands', 'categories', 'materials', 'units'].map(allOptions))
    brands.value = brandRows; categories.value = categoryRows; materials.value = materialRows; units.value = unitRows
    optionsReady.value = true
  } catch (error) { showError(error) } finally { optionsLoading.value = false }
}
function changeSort({ prop, order }) {
  sortBy.value = order ? prop : ''
  sortOrder.value = order === 'ascending' ? 'asc' : order === 'descending' ? 'desc' : ''
  page.value = 1
  loadList()
}
function saveColumnWidth(section, width, column) {
  const key = column.property
  const registry = section === 'list' ? listFieldMap : detailFieldMap
  if (!registry.has(key) || !Number.isFinite(width)) return
  const actor = idOf(userStore.userInfo)
  widthSaveQueue = widthSaveQueue.then(async () => {
    if (idOf(userStore.userInfo) !== actor) return
    const settings = normalizeDisplaySettings(userStore.userInfo.originSetting?.[DISPLAY_SETTING_KEY])
    settings[section].widths[key] = Math.max(80, Math.min(800, Math.round(width)))
    const response = await setSelfSetting({ [DISPLAY_SETTING_KEY]: settings })
    if (response?.code !== 0) throw new Error(response?.msg || '保存列宽失败')
    if (idOf(userStore.userInfo) !== actor) return
    userStore.ResetUserInfo({ originSetting: { ...(userStore.userInfo.originSetting || {}), [DISPLAY_SETTING_KEY]: settings } })
  }).catch(showError)
  return widthSaveQueue
}
function onHeaderDragEnd(newWidth, oldWidth, column) {
  if (listFieldMap.has(column.property)) return saveColumnWidth('list', newWidth, column)
  return headerDragEnd(tableWidthKey, newWidth, oldWidth, column)
}
function onDetailHeaderDragEnd(newWidth, _oldWidth, column) {
  return saveColumnWidth('detail', newWidth, column)
}
async function loadList() {
  const sequence = ++listSequence
  listLoading.value = true
  listError.value = ''
  const params = { page: page.value, pageSize: pageSize.value, number: filters.number.trim() || undefined, brandId: filters.brandId || undefined, status: filters.status || undefined, department: filters.department.trim() || undefined, ...(sortBy.value ? { sortBy: sortBy.value, sortOrder: sortOrder.value } : {}) }
  if (filters.dates?.length === 2) {
    params.deliveredFrom = `${filters.dates[0]}T00:00:00+08:00`
    params.deliveredTo = new Date(new Date(`${filters.dates[1]}T00:00:00+08:00`).getTime() + 86400000).toISOString()
  }
  try {
    const data = await request({ url: `${base}/list`, method: 'get', params })
    if (sequence !== listSequence) return
    notes.value = data.list || []; total.value = data.total; page.value = data.page; pageSize.value = data.pageSize
    selectedNotes.value = []
    listTable.value?.clearSelection?.()
  } catch (error) {
    if (sequence === listSequence) { listError.value = error.message || '送货单列表加载失败'; showError(error) }
  } finally { if (sequence === listSequence) listLoading.value = false }
}
function search() { page.value = 1; loadList() }
function resetSearch() {
  Object.assign(filters, { number: '', dates: getDefaultDateRange(displaySettings.value.dateDefault), brandId: undefined, status: '', department: '' })
  search()
}
function resetEditor() {
  note.value = {}; Object.assign(reservation, { number: '', token: '' }); rows.value = []; cellRefs.clear()
  reservationError.value = ''; conflictMessage.value = ''; lineCategoryId.value = undefined
  Object.assign(form, { brandId: undefined, department: '', deliveredAt: shanghaiInput(new Date()).slice(0, 10) })
}
async function openNew() {
  if (editorLoading.value || saving.value || actionLoading.value || rowsLoading.value || reserving.value) return
  if (detailRoute.value && dirty.value && !(await confirm('新增单据将放弃当前送货单的未保存修改，确定继续？', '新建送货单'))) return
  return enterDetail(null)
}
async function copyDocument() {
  if (!idOf(note.value) || !readonly.value || editorLoading.value || saving.value || actionLoading.value || rowsLoading.value || reserving.value) return
  if (dirty.value && !(await confirm('复制将使用当前页面内容；原单未保存的修改不会写回。确定继续？', '复制单据'))) return
  const copiedRows = rows.value.filter((row) => !isEmptyRow(row)).map((row) => ({
    ...emptyRow(),
    materialId: row.materialId,
    materialUnitId: row.materialUnitId,
    materialCode: row.materialCode,
    materialName: row.materialName,
    specification: row.specification,
    unitName: row.unitName,
    quantity: String(row.quantity),
    unitPrice: String(row.unitPrice),
    units: row.units.map((unit) => ({ ...unit })),
    unitsLoaded: false
  }))
  try {
    sessionStorage.setItem('warehouse.delivery.createDraft', JSON.stringify({
      form: { ...form },
      reservation: {},
      rows: copiedRows.length ? copiedRows : [emptyRow()]
    }))
  } catch (error) {
    showError(error)
    return
  }
  return enterDetail(null)
}
async function initializeNew() {
  resetEditor()
  const draft = readCreateDraft()
  if (draft) {
    Object.assign(form, draft.form)
    Object.assign(reservation, draft.reservation || {})
    rows.value = draft.rows?.length ? draft.rows : [emptyRow()]
  } else {
    rows.value = [emptyRow()]
  }
  editorOpen.value = true
  savedSignature.value = signature()
  if (!reservation.token) await reserveNumber()
}
function readCreateDraft() {
  try { return JSON.parse(sessionStorage.getItem('warehouse.delivery.createDraft') || 'null') } catch { return null }
}
function persistCreateDraft() {
  if (idOf(note.value) || !editorOpen.value) return
  sessionStorage.setItem('warehouse.delivery.createDraft', JSON.stringify({ form: { ...form }, reservation: { ...reservation }, rows: rows.value }))
}
function clearCreateDraft() { sessionStorage.removeItem('warehouse.delivery.createDraft') }
async function reserveNumber() {
  if (reserving.value || reservation.token || idOf(note.value)) return
  reserving.value = true; reservationError.value = ''
  try {
    const data = await request({ url: `${base}/reserveNumber`, method: 'post' })
    if (!data.number || !data.token) throw new Error('服务端未返回有效的正式单号和预占凭证')
    Object.assign(reservation, { number: data.number, token: data.token })
  } catch (error) { reservationError.value = error.message || '正式单号预占失败'; showError(error) } finally { reserving.value = false }
}
async function readAggregate(data) {
  // The current Go aggregate serializes Note/Details; accept tagged JSON as well.
  const current = data.note ?? data.Note
  const details = data.details ?? data.Details
  if (!current || !Array.isArray(details)) throw new Error('送货单详情结构无效')
  note.value = current; form.brandId = current.brandId; form.department = current.department || ''; form.deliveredAt = shanghaiInput(current.deliveredAt).slice(0, 10)
  rows.value = details.map((detail) => ({ ...emptyRow(), id: idOf(detail), materialId: detail.materialId, materialUnitId: detail.materialUnitId, materialCode: detail.materialCodeSnapshot, materialName: detail.materialNameSnapshot, specification: detail.specificationSnapshot || '', unitName: detail.unitNameSnapshot, quantity: String(detail.quantity), unitPrice: String(detail.unitPrice), amount: detail.amount, units: [{ ID: detail.materialUnitId, unitId: detail.unitId, factor: detail.factor, name: detail.unitNameSnapshot, enabled: false, isBase: detail.unitId === detail.baseUnitId }] }))
  if (!readonly.value && !rows.value.length) rows.value.push(emptyRow())
  savedSignature.value = signature(); conflictMessage.value = ''
}
async function loadRowUnits(row) {
  row.unitOpen = true
  if (readonly.value || !row.materialId || row.unitsLoaded || row.loading) return
  const materialId = row.materialId
  row.loading = true
  rowsLoading.value = true
  try {
    let pending = materialUnitsCache.get(materialId)
    if (!pending) {
      pending = request({ url: '/warehouse/master/materials/detail', method: 'get', params: { id: materialId } })
        .then((material) => (material.units || []).filter((binding) => binding.enabled).map((binding) => ({ ...binding })))
      materialUnitsCache.set(materialId, pending)
    }
    const bindings = (await pending).map((binding) => ({ ...binding }))
    if (row.materialId !== materialId) return
    const historical = row.units.find((binding) => idOf(binding) === row.materialUnitId) || row.units[0]
    const matching = bindings.find((binding) => idOf(binding) === row.materialUnitId)
    if (matching) matching.name = row.unitName
    else if (historical) bindings.unshift({ ...historical })
    row.units = bindings
    row.unitsLoaded = true
  } catch {
    materialUnitsCache.delete(materialId)
    // Keep historical labels/bindings if current master data is gone or disabled.
  } finally {
    row.loading = false
    rowsLoading.value = rows.value.some((item) => item.loading)
  }
}
async function initializeExisting(row) {
  if (editorLoading.value) return
  resetEditor(); editorOpen.value = true; editorLoading.value = true; savedSignature.value = signature()
  try { await readAggregate(await request({ url: `${base}/detail`, method: 'get', params: { id: idOf(row) } })) }
  catch (error) { editorOpen.value = false; showError(error) }
  finally { editorLoading.value = false }
}
function openExisting(row) { return enterDetail(idOf(row)) }
async function reloadEditor() {
  if (!idOf(note.value) || !(await confirm('将丢弃当前表单修改并重新读取最新单据，确定继续？'))) return
  await initializeExisting(note.value)
}
async function navigateDocument(direction) {
  if (!idOf(note.value) || editorLoading.value || saving.value || actionLoading.value || rowsLoading.value) return
  if (dirty.value && !(await confirm('切换单据将丢弃当前未保存的修改，确定继续？', '切换送货单'))) return
  const target = notes.value[detailListIndex.value + direction]
  if (!target) return
  const query = { ...route.query, whDelivery: '1', whId: String(idOf(target)) }
  delete query.whNew
  await router.push({ name: route.name, query })
}
async function confirm(message, title = '操作确认') {
  try { await ElMessageBox.confirm(message, title, { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' }); return true }
  catch { return false }
}
async function beforeClose(done) {
  if (saving.value || actionLoading.value || rowsLoading.value || reserving.value || editorLoading.value) return
  if (!dirty.value || await confirm('有未保存的修改，关闭将丢失这些修改。确定关闭？')) { pickerOpen.value = false; done() }
}
function closeEditor() {
  beforeClose(() => router.push({ name: route.name, query: listRouteQuery() }))
}
function changeBrand(brandId) {
  form.brandId = brandId
}
function filterBrands(query) {
  customerSearch.value = query
}
function handleBrandVisibility(visible) {
  if (!visible) customerSearch.value = ''
}
function setCellRef(row, field, element) {
  const key = `${row.rowKey}:${field}`
  if (element) cellRefs.set(key, element)
  else cellRefs.delete(key)
}
async function focusCell(row, field) {
  await nextTick()
  cellRefs.get(`${row.rowKey}:${field}`)?.focus?.()
}
function addBlankRow() { const row = emptyRow(); rows.value.push(row); focusCell(row, 'material') }
function removeRow(row) {
  rows.value = rows.value.filter((item) => item.rowKey !== row.rowKey)
  if (!rows.value.length) rows.value.push(emptyRow())
}
async function currentMaterial(id) {
  const material = await request({ url: '/warehouse/master/materials/detail', method: 'get', params: { id } })
  if (!material) throw new Error('所选物料不存在')
  const bindings = (material.units || []).filter((binding) => binding.enabled)
  const baseBinding = bindings.find((binding) => binding.isBase)
  if (!baseBinding) throw new Error(`物料“${material.name}”没有可用的基本单位`)
  return { material, bindings, baseBinding }
}
function applyMaterial(row, { material, bindings, baseBinding }) {
  Object.assign(row, { materialId: idOf(material), materialUnitId: idOf(baseBinding), materialCode: material.code, materialName: material.name, specification: material.specification || '', unitName: unitNames.value.get(baseBinding.unitId) || '', units: bindings, unitsLoaded: true, keyword: '', errors: {}, amount: undefined })
}
async function selectMaterial(row, materialId) {
  if (!canEdit.value || row.loading) return
  row.loading = true; rowsLoading.value = true
  let selected = false
  try { applyMaterial(row, await currentMaterial(materialId)); selected = true }
  catch (error) { row.errors.materialId = error.message; showError(error) }
  finally { row.loading = false; rowsLoading.value = false }
  if (selected) await focusCell(row, 'unit')
}
function selectUnit(row, materialUnitId) {
  row.materialUnitId = materialUnitId
  const binding = row.units.find((item) => idOf(item) === materialUnitId)
  row.unitName = binding?.name || unitNames.value.get(binding?.unitId) || ''
  row.amount = undefined; delete row.errors.materialUnitId; focusCell(row, 'quantity')
}
async function appendMaterials(selected) {
  if (!canEdit.value) return
  rowsLoading.value = true
  let firstAppended
  try {
    const loaded = await Promise.all(selected.map((item) => currentMaterial(idOf(item))))
    firstAppended = appendLoadedMaterials(loaded)
  } catch (error) { showError(error) } finally { rowsLoading.value = false }
  if (firstAppended) await focusCell(firstAppended, 'unit')
}
function appendLoadedMaterials(loaded) {
  const appended = loaded.map((data) => { const row = emptyRow(); applyMaterial(row, data); return row })
  while (rows.value.length && isEmptyRow(rows.value.at(-1))) rows.value.pop()
  rows.value.push(...appended)
  pickerOpen.value = false
  return appended[0]
}
function clearFieldError(row, field) { delete row.errors[field]; row.amount = undefined }
function validateDecimal(row, field) {
  const value = row[field].trim()
  const label = field === 'quantity' ? '数量' : '单价'
  let message = ''
  if (!value) message = `请输入${label}`
  else if (!/^(0|[1-9]\d{0,13})(\.\d{1,4})?$/.test(value)) message = `${label}须为普通十进制数，整数最多14位、小数最多4位`
  else if (field === 'quantity' && !/[1-9]/.test(value)) message = '数量必须大于0'
  if (message) row.errors[field] = message
  else delete row.errors[field]
  return !message
}
function cellKeydown(event, row, field) {
  if (!canEdit.value || event.isComposing || row.composing || event.keyCode === 229 || event.repeat || event.ctrlKey || event.altKey || event.metaKey) return
  if (!['Enter', 'Tab'].includes(event.key)) return
  if ((field === 'material' && row.materialOpen) || (field === 'unit' && row.unitOpen)) return
  const fields = editingFields.value
  const index = fields.indexOf(field)
  const orderedRows = displayRows.value
  const rowIndex = orderedRows.findIndex((item) => item.rowKey === row.rowKey)
  if (event.shiftKey) {
    if (event.key !== 'Tab') return
    const target = index > 0 ? row : orderedRows[rowIndex - 1]
    if (!target) return
    event.preventDefault(); focusCell(target, fields[index > 0 ? index - 1 : fields.length - 1]); return
  }
  // Enter on a closed select belongs to the select; only its confirmation moves focus.
  if ((field === 'material' || field === 'unit') && event.key === 'Enter') return
  if (field === 'material' && !row.materialId) return
  if (field === 'unit' && !row.materialUnitId) return
  if ((field === 'quantity' || field === 'unitPrice') && !validateDecimal(row, field)) { event.preventDefault(); focusCell(row, field); return }
  event.preventDefault()
  if (index < fields.length - 1) focusCell(row, fields[index + 1])
  else {
    let target = orderedRows[rowIndex + 1]
    if (!target) { target = emptyRow(); rows.value.push(target) }
    focusCell(target, fields[0])
  }
}
function validatedDetails() {
  const submitted = [...rows.value]
  while (submitted.length && isEmptyRow(submitted.at(-1))) submitted.pop()
  if (!submitted.length) { ElMessage.error('请至少录入一行完整明细'); return null }
  let firstInvalid
  const details = submitted.map((row) => {
    row.errors = {}
    if (!row.materialId) row.errors.materialId = '请选择物料'
    if (!row.materialUnitId) row.errors.materialUnitId = '请选择物料单位'
    validateDecimal(row, 'quantity'); validateDecimal(row, 'unitPrice')
    if (!firstInvalid && Object.keys(row.errors).length) firstInvalid = { row, field: Object.keys(row.errors)[0] }
    return { ...(row.id ? { id: row.id } : {}), materialId: row.materialId, materialUnitId: row.materialUnitId, quantity: row.quantity.trim(), unitPrice: row.unitPrice.trim() }
  })
  if (firstInvalid) { ElMessage.error('请补齐或修正标红的明细行'); focusCell(firstInvalid.row, firstInvalid.field === 'materialId' ? 'material' : firstInvalid.field === 'materialUnitId' ? 'unit' : firstInvalid.field); return null }
  return details
}
function reportEditorError(error) {
  const business = error.business || error.data || {}
  if (business.errorCode === 'VERSION_CONFLICT' || business.errorCode === 'STATE_CONFLICT') conflictMessage.value = `${error.message || '单据已被修改'}。请读取最新版本后再操作。`
  if (business.lineNo) {
    const row = rows.value[business.lineNo - 1]
    if (row) {
      const field = ['materialId', 'materialUnitId', 'quantity', 'unitPrice'].includes(business.field) ? business.field : 'materialId'
      row.errors[field] = error.message
    }
  }
  showError(error)
}
async function save(continueNew = false) {
  if (!canEdit.value || (!idOf(note.value) && !reservation.token)) return
  if (!form.brandId || !form.deliveredAt) { ElMessage.error('请选择客户和送货日期'); return }
  const details = validatedDetails()
  if (!details) return
  const id = idOf(note.value)
  const data = { ...(id ? { id, version: note.value.version } : { reservationToken: reservation.token }), brandId: form.brandId, department: form.department.trim(), deliveredAt: `${form.deliveredAt}T00:00:00+08:00`, details }
  saving.value = true
  try {
    await request({ url: `${base}/${id ? 'update' : 'create'}`, method: id ? 'put' : 'post', data })
    clearCreateDraft()
    if (continueNew) {
      const query = { ...route.query, whDelivery: '1', whNew: '1' }
      delete query.whId
      if (route.query.whNew === '1' && !route.query.whId) {
        await initializeNew()
      } else {
        editorOpen.value = false
        await router.push({ name: route.name, query })
      }
    } else {
      editorOpen.value = false
      await router.push({ name: route.name, query: listRouteQuery() })
    }
    ElMessage.success(continueNew ? '保存成功，可继续新增送货单' : '送货单保存成功')
    await loadList()
  } catch (error) { reportEditorError(error) } finally { saving.value = false }
}
async function transition(row, action) {
  if (actionLoading.value) return
  const label = { review: '审核', unreview: '反审核', delete: '删除' }[action]
  const active = editorOpen.value && idOf(note.value) === idOf(row)
  if (active && action === 'review' && dirty.value) { ElMessage.warning('请先保存修改，再审核'); return }
  const warning = active && dirty.value ? '当前未保存的修改也将被丢弃。' : ''
  if (!(await confirm(`确定${label}送货单“${row.number}”？${warning}`, `${label}确认`))) return
  actionLoading.value = true
  try {
    const result = await request({ url: `${base}/${action}`, method: action === 'delete' ? 'delete' : 'post', data: { id: idOf(row), version: row.version } })
    if (active) {
      if (action === 'delete') {
        editorOpen.value = false
        await router.push({ name: route.name, query: listRouteQuery() })
      }
      else {
        note.value = result
        if (action === 'unreview') await readAggregate(await request({ url: `${base}/detail`, method: 'get', params: { id: idOf(row) } }))
      }
    }
    ElMessage.success(`${label}成功`)
    if (action === 'delete' && notes.value.length === 1 && page.value > 1) page.value--
    await loadList()
  } catch (error) { if (active) reportEditorError(error); else showError(error); await loadList() }
  finally { actionLoading.value = false }
}
watch([form, rows, reservation], persistCreateDraft, { deep: true })
onBeforeUnmount(persistCreateDraft)
watch(() => displaySettings.value.dateDefault, (preset) => {
  filters.dates = getDefaultDateRange(preset)
}, { immediate: true })
watch(() => [...displaySettings.value.filters.basic, ...displaySettings.value.filters.more], (visible) => {
  const empty = { number: '', department: '', status: '', brandId: undefined, dates: [] }
  for (const key of Object.keys(empty)) if (!visible.includes(key)) filters[key] = empty[key]
}, { immediate: true })
watch(() => displaySettings.value.list.columns, (columns) => {
  selectedNotes.value = []
  if (sortBy.value && !columns.includes(sortBy.value)) { sortBy.value = ''; sortOrder.value = '' }
})
let activatedOnce = false
onActivated(() => {
  if (activatedOnce && !detailRoute.value) loadList()
  activatedOnce = true
})
watch(() => idOf(userStore.userInfo), () => {
  Object.assign(filters, { number: '', dates: getDefaultDateRange(displaySettings.value.dateDefault), brandId: undefined, status: '', department: '' })
  moreFiltersOpen.value = false
  notes.value = []; selectedNotes.value = []; total.value = 0
  loadColumnWidths(tableWidthKey)
  if (idOf(userStore.userInfo) && !detailRoute.value) search()
})
onMounted(() => { loadList(); loadOptions(); loadDepartments(); loadUsers(); loadColumnWidths(tableWidthKey) })
watch(() => [route.query.whDelivery, route.query.whId, route.query.whNew], async ([detail, id, isNew]) => {
  if (detail !== '1' || (!id && isNew !== '1')) {
    editorOpen.value = false
    return
  }
  if (id) await initializeExisting({ ID: Number(id) })
  else await initializeNew()
}, { immediate: true })
</script>

<style scoped>
.brand-select { width: 230px; }
.status-select { width: 125px; }
 .delivery-detail-page { display: flex; flex-direction: column; height: calc(100vh - 150px); min-height: 320px; overflow: hidden; }
 .delivery-detail-heading { display: flex; align-items: center; gap: 12px; flex: 0 0 auto; padding: 0 0 12px; }
 .delivery-detail-heading h2 { margin: 0; font-size: 20px; font-weight: 600; }
 .delivery-detail-scroll { flex: 1 1 auto; min-height: 0; overflow: auto; padding: 0 4px 16px; }
 .delivery-detail-footer { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex: 0 0 auto; padding: 12px 0 0; border-top: 1px solid var(--el-border-color); background: var(--el-bg-color); }
 .delivery-detail-actions { display: flex; justify-content: flex-end; align-items: center; flex-wrap: wrap; gap: 8px; }
 .delivery-detail-actions :deep(.el-button + .el-button) { margin-left: 0; }
 .delivery-editor { min-height: 240px; }
 .editor-header { margin-top: 20px; }
 .editor-tools { display: flex; align-items: center; flex-wrap: wrap; gap: 12px; margin-bottom: 16px; }
 .editor-tools :deep(.el-select) { min-width: 240px; }
 .editor-hint { font-size: 12px; color: var(--el-text-color-secondary); }
 .cell-error { color: var(--el-color-danger); font-size: 12px; line-height: 1.5; margin-top: 4px; }
 .option-spec { margin-left: 12px; color: var(--el-text-color-secondary); }
 .detail-table :deep(.el-select) { width: 100%; }
 .delivery-row-actions { display: flex; align-items: center; gap: 6px; white-space: nowrap; }
 .delivery-row-actions :deep(.el-button + .el-button) { margin-left: 0; }
 .delivery-editor > .el-alert { margin-bottom: 12px; }
</style>
