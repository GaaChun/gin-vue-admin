<template>
  <el-dialog
    :model-value="modelValue"
    title="选择物料"
    width="94%"
    :close-on-click-modal="false"
    :close-on-press-escape="!loading"
    :show-close="!loading"
    @update:model-value="updateOpen"
  >
    <div class="picker-layout">
      <section class="picker-panel picker-candidates" data-testid="picker-candidates">
        <div class="picker-heading">
          <h4>候选物料（{{ candidates.length }}）</h4>
          <span>已勾选 {{ candidateSelection.length }} 项</span>
        </div>
        <div class="picker-filters">
          <el-tree-select
            v-model="categoryId"
            data-testid="category-filter"
            :data="categories"
            :props="{ label: 'label', children: 'children' }"
            node-key="id"
            check-strictly
            clearable
            filterable
            default-expand-all
            :render-after-expand="false"
            :disabled="loading"
            placeholder="全部类别"
            aria-label="候选物料类别"
          />
          <el-input
            v-model="keyword"
            data-testid="candidate-search"
            clearable
            :disabled="loading"
            placeholder="搜索候选物料：编号、名称、规格或助记符"
            aria-label="搜索候选物料"
          />
        </div>
        <el-table
          ref="candidateTable"
          data-testid="candidate-table"
          :data="candidates"
          :row-key="idOf"
          :row-class-name="candidateRowClass"
          height="400"
          border
          @selection-change="candidateSelectionChanged"
          @row-click="toggleCandidateRow"
        >
          <el-table-column type="selection" width="44" :selectable="canSelect" />
          <el-table-column prop="code" label="物料编号" min-width="120" show-overflow-tooltip />
          <el-table-column prop="name" label="物料名称" min-width="140" show-overflow-tooltip />
          <el-table-column prop="specification" label="规格" min-width="120" show-overflow-tooltip />
        </el-table>
      </section>
      <div class="picker-transfer" aria-label="物料移入移回操作">
        <el-button data-testid="move-all-right" aria-label="全部移入" title="全部移入当前可见候选" :disabled="loading || !candidates.length" @click="moveAllRight">&gt;&gt;</el-button>
        <el-button data-testid="move-right" aria-label="勾选移入" title="移入勾选候选" :disabled="loading || !candidateSelection.length" @click="moveRight">&gt;</el-button>
        <el-button data-testid="move-left" aria-label="勾选移回" title="移回勾选物料" :disabled="loading || !selectedSelection.length" @click="moveLeft">&lt;</el-button>
        <el-button data-testid="move-all-left" aria-label="全部移回" title="全部移回当前可见已选" :disabled="loading || !visibleSelected.length" @click="moveAllLeft">&lt;&lt;</el-button>
      </div>
      <section class="picker-panel picker-selected" data-testid="picker-selected">
        <div class="picker-heading">
          <h4>已选物料（{{ selected.length }}）</h4>
          <span>已勾选 {{ selectedSelection.length }} 项</span>
        </div>
        <div class="picker-filters">
          <p class="picker-hint">按移入顺序追加，确认后才添加到送货明细。</p>
          <el-input
            v-model="selectedKeyword"
            data-testid="selected-search"
            clearable
            :disabled="loading"
            placeholder="搜索已选物料：编号、名称、规格或助记符"
            aria-label="搜索已选物料"
          />
        </div>
        <el-table
          ref="selectedTable"
          data-testid="selected-table"
          :data="visibleSelected"
          :row-key="idOf"
          :row-class-name="selectedRowClass"
          height="400"
          border
          @selection-change="selectedSelectionChanged"
          @row-click="toggleSelectedRow"
        >
          <el-table-column type="selection" width="44" :selectable="canSelect" />
          <el-table-column prop="code" label="物料编号" min-width="120" show-overflow-tooltip />
          <el-table-column prop="name" label="物料名称" min-width="140" show-overflow-tooltip />
          <el-table-column prop="specification" label="规格" min-width="120" show-overflow-tooltip />
        </el-table>
      </section>
    </div>
    <template #footer>
      <el-button data-testid="picker-cancel" :disabled="loading" @click="updateOpen(false)">取消</el-button>
      <el-button data-testid="picker-confirm" type="primary" :disabled="loading || !selected.length" :loading="loading" @click="confirm">追加为独立明细行</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { idOf } from '../api/common'

