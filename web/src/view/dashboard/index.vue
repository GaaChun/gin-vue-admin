<template>
  <main class="h-full overflow-auto bg-slate-50/60 p-4 dark:bg-slate-900 lg:p-6">
    <header class="mb-5">
      <p class="text-xs tracking-[0.2em] text-slate-500 dark:text-slate-400">WAREHOUSE · DELIVERY</p>
      <h1 class="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">送货单工作台</h1>
      <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">送货单状态与本月单据数量</p>
    </header>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" class="mb-4" />
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <el-card v-for="item in cards" :key="item.key" shadow="hover" class="!rounded-xl">
        <div class="flex items-center justify-between gap-3">
          <div>
            <p class="text-sm text-slate-500 dark:text-slate-400">{{ item.label }}</p>
            <strong class="mt-2 block text-3xl font-semibold text-slate-900 dark:text-white">{{ loading ? '—' : (stats[item.key]?.toLocaleString() ?? '—') }}</strong>
            <span class="mt-1 block text-xs text-slate-400">{{ item.caption }}</span>
          </div>
          <div class="flex h-12 w-12 items-center justify-center rounded-xl" :class="item.color">
            <el-icon :size="24"><component :is="item.icon" /></el-icon>
          </div>
        </div>
      </el-card>
    </div>
    <el-empty v-if="!loading && !error && stats.total === 0" description="暂无送货单记录" />
  </main>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { CircleCheck, Clock, Document, Calendar } from '@element-plus/icons-vue'
import { listDeliveryNotes } from '@/plugin/warehouse/api/delivery-dashboard'

const cards = [
  { key: 'total', label: '送货单总数', caption: '全部状态', icon: Document, color: 'bg-blue-50 text-blue-600 dark:bg-blue-400/10' },
  { key: 'draft', label: '待审核', caption: '草稿单据', icon: Clock, color: 'bg-amber-50 text-amber-600 dark:bg-amber-400/10' },
  { key: 'reviewed', label: '已审核', caption: '已完成审核', icon: CircleCheck, color: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-400/10' },
  { key: 'month', label: '本月送货单', caption: '按送货时间统计', icon: Calendar, color: 'bg-violet-50 text-violet-600 dark:bg-violet-400/10' }
]
const stats = reactive({ total: null, draft: null, reviewed: null, month: null })
const loading = ref(false)
const error = ref('')

async function loadStats() {
  loading.value = true
  error.value = ''
  const now = new Date()
  const parts = new Intl.DateTimeFormat('en-US', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit' }).formatToParts(now)
  const year = Number(parts.find(part => part.type === 'year').value)
  const monthNumber = Number(parts.find(part => part.type === 'month').value)
  const monthStart = `${new Date(Date.UTC(year, monthNumber - 1, 1)).toISOString().slice(0, 10)}T00:00:00+08:00`
  const nextMonth = `${new Date(Date.UTC(year, monthNumber, 1)).toISOString().slice(0, 10)}T00:00:00+08:00`
  try {
    const [all, drafts, reviewed, month] = await Promise.all([
      listDeliveryNotes({ page: 1, pageSize: 1 }),
      listDeliveryNotes({ page: 1, pageSize: 1, status: 'draft' }),
      listDeliveryNotes({ page: 1, pageSize: 1, status: 'reviewed' }),
      listDeliveryNotes({ page: 1, pageSize: 1, deliveredFrom: monthStart, deliveredTo: nextMonth })
    ])
    stats.total = all.total
    stats.draft = drafts.total
    stats.reviewed = reviewed.total
    stats.month = month.total
  } catch (e) {
    error.value = e.message || '送货单统计加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadStats)
defineOptions({ name: 'Dashboard' })
</script>
