<template>
  <n-space vertical size="large">
    <n-card title="Import Log Files" hoverable>
      <template #header-extra>
        <n-tag :type="importStatus === 'success' ? 'success' : 'info'">
          {{ importStatusText }}
        </n-tag>
      </template>
      
      <n-steps :current="currentStep" :status="stepStatus">
        <n-step title="Select File" description="Choose log file to import" />
        <n-step title="Configure Parser" description="Set up parsing options" />
        <n-step title="Import" description="Import and process data" />
      </n-steps>

      <n-divider />

      <!-- Step 1: File Selection -->
      <div v-if="currentStep === 1">
        <n-space vertical size="large">
          <n-card size="small" style="width: 100%">
            <n-space vertical size="large">
              <div
                style="
                  border: 1px dashed rgba(255,255,255,0.25);
                  border-radius: 8px;
                  padding: 32px 20px;
                  text-align: center;
                "
              >
                <div style="margin-bottom: 16px">
                  <n-icon size="64" :depth="3">
                    <archive-outline />
                  </n-icon>
                </div>
                <n-text style="font-size: 18px; font-weight: 500; display: block;">
                  Drag & drop a log file here
                </n-text>
                <n-text depth="3" style="margin: 12px 0 0 0; display: block;">
                  Or choose a file using the button below
                </n-text>
                <n-text depth="3" style="margin: 8px 0 0 0; display: block;">
                  Supported: .log .txt .json .ndjson .csv
                </n-text>
              </div>

              <n-space>
                <n-button type="primary" @click="chooseFile">
                  Choose file
                </n-button>
                <n-button v-if="selectedFilePath" @click="clearFile" secondary>
                  Clear
                </n-button>
              </n-space>

              <n-space v-if="selectedFilePath" vertical>
                <n-text>Selected file:</n-text>
                <n-text strong>{{ selectedFilePath }}</n-text>
              </n-space>
            </n-space>
          </n-card>

          <n-space>
            <n-button 
              type="primary" 
              @click="nextStep"
              :disabled="!selectedFilePath"
            >
              Next
            </n-button>
          </n-space>
        </n-space>
      </div>

      <!-- Step 2: Parser Configuration -->
      <div v-if="currentStep === 2">
        <n-space vertical size="large">
          <n-form :model="parserConfig" label-placement="left" label-width="auto">
            <n-form-item label="Parser Type">
              <n-select
                v-model:value="parserConfig.type"
                :options="parserTypeOptions"
                placeholder="Select parser type"
                @update:value="onParserTypeChange"
              />
            </n-form-item>

            <n-form-item v-if="parserConfig.type === 'regex'" label="Regex Pattern">
              <n-input
                v-model:value="parserConfig.pattern"
                placeholder="Enter regex pattern"
                type="textarea"
                :autosize="{ minRows: 2, maxRows: 4 }"
              />
            </n-form-item>

            <n-form-item v-if="parserConfig.type === 'regex'" label="Time Format">
              <n-input
                v-model:value="parserConfig.timeFormat"
                placeholder="e.g., 2006-01-02 15:04:05"
              />
            </n-form-item>

            <n-form-item label="Auto Detect">
              <n-switch v-model:value="autoDetect" />
              <n-text depth="3" style="margin-left: 8px">
                Automatically detect parser type from file content
              </n-text>
            </n-form-item>
          </n-form>

          <n-space>
            <n-button @click="prevStep">Previous</n-button>
            <n-button type="primary" @click="nextStep" :disabled="!isParserConfigValid">
              Next
            </n-button>
          </n-space>
        </n-space>
      </div>

      <!-- Step 3: Import -->
      <div v-if="currentStep === 3">
        <n-space vertical size="large">
          <n-alert type="info" title="Ready to Import">
            <n-p>
              File: {{ selectedFilePath }}<br>
              Parser: {{ parserConfig.type }}<br>
              {{ autoDetect ? 'Auto-detection enabled' : 'Manual configuration' }}
            </n-p>
          </n-alert>

          <n-progress
            type="line"
            :percentage="importProgress"
            :status="importStatus === 'error' ? 'error' : 'default'"
            :show-indicator="true"
          />

          <n-space v-if="importResult">
            <n-statistic label="Total Records" :value="importResult.totalRecords" />
            <n-statistic label="Processed" :value="importResult.processed" />
            <n-statistic label="Duration" :value="formatDuration(importResult.duration)" />
          </n-space>

          <n-space>
            <n-button @click="prevStep" :disabled="importing">Previous</n-button>
            <n-button 
              type="primary" 
              @click="startImport" 
              :loading="importing"
              :disabled="importing"
            >
              {{ importing ? 'Importing...' : 'Start Import' }}
            </n-button>
            <n-button 
              v-if="importResult" 
              @click="resetImport"
            >
              Import Another File
            </n-button>
          </n-space>
        </n-space>
      </div>
    </n-card>
  </n-space>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  NCard,
  NSpace,
  NTag,
  NSteps,
  NStep,
  NDivider,
  NIcon,
  NText,
  NP,
  NButton,
  NForm,
  NFormItem,
  NSelect,
  NInput,
  NSwitch,
  NAlert,
  NProgress,
  NStatistic,
  useMessage,
} from 'naive-ui'
import { ArchiveOutline } from '@vicons/ionicons5'

