<template>
  <el-dialog :model-value="modelValue" title="批量选择物料" width="90%" :close-on-click-modal="false" @update:model-value="$emit('update:modelValue', $event)">
    <div class="picker-layout">
      <aside class="picker-categories">
        <el-button link @click="categoryId = undefined">全部类别</el-button>
        <el-tree :data="categories" node-key="id" :props="{ label: 'label', children: 'children' }" default-expand-all highlight-current @node-click="(node) => categoryId = node.id" />
      </aside>
      <section class="picker-candidates">
        <el-input v-model="keyword" clearable placeholder="编号 / 名称 / 助记符" />
        <p class="picker-hint">双击候选或点击添加；同一物料在本次选择中只出现一次。</p>
        <el-table :data="candidates" :row-key="idOf" height="400" @row-dblclick="choose">
          <el-table-column prop="code" label="物料编号" min-width="120" />
          <el-table-column prop="name" label="物料名称" min-width="140" />
          <el-table-column prop="specification" label="规格" min-width="120" />
          <el-table-column label="操作" width="70"><template #default="{ row }"><el-button link type="primary" :disabled="selectedIds.has(idOf(row))" @click="choose(row)">添加</el-button></template></el-table-column>
        </el-table>
      </section>
      <section class="picker-selected">
        <h4>已选物料（{{ selected.length }}）</h4>
        <el-table :data="selected" :row-key="idOf" height="400">
          <el-table-column type="index" width="45" />
          <el-table-column prop="name" label="按此顺序追加" min-width="120" />
          <el-table-column label="操作" width="70"><template #default="{ row }"><el-button link type="danger" @click="remove(row)">移除</el-button></template></el-table-column>
        </el-table>
      </section>
    </div>
    <template #footer>
      <el-button :disabled="loading" @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :disabled="!selected.length" :loading="loading" @click="$emit('confirm', [...selected])">追加为独立明细行</el-button>
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
defineEmits(['update:modelValue', 'confirm'])
const keyword = ref('')
const categoryId = ref()
const selected = ref([])
const selectedIds = computed(() => new Set(selected.value.map(idOf)))
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
const candidates = computed(() => {
  const search = keyword.value.trim().toLocaleLowerCase()
  return props.materials.filter((row) => (!categoryIds.value || categoryIds.value.has(row.categoryId)) &&
    (!search || [row.code, row.name, row.mnemonic].some((value) => String(value || '').toLocaleLowerCase().includes(search))))
})
watch(() => props.modelValue, (open) => {
  if (open) {
    keyword.value = ''
    categoryId.value = undefined
    selected.value = []
  }
})
function choose(row) {
  if (!props.loading && !selectedIds.value.has(idOf(row))) selected.value.push(row)
}
function remove(row) {
  if (!props.loading) selected.value = selected.value.filter((item) => idOf(item) !== idOf(row))
}
</script>

<style scoped>
.picker-layout { display: grid; grid-template-columns: 190px minmax(300px, 1fr) minmax(230px, .65fr); gap: 18px; }
.picker-categories { max-height: 460px; overflow: auto; border-right: 1px solid var(--el-border-color); }
.picker-candidates, .picker-selected { min-width: 0; }
.picker-selected h4 { margin: 0 0 14px; }
.picker-hint { font-size: 12px; color: var(--el-text-color-secondary); }
@media (max-width: 900px) { .picker-layout { grid-template-columns: 140px minmax(250px, 1fr); } .picker-selected { grid-column: 1 / -1; } }
</style>
