<template>
  <n-space vertical size="large">
    <n-card title="Query Logs" hoverable>
      <template #header-extra>
        <n-space>
          <n-button @click="showExplaination = !showExplaination" quaternary>
            <template #icon>
              <n-icon><information-circle-outline /></n-icon>
            </template>
            Explain
          </n-button>
          <n-button @click="executeQuery" type="primary" :loading="querying">
            <template #icon>
              <n-icon><search-outline /></n-icon>
            </template>
            Run Query
          </n-button>
          <n-button @click="exportReport" secondary>
            Export JSON report
          </n-button>
        </n-space>
      </template>

      <!-- Query Builder -->
      <n-space vertical size="large">
        <!-- Quick Filters -->
        <n-card title="Quick Filters" size="small">
          <n-space vertical>
            <n-space align="center" wrap>
              <n-select
                v-model:value="quick.level"
                :options="levelOptions"
                placeholder="Level"
                clearable
                style="width: 160px"
              />
              <n-select
                v-model:value="quick.service"
                :options="serviceOptions"
                placeholder="Service"
                clearable
                filterable
                tag
                style="min-width: 220px"
              />
              <n-input
                v-model:value="quick.text"
                placeholder="Search in raw/message"
                clearable
                style="min-width: 260px"
                @keyup.enter="applyQuickFilters"
              />
            </n-space>

            <n-space align="center" wrap>
              <n-select
                v-model:value="quick.timePreset"
                :options="timePresetOptions"
                placeholder="Time range"
                style="width: 200px"
              />
              <n-date-picker
                v-if="quick.timePreset === 'custom'"
                v-model:value="quick.timeRange"
                type="datetimerange"
                clearable
                style="min-width: 360px"
              />
              <n-button @click="applyQuickFilters" type="primary" secondary>
                Apply
              </n-button>
              <n-button @click="clearQuickFilters" secondary>
                Clear
              </n-button>
            </n-space>
          </n-space>
        </n-card>

        <!-- Filters -->
        <n-card title="Filters" size="small">
          <n-space vertical>
            <n-space v-for="(filter, index) in query.filters" :key="index" align="center">
              <n-select
                v-model:value="filter.field"
                :options="fieldOptions"
                placeholder="Field"
                style="width: 140px"
              />
              <n-select
                v-model:value="filter.type"
                :options="filterTypeOptions"
                placeholder="Operator"
                style="width: 140px"
                @update:value="onFilterTypeChange(index)"
              />
              <n-input
                v-model:value="filter.value"
                placeholder="Value"
                style="width: 220px"
              />
              <n-select
                v-if="filter.type === 'range'"
                v-model:value="filter.operator"
                :options="rangeOperatorOptions"
                placeholder="Range"
                style="width: 120px"
              />
              <n-button @click="removeFilter(index)" type="error" quaternary circle>
                <template #icon>
                  <n-icon><trash-outline /></n-icon>
                </template>
              </n-button>
            </n-space>
            <n-button @click="addFilter" dashed block>
              <template #icon>
                <n-icon><add-outline /></n-icon>
              </template>
              Add Filter
            </n-button>
          </n-space>
        </n-card>

        <!-- Sort Options -->
        <n-card title="Sort Options" size="small">
          <n-space>
            <n-select
              v-model:value="query.sortBy"
              :options="fieldOptions"
              placeholder="Sort by"
              style="width: 150px"
            />
            <n-switch v-model:value="query.sortDesc">
              <template #checked>Descending</template>
              <template #unchecked>Ascending</template>
            </n-switch>
          </n-space>
        </n-card>

        <!-- Limit Options -->
        <n-card title="Limit Options" size="small">
          <n-space>
            <n-input-number
              v-model:value="query.limit"
              :min="1"
              :max="10000"
              placeholder="Limit"
              style="width: 120px"
            />
            <n-input-number
              v-model:value="query.offset"
              :min="0"
              placeholder="Offset"
              style="width: 120px"
            />
          </n-space>
        </n-card>
      </n-space>
    </n-card>

    <!-- Query Explanation -->
    <n-card v-if="showExplaination" title="Query Explanation" size="small">
      <n-code :code="queryExplanation" language="sql" show-line-numbers />
    </n-card>

    <!-- Results -->
    <n-card title="Results" hoverable>
      <template #header-extra>
        <n-space>
          <n-tag v-if="queryResult" type="info">
            {{ queryResult.total }} records
          </n-tag>
          <n-tag v-if="queryResult" type="success">
            {{ formatDuration(queryResult.took) }}
          </n-tag>
        </n-space>
      </template>

      <n-spin :show="querying">
        <div v-if="!queryResult && !querying">
          <n-empty description="No query results yet">
            <template #extra>
              <n-button @click="executeQuery" type="primary">
                Run your first query
              </n-button>
            </template>
          </n-empty>
        </div>

        <div v-else-if="queryResult">
          <n-card title="Timeline" size="small" style="margin-bottom: 16px">
            <n-space align="center" wrap style="margin-bottom: 12px">
              <n-select
                v-model:value="timelineBucketMs"
                :options="bucketOptions"
                style="width: 180px"
              />
              <n-button @click="loadTimeline" secondary :loading="timelineLoading">Refresh</n-button>
            </n-space>
            <v-chart v-if="timelineOption" :option="timelineOption" autoresize style="height: 240px" />
          </n-card>

          <!-- Aggregations -->
          <n-space
            v-if="queryResult.aggregations && Object.keys(queryResult.aggregations).length > 0"
            vertical
          >
            <template v-for="(value, key) in queryResult.aggregations" :key="String(key)">
              <n-statistic :label="String(key)" :value="String(value)" />
            </template>
          </n-space>

          <!-- Results Table -->
          <n-data-table
            :columns="tableColumns"
            :data="queryResult.records"
            :pagination="pagination"
            :row-key="(row: any) => row.id"
            :max-height="400"
            remote
            striped
            virtual-scroll
            @update:sorter="handleSorterChange"
          />
        </div>
      </n-spin>
    </n-card>
  </n-space>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, h, watch } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import {
  NCard,
  NSpace,
  NButton,
  NIcon,
  NSelect,
  NInput,
  NInputNumber,
  NSwitch,
  NTag,
  NDatePicker,
  NSpin,
  NEmpty,
  NCode,
  NDataTable,
  NStatistic,
  useMessage
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import VChart from 'vue-echarts'
import {
  InformationCircleOutline,
  SearchOutline,
  TrashOutline,
  AddOutline
} from '@vicons/ionicons5'

// Import Wails runtime
import { Query, ExplainQuery, GetTimeline, ExportReport } from '../../wailsjs/go/main/App'
import { domain } from '../../wailsjs/go/models'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent])

