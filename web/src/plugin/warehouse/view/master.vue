<template>
  <div class="warehouse-master">
    <div v-show="!detailRoute">
      <el-form :inline="true" :model="search" @submit.prevent="query">
        <el-form-item :label="resource === 'units' ? '单位名称' : '关键词'">
          <el-input v-model="search.keyword" clearable :placeholder="resource === 'units' ? '单位名称' : '编号 / 名称 / 助记符'" @keyup.enter="query" />
        </el-form-item>
        <el-form-item v-if="isMaterial" label="客户">
          <el-select v-model="search.brandId" clearable filterable placeholder="全部客户" class="brand-select">
            <el-option v-for="item in brands" :key="idOf(item)" :label="`${item.code} · ${item.name}`" :value="idOf(item)" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="query">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>
    <el-alert v-if="pageError" :title="pageError" type="error" show-icon :closable="false" class="error-alert" />
    <div v-show="!detailRoute" class="master-layout">
      <aside v-if="isMaterial && treeVisible" class="category-panel gva-table-box">
        <div class="panel-heading"><strong>物料类别（含下级）</strong><el-button link @click="treeVisible = false">收起</el-button></div>
        <el-input v-model="treeKeyword" placeholder="筛选类别" clearable />
        <el-tree ref="categoryTreeRef" :data="materialTree" node-key="id" :props="treeProps" highlight-current default-expand-all :filter-node-method="filterTreeNode" :current-node-key="search.categoryId || 0" @node-click="selectCategory" />
      </aside>
      <div class="gva-table-box master-table">
        <div class="gva-btn-list master-actions">
          <el-button v-if="isMaterial && !treeVisible" @click="treeVisible = true">展开类别</el-button>
          <el-button type="primary" :disabled="loading" @click="openEditor()">新建{{ title }}</el-button>
          <el-button :loading="fileBusy" @click="runFileAction('template')">下载模板</el-button>
          <el-upload accept=".xlsx" :show-file-list="false" :before-upload="beforeImport" :http-request="handleImport" :disabled="fileBusy">
            <el-button :loading="fileBusy">导入 Excel</el-button>
          </el-upload>
          <el-button :loading="fileBusy" @click="runFileAction('export')">导出筛选全部</el-button>
          <el-button :loading="loading" @click="load">刷新</el-button>
        </div>
        <div v-if="isMaterial && search.categoryId" class="filter-hint">当前类别：{{ categoryName(search.categoryId) }}（含全部后代）</div>
        <div v-if="isCategory" class="filter-hint">树表按顶级分支分页；搜索保留匹配节点的上级路径。</div>
        <el-table :key="resource" v-loading="loading" :data="rows" :row-key="idOf" :tree-props="{ children: 'children' }" default-expand-all border @row-dblclick="openEditor($event)">
          <el-table-column v-if="resource !== 'units'" prop="code" label="编号" min-width="145" />
          <el-table-column prop="name" :label="`${title}名称`" min-width="170" show-overflow-tooltip />
          <el-table-column v-if="resource !== 'units'" prop="mnemonic" label="助记符" min-width="120" show-overflow-tooltip />
          <template v-if="isMaterial">
            <el-table-column label="客户" min-width="130"><template #default="{ row }">{{ brandName(row.brandId) }}</template></el-table-column>
            <el-table-column label="类别" min-width="130"><template #default="{ row }">{{ categoryName(row.categoryId) }}</template></el-table-column>
            <el-table-column prop="specification" label="规格" min-width="120" show-overflow-tooltip />
            <el-table-column prop="alias" label="别名" min-width="100" show-overflow-tooltip />
          </template>
          <el-table-column label="创建人" min-width="125"><template #default="{ row }">{{ userNickname(row.createdBy) }}</template></el-table-column>
          <el-table-column label="创建时间" min-width="170"><template #default="{ row }">{{ displayTime(row.CreatedAt || row.createdAt) }}</template></el-table-column>
          <el-table-column label="修改人" min-width="125"><template #default="{ row }">{{ userNickname(row.updatedBy) }}</template></el-table-column>
          <el-table-column label="修改时间" min-width="170"><template #default="{ row }">{{ displayTime(row.UpdatedAt || row.updatedAt) }}</template></el-table-column>
          <el-table-column label="审核人" min-width="125"><template #default="{ row }">{{ userNickname(row.reviewedBy) }}</template></el-table-column>
          <el-table-column label="审核时间" min-width="170"><template #default="{ row }">{{ displayTime(row.reviewedAt) }}</template></el-table-column>
          <el-table-column prop="version" label="版本" width="75" />
          <el-table-column label="操作" fixed="right" :width="isCategory ? 235 : 145">
            <template #default="{ row }">
              <el-button link type="primary" @click="openEditor(row)">编辑</el-button>
              <el-button v-if="isCategory" link type="primary" @click="openEditor(null, idOf(row))">新增下级</el-button>
              <el-button link type="danger" @click="remove(row)">删除</el-button>
            </template>
          </el-table-column>
          <template #empty><el-empty :description="pageError ? '加载失败，请点击刷新重试' : '暂无资料'" /></template>
        </el-table>
        <div class="gva-pagination">
          <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[10, 20, 50, 100]" :total="total" layout="total, sizes, prev, pager, next, jumper" @size-change="changePageSize" @current-change="load" />
        </div>
      </div>
    </div>
    <div v-if="detailRoute" class="master-detail-page">
      <div class="master-detail-heading"><h2>{{ `${form.id ? '编辑' : '新建'}${title}` }}</h2><el-button @click="closeEditor">返回列表</el-button></div>
      <div v-loading="detailLoading">
        <el-alert v-if="saveError" :title="saveError" type="error" show-icon :closable="false" class="error-alert">
          <template v-if="conflictVersion" #default>
            您的编辑内容已保留，服务器当前版本为 {{ conflictVersion }}。请重新载入最新详情后合并修改。
            <el-button link type="primary" @click="reloadDetail">重新载入（会放弃当前修改）</el-button>
          </template>
        </el-alert>
        <el-form ref="formRef" :model="form" label-width="90px" @submit.prevent="save">
          <el-form-item label="名称" prop="name" required :error="errors.name"><el-input v-model="form.name" :maxlength="isMaterial ? 200 : 100" show-word-limit @blur="suggestMnemonic()" /></el-form-item>
          <template v-if="resource !== 'units'">
            <el-form-item label="编号" prop="code" required :error="errors.code"><el-input v-model="form.code" maxlength="64" show-word-limit /></el-form-item>
            <el-form-item label="助记符" prop="mnemonic" required :error="errors.mnemonic">
              <el-input v-model="form.mnemonic" maxlength="100" @input="editMnemonic"><template #append><el-button :loading="mnemonicLoading" @click="suggestMnemonic(true)">重新生成</el-button></template></el-input>
              <div class="field-hint">根据名称生成；手动修改后不再自动覆盖。</div>
            </el-form-item>
          </template>
          <el-form-item v-if="isCategory" label="上级类别" prop="parentId" :error="errors.parentId">
            <el-tree-select v-model="form.parentId" :data="parentOptions" :props="treeProps" node-key="id" check-strictly filterable default-expand-all :render-after-expand="false" class="full-width" />
          </el-form-item>
          <template v-if="isMaterial">
            <el-form-item label="客户" prop="brandId" :error="errors.brandId">
              <el-select v-model="form.brandId" clearable filterable placeholder="请选择客户（可选）" class="full-width"><el-option v-for="item in brands" :key="idOf(item)" :label="`${item.code} · ${item.name}`" :value="idOf(item)" /></el-select>
            </el-form-item>
            <el-form-item label="类别" prop="categoryId" :error="errors.categoryId">
              <el-tree-select v-model="form.categoryId" :data="categoryOptions" :props="treeProps" node-key="id" check-strictly clearable filterable default-expand-all :render-after-expand="false" class="full-width" />
            </el-form-item>
            <el-form-item label="规格" prop="specification" :error="errors.specification"><el-input v-model="form.specification" maxlength="255" show-word-limit /></el-form-item>
            <el-form-item label="别名" prop="alias" :error="errors.alias"><el-input v-model="form.alias" maxlength="200" show-word-limit /></el-form-item>
            <el-form-item label="物料图片" prop="imageFileId" :error="errors.imageFileId || errors.file">
              <div class="image-editor">
                <el-image v-if="imageUrl" :src="imageUrl" :preview-src-list="[imageUrl]" preview-teleported fit="contain" class="material-image" />
                <span v-else-if="form.imageFileId">已关联文件 #{{ form.imageFileId }}</span>
                <el-upload accept=".jpg,.jpeg,.png,.bmp" :show-file-list="false" :before-upload="beforeImageUpload" :http-request="handleImageUpload" :disabled="uploading || detailLoading">
                  <el-button :loading="uploading">{{ form.imageFileId ? '更换图片' : '上传图片' }}</el-button>
                </el-upload>
                <el-button v-if="form.imageFileId" type="danger" link :disabled="uploading" @click="clearImage">移除关联</el-button>
                <div class="field-hint">仅 jpg/jpeg/png/bmp，最大 1MiB；不压缩，服务端验证实际内容。{{ form.imageFileId ? `文件 ID：${form.imageFileId}` : '' }}</div>
              </div>
            </el-form-item>
            <el-form-item label="单位集合" prop="units" required :error="errors.units || errors.factor">
              <div class="full-width">
                <el-alert title="必须且只能有一个启用的基本单位，系数固定为 1。保存替换完整单位集合；已被送货引用的单位不能删除，可停用。" type="info" :closable="false" class="error-alert" />
                <el-table :data="form.units" :row-key="(row) => row.rowKey" border>
                  <el-table-column label="单位" min-width="130"><template #default="{ row }"><el-select v-model="row.unitId" filterable placeholder="选择单位"><el-option v-for="item in units" :key="idOf(item)" :label="item.name" :value="idOf(item)" :disabled="form.units.some(other => other !== row && other.unitId === idOf(item))" /></el-select></template></el-table-column>
                  <el-table-column label="换算系数" min-width="125"><template #default="{ row }"><el-input v-model="row.factor" :readonly="row.isBase" placeholder="正数，最多6位小数" /></template></el-table-column>
                  <el-table-column label="基本单位" width="100"><template #default="{ row }"><el-tag v-if="row.isBase" type="success">基本单位</el-tag><el-button v-else link type="primary" @click="setBase(row)">设为基本</el-button></template></el-table-column>
                  <el-table-column label="状态" width="80"><template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '停用' }}</el-tag></template></el-table-column>
                  <el-table-column label="操作" width="120"><template #default="{ row }"><el-button link :disabled="row.isBase" @click="row.enabled = !row.enabled">{{ row.enabled ? '停用' : '启用' }}</el-button><el-button link type="danger" @click="removeUnit(row)">删除</el-button></template></el-table-column>
                </el-table>
                <el-button class="add-unit" @click="addUnit">添加单位</el-button>
              </div>
            </el-form-item>
          </template>
          <el-descriptions v-if="form.id" title="审计信息" :column="2" border size="small">
            <el-descriptions-item label="记录 ID">{{ form.id }}</el-descriptions-item>
            <el-descriptions-item label="版本">{{ form.version }}</el-descriptions-item>
            <el-descriptions-item label="创建人">{{ userNickname(audit.createdBy) }}</el-descriptions-item>
            <el-descriptions-item label="修改人">{{ userNickname(audit.updatedBy) }}</el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ displayTime(audit.CreatedAt || audit.createdAt) }}</el-descriptions-item>
            <el-descriptions-item label="修改时间">{{ displayTime(audit.UpdatedAt || audit.updatedAt) }}</el-descriptions-item>
            <el-descriptions-item label="审核人">{{ userNickname(audit.reviewedBy) }}</el-descriptions-item>
            <el-descriptions-item label="审核时间">{{ displayTime(audit.reviewedAt) }}</el-descriptions-item>
          </el-descriptions>
        </el-form>
      </div>
      
      <div class="master-detail-footer"><el-button :disabled="saving || uploading" @click="closeEditor">取消</el-button><el-button type="primary" :loading="saving" :disabled="detailLoading || uploading" @click="save">保存</el-button></div>
    </div>
    <el-dialog v-model="importErrorsVisible" title="导入失败：整份文件未提交" width="min(95%, 900px)">
      <el-alert :title="importErrorMessage" type="error" :closable="false" class="error-alert" />
      <el-table :data="importErrors" border><el-table-column prop="sheet" label="工作表" /><el-table-column prop="row" label="行" width="75" /><el-table-column prop="column" label="列" /><el-table-column prop="message" label="原因" min-width="250" /></el-table>
      <template #footer><el-button @click="importErrorsVisible = false">关闭</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getUserList } from '@/api/user'
