<template>
  <n-config-provider :theme="theme">
    <n-loading-bar-provider>
      <n-dialog-provider>
        <n-notification-provider>
          <n-message-provider>
            <n-layout has-sider style="height: 100vh;">
              <n-layout-sider
                bordered
                collapse-mode="width"
                :collapsed-width="64"
                :width="280"
                :collapsed="collapsed"
                show-trigger
                @collapse="collapsed = true"
                @expand="collapsed = false"
              >
                <div style="padding: 16px 0;">
                  <n-menu
                    :collapsed="collapsed"
                    :collapsed-width="64"
                    :collapsed-icon-size="22"
                    :options="menuOptions"
                    :value="($route.name as string)"
                    @update:value="handleMenuSelect"
                  />
                </div>
              </n-layout-sider>
              <n-layout>
                <n-layout-header bordered style="height: 64px; padding: 0 24px; display: flex; align-items: center;">
                  <n-space>
                    <n-h2 style="margin: 0;">LogLens</n-h2>
                    <n-tag type="info">v1.0.0</n-tag>
                  </n-space>
                </n-layout-header>
                <n-layout-content content-style="padding: 24px; height: calc(100vh - 64px); overflow-y: auto;">
                  <router-view />
                </n-layout-content>
              </n-layout>
            </n-layout>
          </n-message-provider>
        </n-notification-provider>
      </n-dialog-provider>
    </n-loading-bar-provider>
  </n-config-provider>
</template>

<script lang="ts" setup>
import { ref, h } from 'vue'
import { useRouter } from 'vue-router'
import {
  NConfigProvider,
  NLoadingBarProvider,
  NDialogProvider,
  NNotificationProvider,
  NMessageProvider,
  NLayout,
  NLayoutSider,
  NLayoutHeader,
  NLayoutContent,
  NMenu,
  NH2,
  NSpace,
  NTag,
  darkTheme
} from 'naive-ui'
import {
  HomeOutline as HomeIcon,
  DocumentAttachOutline as ImportIcon,
  SearchOutline as QueryIcon
} from '@vicons/ionicons5'

const router = useRouter()
const collapsed = ref(false)
const theme = darkTheme

const menuOptions = [
  {
    label: 'Home',
    key: 'Home',
    icon: () => h(HomeIcon)
  },
  {
    label: 'Import',
    key: 'Import',
    icon: () => h(ImportIcon)
  },
  {
    label: 'Query',
    key: 'Query',
    icon: () => h(QueryIcon)
  }
]

const handleMenuSelect = (key: string) => {
  router.push({ name: key })
}
</script>

<style>
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
    'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
    sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

#app {
  height: 100vh;
}
</style>
