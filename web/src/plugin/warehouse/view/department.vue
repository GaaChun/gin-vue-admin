<template>
  <div class="py-2 warehouse-department">
    <template v-if="!editing">
      <el-form :inline="true" :model="search" @submit.prevent="query">
        <el-form-item label="关键词"><el-input v-model="search.keyword" clearable placeholder="部门名称 / 任一助记符" @keyup.enter="query" /></el-form-item>
        <el-form-item><el-button type="primary" @click="query">查询</el-button><el-button @click="resetSearch">重置</el-button></el-form-item>
      </el-form>
      <el-alert v-if="pageError" :title="pageError" type="error" :closable="false" show-icon />
      <div class="gva-table-box">
        <div class="gva-btn-list"><el-button type="primary" @click="openEditor()">新建部门</el-button><el-button :loading="loading" @click="load">刷新</el-button></div>
        <el-table v-loading="loading" :data="rows" :row-key="idOf" border @row-dblclick="openEditor">
          <el-table-column type="index" label="序号" width="70" :index="index => pageSequenceNumber(index, page, pageSize)" />
          <el-table-column prop="name" label="部门名称" min-width="180" show-overflow-tooltip />
          <el-table-column prop="mnemonic" label="助记符" min-width="180" show-overflow-tooltip />
          <el-table-column label="创建人" width="110"><template #default="{ row }">{{ userName(row.createdBy) }}</template></el-table-column>
          <el-table-column label="创建时间" width="175"><template #default="{ row }">{{ displayTime(row.CreatedAt || row.createdAt) }}</template></el-table-column>
          <el-table-column label="修改人" width="110"><template #default="{ row }">{{ userName(row.updatedBy) }}</template></el-table-column>
          <el-table-column label="修改时间" width="175"><template #default="{ row }">{{ displayTime(row.UpdatedAt || row.updatedAt) }}</template></el-table-column>
          <el-table-column prop="version" label="版本" width="75" />
          <el-table-column label="操作" width="140" fixed="right"><template #default="{ row }"><el-button link type="primary" @click="openEditor(row)">编辑</el-button><el-button link type="danger" @click="remove(row)">删除</el-button></template></el-table-column>
        </el-table>
        <div class="gva-pagination"><el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[10, 20, 50, 100]" :total="total" layout="total, sizes, prev, pager, next, jumper" @current-change="load" @size-change="query" /></div>
      </div>
    </template>
    <div v-else v-loading="detailLoading" class="gva-table-box department-editor">
      <h3>{{ form.id ? '编辑部门' : '新建部门' }}</h3>
      <el-alert v-if="saveError" :title="saveError" type="error" :closable="false" show-icon>
        <template v-if="conflictVersion" #default>服务器当前版本为 {{ conflictVersion }}，本地修改已保留。<el-button link type="primary" @click="reloadDetail">重新载入最新版本（放弃本地修改）</el-button></template>
      </el-alert>
      <el-form :model="form" label-width="90px" @submit.prevent="save">
        <el-form-item label="部门名称" required :error="errors.name"><el-input v-model="form.name" maxlength="100" show-word-limit /></el-form-item>
        <el-form-item label="助记符" :error="errors.mnemonic"><el-input v-model="form.mnemonic" maxlength="100" show-word-limit /><div class="text-sm text-gray-500">可录入多个助记符，以逗号、中文逗号、分号或空白分隔；任一项均可查询。</div></el-form-item>
      </el-form>
      <el-descriptions v-if="form.id" title="审计信息" :column="2" border>
        <el-descriptions-item label="记录 ID">{{ form.id }}</el-descriptions-item><el-descriptions-item label="版本">{{ form.version }}</el-descriptions-item>
        <el-descriptions-item label="创建人">{{ userName(audit.createdBy) }}</el-descriptions-item><el-descriptions-item label="创建时间">{{ displayTime(audit.CreatedAt || audit.createdAt) }}</el-descriptions-item>
        <el-descriptions-item label="修改人">{{ userName(audit.updatedBy) }}</el-descriptions-item><el-descriptions-item label="修改时间">{{ displayTime(audit.UpdatedAt || audit.updatedAt) }}</el-descriptions-item>
      </el-descriptions>
      <div class="mt-4"><el-button :disabled="saving" @click="closeEditor">返回列表</el-button><el-button type="primary" :loading="saving" :disabled="detailLoading" @click="save">保存</el-button></div>
    </div>
  </div>
</template>