import { formatDate } from '@/utils/format'
import { getUrl } from '@/utils/image'
import { allOptions, idOf, showError } from '../api/common'
import { deleteMaster, detailMaster, downloadMasterTemplate, exportMaster, getCategoryTree, importMaster, listMaster, saveMaster, suggestMasterMnemonic, uploadMasterImage } from '../api/master'

const route = useRoute()
const router = useRouter()
const resource = computed(() => {
  const names = { warehouseUnit: 'units', warehouseBrand: 'brands', warehouseCategory: 'categories', warehouseMaterial: 'materials' }
  const aliases = { unit: 'units', units: 'units', brand: 'brands', brands: 'brands', category: 'categories', categories: 'categories', material: 'materials', materials: 'materials' }
  return names[route.name] || aliases[route.query.kind || route.query.resource || route.query.type] || 'units'
})
const title = computed(() => ({ units: '单位', brands: '客户', categories: '类别', materials: '物料' })[resource.value])
const isMaterial = computed(() => resource.value === 'materials')
const isCategory = computed(() => resource.value === 'categories')
const detailRoute = computed(() => route.query.whDetail === resource.value && (route.query.whId || route.query.whNew === '1'))
const search = reactive({ keyword: '', brandId: undefined, categoryId: undefined })
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rows = ref([])
const usersById = ref(new Map())
const loading = ref(false)
const pageError = ref('')
const brands = ref([])
const units = ref([])
const categories = ref([])
const treeVisible = ref(true)
const treeKeyword = ref('')
const categoryTreeRef = ref()
const treeProps = { label: 'name', children: 'children', value: 'id', disabled: 'disabled' }
const visible = ref(false)
const formRef = ref()
const detailLoading = ref(false)
const saving = ref(false)
const uploading = ref(false)
const fileBusy = ref(false)
const errors = ref({})
const saveError = ref('')
const conflictVersion = ref(null)
const audit = ref({})
const imageUrl = ref('')
const mnemonicManual = ref(false)
const mnemonicLoading = ref(false)
const form = reactive(emptyForm())
const importErrorsVisible = ref(false)
const importErrors = ref([])
const importErrorMessage = ref('')
let listSequence = 0
let editorSequence = 0
let mnemonicSequence = 0
let nextUnitKey = 0
let cleanSnapshot = ''
function listRouteQuery() {
  const query = { ...route.query }
  delete query.whDetail; delete query.whId; delete query.whNew; delete query.whParent
  return query
}
function enterEditor(row, parentId = 0) {
  const query = { ...route.query, whDetail: resource.value }
  if (row) query.whId = String(idOf(row))
  else query.whNew = '1'
  if (parentId) query.whParent = String(parentId)
  else delete query.whParent
  return router.push({ name: route.name, query })
}

