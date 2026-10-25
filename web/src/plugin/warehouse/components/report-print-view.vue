<template>
  <Teleport to="body">
    <article v-if="data" class="warehouse-print-sheet">
      <h1>{{ data.title }}</h1>
      <p>{{ data.filterDescription }}</p>
      <p>全量分组数：{{ data.projection.total }}；生成时间：{{ generatedAt }}</p>
      <table>
        <thead><tr><th v-for="column in columns" :key="column.id">{{ column.label }}</th></tr></thead>
        <tbody>
          <tr v-for="(row, index) in data.projection.list" :key="index">
            <td v-for="column in columns" :key="column.id">
              {{ row.values[column.id] ?? '—' }}
              <span v-if="row.unavailable?.[column.id]">（{{ row.unavailable[column.id] }}）</span>
              <small v-if="column.id === 'baseQuantity' && row.baseUnits?.length">
                基本单位：{{ row.baseUnits.map((item) => `${item.unitName} #${item.unitId}`).join('、') }}
              </small>
            </td>
          </tr>
          <tr v-if="!data.projection.list.length"><td :colspan="columns.length">无符合筛选条件的数据</td></tr>
        </tbody>
      </table>
      <ReportSummary :summary="data.projection.summary" :columns="data.projection.columns" />
      <p v-for="warning in data.projection.warnings" :key="warning">{{ warning }}</p>
    </article>
  </Teleport>
</template>
<script setup>
import { computed } from 'vue'
import ReportSummary from './report-summary.vue'
const props = defineProps({ data: { type: Object, default: null }, registry: { type: Array, default: () => [] }, generatedAt: { type: String, default: '' } })
const columns = computed(() => (props.data?.projection.columns || []).map((id) => props.registry.find((column) => column.id === id)).filter(Boolean))
</script>
<style>
.warehouse-print-sheet { display: none; }
@media print {
  @page { size: landscape; margin: 12mm; }
  body.warehouse-report-printing > :not(.warehouse-print-sheet) { display: none !important; }
  body.warehouse-report-printing .warehouse-print-sheet { display: block !important; color: #000; background: #fff; font-size: 10pt; }
  .warehouse-print-sheet h1 { text-align: center; font-size: 18pt; margin: 0 0 12px; }
  .warehouse-print-sheet table { width: 100%; border-collapse: collapse; table-layout: auto; }
  .warehouse-print-sheet thead { display: table-header-group; }
  .warehouse-print-sheet th, .warehouse-print-sheet td { border: 1px solid #333; padding: 5px; overflow-wrap: anywhere; white-space: pre-wrap; }
  .warehouse-print-sheet tr { break-inside: avoid; }
  .warehouse-print-sheet small { display: block; }
}
</style>
