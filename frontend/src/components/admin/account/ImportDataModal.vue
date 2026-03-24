<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.dataImportTitle')"
    width="normal"
    close-on-click-outside
    @close="handleClose"
  >
    <form id="import-data-form" class="space-y-4" @submit.prevent="handleImport">
      <div class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.accounts.dataImportHint') }}
      </div>
      <div
        class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-600 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-400"
      >
        {{ t('admin.accounts.dataImportWarning') }}
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.dataImportFile') }}</label>
        <div
          class="flex items-center justify-between gap-3 rounded-lg border border-dashed border-gray-300 bg-gray-50 px-4 py-3 dark:border-dark-600 dark:bg-dark-800"
        >
          <div class="min-w-0">
            <div class="truncate text-sm text-gray-700 dark:text-dark-200">
              {{ fileName || t('admin.accounts.dataImportSelectFile') }}
            </div>
            <div class="text-xs text-gray-500 dark:text-dark-400">JSON (.json)</div>
          </div>
          <button type="button" class="btn btn-secondary shrink-0" @click="openFilePicker">
            {{ t('common.chooseFile') }}
          </button>
        </div>
        <input
          ref="fileInput"
          type="file"
          class="hidden"
          accept="application/json,.json"
          @change="handleFileChange"
        />
      </div>

      <div v-if="groups.length > 0" class="space-y-2">
        <div class="text-sm text-gray-700 dark:text-dark-200">
          {{ t('admin.accounts.importBindGroups') }}
        </div>
        <div class="text-xs text-gray-500 dark:text-dark-400">
          {{ t('admin.accounts.importBindGroupsHint') }}
        </div>
        <GroupSelector v-model="selectedGroupIDs" :groups="groups" />
      </div>

      <div
        v-if="result"
        class="space-y-2 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
      >
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.dataImportResult') }}
        </div>
        <div class="text-sm text-gray-700 dark:text-dark-300">
          {{ t('admin.accounts.dataImportResultSummary', result) }}
        </div>

        <div v-if="errorItems.length" class="mt-2">
          <div class="text-sm font-medium text-red-600 dark:text-red-400">
            {{ t('admin.accounts.dataImportErrors') }}
          </div>
          <div
            class="mt-2 max-h-48 overflow-auto rounded-lg bg-gray-50 p-3 font-mono text-xs dark:bg-dark-800"
          >
            <div v-for="(item, idx) in errorItems" :key="idx" class="whitespace-pre-wrap">
              {{ item.kind }} {{ item.name || item.proxy_key || '-' }} — {{ item.message }}
            </div>
          </div>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" type="button" :disabled="importing" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          class="btn btn-primary"
          type="submit"
          form="import-data-form"
          :disabled="importing"
        >
          {{ importing ? t('admin.accounts.dataImporting') : t('admin.accounts.dataImportButton') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AdminDataImportResult, AdminDataImportTask, AdminGroup } from '@/types'

interface Props {
  show: boolean
  groups?: AdminGroup[]
}

interface Emits {
  (e: 'close'): void
  (e: 'imported'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const appStore = useAppStore()

const importing = ref(false)
const file = ref<File | null>(null)
const result = ref<AdminDataImportResult | null>(null)
const selectedGroupIDs = ref<number[]>([])

const fileInput = ref<HTMLInputElement | null>(null)
const fileName = computed(() => file.value?.name || '')
const groups = computed(() => props.groups || [])
let pollTimer: ReturnType<typeof setTimeout> | null = null

const errorItems = computed(() => result.value?.errors || [])

watch(
  () => props.show,
  (open) => {
    if (open) {
      file.value = null
      result.value = null
      selectedGroupIDs.value = []
      if (fileInput.value) {
        fileInput.value.value = ''
      }
    }
  }
)

const openFilePicker = () => {
  fileInput.value?.click()
}

const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  file.value = target.files?.[0] || null
}

const handleClose = () => {
  if (importing.value) return
  emit('close')
}

const clearPollTimer = () => {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

const buildTaskSubtitle = (task: AdminDataImportTask) => {
  if (task.total > 0) {
    return `${task.current}/${task.total}`
  }
  return task.stage || task.status
}

const buildTaskMessage = (task: AdminDataImportTask) => {
  return task.message || task.stage || task.status
}

const scheduleTaskPoll = (taskID: string, toastID: string) => {
  clearPollTimer()
  pollTimer = setTimeout(async () => {
    try {
      const task = await adminAPI.accounts.getImportTask(taskID)
      appStore.updateToast(toastID, {
        title: t('admin.accounts.dataImportTitle'),
        subtitle: buildTaskSubtitle(task),
        message: buildTaskMessage(task),
        progress: task.progress
      })

      if (task.status === 'completed') {
        result.value = task.result || null
        const summary = task.result || {
          proxy_created: 0,
          proxy_reused: 0,
          proxy_failed: 0,
          account_created: 0,
          account_skipped: 0,
          account_failed: 0
        }
        const msgParams: Record<string, unknown> = {
          account_created: summary.account_created,
          account_skipped: summary.account_skipped,
          account_failed: summary.account_failed,
          proxy_created: summary.proxy_created,
          proxy_reused: summary.proxy_reused,
          proxy_failed: summary.proxy_failed,
        }
        appStore.updateToast(toastID, {
          type: summary.account_failed > 0 || summary.proxy_failed > 0 ? 'warning' : 'success',
          subtitle: buildTaskSubtitle(task),
          message: summary.account_failed > 0 || summary.proxy_failed > 0
            ? t('admin.accounts.dataImportCompletedWithErrors', msgParams)
            : t('admin.accounts.dataImportSuccess', msgParams),
          progress: 100,
          duration: 5000
        })
        emit('imported')
        clearPollTimer()
        return
      }

      if (task.status === 'failed') {
        appStore.updateToast(toastID, {
          type: 'error',
          subtitle: buildTaskSubtitle(task),
          message: task.message || t('admin.accounts.dataImportFailed'),
          progress: task.progress,
          duration: 6000
        })
        clearPollTimer()
        return
      }

      scheduleTaskPoll(taskID, toastID)
    } catch (error: any) {
      appStore.updateToast(toastID, {
        type: 'error',
        message: error?.message || t('admin.accounts.dataImportFailed'),
        duration: 6000
      })
      clearPollTimer()
    }
  }, 1200)
}

const handleImport = async () => {
  if (!file.value) {
    appStore.showError(t('admin.accounts.dataImportSelectFile'))
    return
  }

  importing.value = true
  const toastID = appStore.showToast('info', t('admin.accounts.dataImportUploading'), undefined, {
    title: t('admin.accounts.dataImportTitle'),
    subtitle: '0%',
    progress: 0
  })
  try {
    const task = await adminAPI.accounts.createImportTask({
      file: file.value,
      group_ids: selectedGroupIDs.value,
      skip_default_group_bind: true,
      onUploadProgress: (progress) => {
        appStore.updateToast(toastID, {
          title: t('admin.accounts.dataImportTitle'),
          subtitle: `${progress}%`,
          message: t('admin.accounts.dataImportUploading'),
          progress
        })
      }
    })
    appStore.updateToast(toastID, {
      type: 'info',
      title: t('admin.accounts.dataImportTitle'),
      subtitle: buildTaskSubtitle(task),
      message: buildTaskMessage(task),
      progress: task.progress
    })
    emit('close')
    scheduleTaskPoll(task.task_id, toastID)
  } catch (error: any) {
    const message = String(error?.response?.data?.message || error?.message || '')
    if (message.toLowerCase().includes('invalid import file') || message.toLowerCase().includes('unexpected token')) {
      appStore.showError(t('admin.accounts.dataImportParseFailed'))
    } else {
      appStore.showError(message || t('admin.accounts.dataImportFailed'))
    }
    appStore.hideToast(toastID)
  } finally {
    importing.value = false
  }
}

onUnmounted(() => {
  clearPollTimer()
})
</script>
