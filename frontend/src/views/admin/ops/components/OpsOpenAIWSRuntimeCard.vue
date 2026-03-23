<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { opsAPI, type OpenAIWSRuntimeResponse } from '@/api/admin/ops'
import EmptyState from '@/components/common/EmptyState.vue'

interface Props {
  platformFilter?: string
  refreshToken?: number
  active?: boolean
}

const props = defineProps<Props>()
const { t } = useI18n()

const loading = ref(false)
const errorMessage = ref('')
const runtime = ref<OpenAIWSRuntimeResponse | null>(null)

const fallbackRows = computed(() =>
  Object.entries(runtime.value?.fallback_reason_counts ?? {})
    .sort((a, b) => b[1] - a[1])
    .slice(0, 4)
)

const hasAnyRuntimeData = computed(() => {
  const current = runtime.value
  if (!current) return false
  return (
    current.acquire_total > 0 ||
    current.reuse_total > 0 ||
    current.create_total > 0 ||
    current.queue_wait_ms_total > 0 ||
    current.conn_pick_ms_total > 0 ||
    current.scale_up_total > 0 ||
    current.scale_down_total > 0 ||
    current.prewarm_success_total > 0 ||
    current.prewarm_fallback_total > 0 ||
    (current.relay?.incomplete_close_total ?? 0) > 0 ||
    (current.circuits?.open_proxy_count ?? 0) > 0 ||
    (current.circuits?.open_account_count ?? 0) > 0
  )
})

const reuseRate = computed(() => {
  const current = runtime.value
  if (!current || current.acquire_total <= 0) return null
  return current.reuse_total / current.acquire_total
})

async function fetchData() {
  if (props.active === false) return
  loading.value = true
  errorMessage.value = ''
  try {
    runtime.value = await opsAPI.getOpenAIWSRuntime(props.platformFilter)
  } catch (err: any) {
    runtime.value = null
    errorMessage.value = err?.response?.data?.detail || err?.message || t('common.loadFailed')
  } finally {
    loading.value = false
  }
}

function formatPercent(value?: number | null, digits = 1) {
  if (value == null || Number.isNaN(value)) return '-'
  return `${(value * 100).toFixed(digits)}%`
}

function formatInt(value?: number | null) {
  if (value == null || Number.isNaN(value)) return '-'
  return Intl.NumberFormat().format(value)
}

watch(() => [props.platformFilter, props.refreshToken, props.active], fetchData)
onMounted(() => {
  if (props.active !== false) {
    fetchData()
  }
})
</script>

<template>
  <div class="flex h-full flex-col rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700">
    <div class="mb-4 flex items-center justify-between">
      <div>
        <h3 class="text-sm font-bold text-gray-900 dark:text-white">OpenAI WS Runtime</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Pool reuse, queue wait, and prewarm fallback visibility</p>
      </div>
    </div>

    <div v-if="errorMessage" class="mb-4 rounded-2xl bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-900/20 dark:text-red-400">
      {{ errorMessage }}
    </div>

    <div class="grid grid-cols-2 gap-3 text-sm">
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Reuse Rate</div>
        <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">{{ formatPercent(reuseRate) }}</div>
      </div>
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Queue Wait</div>
        <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">{{ formatInt(runtime?.queue_wait_ms_total) }}ms</div>
      </div>
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Conn Pick</div>
        <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">{{ formatInt(runtime?.conn_pick_ms_total) }}ms</div>
      </div>
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Prewarm</div>
        <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">{{ formatInt(runtime?.prewarm_success_total) }}/{{ formatInt(runtime?.prewarm_fallback_total) }}</div>
      </div>
    </div>

    <div class="mt-4 grid grid-cols-2 gap-3 text-sm xl:grid-cols-4">
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Acquire</div>
        <div class="mt-1 text-base font-bold text-gray-900 dark:text-white">{{ formatInt(runtime?.acquire_total) }}</div>
      </div>
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Reuse / Create</div>
        <div class="mt-1 text-base font-bold text-gray-900 dark:text-white">{{ formatInt(runtime?.reuse_total) }} / {{ formatInt(runtime?.create_total) }}</div>
      </div>
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Scale</div>
        <div class="mt-1 text-base font-bold text-gray-900 dark:text-white">+{{ formatInt(runtime?.scale_up_total) }} / -{{ formatInt(runtime?.scale_down_total) }}</div>
      </div>
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Retry</div>
        <div class="mt-1 text-base font-bold text-gray-900 dark:text-white">{{ formatInt(runtime?.retry?.retry_attempts_total) }}</div>
      </div>
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Incomplete</div>
        <div class="mt-1 text-base font-bold text-gray-900 dark:text-white">{{ formatInt(runtime?.relay?.incomplete_close_total) }}</div>
      </div>
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Open Circuits</div>
        <div class="mt-1 text-base font-bold text-gray-900 dark:text-white">
          P{{ formatInt(runtime?.circuits?.open_proxy_count) }} / A{{ formatInt(runtime?.circuits?.open_account_count) }}
        </div>
      </div>
    </div>

    <div class="mt-4 min-h-0 flex-1 overflow-auto">
      <div v-if="loading" class="flex h-full items-center justify-center text-sm text-gray-400">{{ t('common.loading') }}</div>
      <div v-else class="space-y-4">
        <div
          class="rounded-2xl border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:border-dark-700 dark:bg-dark-700 dark:text-gray-300"
        >
          <template v-if="hasAnyRuntimeData">
            Showing current OpenAI WebSocket pool activity. Fallback reasons appear below when they occur.
          </template>
          <template v-else>
            No OpenAI WS runtime activity recorded yet.
          </template>
        </div>

        <div>
          <div class="mb-2 text-[11px] font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Fallback Reasons</div>
          <EmptyState
            v-if="fallbackRows.length === 0"
            :title="t('common.noData')"
            description="No WS fallback reasons recorded yet."
          />
          <div v-else class="space-y-2">
            <div
              v-for="[reason, count] in fallbackRows"
              :key="reason"
              class="flex items-center justify-between rounded-2xl bg-gray-50 px-3 py-2 text-xs dark:bg-dark-700"
            >
              <span class="truncate pr-3 text-gray-600 dark:text-gray-300">{{ reason }}</span>
              <span class="font-semibold text-gray-900 dark:text-white">{{ formatInt(count) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
