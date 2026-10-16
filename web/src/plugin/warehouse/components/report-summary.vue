<template>
  <section class="warehouse-report-summary">
    <strong v-if="summary.amount !== undefined && summary.amount !== null">全局金额合计：{{ summary.amount }}</strong>
    <div v-for="group in groups" :key="group.title" class="summary-group">
      <h4>{{ group.title }}（不同桶不可相加）</h4>
      <div v-if="!group.buckets.length">无数据</div>
      <ul v-else>
        <li v-for="(bucket, index) in group.buckets" :key="index">
          {{ bucket.materialCode }} {{ bucket.materialName }} #{{ bucket.materialId }} / {{ bucket.unitName }} #{{ bucket.unitId }}：{{ bucket.value }}
        </li>
      </ul>
    </div>
  </section>
</template>
<script setup>
import { computed } from 'vue'
const props = defineProps({ summary: { type: Object, default: () => ({}) }, columns: { type: Array, default: () => [] } })
const groups = computed(() => [
  { id: 'quantity', title: '全局数量分桶', buckets: props.summary.quantityBuckets || [] },
  { id: 'baseQuantity', title: '全局基本数量分桶', buckets: props.summary.baseQuantityBuckets || [] }
].filter((item) => props.columns.includes(item.id)))
</script>
<style scoped>
.warehouse-report-summary { margin: 16px 0; overflow-wrap: anywhere; }
.summary-group { margin-top: 12px; }
h4 { margin: 0 0 6px; font-size: 14px; }
ul { margin: 0; padding-left: 20px; }
</style>
