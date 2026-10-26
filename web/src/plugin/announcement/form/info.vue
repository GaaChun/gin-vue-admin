<template>
  <div>
    <div class="gva-form-box">
      <el-form
        :model="formData"
        ref="elFormRef"
        label-position="right"
        :rules="rule"
        label-width="80px"
      >
        <el-form-item label="标题:" prop="title">
          <el-input
            v-model="formData.title"
            :clearable="true"
            placeholder="请输入标题"
          />
        </el-form-item>
        <el-form-item label="内容:" prop="content">
          <RichEdit v-model="formData.content" />
        </el-form-item>
        <el-form-item label="作者:" prop="userID">
          <el-select
            v-model="formData.userID"
            placeholder="请选择作者"
            style="width: 100%"
            :clearable="true"
          >
            <el-option
              v-for="(item, key) in dataSource.userID"
              :key="key"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="附件:" prop="attachments">
          <SelectFile v-model="formData.attachments" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" :disabled="initializing" @click="save">保存</el-button>
          <el-button type="primary" @click="back">返回</el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup>
  import {
    getInfoDataSource,
    createInfo,
    updateInfo,
    findInfo
  } from '@/plugin/announcement/api/info'

  defineOptions({
    name: 'InfoForm'
  })

  // 自动获取字典
  import { useRoute, useRouter } from 'vue-router'
  import { ElMessage } from 'element-plus'
  import { onActivated, onDeactivated, onUnmounted, ref, reactive } from 'vue'
  import SelectFile from '@/components/selectFile/selectFile.vue'
  // 富文本组件
  import RichEdit from '@/components/richtext/rich-edit.vue'
  import { useUserStore } from '@/pinia/modules/user'
  import { accountIdentity } from '@/utils/route-state'

  const route = useRoute()
  const router = useRouter()
  const userStore = useUserStore()
  const saving = ref(false)
  const initializing = ref(true)
  let active = true
  let alive = true
  const savedRoute = () => ({ name: route.name, params: { ...route.params }, query: { ...route.query, id: String(formData.value.ID || formData.value.id) }, hash: route.hash || '' })
  onActivated(() => {
    active = true
    if (!route.query.id && (formData.value.ID || formData.value.id)) router.replace(savedRoute())
  })
  onDeactivated(() => { active = false })
  onUnmounted(() => { alive = false })

  const formData = ref({
    title: '',
    content: '',
    userID: undefined,
    attachments: []
  })
  // 验证规则
  const rule = reactive({})

  const elFormRef = ref()
  const dataSource = ref([])
  const getDataSourceFunc = async () => {
    const res = await getInfoDataSource()
    if (res.code === 0) {
      dataSource.value = res.data
    }
  }
  getDataSourceFunc()

  // 初始化方法
  const init = async () => {
    const owner = accountIdentity(userStore.userInfo)
    try {
      if (route.query.id) {
        const res = await findInfo({ ID: route.query.id })
        if (alive && owner === accountIdentity(userStore.userInfo) && res.code === 0) formData.value = res.data
      }
    } finally {
      if (alive) initializing.value = false
    }
  }

  init()
  // 保存按钮
  const save = async () => {
    if (saving.value || initializing.value) return
    saving.value = true
    const owner = accountIdentity(userStore.userInfo)
    try {
      const valid = await elFormRef.value?.validate().catch(() => false)
      if (!valid || !alive) return
      const creating = !(formData.value.ID || formData.value.id)
      const res = await (creating ? createInfo(formData.value) : updateInfo(formData.value))
      if (!alive || owner !== accountIdentity(userStore.userInfo) || res.code !== 0) return
      if (creating) formData.value = res.data
      ElMessage({ type: 'success', message: '创建/更改成功' })
      if (creating && active) await router.replace(savedRoute())
    } finally {
      if (alive) saving.value = false
    }
  }

  // 返回按钮
  const back = () => {
    router.go(-1)
  }
</script>

<style></style>