function emptyForm() {
  return { id: 0, version: undefined, name: '', code: '', mnemonic: '', parentId: 0, brandId: undefined, categoryId: undefined, specification: '', alias: '', imageFileId: null, units: [] }
}
function mapTree(nodes) {
  return (nodes || []).map(node => ({ ...node, id: idOf(node), children: mapTree(node.children) }))
}
const categoryOptions = computed(() => mapTree(categories.value))
const materialTree = computed(() => [{ id: 0, name: '全部类别', children: categoryOptions.value }])
const categoryMap = computed(() => {
  const map = new Map()
  const walk = nodes => nodes.forEach(node => { map.set(idOf(node), node); walk(node.children || []) })
  walk(categories.value)
  return map
})
const brandMap = computed(() => new Map(brands.value.map(item => [idOf(item), item])))
const parentOptions = computed(() => {
  const mark = (nodes, disabled = false) => nodes.map(node => {
    const blocked = disabled || node.id === form.id
    return { ...node, disabled: blocked, children: mark(node.children || [], blocked) }
  })
  return [{ id: 0, name: '无上级', children: mark(categoryOptions.value) }]
})
function categoryName(id) { return categoryMap.value.get(id)?.name || `#${id}` }
function brandName(id) { return brandMap.value.get(id)?.name || `#${id}` }
function userNickname(id) { return usersById.value.get(Number(id)) || '—' }
function displayTime(value) { return value ? formatDate(value) : '—' }
async function loadUsers() {
  const response = await getUserList({ page: 1, pageSize: 999 })
  if (response?.code === 0) {
    usersById.value = new Map((response.data?.list || []).map(user => [Number(idOf(user)), user.nickName || '—']))
  }
}
function currentFilters() {
  const keyword = search.keyword.trim()
  if (resource.value === 'units') return keyword ? { name: keyword } : {}
  const filters = keyword ? { keyword } : {}
  if (isMaterial.value) {
    if (search.brandId) filters.brandId = search.brandId
    if (search.categoryId) filters.categoryId = search.categoryId
  }
  return filters
}
function filterCategories(nodes, keyword) {
  return nodes.reduce((result, node) => {
    const matches = !keyword || [node.code, node.name, node.mnemonic].some(value => String(value || '').toLocaleLowerCase().includes(keyword))
    const children = matches ? node.children || [] : filterCategories(node.children || [], keyword)
    if (matches || children.length) result.push({ ...node, children })
    return result
  }, [])
}
async function load() {
  const sequence = ++listSequence
  const kind = resource.value
  loading.value = true
  pageError.value = ''
  try {
    if (kind === 'categories') {
      const tree = await getCategoryTree()
      if (sequence !== listSequence) return
      categories.value = tree || []
      const filtered = filterCategories(tree || [], search.keyword.trim().toLocaleLowerCase())
      total.value = filtered.length
      page.value = Math.max(1, Math.min(page.value, Math.ceil(total.value / pageSize.value) || 1))
      rows.value = filtered.slice((page.value - 1) * pageSize.value, page.value * pageSize.value)
    } else {
      const result = await listMaster(kind, { ...currentFilters(), page: page.value, pageSize: pageSize.value })
      if (sequence !== listSequence) return
      total.value = result.total || 0
      rows.value = result.list || []
      if (!rows.value.length && page.value > 1) {
        page.value = Math.max(1, Math.ceil(total.value / pageSize.value))
        await load()
      }
    }
  } catch (error) {
    if (sequence !== listSequence) return
    rows.value = []
    total.value = 0
    pageError.value = error.message || '资料加载失败'
    showError(error)
  } finally {
    if (sequence === listSequence) loading.value = false
  }
}
async function refreshOptions(kind = resource.value) {
  if (kind === 'materials') {
    const [brandRows, unitRows, tree] = await Promise.all([allOptions('brands'), allOptions('units'), getCategoryTree()])
    if (resource.value !== kind) return
    brands.value = brandRows
    units.value = unitRows
    categories.value = tree || []
  } else if (kind === 'categories') {
    const tree = await getCategoryTree()
    if (resource.value === kind) categories.value = tree || []
  }
}
function query() { page.value = 1; return load() }
function changePageSize() { page.value = 1; return load() }
function resetSearch() { Object.assign(search, { keyword: '', brandId: undefined, categoryId: undefined }); categoryTreeRef.value?.setCurrentKey(0); return query() }
function selectCategory(node) { search.categoryId = node.id || undefined; query() }
function filterTreeNode(value, node) { return !value || [node.name, node.code, node.mnemonic].some(text => String(text || '').toLocaleLowerCase().includes(value.toLocaleLowerCase())) }
watch(treeKeyword, value => categoryTreeRef.value?.filter(value))

