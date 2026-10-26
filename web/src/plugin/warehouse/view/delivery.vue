<template>
  <div class="delivery-page py-2">
    <div v-show="!detailRoute">
      <el-form inline class="delivery-filters" @submit.prevent="search">
        <template v-for="group in filterGroups" :key="group.key">
          <div v-show="group.key === 'basic' || moreFiltersOpen" :data-filter-group="group.key">
            <el-form-item v-for="field in group.fields" :key="field.key" :label="field.label" :data-filter-key="field.key">
              <el-input v-if="field.key === 'number'" v-model="filters.number" clearable placeholder="完整送货单号" @keyup.enter="search" />
              <el-date-picker v-else-if="field.key === 'dates'" v-model="filters.dates" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期" />
              <el-select v-else-if="field.key === 'brandId'" v-model="filters.brandId" clearable filterable default-first-option automatic-dropdown :filter-method="(value) => listCustomerSearch = value" class="brand-select" @visible-change="listCustomerSearch = ''">
                <el-option v-for="brand in filteredListBrands" :key="idOf(brand)" :value="idOf(brand)" :label="brand.name"><div class="delivery-option-main"><span>{{ brand.name }}</span><small>{{ brand.code }}</small></div></el-option>
              </el-select>
              <MaterialFilter v-else-if="field.key === 'materialIds'" v-model="filters.materialIds" :materials="materials" />
              <el-input v-else-if="field.key === 'department'" v-model="filters.department" clearable placeholder="输入部门" @keyup.enter="search" />
              <el-input v-else-if="field.key === 'detailRemark'" v-model="filters.detailRemark" clearable placeholder="输入明细备注关键词" @keyup.enter="search" />
              <el-select v-else-if="field.key === 'status'" v-model="filters.status" clearable class="status-select"><el-option label="未审核" value="draft" /><el-option label="已审核" value="reviewed" /><el-option label="已复审" value="reReviewed" /></el-select>
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
        <el-button data-action="review-selected" type="success" :disabled="!selectedDrafts.length || actionLoading" :loading="batchLoading === 'review'" @click="batchTransition('review')">审核</el-button>
        <el-button data-action="unreview-selected" type="warning" :disabled="!selectedReviewed.length || actionLoading" :loading="batchLoading === 'unreview'" @click="batchTransition('unreview')">反审核</el-button>
        <el-button data-action="rereview-selected" type="success" :disabled="!selectedReviewed.length || actionLoading" :loading="batchLoading === 'reReview'" @click="openReReview(selectedReviewed, true)">复审</el-button>
        <el-button data-action="cancel-rereview-selected" type="warning" :disabled="!selectedReReviewed.length || actionLoading" :loading="batchLoading === 'cancelReReview'" @click="batchTransition('cancelReReview')">取消复审</el-button>
        <el-button data-action="delete-selected" type="danger" :disabled="!selectedDrafts.length || actionLoading" :loading="batchLoading === 'delete'" @click="batchTransition('delete')">删除</el-button>
        <el-button @click="router.push({ name: 'warehouseDeliveryDisplaySettings' })">单据展示设置</el-button>
      </div>
      <el-alert v-if="listError" :title="listError" type="error" show-icon :closable="false" />
      <el-table :key="`${displaySettings.list.columns.join(',')}:${displaySettings.list.showActions}`" ref="listTable" v-loading="listLoading" :data="notes" :row-key="idOf" border size="small" :default-sort="sortBy ? { prop: sortBy, order: sortOrder === 'asc' ? 'ascending' : 'descending' } : undefined" @sort-change="changeSort" @header-dragend="onHeaderDragEnd" @selection-change="selectionChanged" @row-dblclick="openExisting($event)">
        <el-table-column type="selection" column-key="__selection" :width="columnWidths.__selection || 48" :selectable="selectableNote" />
        <el-table-column type="index" column-key="__index" label="序号" :width="columnWidths.__index || 65" align="center" :index="pageIndex" />
        <el-table-column v-for="column in orderedDisplayColumns" :key="column.key" :prop="column.key" :label="column.label" :width="displaySettings.list.widths[column.key] || column.width" min-width="80" :align="column.key === 'totalAmount' ? 'center' : 'left'" sortable="custom">
          <template #default="{ row }">
            <el-tag v-if="column.key === 'status'" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag>
            <span v-else-if="column.key === 'deliveredAt'">{{ displayDate(row.deliveredAt) }}</span>
            <span v-else-if="column.key === 'createdAt'">{{ displayTime(row.CreatedAt ?? row.createdAt) }}</span>
            <span v-else-if="column.key === 'updatedAt'">{{ displayTime(row.updatedAt ?? row.UpdatedAt) }}</span>
            <span v-else-if="column.key === 'reviewedAt'">{{ displayTime(row.reviewedAt) }}</span>
            <span v-else-if="['createdByNameSnapshot', 'updatedByName', 'reviewedByNameSnapshot'].includes(column.key)">{{ listUserName(row, column.key) }}</span>
            <span v-else>{{ row[column.key] }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="displaySettings.list.showActions" column-key="__actions" label="操作" fixed="right" align="right" :width="columnWidths.__actions || 210"><template #default="{ row }">
          <div class="delivery-row-actions">
            <el-button link size="small" type="primary" :disabled="editorLoading || actionLoading" @click="openExisting(row)">{{ row.status !== 'draft' ? '查看' : '编辑' }}</el-button>
            <el-button v-if="row.status === 'draft'" link size="small" type="success" :disabled="actionLoading" @click="transition(row, 'review')">审核</el-button>
            <el-button v-if="row.status === 'reviewed'" link size="small" type="warning" :disabled="actionLoading" @click="transition(row, 'unreview')">反审核</el-button>
            <el-button v-if="row.status === 'reviewed'" link size="small" type="success" :disabled="actionLoading" @click="openReReview([row], true)">复审</el-button>
            <el-button v-if="row.status === 'reReviewed'" link size="small" type="warning" :disabled="actionLoading" @click="transition(row, 'cancelReReview')">取消复审</el-button>
            <el-button v-if="row.status === 'draft'" link size="small" type="danger" :disabled="actionLoading" @click="transition(row, 'delete')">删除</el-button>
          </div>
        </template></el-table-column>
      </el-table>
      <div class="gva-pagination"><el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[20, 50, 100]" :total="total" layout="total, sizes, prev, pager, next, jumper" @current-change="loadList" @size-change="search" /></div>
    </div>
    

    <div v-if="detailRoute" v-loading="editorLoading" class="delivery-detail-page">
      <header class="delivery-detail-heading">
        <h2>{{ idOf(note) ? '送货单详情' : '新建送货单' }}</h2>
        <el-tag v-if="idOf(note)" :type="statusType(note.status)">{{ statusLabel(note.status) }}</el-tag>
        <el-tag v-else type="info">未保存</el-tag>
      </header>
      <div class="delivery-detail-scroll">
      <div class="delivery-editor">
        <el-alert v-if="reservationError" :title="reservationError" type="error" :closable="false" show-icon><template #default><el-button :loading="reserving" @click="reserveNumber">重新预占正式单号</el-button></template></el-alert>
        <el-alert v-if="conflictMessage" :title="conflictMessage" type="error" :closable="false" show-icon><template #default>当前表单未被覆盖。<el-button link type="primary" @click="reloadEditor">放弃本地修改并读取最新版本</el-button></template></el-alert>
        <el-alert v-if="readonly" :title="note.status === 'reReviewed' ? '已复审送货单只读。请先取消复审，再反审核后修改。' : '已审核送货单只读。修改前请单独执行反审核。'" type="info" :closable="false" />
        <el-form inline class="editor-header" :disabled="readonly || saving || actionLoading || editorLoading" @keydown.capture="headerKeydown">
          <el-form-item label="正式单号"><strong>{{ note.number || reservation.number || (reserving ? '正在预占正式单号…' : '尚未取得正式单号') }}</strong></el-form-item>
          <el-form-item label="客户" required>
            <el-select ref="brandSelect" :model-value="form.brandId" filterable default-first-option automatic-dropdown :filter-method="filterBrands" :disabled="rowsLoading" class="brand-select" @focus="openSelect(brandSelect)" @change="changeBrand" @visible-change="handleBrandVisibility">
              <el-option v-if="form.brandId && !brands.some((brand) => idOf(brand) === form.brandId)" :value="form.brandId" :label="note.brandNameSnapshot || `历史客户 #${form.brandId}`" />
              <el-option v-for="brand in filteredBrands" :key="idOf(brand)" :value="idOf(brand)" :label="brand.name"><div class="delivery-option-main"><span>{{ brand.name }}</span><small>{{ brand.code }}</small></div></el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="部门"><el-autocomplete v-model="form.department" :fetch-suggestions="suggestDepartments" :trigger-on-focus="true" highlight-first-item maxlength="100" placeholder="请输入部门" /></el-form-item>
          <el-form-item label="送货日期" required><div @keydown.capture="adjustDeliveryDate"><el-date-picker v-model="form.deliveredAt" type="date" value-format="YYYY-MM-DD" placeholder="选择送货日期" /></div></el-form-item>
          <el-form-item v-if="idOf(note)" label="创建人"><el-input :model-value="note.createdByNameSnapshot || '—'" readonly /></el-form-item>
          <el-form-item v-if="idOf(note)" label="创建时间"><el-input :model-value="displayTime(note.createdAt ?? note.CreatedAt)" readonly /></el-form-item>
          <el-form-item v-if="idOf(note)" label="修改人"><el-input :model-value="note.updatedByName || userNickname(note.updatedBy)" readonly /></el-form-item>
          <el-form-item v-if="idOf(note)" label="修改时间"><el-input :model-value="displayTime(note.updatedAt ?? note.UpdatedAt)" readonly /></el-form-item>
          <el-form-item v-if="idOf(note)" label="审核人"><el-input :model-value="note.reviewedByNameSnapshot || '—'" readonly /></el-form-item>
          <el-form-item v-if="idOf(note)" label="审核时间"><el-input :model-value="displayTime(note.reviewedAt)" readonly /></el-form-item>
          <el-form-item v-if="idOf(note)" label="复审备注"><el-input :model-value="note.reReviewOpinion || '—'" type="textarea" autosize readonly /></el-form-item>
        </el-form>
        <div v-if="!readonly" class="editor-tools">
          <div class="editor-tool-actions">
            <el-button :disabled="!canEdit" @click="addBlankRow">新增空行</el-button>
            <el-button type="primary" :disabled="!canEdit || !optionsReady" @click="pickerOpen = true">选择物料</el-button>
            <el-button :disabled="!canEdit || !optionsReady" @click="openQuickMaterialCreate">新建物料并添加</el-button>
            <el-button type="danger" :disabled="!canEdit || !selectedDetailRows.length" @click="removeSelectedRows">移除所选</el-button>
          </div>
          <span class="editor-hint">数量、单价最多4位小数，金额最多2位小数，均可为0且不小于0；Enter / Tab 前进，Shift+Enter / Shift+Tab 返回。</span>
        </div>
        <div v-if="dragPreview" class="detail-drag-preview" role="status"><strong>正在移动：{{ dragPreview }}</strong><span>{{ dragTargetKey ? `松开以插入目标行${dragPosition === 'before' ? '之前' : '之后'}` : '拖到目标行的上半部或下半部；松开确认，Esc 取消' }}</span></div>
        <el-table :key="detailColumnSignature" ref="detailTable" :data="displayRows" row-key="rowKey" :row-class-name="detailRowClass" border size="small" class="detail-table" show-summary :summary-method="tableSummary" @selection-change="detailSelectionChanged" @row-click="toggleDetailRow" @row-dblclick="activateRow" @sort-change="changeDetailSort" @header-dragend="onDetailHeaderDragEnd" @dragover="hoverRowDrag" @dragleave="leaveRowDrag" @drop="handleRowDrop">
          <el-table-column v-if="!readonly" type="selection" width="45" />
          <el-table-column label="序号" width="80" align="center" :fixed="displaySettings.detail.frozen ? 'left' : false">
            <template #default="{ row, $index }"><span class="detail-order-handle" :data-row-key="row.rowKey" :draggable="canEdit" :title="canEdit ? '拖动明细序号调整顺序，保存后生效' : ''" @click.stop @dragstart="startRowDrag($event, row)" @dragend="endRowDrag"><span v-if="canEdit" class="detail-order-grip" aria-hidden="true">⠿</span>{{ $index + 1 }}</span></template>
          </el-table-column>
          <el-table-column v-for="(column, index) in orderedDetailColumns" :key="column.key" :prop="column.key" :label="column.label" :width="displaySettings.detail.widths[column.key] || column.width" min-width="80" :fixed="index < displaySettings.detail.frozen ? 'left' : false" :align="column.key === 'amount' ? 'right' : 'left'" sortable="custom">
            <template #default="{ row }">
              <template v-if="column.key === 'materialCode'">
                <span v-if="readonly || activeRowKey !== row.rowKey" class="editable-cell" @click.stop="activateCell(row, 'material')">{{ row.materialCode || '点击选择物料' }}</span>
                <el-select v-else :ref="(element) => setCellRef(row, 'material', element)" :model-value="row.materialId" filterable default-first-option automatic-dropdown :filter-method="(value) => row.keyword = value" :loading="row.loading" :disabled="!canEdit" placeholder="编号" @focus="openMaterialSelect(row)" @change="(value) => selectMaterial(row, value)" @visible-change="(open) => row.materialOpen = open" @keydown.capture="cellKeydown($event, row, 'material')">
                  <el-option v-if="row.materialId && !inlineMaterials(row).some((item) => idOf(item) === row.materialId)" :value="row.materialId" :label="row.materialCode" disabled />
                  <el-option v-for="material in inlineMaterials(row)" :key="idOf(material)" :value="idOf(material)" :label="material.code"><div class="delivery-option-main"><span>{{ material.name }}</span><small>{{ material.code }}</small></div><small class="option-spec">{{ material.specification }}</small></el-option>
                </el-select>
                <div v-if="row.errors.materialId" class="cell-error">{{ row.errors.materialId }}</div>
              </template>
              <template v-else-if="column.key === 'unitName'">
                <span v-if="readonly || activeRowKey !== row.rowKey" class="editable-cell" @click.stop="activateCell(row, 'unit')">{{ row.unitName }}</span>
                <el-select v-else :ref="(element) => setCellRef(row, 'unit', element)" :model-value="row.materialUnitId" :disabled="!canEdit || !row.materialId || row.loading" @change="(value) => selectUnit(row, value)" @visible-change="(open) => open ? loadRowUnits(row) : row.unitOpen = false" @keydown.capture="cellKeydown($event, row, 'unit')">
                  <el-option v-for="binding in row.units" :key="idOf(binding)" :value="idOf(binding)" :label="bindingLabel(binding)" :disabled="!binding.enabled" />
                </el-select>
                <div v-if="row.errors.materialUnitId" class="cell-error">{{ row.errors.materialUnitId }}</div>
              </template>
              <template v-else-if="['quantity', 'unitPrice', 'amount'].includes(column.key)">
                <span v-if="readonly || activeRowKey !== row.rowKey" class="editable-cell" @click.stop="activateCell(row, column.key)">{{ column.key === 'amount' ? rowAmount(row) : row[column.key] }}</span>
                <el-input v-else :ref="(element) => setCellRef(row, column.key, element)" v-model="row[column.key]" :disabled="!canEdit" inputmode="decimal" @focus="selectInputText" @input="clearFieldError(row, column.key)" @blur="commitNumber(row, column.key)" @compositionstart="row.composing = true" @compositionend="row.composing = false" @keydown="cellKeydown($event, row, column.key)" />
                <div v-if="row.errors[column.key]" class="cell-error">{{ row.errors[column.key] }}</div>
              </template>
              <template v-else-if="column.key === 'remark'">
                <span v-if="readonly || activeRowKey !== row.rowKey" class="editable-cell" data-detail-field="remark" @click.stop="activateCell(row, 'remark')">{{ row.remark }}</span>
                <el-input v-else :ref="(element) => setCellRef(row, 'remark', element)" v-model="row.remark" data-detail-field="remark" :tabindex="-1" :disabled="!canEdit" @compositionstart="row.composing = true" @compositionend="row.composing = false" @keydown="cellKeydown($event, row, 'remark')" />
                <div v-if="row.errors.remark" class="cell-error">{{ row.errors.remark }}</div>
              </template>
              <span v-else>{{ row[column.key] }}</span>
            </template>
          </el-table-column>
          <el-table-column v-if="!readonly" label="操作" width="80" fixed="right" align="right"><template #default="{ row }"><el-button link type="danger" :disabled="!canEdit" @click="removeRow(row)">移除</el-button></template></el-table-column>
        </el-table>
        <p class="editor-hint">单击明细单元格进入编辑，鼠标悬停虚线提示；点击行外结束编辑。Enter / Tab 前进，Shift+Enter / Shift+Tab 返回；拖动明细序号可预览插入位置，松开确认顺序，Esc 取消。金额按行四舍五入至2位小数后实时合计，修改金额会反算单价。移除仅在保存后生效。</p>
        
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
          <el-button v-if="!readonly" type="success" :loading="saving" :disabled="!canEdit || (!idOf(note) && !reservation.token)" @click="save(false, true)">审核</el-button>
          <el-button v-if="note.status === 'reviewed'" type="warning" :loading="actionLoading" :disabled="actionLoading || editorLoading" @click="transition(note, 'unreview')">反审核</el-button>
          <el-button v-if="note.status === 'reviewed'" type="success" :disabled="actionLoading || editorLoading" @click="openReReview([note])">复审</el-button>
          <el-button v-if="note.status === 'reReviewed'" type="warning" :loading="actionLoading" :disabled="actionLoading || editorLoading" @click="transition(note, 'cancelReReview')">取消复审</el-button>
        </div>
      </footer>
    </div>
    <DeliveryMaterialPicker v-model="pickerOpen" :materials="materials" :categories="categoryTree" :loading="rowsLoading" @confirm="appendMaterials" />
    <el-dialog v-model="reReviewDialog.open" title="复审确认" width="480px" :close-on-click-modal="false" :close-on-press-escape="!actionLoading && !batchLoading" :show-close="!actionLoading && !batchLoading">
      <p>是否确定复审所选单据？</p>
      <p>已选中{{ reReviewDialog.targets.length }}条数据</p>
      <el-form label-position="top"><el-form-item :label="reReviewDialog.batch ? '复审备注：' : '复审意见：'"><el-input v-model="reReviewDialog.opinion" type="textarea" :rows="3" placeholder="选填" :disabled="actionLoading || !!batchLoading" /></el-form-item></el-form>
      <template #footer><el-button :disabled="actionLoading || !!batchLoading" @click="reReviewDialog.open = false">取消</el-button><el-button type="primary" :loading="actionLoading || batchLoading === 'reReview'" @click="confirmReReview">确定复审</el-button></template>
    </el-dialog>
    <el-dialog v-model="quickMaterialDialogOpen" title="新建物料并添加到送货单" width="480px" :close-on-click-modal="false">
      <el-form label-width="90px" @submit.prevent="createQuickMaterial">
        <el-form-item label="物料名称" required><el-input v-model="quickMaterialForm.name" maxlength="200" @input="suggestQuickMaterialMnemonic" /></el-form-item>
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
import { computed, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import DeliveryMaterialPicker from '../components/delivery-material-picker.vue'
import MaterialFilter from '../components/material-filter.vue'
import { getUserList, setSelfSetting } from '@/api/user'
import { useUserStore } from '@/pinia/modules/user'
import { allOptions, idOf, pageSequenceNumber, request, showError } from '../api/common'
import { reserveMasterCode, saveMaster, suggestMasterMnemonic } from '../api/master'
import { useTableWidths } from '../use-table-widths'
import { DISPLAY_SETTING_KEY, FILTER_FIELDS, LIST_FIELDS, DETAIL_FIELDS, normalizeDisplaySettings, getDefaultDateRange } from '../delivery-display'
import { calculateAmount, decimalError, deliveryCellTarget, deliveryForwardTarget, formatScaled, parseScaled, settleDeliveryNumber, trimDecimal, updateDeliveryNumber } from '../delivery-input'
import { matchesMasterKeyword } from '../master-search'

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
const filters = reactive({ number: '', dates: [], brandId: undefined, materialIds: [], status: '', department: '', detailRemark: '' })
const tableWidthKey = 'warehouse.delivery.list'
const { widths: columnWidths, load: loadColumnWidths, headerDragEnd } = useTableWidths()
const userStore = useUserStore()
const viewRouteKey = route.fullPath
let stateAccount = accountKey()
let viewActive = true
let editorDocument = ''
let editorReady = false
let editorSequence = 0
function accountKey() { return String(idOf(userStore.userInfo) || '') }
function currentView() { return viewActive && route.fullPath === viewRouteKey }
function sessionKey(kind, account = stateAccount) { return `warehouse.delivery.${account}.${kind}` }
function readSession(kind) {
  if (!stateAccount) return null
  try { return JSON.parse(sessionStorage.getItem(sessionKey(kind)) || 'null') } catch { return null }
}
function writeSession(kind, value, account = stateAccount) {
  if (!account) return false
  try { sessionStorage.setItem(sessionKey(kind, account), JSON.stringify(value)); return true } catch (error) { showError(error); return false }
}
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
  ({ materialCode: 'material', unitName: 'unit', quantity: 'quantity', unitPrice: 'unitPrice', amount: 'amount', remark: 'remark' })[key]
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
const selectedReReviewed = computed(() => selectedNotes.value.filter((row) => row.status === 'reReviewed'))
const reReviewDialog = reactive({ open: false, batch: false, targets: [], opinion: '' })
function statusLabel(status) { return { draft: '未审核', reviewed: '已审核', reReviewed: '已复审' }[status] || status }
function statusType(status) { return { draft: 'info', reviewed: 'success', reReviewed: 'primary' }[status] || 'info' }
function openReReview(targets, batch = false) {
  if (actionLoading.value || batchLoading.value) return
  const eligible = targets.filter((row) => row.status === 'reviewed')
  if (!eligible.length) return
  Object.assign(reReviewDialog, { open: true, batch, targets: [...eligible], opinion: '' })
}
async function confirmReReview() {
  if (!reReviewDialog.open || actionLoading.value || batchLoading.value) return
  if (reReviewDialog.batch) await batchTransition('reReview', reReviewDialog.targets, reReviewDialog.opinion)
  else await transition(reReviewDialog.targets[0], 'reReview', reReviewDialog.opinion)
  reReviewDialog.open = false
}
function selectionChanged(rows) { selectedNotes.value = rows }
function selectableNote(row) { return ['draft', 'reviewed', 'reReviewed'].includes(row.status) }
async function batchTransition(action, confirmedTargets, opinion) {
  if (batchLoading.value || actionLoading.value) return
  const actor = stateAccount
  const targets = confirmedTargets || (['review', 'delete'].includes(action) ? selectedDrafts.value : action === 'cancelReReview' ? selectedReReviewed.value : selectedReviewed.value)
  if (!targets.length) return
  const label = { review: '审核', unreview: '反审核', reReview: '复审', cancelReReview: '取消复审', delete: '删除' }[action]
  const skipped = selectedNotes.value.length - targets.length
  const message = action === 'delete'
    ? `确定删除选中的 ${targets.length} 张未审核送货单？${skipped ? `另有 ${skipped} 张已审核或已复审单据不会删除。` : ''}此操作不可恢复。`
    : `确定${label}选中的 ${targets.length} 张送货单？`
  if (!confirmedTargets && !(await confirm(message, `${label}确认`))) return
  if (actor !== accountKey()) return
  batchLoading.value = action
  let completed = 0
  try {
    for (const row of targets) {
      await request({ url: `${base}/${action}`, method: action === 'delete' ? 'delete' : 'post', data: { id: idOf(row), version: row.version, ...(action === 'reReview' ? { opinion: opinion || '' } : {}) } })
      if (action === 'delete') clearEditorDraft(String(idOf(row)), actor)
      if (actor !== accountKey()) return
      completed++
    }
    ElMessage.success(`${label}成功：${completed} 张`)
  } catch (error) {
    if (actor === accountKey()) {
      showError(error)
      if (completed) ElMessage.warning(`已${label} ${completed} 张，其余未完成；请刷新后重试`)
    }
  } finally {
    if (actor === accountKey()) {
      batchLoading.value = ''
      selectedNotes.value = []
      listTable.value?.clearSelection?.()
      await loadList()
    }
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
const brandSelect = ref()
const form = reactive({ brandId: undefined, department: '', deliveredAt: '' })
const departments = ref([])
const quickMaterialSearch = ref('')
const quickMaterialDialogOpen = ref(false)
const quickMaterialSaving = ref(false)
const quickMaterialForm = reactive({ name: '', code: '', mnemonic: '', unitId: undefined })
let quickMaterialDialogSequence = 0
let quickMaterialMnemonicSequence = 0
const rows = ref([])
const detailTable = ref()
const selectedDetailRows = ref([])
const activeRowKey = ref('')
const draggedRowKey = ref('')
const dragTargetKey = ref('')
const dragPosition = ref('before')
const dragPreview = computed(() => {
  const index = displayRows.value.findIndex((row) => row.rowKey === draggedRowKey.value)
  const row = displayRows.value[index]
  return row ? `第 ${index + 1} 行 · ${row.materialName || '未选择物料'} · ${row.materialCode || '无编号'}` : ''
})
let dragGhost
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
const readonly = computed(() => ['reviewed', 'reReviewed'].includes(note.value.status))
const canEdit = computed(() => !readonly.value && !saving.value && !actionLoading.value && !editorLoading.value && !rowsLoading.value)
const dirty = computed(() => editorOpen.value && signature() !== savedSignature.value)
const detailListIndex = computed(() => notes.value.findIndex((row) => idOf(row) === idOf(note.value)))
const canNavigatePrevious = computed(() => detailListIndex.value > 0 && !editorLoading.value && !saving.value && !actionLoading.value && !rowsLoading.value)
const canNavigateNext = computed(() => detailListIndex.value >= 0 && detailListIndex.value < notes.value.length - 1 && !editorLoading.value && !saving.value && !actionLoading.value && !rowsLoading.value)
const filteredBrands = computed(() => brands.value.filter(brand => matchesMasterKeyword(brand, customerSearch.value)))
const listCustomerSearch = ref('')
const filteredListBrands = computed(() => brands.value.filter(brand => matchesMasterKeyword(brand, listCustomerSearch.value)))
function userNickname(id) {
  return usersById.value.get(Number(id)) || '—'
}
function listUserName(row, key) {
  const userKey = { createdByNameSnapshot: 'createdBy', updatedByName: 'updatedBy', reviewedByNameSnapshot: 'reviewedBy' }[key]
  const id = row[userKey]
  return usersById.value.get(Number(id)) || row[key] || '—'
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
    departments.value = await allOptions('departments')
  } catch (error) {
    showError(error)
  }
}
function suggestDepartments(queryString, callback) {
  callback(departments.value
    .filter(department => matchesMasterKeyword(department, queryString, ['name']))
    .map(department => ({ value: department.name })))
}
function suggestQuickMaterials(queryString, callback) {
  callback(materials.value
    .filter(item => matchesMasterKeyword(item, queryString))
    .slice(0, 20)
    .map((item) => ({ ...item, value: `${item.code} ${item.name}` })))
}
async function quickAddMaterial(item) {
  quickMaterialSearch.value = ''
  await appendMaterials([item])
}
async function openQuickMaterialCreate() {
  const sequence = ++quickMaterialDialogSequence
  ++quickMaterialMnemonicSequence
  Object.assign(quickMaterialForm, { name: quickMaterialSearch.value.trim(), code: '', mnemonic: '', unitId: undefined })
  quickMaterialDialogOpen.value = true
  if (quickMaterialForm.name) void suggestQuickMaterialMnemonic(quickMaterialForm.name)
  try {
    const result = await reserveMasterCode('materials')
    if (sequence === quickMaterialDialogSequence && quickMaterialDialogOpen.value) quickMaterialForm.code = result.code || ''
  } catch (error) {
    if (sequence === quickMaterialDialogSequence && quickMaterialDialogOpen.value) showError(error)
  }
}
async function suggestQuickMaterialMnemonic(name) {
  const sequence = ++quickMaterialMnemonicSequence
  const materialName = String(name || '').trim()
  quickMaterialForm.mnemonic = ''
  if (!materialName) return
  try {
    const result = await suggestMasterMnemonic(materialName)
    if (sequence === quickMaterialMnemonicSequence && materialName === quickMaterialForm.name.trim()) {
      quickMaterialForm.mnemonic = result.mnemonic || ''
    }
  } catch (error) {
    if (sequence === quickMaterialMnemonicSequence) showError(error)
  }
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
    if (column.property === 'amount') return formatScaled(rows.value.reduce((total, row) => total + (parseScaled(rowAmount(row), 2) ?? BigInt(0)), BigInt(0)), 2)
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
  return JSON.stringify({ brandId: form.brandId, department: form.department, deliveredAt: form.deliveredAt, details: rows.value.map((row) => [row.id, row.materialId, row.materialUnitId, row.quantity, row.unitPrice, row.amountEdited ? row.amount : undefined, row.remark]) })
}
function emptyRow() {
  return { rowKey: `delivery-row-${++nextRowKey}`, id: undefined, materialId: undefined, materialUnitId: undefined, materialCode: '', materialName: '', specification: '', unitName: '', quantity: '', unitPrice: '', remark: '', units: [], unitsLoaded: false, keyword: '', errors: {}, loading: false, composing: false, materialOpen: false, unitOpen: false }
}
function isEmptyRow(row) { return !row.materialId && !row.materialUnitId && !row.quantity.trim() && !row.unitPrice.trim() && !String(row.amount || '').trim() && !String(row.remark || '').trim() }
function inlineMaterials(row) {
  return materials.value.filter((item) => (!lineCategoryIds.value || lineCategoryIds.value.has(item.categoryId)) &&
    matchesMasterKeyword(item, row.keyword))
}
function bindingLabel(binding) {
  return binding.name || unitNames.value.get(binding.unitId) || `单位 #${binding.unitId}`
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
  if (!prop && !order) return
  const nextBy = order ? prop : ''
  const nextOrder = order === 'ascending' ? 'asc' : order === 'descending' ? 'desc' : ''
  if (sortBy.value === nextBy && sortOrder.value === nextOrder) return
  sortBy.value = nextBy
  sortOrder.value = nextOrder
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
  const params = { page: page.value, pageSize: pageSize.value, number: filters.number.trim() || undefined, brandId: filters.brandId || undefined, materialIds: filters.materialIds.length ? filters.materialIds.join(',') : undefined, status: filters.status || undefined, department: filters.department.trim() || undefined, detailRemark: filters.detailRemark.trim() || undefined, ...(sortBy.value ? { sortBy: sortBy.value, sortOrder: sortOrder.value } : {}) }
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
  Object.assign(filters, { number: '', dates: getDefaultDateRange(displaySettings.value.dateDefault), brandId: undefined, materialIds: [], status: '', department: '', detailRemark: '' })
  search()
}
function resetEditor() {
  note.value = {}; Object.assign(reservation, { number: '', token: '' }); rows.value = []; cellRefs.clear()
  activeRowKey.value = ''; selectedDetailRows.value = []; customerSearch.value = ''
  reservationError.value = ''; conflictMessage.value = ''; lineCategoryId.value = undefined
  detailSortBy.value = ''; detailSortOrder.value = ''; endRowDrag()
  reserving.value = false
  Object.assign(form, { brandId: undefined, department: '', deliveredAt: shanghaiInput(new Date()).slice(0, 10) })
}
async function openNew() {
  if (editorLoading.value || saving.value || actionLoading.value || rowsLoading.value || reserving.value) return
  if (detailRoute.value) {
    if (dirty.value && !(await confirm('新增单据将放弃当前送货单的未保存修改，确定继续？', '新建送货单'))) return
    clearEditorDraft()
    clearEditorDraft('new')
    editorReady = false
    if (!idOf(note.value)) return initializeNew()
  }
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
    amount: rowAmount(row),
    amountEdited: !!row.amountEdited,
    remark: row.remark,
    units: row.units.map((unit) => ({ ...unit })),
    unitsLoaded: false
  }))
  if (!writeSession('draft.new', {
    form: { ...form },
    reservation: {},
    rows: copiedRows.length ? copiedRows : [emptyRow()]
  })) return
  return enterDetail(null)
}
async function initializeNew() {
  const sequence = ++editorSequence
  const draft = readSession('draft.new')
  editorReady = false
  editorDocument = 'new'
  editorLoading.value = false
  resetEditor()
  rows.value = [emptyRow()]
  savedSignature.value = signature()
  if (draft) restoreEditorDraft(draft, true)
  editorOpen.value = true
  editorReady = true
  if (!reservation.token) await reserveNumber()
  if (sequence === editorSequence) persistEditorDraft()
}
function restoreEditorDraft(draft, isNew = false) {
  if (!isNew) note.value = { ...draft.note }
  Object.assign(form, draft.form)
  Object.assign(reservation, isNew ? draft.reservation || {} : {})
  rows.value = draft.rows?.length ? draft.rows.map((row) => {
    const fresh = emptyRow()
    return { ...fresh, ...row, id: isNew ? undefined : row.id, rowKey: fresh.rowKey, loading: false, composing: false, materialOpen: false, unitOpen: false }
  }) : [emptyRow()]
  savedSignature.value = draft.savedSignature || savedSignature.value
  detailSortBy.value = draft.detailSortBy || ''
  detailSortOrder.value = draft.detailSortOrder || ''
  lineCategoryId.value = draft.lineCategoryId
}
function persistEditorDraft() {
  if (!viewActive || !editorReady || !editorOpen.value || editorLoading.value || saving.value || !stateAccount) return
  if (editorDocument !== 'new' && !dirty.value) { clearEditorDraft(); return }
  if (editorDocument === 'new' && idOf(note.value)) return
  writeSession(`draft.${editorDocument}`, {
    note: editorDocument === 'new' ? undefined : { ...note.value },
    form: { ...form }, reservation: { ...reservation }, rows: rows.value,
    savedSignature: savedSignature.value, detailSortBy: detailSortBy.value,
    detailSortOrder: detailSortOrder.value, lineCategoryId: lineCategoryId.value
  })
}
function clearEditorDraft(document = editorDocument, account = stateAccount) {
  if (account && document) sessionStorage.removeItem(sessionKey(`draft.${document}`, account))
}
function persistListState() {
  if (!viewActive) return
  writeSession('listState', { filters: { ...filters }, page: page.value, pageSize: pageSize.value, sortBy: sortBy.value, sortOrder: sortOrder.value, moreFiltersOpen: moreFiltersOpen.value })
}
function restoreListState() {
  const state = readSession('listState')
  Object.assign(filters, { number: '', dates: getDefaultDateRange(displaySettings.value.dateDefault), brandId: undefined, materialIds: [], status: '', department: '', detailRemark: '' }, state?.filters)
  clearHiddenFilters()
  page.value = state?.page || 1
  pageSize.value = state?.pageSize || 20
  sortBy.value = state?.sortBy || ''
  sortOrder.value = state?.sortOrder || ''
  moreFiltersOpen.value = !!state?.moreFiltersOpen
}
function clearHiddenFilters() {
  const visible = new Set([...displaySettings.value.filters.basic, ...displaySettings.value.filters.more])
  for (const { key } of FILTER_FIELDS) {
    if (visible.has(key)) continue
    filters[key] = key === 'materialIds' || key === 'dates' ? [] : key === 'brandId' ? undefined : ''
  }
}
async function reserveNumber() {
  if (reserving.value || reservation.token || idOf(note.value)) return
  const sequence = editorSequence
  const actor = stateAccount
  reserving.value = true; reservationError.value = ''
  try {
    const data = await request({ url: `${base}/reserveNumber`, method: 'post' })
    if (sequence !== editorSequence || actor !== accountKey()) return
    if (!data.number || !data.token) throw new Error('服务端未返回有效的正式单号和预占凭证')
    Object.assign(reservation, { number: data.number, token: data.token })
  } catch (error) {
    if (sequence === editorSequence && actor === accountKey()) { reservationError.value = error.message || '正式单号预占失败'; showError(error) }
  } finally { if (sequence === editorSequence && actor === accountKey()) reserving.value = false }
}
async function readAggregate(data) {
  // The current Go aggregate serializes Note/Details; accept tagged JSON as well.
  const current = data.note ?? data.Note
  const details = data.details ?? data.Details
  if (!current || !Array.isArray(details)) throw new Error('送货单详情结构无效')
  note.value = current; form.brandId = current.brandId; form.department = current.department || ''; form.deliveredAt = shanghaiInput(current.deliveredAt).slice(0, 10)
  rows.value = details.map((detail) => ({
    ...emptyRow(), id: idOf(detail), materialId: detail.materialId, materialUnitId: detail.materialUnitId,
    materialCode: detail.materialCodeSnapshot, materialName: detail.materialNameSnapshot,
    specification: detail.specificationSnapshot || '', unitName: detail.unitNameSnapshot,
    quantity: trimDecimal(detail.quantity), unitPrice: trimDecimal(detail.unitPrice),
    remark: detail.remark || '',
    amount: compactAmount(detail.amount == null ? calculateAmount(detail.quantity, detail.unitPrice) : detail.amount),
    amountEdited: detail.amount != null && parseScaled(detail.amount, 2) !== parseScaled(calculateAmount(detail.quantity, detail.unitPrice), 2),
    units: [{ ID: detail.materialUnitId, unitId: detail.unitId, factor: detail.factor, name: detail.unitNameSnapshot, enabled: false, isBase: detail.unitId === detail.baseUnitId }]
  }))
  if (!readonly.value && !rows.value.length) rows.value.push(emptyRow())
  activeRowKey.value = ''; selectedDetailRows.value = []
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
async function initializeExisting(row, restoreDraft = true) {
  const sequence = ++editorSequence
  const actor = stateAccount
  const id = String(idOf(row))
  const draft = restoreDraft ? readSession(`draft.${id}`) : null
  editorReady = false
  editorDocument = id
  resetEditor(); editorOpen.value = true; editorLoading.value = true
  try {
    const aggregate = await request({ url: `${base}/detail`, method: 'get', params: { id: idOf(row) } })
    if (sequence !== editorSequence || actor !== accountKey()) return
    await readAggregate(aggregate)
    if (sequence !== editorSequence || actor !== accountKey()) return
    if (draft?.note && String(idOf(draft.note)) === id) {
      const latestVersion = note.value.version
      restoreEditorDraft(draft)
      if (latestVersion !== note.value.version) conflictMessage.value = '服务端单据已有新版本。本地草稿保留原版本，请读取最新版本后再操作。'
    }
    editorReady = true
  } catch (error) {
    if (sequence === editorSequence && actor === accountKey()) { editorOpen.value = false; showError(error) }
  } finally { if (sequence === editorSequence && actor === accountKey()) editorLoading.value = false }
}
function openExisting(row) { return enterDetail(idOf(row)) }
async function reloadEditor() {
  if (!idOf(note.value) || !(await confirm('将丢弃当前表单修改并重新读取最新单据，确定继续？'))) return
  clearEditorDraft()
  await initializeExisting(note.value, false)
}
async function navigateDocument(direction) {
  if (!idOf(note.value) || editorLoading.value || saving.value || actionLoading.value || rowsLoading.value) return
  if (dirty.value && !(await confirm('切换单据将丢弃当前未保存的修改，确定继续？', '切换送货单'))) return
  const target = notes.value[detailListIndex.value + direction]
  if (!target) return
  clearEditorDraft()
  editorReady = false
  const query = { ...route.query, whDelivery: '1', whId: String(idOf(target)) }
  delete query.whNew
  await router.push({ name: route.name, query })
}
async function confirm(message, title = '操作确认') {
  try { await ElMessageBox.confirm(message, title, { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' }); return true }
  catch { return false }
}
function beforeClose(done) {
  if (saving.value || actionLoading.value || rowsLoading.value || reserving.value || editorLoading.value) return
  persistEditorDraft()
  persistListState()
  pickerOpen.value = false
  done()
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
function openSelect(select) {
  nextTick(() => { if (select) select.expanded = true })
}
function openMaterialSelect(row) {
  row.materialOpen = true
  openSelect(cellRefs.get(`${row.rowKey}:material`))
}
function detailRowClass({ row }) {
  return [
    activeRowKey.value === row.rowKey ? 'delivery-editing-row' : '',
    selectedDetailRows.value.some((selected) => selected.rowKey === row.rowKey) ? 'delivery-selected-row' : '',
    draggedRowKey.value === row.rowKey ? 'delivery-drag-source' : '',
    dragTargetKey.value === row.rowKey ? `delivery-drop-${dragPosition.value}` : ''
  ].filter(Boolean).join(' ')
}
function detailSelectionChanged(selected) { selectedDetailRows.value = selected }
function toggleDetailRow(row, column, event) {
  if (!canEdit.value || column?.type === 'selection' || event?.detail > 1) return
  if (event?.target?.closest?.('input, button, .el-select, .el-checkbox, .el-input')) return
  detailTable.value?.toggleRowSelection?.(row)
}
async function activateRow(row) {
  if (!canEdit.value) return
  await focusCell(row, 'material')
}
function startRowDrag(event, row) {
  if (!canEdit.value) { event.preventDefault(); return }
  endRowDrag()
  draggedRowKey.value = row.rowKey
  event.dataTransfer.effectAllowed = 'move'
  event.dataTransfer.setData('text/plain', row.rowKey)
  dragGhost = document.createElement('div')
  dragGhost.textContent = dragPreview.value
  Object.assign(dragGhost.style, { position: 'fixed', top: '12px', right: '12px', zIndex: '3000', pointerEvents: 'none', padding: '12px 18px', background: '#ecf5ff', border: '2px solid #409eff', borderRadius: '6px', color: '#303133', fontSize: '14px', whiteSpace: 'nowrap' })
  document.body.appendChild(dragGhost)
  event.dataTransfer.setDragImage?.(dragGhost, 20, 20)
}
function endRowDrag() {
  draggedRowKey.value = ''; dragTargetKey.value = ''; dragPosition.value = 'before'
  dragGhost?.remove()
  dragGhost = undefined
}
function hoverRowDrag(event) {
  if (!canEdit.value || !draggedRowKey.value) return
  const element = event.target?.closest?.('.el-table__row')
  if (!element) { dragTargetKey.value = ''; return }
  const index = Array.from(element.parentElement.children).indexOf(element)
  const target = displayRows.value[index]
  if (!target || target.rowKey === draggedRowKey.value) { dragTargetKey.value = ''; return }
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'
  const bounds = element.getBoundingClientRect()
  dragTargetKey.value = target.rowKey
  dragPosition.value = event.clientY < bounds.top + bounds.height / 2 ? 'before' : 'after'
}
function leaveRowDrag(event) {
  if (!event.currentTarget?.contains(event.relatedTarget)) dragTargetKey.value = ''
}
function handleRowDrop(event) {
  if (!draggedRowKey.value) return
  event.preventDefault()
  const target = displayRows.value.find((row) => row.rowKey === dragTargetKey.value)
  if (target) dropRow(target, dragPosition.value)
  else endRowDrag()
}
function dropRow(target, position = 'before') {
  const sourceKey = draggedRowKey.value
  endRowDrag()
  if (!canEdit.value || !sourceKey || sourceKey === target.rowKey) return
  const ordered = [...displayRows.value]
  const from = ordered.findIndex((row) => row.rowKey === sourceKey)
  if (from < 0) return
  const [moved] = ordered.splice(from, 1)
  const to = ordered.findIndex((row) => row.rowKey === target.rowKey)
  if (to < 0) return
  ordered.splice(to + (position === 'after' ? 1 : 0), 0, moved)
  detailSortBy.value = ''; detailSortOrder.value = ''
  detailTable.value?.clearSort?.()
  rows.value = ordered
}
function handleEditorPointerDown(event) {
  if (!activeRowKey.value) return
  if (event.target?.closest?.('.delivery-editing-row, .el-select-dropdown, .el-autocomplete-suggestion')) return
  activeRowKey.value = ''
}
async function removeSelectedRows() {
  if (!canEdit.value || !selectedDetailRows.value.length) return
  const keys = new Set(selectedDetailRows.value.map((row) => row.rowKey))
  if (!(await confirm(`确定移除选中的 ${keys.size} 行物料明细？保存后生效。`, '移除确认'))) return
  rows.value = rows.value.filter((row) => !keys.has(row.rowKey))
  if (keys.has(activeRowKey.value)) activeRowKey.value = ''
  selectedDetailRows.value = []
  detailTable.value?.clearSelection?.()
  if (!rows.value.length) rows.value.push(emptyRow())
}
function setCellRef(row, field, element) {
  const key = `${row.rowKey}:${field}`
  if (element) cellRefs.set(key, element)
  else cellRefs.delete(key)
}
async function focusCell(row, field) {
  activeRowKey.value = row.rowKey
  await nextTick()
  cellRefs.get(`${row.rowKey}:${field}`)?.focus?.()
}
function selectInputText(event) {
  const input = event.target
  if (typeof input?.select === 'function') input.select()
}
function headerKeydown(event) {
  if (!canEdit.value || !event.shiftKey || event.key !== 'Enter' || event.isComposing || event.keyCode === 229) return
  const inputs = [...event.currentTarget.querySelectorAll('input:not([readonly]):not([disabled]), textarea:not([readonly]):not([disabled])')]
  const previous = inputs[inputs.indexOf(event.target) - 1]
  if (!previous) return
  event.preventDefault(); event.stopImmediatePropagation()
  previous.focus(); previous.select()
}
async function activateCell(row, field) {
  if (!canEdit.value) return
  await focusCell(row, field)
}
function adjustDeliveryDate(event) {
  if (!canEdit.value || !['ArrowLeft', 'ArrowRight'].includes(event.key) || !form.deliveredAt) return
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(form.deliveredAt)
  if (!match) return
  event.preventDefault()
  event.stopImmediatePropagation?.()
  const day = new Date(Date.UTC(Number(match[1]), Number(match[2]) - 1, Number(match[3]) + (event.key === 'ArrowRight' ? 1 : -1)))
  form.deliveredAt = `${day.getUTCFullYear()}-${String(day.getUTCMonth() + 1).padStart(2, '0')}-${String(day.getUTCDate()).padStart(2, '0')}`
}
function addBlankRow() { const row = emptyRow(); rows.value.push(row); focusCell(row, 'material') }
function removeRow(row) {
  rows.value = rows.value.filter((item) => item.rowKey !== row.rowKey)
  if (activeRowKey.value === row.rowKey) activeRowKey.value = ''
  selectedDetailRows.value = selectedDetailRows.value.filter((item) => item.rowKey !== row.rowKey)
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
  Object.assign(row, { materialId: idOf(material), materialUnitId: idOf(baseBinding), materialCode: material.code, materialName: material.name, specification: material.specification || '', unitName: baseBinding.name || unitNames.value.get(baseBinding.unitId) || '', units: bindings, unitsLoaded: true, quantity: '0', unitPrice: '0', keyword: '', errors: {}, amount: '0', amountEdited: false })
}
async function selectMaterial(row, materialId) {
  if (!canEdit.value || row.loading) return
  row.loading = true; rowsLoading.value = true
  let selected = false
  try { applyMaterial(row, await currentMaterial(materialId)); selected = true }
  catch (error) { row.errors.materialId = error.message; showError(error) }
  finally { row.loading = false; rowsLoading.value = false }
  if (selected) await focusCell(row, 'quantity')
}
function selectUnit(row, materialUnitId) {
  row.materialUnitId = materialUnitId
  const binding = row.units.find((item) => idOf(item) === materialUnitId)
  row.unitName = binding?.name || unitNames.value.get(binding?.unitId) || ''
  delete row.errors.materialUnitId; focusCell(row, 'quantity')
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
function rowAmount(row) {
  return compactAmount(row.amountEdited ? row.amount : calculateAmount(row.quantity, row.unitPrice))
}
function compactAmount(value) {
  return value == null ? value : trimDecimal(value)
}
function clearFieldError(row, field) {
  delete row.errors[field]
  Object.assign(row, updateDeliveryNumber(row, field))
  if (field === 'amount') {
    if (parseScaled(row.unitPrice, 4) !== null) delete row.errors.unitPrice
  } else delete row.errors.amount
}
function commitNumber(row, field) {
  // Do not turn a completely untouched trailing row into a zero-valued detail.
  if (!row.materialId && isEmptyRow(row)) return
  Object.assign(row, settleDeliveryNumber(row, field))
  if (!decimalError(row[field], field)) delete row.errors[field]
}
function validateDecimal(row, field) {
  Object.assign(row, settleDeliveryNumber(row, field))
  const message = decimalError(row[field], field)
  if (message) row.errors[field] = message
  else delete row.errors[field]
  return !message
}
function cellKeydown(event, row, field) {
  if (!canEdit.value || event.isComposing || row.composing || event.keyCode === 229 || event.repeat || event.ctrlKey || event.altKey || event.metaKey) return
  if (!['Enter', 'Tab'].includes(event.key)) return
  const fields = editingFields.value
  const orderedRows = displayRows.value
  const rowIndex = orderedRows.findIndex((item) => item.rowKey === row.rowKey)
  const destination = event.shiftKey
    ? deliveryCellTarget(fields, field, true, !!row.materialId)
    : deliveryForwardTarget(fields, field, !!row.materialId, rowIndex < orderedRows.length - 1)
  if (event.shiftKey) {
    const target = destination.rowOffset ? orderedRows[rowIndex + destination.rowOffset] : row
    if (!target) return
    event.preventDefault()
    event.stopImmediatePropagation?.()
    const select = cellRefs.get(`${row.rowKey}:${field}`)
    if (field === 'material' || field === 'unit') {
      if (select) select.expanded = false
      select?.blur?.()
    }
    if (['quantity', 'unitPrice', 'amount'].includes(field)) commitNumber(row, field)
    focusCell(target, destination.field)
    return
  }
  // Let ElSelect confirm its highlighted option and close its own popper.
  if (field === 'material' && event.key === 'Enter') {
    const select = cellRefs.get(`${row.rowKey}:material`)
    nextTick(() => {
      if (select && !select.expanded && row.materialId && !row.loading) focusCell(row, 'quantity')
    })
    return
  }
  if (field === 'unit' && row.unitOpen) return
  if (field === 'material' && !row.materialId) return
  if (field === 'unit' && !row.materialUnitId) return
  if (!row.materialId && !destination.rowOffset && destination.field === 'material') {
    event.preventDefault()
    focusCell(row, 'material')
    return
  }
  if (['quantity', 'unitPrice', 'amount'].includes(field) && !validateDecimal(row, field)) { event.preventDefault(); focusCell(row, field); return }
  event.preventDefault()
  if (!destination.rowOffset) focusCell(row, destination.field)
  else {
    let target = orderedRows[rowIndex + destination.rowOffset]
    if (destination.append) { target = emptyRow(); rows.value.push(target) }
    focusCell(target, destination.field)
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
    if (row.amountEdited) validateDecimal(row, 'amount')
    if (!firstInvalid && Object.keys(row.errors).length) firstInvalid = { row, field: Object.keys(row.errors)[0] }
    return { ...(row.id ? { id: row.id } : {}), materialId: row.materialId, materialUnitId: row.materialUnitId, quantity: row.quantity.trim(), unitPrice: row.unitPrice.trim(), remark: row.remark || '', ...(row.amountEdited ? { amount: String(row.amount).trim() } : {}) }
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
      const field = ['materialId', 'materialUnitId', 'quantity', 'unitPrice', 'amount', 'remark'].includes(business.field) ? business.field : 'materialId'
      row.errors[field] = error.message
    }
  }
  showError(error)
}
async function save(continueNew = false, review = false) {
  if (!canEdit.value || (!idOf(note.value) && !reservation.token)) return
  if (!form.brandId || !form.deliveredAt) { ElMessage.error('请选择客户和送货日期'); return }
  const details = validatedDetails()
  if (!details) return
  const id = idOf(note.value)
  const actor = stateAccount
  const data = { ...(id ? { id, version: note.value.version } : { reservationToken: reservation.token }), brandId: form.brandId, department: form.department.trim(), deliveredAt: `${form.deliveredAt}T00:00:00+08:00`, details }
  if (review) data.review = true
  saving.value = true
  try {
    const saved = await request({ url: `${base}/${id ? 'update' : 'create'}`, method: id ? 'put' : 'post', data })
    clearEditorDraft(id ? String(id) : 'new', actor)
    if (actor !== accountKey()) return
    if (continueNew) {
      clearEditorDraft('new')
      editorReady = false
      const query = { ...route.query, whDelivery: '1', whNew: '1' }
      delete query.whId
      if (route.query.whNew === '1' && !route.query.whId) {
        await initializeNew()
      } else {
        editorOpen.value = false
        await router.push({ name: route.name, query })
      }
    } else {
      const savedNote = saved?.note ?? saved?.Note
      const savedId = id || idOf(savedNote)
      editorDocument = String(savedId)
      await readAggregate(saved)
      if (!id) {
        const query = { ...route.query, whDelivery: '1', whId: String(savedId) }
        delete query.whNew
        await router.push({ name: route.name, query })
      }
    }
    ElMessage.success(review ? '送货单保存并审核成功' : continueNew ? '保存成功，可继续新增送货单' : '送货单保存成功')
    await loadList()
  } catch (error) { if (actor === accountKey()) reportEditorError(error) } finally { if (actor === accountKey()) saving.value = false }
}
async function transition(row, action, opinion) {
  if (actionLoading.value || batchLoading.value) return
  if (action === 'unreview' && row.status === 'reReviewed') return
  const actor = stateAccount
  const label = { review: '审核', unreview: '反审核', delete: '删除', reReview: '复审', cancelReReview: '取消复审' }[action]
  const active = editorOpen.value && idOf(note.value) === idOf(row)
  if (active && action === 'review' && dirty.value) { ElMessage.warning('请先保存修改，再审核'); return }
  const warning = active && dirty.value ? '当前未保存的修改也将被丢弃。' : ''
  if (action !== 'reReview' && !(await confirm(`确定${label}送货单“${row.number}”？${warning}`, `${label}确认`))) return
  if (actor !== accountKey()) return
  actionLoading.value = true
  try {
    const result = await request({ url: `${base}/${action}`, method: action === 'delete' ? 'delete' : 'post', data: { id: idOf(row), version: row.version, ...(action === 'reReview' ? { opinion: opinion || '' } : {}) } })
    if (action === 'delete' || active) clearEditorDraft(String(idOf(row)), actor)
    if (actor !== accountKey()) return
    if (active) {
      if (action === 'delete') {
        editorReady = false
        editorOpen.value = false
        await router.push({ name: route.name, query: listRouteQuery() })
      }
      else {
        note.value = result
        activeRowKey.value = ''
        savedSignature.value = signature()
        if (action === 'unreview') {
          const aggregate = await request({ url: `${base}/detail`, method: 'get', params: { id: idOf(row) } })
          if (actor !== accountKey()) return
          await readAggregate(aggregate)
        }
      }
    }
    ElMessage.success(`${label}成功`)
    if (action === 'delete' && notes.value.length === 1 && page.value > 1) page.value--
    await loadList()
  } catch (error) {
    if (actor === accountKey()) { if (active) reportEditorError(error); else showError(error); await loadList() }
  } finally { if (actor === accountKey()) actionLoading.value = false }
}
restoreListState()
watch(() => displaySettings.value.filters, clearHiddenFilters, { deep: true })
watch([form, rows, reservation, detailSortBy, detailSortOrder, lineCategoryId], persistEditorDraft, { deep: true })
watch([filters, page, pageSize, sortBy, sortOrder, moreFiltersOpen], () => {
  if (currentView() && stateAccount === accountKey()) persistListState()
}, { deep: true })
onBeforeUnmount(() => {
  persistEditorDraft()
  persistListState()
  endRowDrag()
  ++editorSequence
  document.removeEventListener('pointerdown', handleEditorPointerDown, true)
})
onMounted(() => document.addEventListener('pointerdown', handleEditorPointerDown, true))
watch(() => displaySettings.value.list.columns, () => { selectedNotes.value = [] })
async function restoreRouteEditor() {
  if (!currentView()) return
  const { whDelivery: detail, whId: id, whNew: isNew } = route.query
  persistEditorDraft()
  if (detail !== '1' || (!id && isNew !== '1')) {
    ++editorSequence
    editorReady = false
    editorOpen.value = false
    return
  }
  if (id) await initializeExisting({ ID: Number(id) })
  else await initializeNew()
}
let activatedOnce = false
onActivated(() => {
  viewActive = true
  if (activatedOnce) {
    restoreListState()
    restoreRouteEditor()
    loadList()
  }
  activatedOnce = true
})
onDeactivated(() => {
  persistEditorDraft()
  persistListState()
  viewActive = false
  editorReady = false
  ++editorSequence
})
watch(() => idOf(userStore.userInfo), () => {
  persistListState()
  persistEditorDraft()
  editorReady = false
  ++editorSequence
  stateAccount = accountKey()
  resetEditor()
  editorOpen.value = false
  saving.value = false; actionLoading.value = false; batchLoading.value = ''
  reserving.value = false
  restoreListState()
  notes.value = []; selectedNotes.value = []; total.value = 0
  ++listSequence
  loadColumnWidths(tableWidthKey)
  if (stateAccount && currentView()) { loadList(); restoreRouteEditor() }
})
onMounted(() => { loadList(); loadOptions(); loadDepartments(); loadUsers(); loadColumnWidths(tableWidthKey) })
watch(() => [route.query.whDelivery, route.query.whId, route.query.whNew], restoreRouteEditor, { immediate: true })
</script>

<style scoped>
.delivery-page :deep(.el-table) { font-size: 14px; }
 .detail-table :deep(.el-input), .detail-table :deep(.el-select) { width: 100%; min-height: 22px; }
 .detail-table :deep(.el-input__wrapper), .detail-table :deep(.el-select__wrapper) { min-height: 22px; padding-top: 0; padding-bottom: 0; }
 .detail-table :deep(.editable-cell) { display: block; width: 100%; height: 24px; line-height: 22px; box-sizing: border-box; border: 1px dashed transparent; border-radius: 2px; cursor: text; }
 .detail-table :deep(.editable-cell:hover) { border-color: var(--el-border-color); }
 .detail-table :deep(.el-table__row > td.el-table__cell) { height: 34px; }
 .detail-table :deep(.el-table__cell) { padding: 3px 0; }
 .detail-table :deep(.cell) { line-height: 24px; }
 .detail-table :deep(.el-input__inner) { height: 24px; line-height: 24px; }
 .detail-table :deep(.el-select__wrapper) { height: 24px; min-height: 24px; }
 .detail-order-handle { display: flex; align-items: center; justify-content: center; gap: 6px; cursor: grab; user-select: none; border: 1px dashed var(--el-border-color); border-radius: 4px; }
 .detail-order-handle:active { cursor: grabbing; }
 .detail-order-grip { color: var(--el-color-primary); font-size: 20px; line-height: 22px; }
 .detail-drag-preview { display: flex; flex-wrap: wrap; gap: 8px 20px; padding: 10px 12px; margin-bottom: 8px; color: var(--el-color-primary); background: var(--el-color-primary-light-9); border: 1px solid var(--el-color-primary); border-radius: 4px; }
 .detail-drag-preview span { font-size: 12px; }
 .detail-table :deep(.delivery-drag-source > td.el-table__cell) { opacity: 0.5; }
 .detail-table :deep(.delivery-drop-before > td.el-table__cell) { box-shadow: inset 0 3px 0 var(--el-color-primary); }
 .detail-table :deep(.delivery-drop-after > td.el-table__cell) { box-shadow: inset 0 -3px 0 var(--el-color-primary); }
 .detail-order-handle[draggable="false"] { cursor: default; }
.detail-table :deep(.delivery-editing-row > td.el-table__cell),
.detail-table :deep(.delivery-selected-row > td.el-table__cell) { background: var(--el-color-primary-light-9); }
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
 .editor-tools { position: sticky; top: 0; z-index: 5; display: flex; align-items: center; flex-wrap: wrap; gap: 12px; margin-bottom: 16px; background: var(--el-bg-color); padding: 8px 0; }
 .editor-tools :deep(.el-select) { min-width: 240px; }
 .editor-hint { font-size: 12px; color: var(--el-text-color-secondary); }
 .cell-error { color: var(--el-color-danger); font-size: 12px; line-height: 1.5; margin-top: 4px; }
 .option-spec { margin-left: 12px; color: var(--el-text-color-secondary); }
 .delivery-option-main { display: flex; align-items: center; justify-content: space-between; gap: 16px; width: 100%; }
 .delivery-option-main small { color: var(--el-text-color-secondary); }
 .detail-table :deep(.el-select) { width: 100%; }
 .delivery-row-actions { display: flex; justify-content: flex-end; align-items: center; gap: 6px; white-space: nowrap; }
 .delivery-row-actions :deep(.el-button + .el-button) { margin-left: 0; }
 .delivery-editor > .el-alert { margin-bottom: 12px; }
</style>