// Import Wails runtime
import { AutoImportFile, ImportFile, SelectLogFile } from '../../wailsjs/go/main/App'
import { OnFileDrop, OnFileDropOff } from '../../wailsjs/runtime/runtime'

const router = useRouter()
const message = useMessage()

const currentStep = ref(1)
const stepStatus = ref<'process' | 'finish' | 'error' | 'wait'>('process')
const importStatus = ref<'idle' | 'importing' | 'success' | 'error'>('idle')
const importStatusText = computed(() => {
  switch (importStatus.value) {
    case 'idle': return 'Ready'
    case 'importing': return 'Importing'
    case 'success': return 'Complete'
    case 'error': return 'Error'
    default: return 'Ready'
  }
})

const selectedFilePath = ref<string>('')
const autoDetect = ref(true)
const importing = ref(false)
const importProgress = ref(0)
const importResult = ref<any>(null)

const parserConfig = ref({
  type: '',
  pattern: '',
  timeFormat: '',
  fields: {} as Record<string, string>
})

const parserTypeOptions = [
  { label: 'Plain Text', value: 'plain' },
  { label: 'JSON/NDJSON', value: 'json' },
  { label: 'Regex', value: 'regex' },
  { label: 'Grok', value: 'grok' }
]

const isParserConfigValid = computed(() => {
  if (autoDetect.value) return true
  if (!parserConfig.value.type) return false
  if (parserConfig.value.type === 'regex' && !parserConfig.value.pattern) return false
  return true
})

const chooseFile = async () => {
  try {
    const path = await SelectLogFile()
    if (!path) return
    selectedFilePath.value = path
  } catch (error) {
    console.error('SelectLogFile failed:', error)
    message.error(`Failed to select file: ${error}`)
  }
}

const clearFile = () => {
  selectedFilePath.value = ''
}

const onParserTypeChange = (value: string) => {
  parserConfig.value.type = value
}

const nextStep = () => {
  if (currentStep.value < 3) {
    currentStep.value++
  }
}

const prevStep = () => {
  if (currentStep.value > 1) {
    currentStep.value--
  }
}

const startImport = async () => {
  if (!selectedFilePath.value) return

  importing.value = true
  importStatus.value = 'importing'
  importProgress.value = 0
  stepStatus.value = 'process'

  try {
    let result
    
    if (autoDetect.value) {
      result = await AutoImportFile(selectedFilePath.value)
    } else {
      result = await ImportFile(selectedFilePath.value, parserConfig.value)
    }

    importResult.value = result
    importProgress.value = 100
    importStatus.value = 'success'
    stepStatus.value = 'finish'
    
    message.success(`Successfully imported ${result.processed} records`)
  } catch (error) {
    console.error('Import failed:', error)
    importStatus.value = 'error'
    stepStatus.value = 'error'
    message.error(`Import failed: ${error}`)
  } finally {
    importing.value = false
  }
}

const resetImport = () => {
  currentStep.value = 1
  stepStatus.value = 'process'
  importStatus.value = 'idle'
  selectedFilePath.value = ''
  importResult.value = null
  importProgress.value = 0
  parserConfig.value = {
    type: '',
    pattern: '',
    timeFormat: '',
    fields: {}
  }
}

const formatFileSize = (bytes: number) => {
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  if (bytes === 0) return '0 Bytes'
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i]
}

const formatDuration = (nanoseconds: number) => {
  const seconds = nanoseconds / 1000000000
  if (seconds < 1) return `${Math.round(nanoseconds / 1000000)}ms`
  if (seconds < 60) return `${Math.round(seconds * 100) / 100}s`
  return `${Math.round(seconds / 60 * 100) / 100}m`
}

onMounted(() => {
  OnFileDrop((_x, _y, paths) => {
    if (paths && paths.length > 0) {
      selectedFilePath.value = paths[0]
      message.success('File selected')
    }
  }, false)
})

onUnmounted(() => {
  OnFileDropOff()
})
</script>
