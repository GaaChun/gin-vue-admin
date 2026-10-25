<template>
  <div v-click-outside="closeOnOutside" class="material-filter">
    <el-popover :visible="open" :trigger="[]" :trigger-keys="[]" placement="bottom-start" :width="360" @after-enter="focusSearch" @after-leave="closing = false">
      <template #reference>
        <div class="material-filter-reference">
          <el-tooltip :disabled="open || !selectedMaterials.length" :trigger-keys="[]" :show-after="200" placement="top" popper-class="material-filter-selection-tooltip">
            <template #content>
              <div v-for="material in selectedMaterials" :key="idOf(material)" class="material-filter-selection">{{ material.name }}（{{ material.code }}）</div>
            </template>
            <el-input :model-value="summary" readonly placeholder="选择物料" aria-label="筛选物料" :aria-expanded="open" @focus="focusReference" @click="openFilter" @keydown.esc.stop="closeFilter" @keydown.enter.prevent.stop="openFilter" @keydown.space.prevent.stop="openFilter">
              <template #suffix>
                <button v-if="modelValue.length" type="button" class="material-filter-clear" aria-label="清空物料筛选" @mousedown.prevent @keydown.stop @click.stop="clearSelection">×</button>
                <span class="material-filter-arrow" aria-hidden="true">▾</span>
              </template>
            </el-input>
          </el-tooltip>
        </div>
      </template>
      <div ref="panel" class="material-filter-panel" @keydown.esc.stop="closeFilter">
        <el-input ref="searchInput" v-model="keyword" class="material-filter-search" clearable placeholder="搜索物料名称、编号或助记符" aria-label="搜索物料" @keydown.esc.stop="closeFilter" />
        <div class="material-filter-actions">
          <button type="button" class="material-filter-select-all" :disabled="!candidates.length" @click="selectAll">全选当前可见</button>
          <button type="button" class="material-filter-deselect-all" :disabled="!modelValue.length" @click="clearSelection">取消全选</button>
          <span>已选 {{ modelValue.length }} 项</span>
        </div>
        <div class="material-filter-options" aria-label="物料候选">
          <button v-for="material in candidates" :key="idOf(material)" type="button" class="material-filter-option" :class="{ selected: selectedIds.has(idOf(material)) }" :aria-pressed="selectedIds.has(idOf(material))" @click="toggle(material)">
            <span class="material-filter-label"><span>{{ material.name }}</span><small>{{ material.code }}</small></span>
            <span v-if="selectedIds.has(idOf(material))" class="material-filter-check" aria-hidden="true">✓</span>
          </button>
          <div v-if="!candidates.length" class="material-filter-empty">暂无匹配物料</div>
        </div>
      </div>
    </el-popover>
  </div>
</template>

<script setup>
import { computed, nextTick, ref } from 'vue'
import { ClickOutside as vClickOutside } from 'element-plus'
import { idOf } from '../api/common'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  materials: { type: Array, default: () => [] }
})
const emit = defineEmits(['update:modelValue'])
const open = ref(false)
const closing = ref(false)
const keyword = ref('')
const searchInput = ref(null)
const panel = ref(null)
const selectedIds = computed(() => new Set(props.modelValue))
const summary = computed(() => props.modelValue.length ? `已选 ${props.modelValue.length} 种物料` : '')
const selectedMaterials = computed(() => props.materials.filter(material => selectedIds.value.has(idOf(material))))
const candidates = computed(() => {
  const search = keyword.value.trim().toLocaleLowerCase()
  return props.materials.filter((material) => !search || [material.name, material.code, material.mnemonic].some((value) => String(value || '').toLocaleLowerCase().includes(search)))
})
async function focusSearch() {
  await nextTick()
  if (open.value) searchInput.value?.focus()
}
function openFilter() {
  closing.value = false
  if (!open.value) keyword.value = ''
  open.value = true
  focusSearch()
}
function focusReference() {
  // The popover restores reference focus while closing; that must not reopen it.
  if (!closing.value) openFilter()
}
function closeFilter() {
  if (!open.value) return
  closing.value = true
  open.value = false
}
function closeOnOutside(event) {
  if (!panel.value?.contains(event.target)) closeFilter()
}
function toggle(material) {
  const id = idOf(material)
  emit('update:modelValue', selectedIds.value.has(id) ? props.modelValue.filter((value) => value !== id) : [...props.modelValue, id])
}
function selectAll() {
  emit('update:modelValue', [...new Set([...props.modelValue, ...candidates.value.map(idOf)])])
}
function clearSelection() { emit('update:modelValue', []) }
</script>

<style scoped>
.material-filter { width: 220px; max-width: 100%; }
.material-filter :deep(.el-input__inner) { cursor: pointer; }
.material-filter-clear { border: 0; padding: 0 4px; color: var(--el-text-color-placeholder); background: transparent; cursor: pointer; font-size: 18px; }
.material-filter-clear:hover { color: var(--el-color-primary); }
.material-filter-arrow { color: var(--el-text-color-placeholder); }
.material-filter-panel { color: var(--el-text-color-regular); }
.material-filter-selection { max-width: 360px; overflow-wrap: anywhere; }
.material-filter-actions { display: flex; align-items: center; gap: 14px; padding: 12px 0; font-size: 12px; }
.material-filter-actions button { border: 0; padding: 0; background: transparent; color: var(--el-color-primary); cursor: pointer; font: inherit; }
.material-filter-actions button:disabled { color: var(--el-text-color-disabled); cursor: not-allowed; }
.material-filter-actions > span { margin-left: auto; color: var(--el-text-color-secondary); }
.material-filter-options { max-height: 280px; overflow-y: auto; border-top: 1px solid var(--el-border-color-lighter); }
.material-filter-option { display: flex; align-items: center; justify-content: space-between; gap: 12px; width: 100%; padding: 9px 10px; border: 0; background: transparent; color: var(--el-text-color-regular); font: inherit; text-align: left; cursor: pointer; }
.material-filter-option:hover, .material-filter-option:focus-visible { background: var(--el-fill-color-light); }
.material-filter-option.selected { color: var(--el-color-primary); background: var(--el-color-primary-light-9); }
.material-filter-label { display: flex; flex-direction: column; gap: 3px; min-width: 0; overflow-wrap: anywhere; }
.material-filter-label small { color: var(--el-text-color-secondary); font-size: 12px; }
.material-filter-check { flex-shrink: 0; color: var(--el-color-primary); font-weight: 700; }
.material-filter-empty { padding: 24px 0; text-align: center; color: var(--el-text-color-secondary); }
</style>
