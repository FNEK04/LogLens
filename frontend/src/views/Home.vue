<template>
  <n-space vertical size="large">
    <n-card title="Welcome to LogLens" hoverable>
      <template #header-extra>
        <n-tag type="success">Ready</n-tag>
      </template>
      <n-p>
        LogLens is a powerful desktop application for analyzing large log files and data dumps.
        Built with Go + Wails + Vue for optimal performance and user experience.
      </n-p>
      <n-divider />
      <n-space>
        <n-statistic label="Total Records" :value="stats?.totalRecords || 0">
          <template #suffix>
            <n-text depth="3" style="font-size: 16px">logs</n-text>
          </template>
        </n-statistic>
        <n-statistic label="Error Count" :value="stats?.levelCounts?.ERROR || 0">
          <template #suffix>
            <n-text depth="3" style="font-size: 16px">errors</n-text>
          </template>
        </n-statistic>
        <n-statistic label="Warning Count" :value="stats?.levelCounts?.WARN || 0">
          <template #suffix>
            <n-text depth="3" style="font-size: 16px">warnings</n-text>
          </template>
        </n-statistic>
      </n-space>
    </n-card>

    <n-grid x-gap="12" y-gap="12" cols="1 s:2 m:2 l:3" responsive="screen">
      <n-gi>
        <n-card title="Quick Actions" hoverable>
          <n-space vertical>
            <n-button 
              type="primary" 
              block 
              size="large"
              @click="$router.push('/import')"
              :loading="false"
            >
              <template #icon>
                <n-icon><document-attach-outline /></n-icon>
              </template>
              Import Logs
            </n-button>
            <n-button 
              type="info" 
              block 
              size="large"
              @click="$router.push('/query')"
              :disabled="stats?.totalRecords === 0"
            >
              <template #icon>
                <n-icon><search-outline /></n-icon>
              </template>
              Query Logs
            </n-button>
          </n-space>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card title="Supported Formats" hoverable>
          <n-list>
            <n-list-item>
              <n-thing title="Plain Text" description="Standard log files">
                <template #avatar>
                  <n-avatar round>📄</n-avatar>
                </template>
              </n-thing>
            </n-list-item>
            <n-list-item>
              <n-thing title="JSON/NDJSON" description="Structured log data">
                <template #avatar>
                  <n-avatar round>🔧</n-avatar>
                </template>
              </n-thing>
            </n-list-item>
            <n-list-item>
              <n-thing title="CSV" description="Tabular data">
                <template #avatar>
                  <n-avatar round>📊</n-avatar>
                </template>
              </n-thing>
            </n-list-item>
          </n-list>
        </n-card>
      </n-gi>

      <n-gi>
        <n-card title="Features" hoverable>
          <n-list>
            <n-list-item>
              <n-space>
                <n-icon size="20" color="#18a058">
                  <checkmark-circle-outline />
                </n-icon>
                <n-text>Stream processing for large files</n-text>
              </n-space>
            </n-list-item>
            <n-list-item>
              <n-space>
                <n-icon size="20" color="#18a058">
                  <checkmark-circle-outline />
                </n-icon>
                <n-text>Advanced filtering and search</n-text>
              </n-space>
            </n-list-item>
            <n-list-item>
              <n-space>
                <n-icon size="20" color="#18a058">
                  <checkmark-circle-outline />
                </n-icon>
                <n-text>Data visualization</n-text>
              </n-space>
            </n-list-item>
          </n-list>
        </n-card>
      </n-gi>
    </n-grid>

    <n-card title="Recent Activity" hoverable>
      <n-empty v-if="!stats || stats.totalRecords === 0" description="No data imported yet">
        <template #extra>
          <n-button size="small" @click="$router.push('/import')">
            Import your first log file
          </n-button>
        </template>
      </n-empty>
      <n-list v-else>
        <n-list-item>
          <n-thing 
            title="Database Ready" 
            :description="`${stats.totalRecords} records available for querying`"
          >
            <template #avatar>
              <n-avatar round style="background-color: #18a058">
                <n-icon><database-outline /></n-icon>
              </n-avatar>
            </template>
          </n-thing>
        </n-list-item>
      </n-list>
    </n-card>
  </n-space>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import {
  NCard,
  NSpace,
  NP,
  NDivider,
  NStatistic,
  NTag,
  NText,
  NGrid,
  NGi,
  NButton,
  NList,
  NListItem,
  NThing,
  NAvatar,
  NIcon,
  NEmpty,
  useMessage
} from 'naive-ui'
import {
  CheckmarkCircleOutline,
  ServerOutline as DatabaseOutline
} from '@vicons/ionicons5'

// Import Wails runtime
import { GetStats } from '../../wailsjs/go/main/App'

const message = useMessage()
const stats = ref<any>(null)

const loadStats = async () => {
  try {
    const result = await GetStats()
    stats.value = result
  } catch (error) {
    console.error('Failed to load stats:', error)
    message.error('Failed to load statistics')
  }
}

onMounted(() => {
  loadStats()
})
</script>