const message = useMessage()

const querying = ref(false)
const showExplaination = ref(false)
const queryExplanation = ref('')
const queryResult = ref<any>(null)

const withTimeout = async <T,>(p: Promise<T>, ms: number, label: string): Promise<T> => {
  let timeoutId: any
  const timeout = new Promise<T>((_, reject) => {
    timeoutId = setTimeout(() => reject(new Error(`${label} timed out after ${ms}ms`)), ms)
  })
  try {
    return await Promise.race([p, timeout])
  } finally {
    clearTimeout(timeoutId)
  }
}

const timeline = ref<any[]>([])
const timelineLoading = ref(false)
const timelineBucketMs = ref<number>(60_000)
const bucketOptions = [
  { label: '1 minute', value: 60_000 },
  { label: '5 minutes', value: 300_000 },
  { label: '15 minutes', value: 900_000 },
  { label: '1 hour', value: 3_600_000 }
]

const timelineOption = computed(() => {
  if (!timeline.value || timeline.value.length === 0) return null
  const x = timeline.value.map((p: any) => new Date(p.bucketStart).toLocaleString())
  const y = timeline.value.map((p: any) => p.count)
  return {
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 20, top: 20, bottom: 40 },
    xAxis: { type: 'category', data: x, axisLabel: { rotate: 30 } },
    yAxis: { type: 'value' },
    series: [{ type: 'line', smooth: true, data: y }]
  }
})

const query = ref<any>({
  filters: [] as any[],
  sortBy: 'timestamp',
  sortDesc: true,
  limit: 100,
  offset: 0
})

