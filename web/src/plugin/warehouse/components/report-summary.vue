<template>
  <section v-if="lines.length" class="warehouse-report-summary">
    <strong v-for="line in lines" :key="line.label">{{ line.label }}：{{ line.value }}</strong>
  </section>
</template>
<script setup>
import { computed } from 'vue'
const props = defineProps({
  summary: { type: Object, default: () => ({}) },
  columns: { type: Array, default: () => [] },
  showAmount: { type: Boolean, default: true }
})
const lines = computed(() => [
  { id: 'amount', label: '全局金额合计', value: props.summary.amount },
  { id: 'quantity', label: '全局数量合计', value: props.summary.quantity },
  { id: 'baseQuantity', label: '全局基本数量合计', value: props.summary.baseQuantity }
].filter((line) => props.columns.includes(line.id) && (line.id !== 'amount' || props.showAmount) && line.value != null))
</script>
<style scoped>
.warehouse-report-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin: 16px 0;
  color: var(--el-text-color-secondary);
}
</style>