<script setup>
import { computed, onActivated, onBeforeUnmount, onDeactivated, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '@/pinia/modules/user'
import { getUserList } from '@/api/user'
import { formatDate } from '@/utils/format'
import { accountIdentity, routeTabKey, savedMasterRoute, snapshotRoute } from '@/utils/route-state'
import { idOf, pageSequenceNumber, showError } from '../api/common'
import { deleteMaster, detailMaster, listMaster, saveMaster } from '../api/master'

defineOptions({ name: 'WarehouseDepartment' })
const route = snapshotRoute(useRoute())
const router = useRouter()
const userStore = useUserStore()
const viewRouteKey = routeTabKey(route)
let viewActive = true
function currentView() { return viewActive && routeTabKey(router.currentRoute.value) === viewRouteKey }
const editing = computed(() => route.query.whDetail === 'departments' && (route.query.whId || route.query.whNew === '1'))
const search = reactive({ keyword: '' })
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rows = ref([])
const users = ref(new Map())
const loading = ref(false)
const detailLoading = ref(false)
const saving = ref(false)
const pageError = ref('')
const saveError = ref('')
const errors = ref({})
const conflictVersion = ref(null)
const form = reactive({ id: 0, version: undefined, name: '', mnemonic: '' })
const audit = ref({})
let clean = ''
let editorReady = false
let editorSequence = 0
let listSequence = 0
let account = accountIdentity(userStore.userInfo)
const documentKey = () => route.query.whId ? String(route.query.whId) : 'new'
const storageScope = routeTabKey({ ...route, query: listQuery() })
const storageKey = (kind, document = documentKey()) => `warehouse.department.${account}.${storageScope}.${kind}.${kind === 'editor' ? document : 'list'}`
function stored(key) { try { return JSON.parse(sessionStorage.getItem(key) || 'null') } catch { return null } }
function store(key, value) { try { sessionStorage.setItem(key, JSON.stringify(value)) } catch { /* Browser storage may be unavailable. */ } }
function forget(key) { try { sessionStorage.removeItem(key) } catch { /* Browser storage may be unavailable. */ } }
function snapshot() { return JSON.stringify({ name: form.name, mnemonic: form.mnemonic }) }
function persist() {
  if (!account) return
  if (!editing.value) store(storageKey('list'), { keyword: search.keyword, page: page.value, pageSize: pageSize.value })
  else if (editorReady && !(documentKey() === 'new' && form.id)) store(storageKey('editor'), { form: { ...form }, audit: audit.value, clean })
}
function restoreList() {
  const state = stored(storageKey('list'))
  search.keyword = typeof state?.keyword === 'string' ? state.keyword : ''
  page.value = Number.isInteger(state?.page) && state.page > 0 ? state.page : 1
  pageSize.value = [10, 20, 50, 100].includes(state?.pageSize) ? state.pageSize : 20
}
const userName = id => users.value.get(Number(id)) || (id ? `#${id}` : '—')
const displayTime = value => value ? formatDate(value) : '—'
async function loadUsers() {
  const owner = account
  try {
    const response = await getUserList({ page: 1, pageSize: 999 })
    if (owner === account && response?.code === 0) users.value = new Map((response.data?.list || []).map(item => [Number(idOf(item)), item.nickName || '—']))
  } catch { /* Audit IDs remain visible when user-directory access is unavailable. */ }
}
async function load() {
  if (!currentView() || editing.value) return
  const sequence = ++listSequence
  const owner = account
  loading.value = true; pageError.value = ''
  try {
    const result = await listMaster('departments', { keyword: search.keyword.trim(), page: page.value, pageSize: pageSize.value })
    if (sequence !== listSequence || owner !== account) return
    rows.value = result.list || []; total.value = result.total || 0
  } catch (error) { if (sequence === listSequence && owner === account) { pageError.value = error.message || '加载失败'; showError(error) } }
  finally { if (sequence === listSequence) loading.value = false }
}
function query() { page.value = 1; persist(); load() }
function resetSearch() { search.keyword = ''; query() }
function listQuery() { const query = { ...route.query }; delete query.whDetail; delete query.whId; delete query.whNew; return query }
function openEditor(row) {
  persist()
  router.push({ name: route.name, query: { ...listQuery(), whDetail: 'departments', ...(idOf(row) ? { whId: String(idOf(row)) } : { whNew: '1' }) } })
}
function applyDetail(item) {
  Object.assign(form, { id: idOf(item) || 0, version: item.version, name: item.name || '', mnemonic: item.mnemonic || '' })
  audit.value = item; clean = snapshot(); editorReady = true
}
async function initializeEditor(force = false) {
  if (!currentView() || !editing.value) return
  const sequence = ++editorSequence
  editorReady = false; saveError.value = ''; errors.value = {}; conflictVersion.value = null
  const draft = force ? null : stored(storageKey('editor'))
  if (draft?.form && String(draft.form.id || 'new') === documentKey()) {
    Object.assign(form, draft.form); audit.value = draft.audit || {}; clean = draft.clean || ''; editorReady = true; return
  }
  if (!route.query.whId) { applyDetail({}); return }
  detailLoading.value = true
  try {
    const item = await detailMaster('departments', Number(route.query.whId))
    if (sequence === editorSequence) applyDetail(item)
  } catch (error) { if (sequence === editorSequence) { saveError.value = error.message || '加载失败'; showError(error) } }
  finally { if (sequence === editorSequence) detailLoading.value = false }
}
async function save() {
  if (!currentView() || saving.value || detailLoading.value || !editorReady) return
  errors.value = {}; saveError.value = ''; conflictVersion.value = null
  if (!form.name.trim()) { errors.value.name = '请输入部门名称'; return }
  const owner = account
  const oldKey = storageKey('editor')
  saving.value = true
  try {
    const item = await saveMaster('departments', { id: form.id || undefined, version: form.version, name: form.name.trim(), mnemonic: form.mnemonic.trim() })
    if (owner !== account) return
    forget(oldKey); applyDetail(item)
    store(storageKey('editor', String(idOf(item))), { form: { ...form }, audit: audit.value, clean })
    // Replace the new-document tab instead of leaving a second saveable draft behind.
    if (!route.query.whId) {
      editorReady = false
      if (currentView()) await router.replace(savedMasterRoute(route, 'departments', idOf(item)))
    } else persist()
    ElMessage.success('保存成功')
  } catch (error) {
    if (owner !== account) return
    saveError.value = error.message || '保存失败，编辑内容已保留'
    const business = error.business || {}
    if (business.errorCode === 'VERSION_CONFLICT') conflictVersion.value = business.currentVersion
    if (business.field) errors.value = { [business.field]: saveError.value }
    persist(); showError(error)
  } finally { if (owner === account) saving.value = false }
}
async function closeEditor() {
  if (!currentView() || saving.value) return
  if (editorReady && snapshot() !== clean) {
    try { await ElMessageBox.confirm('当前修改尚未保存，确认放弃？', '返回列表', { type: 'warning' }) } catch { return }
    if (!currentView()) return
  }
  forget(storageKey('editor')); editorReady = false
  router.push({ name: route.name, query: listQuery() })
}
async function reloadDetail() {
  try { await ElMessageBox.confirm('重新载入会放弃当前修改，是否继续？', '载入最新版本', { type: 'warning' }) } catch { return }
  if (!currentView()) return
  forget(storageKey('editor')); await initializeEditor(true)
}
async function remove(row) {
  const owner = account
  try {
    await ElMessageBox.confirm(`确认删除部门“${row.name}”？历史送货单不受影响。`, '删除确认', { type: 'warning' })
    if (owner !== account || !currentView()) return
    await deleteMaster('departments', { id: idOf(row), version: row.version })
    if (owner !== account) return
    forget(storageKey('editor', String(idOf(row)))); ElMessage.success('删除成功')
    if (rows.value.length === 1 && page.value > 1) page.value--
    await load()
  } catch (error) { if (owner === account) showError(error) }
}
watch([form, search, page, pageSize], () => {
  if (currentView() && account === accountIdentity(userStore.userInfo)) persist()
}, { deep: true })
watch(() => accountIdentity(userStore.userInfo), value => {
  persist(); account = value; ++editorSequence; ++listSequence; editorReady = false
  Object.assign(form, { id: 0, version: undefined, name: '', mnemonic: '' }); audit.value = {}; rows.value = []; users.value = new Map(); saving.value = false; detailLoading.value = false
  restoreList()
  if (account && currentView()) { load(); loadUsers(); initializeEditor() }
}, { flush: 'sync' })
onDeactivated(() => { persist(); viewActive = false })
onBeforeUnmount(() => { persist(); viewActive = false; ++editorSequence; ++listSequence })
let activatedOnce = false
onActivated(() => {
  viewActive = true
  if (!currentView()) return
  if (route.query.whNew === '1' && form.id) {
    router.replace(savedMasterRoute(route, 'departments', form.id))
  } else if (activatedOnce) {
    if (!editing.value) load()
    else if (!editorReady) initializeEditor()
  }
  activatedOnce = true
})
restoreList(); load(); loadUsers(); initializeEditor()
</script>

<style scoped>
.department-editor { max-width: 900px; }
.department-editor h3 { margin-bottom: 20px; }
.department-editor .el-alert { margin-bottom: 16px; }
</style>