const quick = ref<{ level: string | null; service: string | null; text: string; timePreset: string; timeRange: [number, number] | null }>({
  level: null,
  service: null,
  text: '',
  timePreset: 'all',
  timeRange: null
})

const levelOptions = [
  { label: 'ERROR', value: 'ERROR' },
  { label: 'WARN', value: 'WARN' },
  { label: 'INFO', value: 'INFO' },
  { label: 'DEBUG', value: 'DEBUG' }
]

const timePresetOptions = [
  { label: 'All time', value: 'all' },
  { label: 'Last 15 minutes', value: '15m' },
  { label: 'Last 1 hour', value: '1h' },
  { label: 'Last 24 hours', value: '24h' },
  { label: 'Last 7 days', value: '7d' },
  { label: 'Custom range', value: 'custom' }
]

const serviceOptions = computed(() => {
  const services = new Set<string>()
  const records = queryResult.value?.records || []
  for (const r of records) {
    if (r?.service) services.add(String(r.service))
  }
  return Array.from(services)
    .sort()
    .slice(0, 200)
    .map((s) => ({ label: s, value: s }))
})

const fieldOptions = [
  { label: 'ID', value: 'id' },
  { label: 'Timestamp', value: 'timestamp' },
  { label: 'Level', value: 'level' },
  { label: 'Message', value: 'message' },
  { label: 'Service', value: 'service' },
  { label: 'Raw', value: 'raw' }
]

const filterTypeOptions = [
  { label: 'Equals', value: 'equality' },
  { label: 'Not Equals', value: 'exclusion' },
  { label: 'Contains', value: 'contains' },
  { label: 'Regex', value: 'regexp' },
  { label: 'Range', value: 'range' }
]

const rangeOperatorOptions = [
  { label: '>', value: 'gt' },
  { label: '>=', value: 'gte' },
  { label: '<', value: 'lt' },
  { label: '<=', value: 'lte' }
]

const tableColumns: DataTableColumns<any> = [
  {
    title: 'ID',
    key: 'id',
    width: 110,
    ellipsis: {
      tooltip: true
    }
  },
  {
    title: 'Timestamp',
    key: 'timestamp',
    width: 180,
    sorter: 'default',
    render: (row: any) => new Date(row.timestamp).toLocaleString()
  },
  {
    title: 'Level',
    key: 'level',
    width: 80,
    sorter: 'default',
    render: (row: any) => {
      const level = row.level?.toUpperCase()
      const type = level === 'ERROR' ? 'error' : level === 'WARN' ? 'warning' : 'info'
      return h(NTag, { type, size: 'small' }, { default: () => String(level || '') })
    }
  },
  {
    title: 'Service',
    key: 'service',
    width: 120,
    sorter: 'default',
    ellipsis: {
      tooltip: true
    }
  },
  {
    title: 'Message',
    key: 'message',
    sorter: 'default',
    ellipsis: {
      tooltip: true
    }
  }
]

const pagination = ref({
  page: 1,
  pageSize: 50,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [25, 50, 100, 200],
  onUpdatePage: (page: number) => {
    pagination.value.page = page
    query.value.offset = (page - 1) * pagination.value.pageSize
    query.value.limit = pagination.value.pageSize
    executeQuery()
  },
  onUpdatePageSize: (pageSize: number) => {
    pagination.value.pageSize = pageSize
    pagination.value.page = 1
    query.value.offset = 0
    query.value.limit = pageSize
    executeQuery()
  }
})