const props = defineProps({
  modelValue: Boolean,
  loading: Boolean,
  materials: { type: Array, default: () => [] },
  categories: { type: Array, default: () => [] }
})
const emit = defineEmits(['update:modelValue', 'confirm'])
const keyword = ref('')
const selectedKeyword = ref('')
const categoryId = ref()
const selected = ref([])
const candidateSelection = ref([])
const selectedSelection = ref([])
const candidateTable = ref()
const selectedTable = ref()
const selectedIds = computed(() => new Set(selected.value.map(idOf)))
const candidateSelectionIds = computed(() => new Set(candidateSelection.value.map(idOf)))
const selectedSelectionIds = computed(() => new Set(selectedSelection.value.map(idOf)))
const categoryIds = computed(() => {
  if (!categoryId.value) return null
  const result = new Set()
  function visit(nodes, include = false) {
    for (const node of nodes) {
      const matched = include || node.id === categoryId.value
      if (matched) result.add(node.id)
      visit(node.children || [], matched)
    }
  }
  visit(props.categories)
  return result
})
function matchesKeyword(row, search) {
  return !search || [row.code, row.name, row.specification, row.mnemonic]
    .some((value) => String(value || '').toLocaleLowerCase().includes(search))
}
const candidates = computed(() => {
  const search = keyword.value.trim().toLocaleLowerCase()
  const seen = new Set(selectedIds.value)
  return props.materials.filter((row) => {
    const id = idOf(row)
    if (seen.has(id) || (categoryIds.value && !categoryIds.value.has(row.categoryId)) || !matchesKeyword(row, search)) return false
    seen.add(id)
    return true
  })
})
const visibleSelected = computed(() => {
  const search = selectedKeyword.value.trim().toLocaleLowerCase()
  return selected.value.filter((row) => matchesKeyword(row, search))
})
watch(candidates, () => {
  candidateSelection.value = []
  candidateTable.value?.clearSelection()
})
watch(visibleSelected, () => {
  selectedSelection.value = []
  selectedTable.value?.clearSelection()
})
watch(() => props.modelValue, (open) => {
  if (!open) return
  keyword.value = ''
  selectedKeyword.value = ''
  categoryId.value = undefined
  selected.value = []
  candidateSelection.value = []
  selectedSelection.value = []
  candidateTable.value?.clearSelection()
  selectedTable.value?.clearSelection()
}, { immediate: true })

function canSelect() {
  return !props.loading
}
function candidateSelectionChanged(rows) {
  if (!props.loading) candidateSelection.value = rows
}
function selectedSelectionChanged(rows) {
  if (!props.loading) selectedSelection.value = rows
}
function candidateRowClass({ row }) {
  return candidateSelectionIds.value.has(idOf(row)) ? 'picker-checked-row' : ''
}
function selectedRowClass({ row }) {
  return selectedSelectionIds.value.has(idOf(row)) ? 'picker-checked-row' : ''
}
function toggleRow(table, row, column, event) {
  if (props.loading || column?.type === 'selection') return
  if (event?.target?.closest?.('input, button, .el-checkbox')) return
  table?.toggleRowSelection(row)
}
function toggleCandidateRow(row, column, event) {
  toggleRow(candidateTable.value, row, column, event)
}
function toggleSelectedRow(row, column, event) {
  toggleRow(selectedTable.value, row, column, event)
}
function moveAllRight() {
  if (!props.loading) selected.value.push(...candidates.value)
}
function moveRight() {
  if (props.loading) return
  selected.value.push(...candidates.value.filter((row) => candidateSelectionIds.value.has(idOf(row))))
}
function moveLeft() {
  if (props.loading) return
  selected.value = selected.value.filter((row) => !selectedSelectionIds.value.has(idOf(row)))
}
function moveAllLeft() {
  if (props.loading) return
  const visibleIds = new Set(visibleSelected.value.map(idOf))
  selected.value = selected.value.filter((row) => !visibleIds.has(idOf(row)))
}
function updateOpen(open) {
  if (!props.loading) emit('update:modelValue', open)
}
function confirm() {
  if (!props.loading && selected.value.length) emit('confirm', [...selected.value])
}
</script>

<style scoped>
.picker-layout { display: grid; grid-template-columns: minmax(0, 1fr) 52px minmax(0, 1fr); align-items: start; gap: 16px; }
.picker-panel { min-width: 0; padding: 14px; border: 1px solid var(--el-border-color); border-radius: var(--el-border-radius-base); background: var(--el-bg-color); }
.picker-heading { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; min-height: 24px; margin-bottom: 12px; }
.picker-heading h4 { margin: 0; color: var(--el-text-color-primary); }
.picker-heading span, .picker-hint { color: var(--el-text-color-secondary); font-size: 12px; }
.picker-filters { display: grid; gap: 10px; margin-bottom: 12px; }
.picker-filters :deep(.el-select) { width: 100%; }
.picker-hint { display: flex; align-items: center; min-height: 32px; margin: 0; }
.picker-transfer { display: flex; flex-direction: column; justify-content: center; gap: 12px; align-self: stretch; }
.picker-transfer .el-button { width: 52px; margin: 0; padding: 8px; font-weight: 600; }
.picker-panel :deep(.picker-checked-row) { --el-table-tr-bg-color: var(--el-color-primary-light-9); }
@media (max-width: 900px) {
  .picker-layout { gap: 8px; grid-template-columns: minmax(0, 1fr) 44px minmax(0, 1fr); }
  .picker-panel { padding: 8px; }
  .picker-transfer .el-button { width: 44px; }
}
@media (max-width: 600px) {
  .picker-layout { grid-template-columns: minmax(0, 1fr); }
  .picker-transfer { flex-direction: row; justify-content: center; }
}
</style>
