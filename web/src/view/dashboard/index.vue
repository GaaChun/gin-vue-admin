<template>
  <main class="h-full overflow-auto bg-slate-50/60 p-4 dark:bg-slate-900 lg:p-6">
    <header class="mb-5">
      <p class="text-xs tracking-[0.2em] text-slate-500 dark:text-slate-400">WAREHOUSE · DELIVERY</p>
      <h1 class="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">送货单工作台</h1>
      <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">送货单状态与本月单据数量</p>
    </header>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" class="mb-4" />
    <section class="mb-6">
      <h2 class="mb-3 text-base font-semibold text-slate-900 dark:text-white">最近访问</h2>
      <div v-if="recentVisits.length" class="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-6">
        <button
          v-for="item in recentVisits"
          :key="item.id"
          type="button"
          class="truncate rounded-lg border border-slate-100 bg-white px-3 py-3 text-left text-sm text-slate-700 hover:border-[var(--el-color-primary)] hover:text-[var(--el-color-primary)] dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200"
          @click="openRecentMenu(item)"
        >{{ item.title }}</button>
      </div>
      <el-empty v-else :description="recentVisitError || '暂无可访问的菜单'" :image-size="72" />
    </section>
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
  </main>
</template>

<script setup>
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useRouterStore } from '@/pinia/modules/router'
import { CircleCheck, Clock, Document, Calendar } from '@element-plus/icons-vue'
import { getRecentMenuVisits } from '@/api/menu'
import { listDeliveryNotes } from '@/plugin/warehouse/api/delivery-dashboard'
import { defaultMenuEntries, sortRecentVisits } from './recentVisits'
import { emitter } from '@/utils/bus'

const router = useRouter()
const routerStore = useRouterStore()

const cards = [
  { key: 'total', label: '送货单总数', caption: '全部状态', icon: Document, color: 'bg-blue-50 text-blue-600 dark:bg-blue-400/10' },
  { key: 'draft', label: '待审核', caption: '草稿单据', icon: Clock, color: 'bg-amber-50 text-amber-600 dark:bg-amber-400/10' },
  { key: 'reviewed', label: '已审核', caption: '已完成审核', icon: CircleCheck, color: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-400/10' },
  { key: 'month', label: '本月送货单', caption: '按送货时间统计', icon: Calendar, color: 'bg-violet-50 text-violet-600 dark:bg-violet-400/10' }
]
const stats = reactive({ total: null, draft: null, reviewed: null, month: null })
const loading = ref(false)
const error = ref('')
const recentVisits = ref([])
const recentVisitError = ref('')


async function loadRecentVisits() {
  recentVisitError.value = ''
  const routeMap = routerStore.routeMap
  try {
    const res = await getRecentMenuVisits()
    if (res.code !== 0 || !Array.isArray(res.data)) throw new Error('响应格式无效')
    const visits = res.data
      .filter((item) => {
        const menu = routeMap[item.name]
        return menu && !menu.hidden
      })
      .map((item) => ({
        id: item.menuId,
        name: item.name,
        title: item.title,
        visitCount: item.visitCount,
        lastVisitedAt: item.lastVisitedAt
      }))
    recentVisits.value = visits.length
      ? sortRecentVisits(visits)
      : defaultMenuEntries(routerStore.asyncRouters?.[0]?.children)
  } catch {
    recentVisits.value = []
    recentVisitError.value = '最近访问暂不可用'
  }
}
function openRecentMenu(item) {
  const query = {}
  const params = {}
  routerStore.routeMap[item.name]?.parameters?.forEach((parameter) => {
    if (parameter.type === 'query') {
      query[parameter.key] = parameter.value
    } else {
      params[parameter.key] = parameter.value
    }
  })
  router.push({ name: item.name, query, params })
}

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

function refreshRecentVisits() {
  loadRecentVisits()
}

onMounted(() => {
  emitter.on('recent-visits-updated', refreshRecentVisits)
  loadStats()
  loadRecentVisits()
})
onBeforeUnmount(() => emitter.off('recent-visits-updated', refreshRecentVisits))
defineOptions({ name: 'Dashboard' })
</script>