const buildQuickFilters = () => {
  const filters: any[] = []

  if (quick.value.level) {
    filters.push({ type: 'equality', field: 'level', value: quick.value.level })
  }

  if (quick.value.service) {
    filters.push({ type: 'equality', field: 'service', value: quick.value.service })
  }

  if (quick.value.text && quick.value.text.trim() !== '') {
    // Search in raw by default (it's always present). Users can add more advanced filters if needed.
    filters.push({ type: 'contains', field: 'raw', value: quick.value.text.trim() })
  }

  const now = Date.now()
  let start: number | null = null
  let end: number | null = null

  switch (quick.value.timePreset) {
    case '15m':
      start = now - 15 * 60 * 1000
      end = now
      break
    case '1h':
      start = now - 60 * 60 * 1000
      end = now
      break
    case '24h':
      start = now - 24 * 60 * 60 * 1000
      end = now
      break
    case '7d':
      start = now - 7 * 24 * 60 * 60 * 1000
      end = now
      break
    case 'custom':
      if (quick.value.timeRange && quick.value.timeRange.length === 2) {
        start = quick.value.timeRange[0]
        end = quick.value.timeRange[1]
      }
      break
    default:
      break
  }

  if (start != null) {
    filters.push({ type: 'range', field: 'timestamp', operator: 'gte', value: start })
  }
  if (end != null) {
    filters.push({ type: 'range', field: 'timestamp', operator: 'lte', value: end })
  }

  return filters
}

const getEffectiveQuery = () => {
  return {
    ...query.value,
    filters: [...(query.value.filters || []), ...buildQuickFilters()]
  }
}

const applyQuickFilters = () => {
  pagination.value.page = 1
  query.value.offset = 0
  executeQuery()
}

const clearQuickFilters = () => {
  quick.value.level = null
  quick.value.service = null
  quick.value.text = ''
  quick.value.timePreset = 'all'
  quick.value.timeRange = null
  applyQuickFilters()
}

const loadTimeline = async () => {
  timelineLoading.value = true
  try {
    const q = getEffectiveQuery()
    const req = domain.TimelineRequest.createFrom({
      filters: q.filters,
      bucketMs: timelineBucketMs.value
    })
    const points = await withTimeout(GetTimeline(req), 15000, 'Timeline')
    timeline.value = points || []
  } catch (err) {
    console.error('GetTimeline failed:', err)
    message.error(`Timeline failed: ${err}`)
  } finally {
    timelineLoading.value = false
  }
}

const exportReport = async () => {
  try {
    const path = await ExportReport(getEffectiveQuery(), timelineBucketMs.value)
    if (!path) {
      message.info('Export canceled')
      return
    }
    message.success(`Report saved: ${path}`)
  } catch (err) {
    console.error('ExportReport failed:', err)
    message.error(`Export failed: ${err}`)
  }
}

const addFilter = () => {
  query.value.filters.push({
    field: '',
    type: 'equality',
    value: '',
    operator: 'eq'
  })
}

const removeFilter = (index: number) => {
  query.value.filters.splice(index, 1)
}

const onFilterTypeChange = (index: number) => {
  const filter = query.value.filters[index]
  if (filter.type !== 'range') {
    filter.operator = ''
  }
}

const executeQuery = async () => {
  querying.value = true
  
  try {
    const result = await withTimeout(Query(getEffectiveQuery()), 30000, 'Query')
    queryResult.value = result
    pagination.value.itemCount = Number(result.total || 0)
    loadTimeline()
    message.success(`Query executed successfully: ${result.total} records`)
  } catch (error) {
    console.error('Query failed:', error)
    message.error(`Query failed: ${error}`)
  } finally {
    querying.value = false
  }
}

const handleSorterChange = (sorter: any) => {
  if (!sorter || !sorter.columnKey || sorter.order === false) {
    query.value.sortBy = 'timestamp'
    query.value.sortDesc = true
  } else {
    query.value.sortBy = String(sorter.columnKey)
    query.value.sortDesc = sorter.order === 'descend'
  }

  pagination.value.page = 1
  query.value.offset = 0
  executeQuery()
}

const explainQuery = async () => {
  try {
    const explanation = await ExplainQuery(getEffectiveQuery())
    queryExplanation.value = explanation
  } catch (error) {
    console.error('Explain failed:', error)
    message.error(`Explain failed: ${error}`)
  }
}

const formatDuration = (milliseconds: number) => {
  if (milliseconds < 1000) return `${Math.round(milliseconds)}ms`
  const seconds = milliseconds / 1000
  if (seconds < 60) return `${Math.round(seconds * 100) / 100}s`
  return `${Math.round((seconds / 60) * 100) / 100}m`
}

// Watch for showExplaination changes
watch(showExplaination, (newValue) => {
  if (newValue) {
    explainQuery()
  }
})

onMounted(() => {
  executeQuery()
})
</script>