async function initializeEditor(row, parentId = 0) {
  const sequence = ++editorSequence
  ++mnemonicSequence
  mnemonicLoading.value = false
  Object.assign(form, emptyForm(), { parentId, categoryId: search.categoryId })
  errors.value = {}
  saveError.value = ''
  conflictVersion.value = null
  audit.value = {}
  imageUrl.value = ''
  mnemonicManual.value = false
  visible.value = true
  detailLoading.value = true
  cleanSnapshot = JSON.stringify(form)
  await nextTick()
  formRef.value?.clearValidate()
  try {
    await refreshOptions()
    if (sequence !== editorSequence) return
    if (row) {
      const detail = await detailMaster(resource.value, idOf(row))
      if (sequence !== editorSequence) return
      Object.assign(form, {
        id: idOf(detail), version: detail.version, name: detail.name || '', code: detail.code || '', mnemonic: detail.mnemonic || '',
        parentId: detail.parentId || 0, brandId: detail.brandId, categoryId: detail.categoryId,
        specification: detail.specification || '', alias: detail.alias || '', imageFileId: detail.imageFileId || null,
        units: (detail.units || []).map(unit => ({ rowKey: `unit-${++nextUnitKey}`, unitId: unit.unitId, factor: String(unit.factor), isBase: unit.isBase, enabled: unit.enabled }))
      })
      audit.value = detail
      imageUrl.value = detail.imageUrl ? getUrl(detail.imageUrl) : ''
      mnemonicManual.value = Boolean(detail.mnemonic)
    }
    if (sequence === editorSequence) cleanSnapshot = JSON.stringify(form)
  } catch (error) {
    if (sequence !== editorSequence) return
    visible.value = false
    showError(error)
  } finally {
    if (sequence === editorSequence) detailLoading.value = false
  }
}
function openEditor(row, parentId = 0) { return enterEditor(row, parentId) }
function editMnemonic() {
  mnemonicManual.value = true
  ++mnemonicSequence
  mnemonicLoading.value = false
}
async function suggestMnemonic(force = false) {
  if (resource.value === 'units' || (!force && mnemonicManual.value) || !form.name.trim()) return
  const sequence = ++mnemonicSequence
  const session = editorSequence
  const name = form.name.trim()
  mnemonicLoading.value = true
  try {
    const result = await suggestMasterMnemonic(name)
    if (session === editorSequence && sequence === mnemonicSequence && name === form.name.trim() && (force || !mnemonicManual.value)) {
      form.mnemonic = result.mnemonic
      mnemonicManual.value = false
    }
  } catch (error) {
    if (session === editorSequence && sequence === mnemonicSequence) showError(error)
  } finally {
    if (sequence === mnemonicSequence) mnemonicLoading.value = false
  }
}
function addUnit() {
  const isBase = !form.units.length
  form.units.push({ rowKey: `unit-${++nextUnitKey}`, unitId: undefined, factor: '1', isBase, enabled: true })
}
function setBase(row) { form.units.forEach(unit => { unit.isBase = unit === row }); row.factor = '1'; row.enabled = true }
async function removeUnit(row) {
  if (row.isBase && form.units.length > 1) { ElMessage.warning('请先将其他单位设为基本单位'); return }
  form.units = form.units.filter(unit => unit !== row)
}
function validate() {
  const next = {}
  if (!form.name.trim()) next.name = '请输入名称'
  if (resource.value !== 'units') {
    if (!form.code.trim()) next.code = '请输入编号'
    if (!form.mnemonic.trim()) next.mnemonic = '请输入助记符或根据名称生成'
  }
  if (isCategory.value && form.code.trim().toUpperCase() === 'ROOT') next.code = 'ROOT 为保留根编号，不能使用'
  if (isMaterial.value) {
    const seen = new Set()
    if (form.units.filter(unit => unit.isBase && unit.enabled).length !== 1) next.units = '必须且只能有一个启用基本单位'
    for (const unit of form.units) {
      if (!unit.unitId) next.units = '请选择每一行单位'
      if (seen.has(unit.unitId)) next.units = '同一单位不能重复绑定'
      seen.add(unit.unitId)
      if (!/^\d+(\.\d{1,6})?$/.test(unit.factor) || !/[1-9]/.test(unit.factor) || unit.factor.split('.')[0].replace(/^0+/, '').length > 12) next.factor = '换算系数必须为正数，整数最多12位、小数最多6位'
      if (unit.isBase && (!unit.enabled || !/^0*1(?:\.0{1,6})?$/.test(unit.factor))) next.units = '基本单位必须启用且系数为1'
    }
  }
  errors.value = next
  return !Object.keys(next).length
}
async function locateError() {
  const first = Object.keys(errors.value)[0]
  if (!first) return
  await nextTick()
  formRef.value?.scrollToField(first === 'factor' ? 'units' : first)
  const field = formRef.value?.fields?.find(item => item.prop === first)
  field?.$el?.querySelector('input')?.focus()
}
function writePayload() {
  const data = { name: form.name.trim() }
  if (form.id) Object.assign(data, { id: form.id, version: form.version })
  if (resource.value !== 'units') Object.assign(data, { code: form.code.trim(), mnemonic: form.mnemonic.trim() })
  if (isCategory.value) data.parentId = form.parentId || 0
  if (isMaterial.value) Object.assign(data, {
    brandId: form.brandId, categoryId: form.categoryId, specification: form.specification || null, alias: form.alias || null,
    imageFileId: form.imageFileId || null,
    units: form.units.map(unit => ({ unitId: unit.unitId, factor: String(unit.factor), isBase: unit.isBase, enabled: unit.enabled }))
  })
  return data
}
async function save() {
  if (saving.value || uploading.value || detailLoading.value) return
  saving.value = true
  saveError.value = ''
  conflictVersion.value = null
  try {
    if (!mnemonicManual.value) await suggestMnemonic()
    if (!validate()) { await locateError(); return }
    await saveMaster(resource.value, writePayload())
    ++editorSequence
    await router.push({ name: route.name, query: listRouteQuery() })
    ElMessage.success('保存成功')
    await load()
  } catch (error) {
    const business = error.business || {}
    saveError.value = error.message || '保存失败，编辑内容已保留'
    if (business.errorCode === 'VERSION_CONFLICT') conflictVersion.value = business.currentVersion
    if (business.field && business.field !== 'version') errors.value = { [business.field]: saveError.value }
    showError(error)
    await locateError()
  } finally { saving.value = false }
}
async function remove(row) {
  try {
    await ElMessageBox.confirm(`确认删除${title.value}“${row.name}”？被引用的资料无法删除。`, '删除确认', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' })
    await deleteMaster(resource.value, { id: idOf(row), version: row.version })
    ElMessage.success('删除成功')
    await load()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    pageError.value = error.message || '删除失败'
    showError(error)
  }
}
async function beforeClose(done) {
  if (saving.value || uploading.value) return
  if (JSON.stringify(form) !== cleanSnapshot && !detailLoading.value) {
    try { await ElMessageBox.confirm('当前修改尚未保存，确认放弃？', '关闭编辑', { type: 'warning', confirmButtonText: '放弃修改', cancelButtonText: '继续编辑' }) } catch { return }
  }
  ++editorSequence
  ++mnemonicSequence
  done()
}
function closeEditor() {
  if (saving.value || uploading.value) return
  if (JSON.stringify(form) !== cleanSnapshot && !detailLoading.value) {
    ElMessageBox.confirm('当前修改尚未保存，确认放弃？', '关闭编辑', { type: 'warning', confirmButtonText: '放弃修改', cancelButtonText: '继续编辑' })
      .then(() => router.push({ name: route.name, query: listRouteQuery() })).catch(() => {})
    return
  }
  router.push({ name: route.name, query: listRouteQuery() })
}
async function reloadDetail() {
  try { await ElMessageBox.confirm('重新载入将放弃当前修改，请先记录需要保留的内容。', '载入最新版本', { type: 'warning', confirmButtonText: '重新载入', cancelButtonText: '保留修改' }) } catch { return }
  await initializeEditor({ ID: form.id })
}
function beforeImageUpload(file) {
  let message = ''
  if (!/\.(jpg|jpeg|png|bmp)$/i.test(file.name)) message = '仅支持 jpg、jpeg、png、bmp 图片'
  else if (!file.size || file.size > 1024 * 1024) message = '图片必须为非空文件且不超过1MiB，不自动压缩'
  if (message) { errors.value = { ...errors.value, imageFileId: message }; ElMessage.error(message); return false }
  return true
}
async function handleImageUpload(options) {
  uploading.value = true
  const session = editorSequence
  try {
    const image = await uploadMasterImage(options.file)
    if (session !== editorSequence) return
    form.imageFileId = image.fileId
    imageUrl.value = getUrl(image.url)
    delete errors.value.imageFileId
    delete errors.value.file
    options.onSuccess(image)
    ElMessage.success('上传成功，保存物料后生效')
  } catch (error) {
    if (session === editorSequence) { errors.value = { ...errors.value, imageFileId: error.message }; showError(error) }
    options.onError(error)
  } finally { uploading.value = false }
}
function clearImage() { form.imageFileId = null; imageUrl.value = '' }
function beforeImport(file) {
  if (!/\.xlsx$/i.test(file.name)) { ElMessage.error('请选择 .xlsx 文件'); return false }
  return true
}
function collectImportErrors(business) {
  const collected = []
  function walk(item) {
    if (item.errors?.length) item.errors.forEach(walk)
    else collected.push({ sheet: item.sheet || '—', row: item.row || item.lineNo || '—', column: item.column || item.field || '—', message: item.reason || item.message || importErrorMessage.value })
  }
  walk(business)
  return collected
}
async function handleImport(options) {
  fileBusy.value = true
  const kind = resource.value
  try {
    await ElMessageBox.confirm('导入按单位名称或资料编号新增/更新；空白可选值会清空，物料单位按完整集合替换。任一行失败整份文件回滚。是否继续？', '导入确认', { type: 'warning', confirmButtonText: '开始导入', cancelButtonText: '取消' })
    const result = await importMaster(kind, options.file)
    options.onSuccess(result)
    ElMessage.success(`导入成功：新增 ${result.createdCount || 0} 条，更新 ${result.updatedCount || 0} 条`)
    if (resource.value === kind) { await refreshOptions(); await query() }
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    options.onError(error)
    importErrorMessage.value = error.message || '导入失败'
    importErrors.value = collectImportErrors(error.business || {})
    importErrorsVisible.value = true
    showError(error)
  } finally { fileBusy.value = false }
}
async function runFileAction(action) {
  if (fileBusy.value) return
  fileBusy.value = true
  try {
    if (action === 'template') await downloadMasterTemplate(resource.value)
    else await exportMaster(resource.value, currentFilters())
  } catch (error) { showError(error) } finally { fileBusy.value = false }
}
watch(resource, async () => {
  ++editorSequence
  ++mnemonicSequence
  visible.value = false
  detailLoading.value = false
  mnemonicLoading.value = false
  Object.assign(search, { keyword: '', brandId: undefined, categoryId: undefined })
  page.value = 1
  rows.value = []
  categories.value = []
  try { await refreshOptions() } catch (error) { showError(error) }
  try { await loadUsers() } catch (error) { showError(error) }
  await load()
}, { immediate: true })
watch(() => [route.query.whDetail, route.query.whId, route.query.whNew, route.query.whParent], async ([kind, id, isNew, parent]) => {
  if (kind !== resource.value || (!id && isNew !== '1')) {
    ++editorSequence
    visible.value = false
    detailLoading.value = false
    return
  }
  const selected = id ? { ID: Number(id) } : null
  await initializeEditor(selected, Number(parent) || 0)
}, { immediate: true })
</script>

<style scoped>
.master-layout { display: flex; gap: 16px; align-items: flex-start; }
.category-panel { width: 240px; flex-shrink: 0; overflow: auto; }
.panel-heading { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.category-panel :deep(.el-tree) { margin-top: 12px; }
.master-table { min-width: 0; flex: 1; }
.master-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; }
.master-actions :deep(.el-button + .el-button) { margin-left: 0; }
.brand-select { width: 220px; }
.full-width { width: 100%; }
.field-hint, .filter-hint { font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.6; }
.filter-hint { margin-bottom: 12px; }
.error-alert { margin-bottom: 16px; }
.image-editor { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; }
.image-editor .field-hint { flex-basis: 100%; }
.material-image { width: 120px; height: 120px; border: 1px solid var(--el-border-color); border-radius: 4px; }
.add-unit { margin-top: 12px; }
@media (max-width: 760px) { .master-layout { flex-direction: column; } .category-panel { width: 100%; box-sizing: border-box; max-height: 300px; } .master-table { width: 100%; box-sizing: border-box; } }
</style>
