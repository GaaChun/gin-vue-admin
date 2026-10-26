<template>
  <div
    class="bg-gray-50 text-slate-700 dark:text-slate-500 dark:bg-slate-800 w-screen h-screen"
  >
    <el-watermark
      v-if="config.show_watermark"
      :font="font"
      :z-index="9999"
      :gap="[180, 150]"
      class="!absolute !inset-0 !pointer-events-none"
      :content="userStore.userInfo.nickName"
    />
    <gva-header />
    <div class="flex flex-row w-full gva-container pt-16 box-border !h-full">
      <gva-aside
        v-if="
          config.side_mode === 'normal' ||
          config.side_mode === 'sidebar' ||
          (device === 'mobile' && config.side_mode == 'head') ||
          (device === 'mobile' && config.side_mode == 'combination')
        "
      />
      <gva-aside
        v-if="config.side_mode === 'combination' && device !== 'mobile'"
        mode="normal"
      />
      <div class="flex-1 w-0 h-full">
        <gva-tabs v-if="config.showTabs" :key="cacheOwner" />
        <div
          class="overflow-auto px-2"
          :class="config.showTabs ? 'gva-container2' : 'gva-container pt-1'"
        >
          <router-view v-if="reloadFlag" v-slot="{ Component, route }">
            <div
              id="gva-base-load-dom"
              class="gva-body-h bg-gray-50 dark:bg-slate-800"
            >
              <transition
                mode="out-in"
                :name="route.meta.transitionType || config.transition_type"
              >
                <keep-alive :key="cacheOwner" :include="cachedPageNames">
                  <component :is="routePage(Component, route)" :key="routeTabKey(route)" />
                </keep-alive>
              </transition>
            </div>
          </router-view>
          <BottomInfo />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
  import GvaAside from '@/view/layout/aside/index.vue'
  import GvaHeader from '@/view/layout/header/index.vue'
  import useResponsive from '@/hooks/responsive'
  import GvaTabs from './tabs/index.vue'
  import BottomInfo from '@/components/bottomInfo/bottomInfo.vue'
  import { emitter } from '@/utils/bus.js'
  import { cloneVNode, computed, defineComponent, markRaw, onUnmounted, provide, shallowReadonly, ref, shallowRef, onMounted, nextTick, reactive, shallowReactive, watch, watchEffect } from 'vue'
  import { useRouter, useRoute, routeLocationKey, routerViewLocationKey } from 'vue-router'
  import { useRouterStore } from '@/pinia/modules/router'
  import { useUserStore } from '@/pinia/modules/user'
  import { useAppStore } from '@/pinia'
  import { storeToRefs } from 'pinia'
  import { accountIdentity, routeTabKey, snapshotRoute } from '@/utils/route-state'
  import '@/style/transition.scss'
  const appStore = useAppStore()
  const { config, isDark, device } = storeToRefs(appStore)

  defineOptions({
    name: 'GvaLayout'
  })

  useResponsive(true)
  const font = reactive({
    color: 'rgba(0, 0, 0, .15)'
  })

  watchEffect(() => {
    font.color = isDark.value ? 'rgba(255,255,255, .15)' : 'rgba(0, 0, 0, .15)'
  })

  const router = useRouter()
  const route = useRoute()
  const routerStore = useRouterStore()

  onMounted(() => {
    // 挂载一些通用的事件
    emitter.on('reload', reload)
    if (userStore.loadingInstance) {
      userStore.loadingInstance.close()
    }
  })

  const userStore = useUserStore()
  const cacheOwner = computed(() => `${accountIdentity(userStore.userInfo)}:${Boolean(userStore.token)}`)
  const pages = shallowReactive(new Map())
  let pageSequence = 0
  const cachedPageNames = computed(() => [...pages.values()].map(page => page.name))
  function routePage(component, pageRoute) {
    if (!component) return
    const key = routeTabKey(pageRoute)
    if (!pages.has(key)) {
      const ownRoute = snapshotRoute(pageRoute)
      const name = `RouteTab${++pageSequence}`
      const page = markRaw(defineComponent({
        name,
        setup() {
          provide(routeLocationKey, shallowReadonly(ownRoute))
          provide(routerViewLocationKey, shallowRef(ownRoute))
          return () => cloneVNode(component)
        }
      }))
      pages.set(key, { name, page })
    }
    return pages.get(key).page
  }
  watch(() => routerStore.cachedTabs, tabs => {
    const retained = new Set(tabs.map(routeTabKey))
    retained.add(routeTabKey(route))
    for (const key of pages.keys()) {
      if (!retained.has(key)) pages.delete(key)
    }
  })
  watch(cacheOwner, () => pages.clear(), { flush: 'sync' })
  onUnmounted(() => {
    emitter.off('reload', reload)
    if (reloadTimer) window.clearTimeout(reloadTimer)
  })

  const reloadFlag = ref(true)
  let reloadTimer = null
  const reload = async () => {
    if (reloadTimer) {
      window.clearTimeout(reloadTimer)
    }
    reloadTimer = window.setTimeout(async () => {
      if (route.meta.keepAlive) {
        reloadFlag.value = false
        await nextTick()
        reloadFlag.value = true
      } else {
        const title = route.meta.title
        router.push({ name: 'Reload', params: { title } })
      }
    }, 400)
  }
</script>

<style lang="scss"></style>
