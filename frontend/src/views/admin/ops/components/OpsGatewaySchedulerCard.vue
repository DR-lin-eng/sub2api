<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { opsAPI, type GatewaySchedulerRuntimeEntry, type GatewaySchedulerRuntimeResponse } from '@/api/admin/ops'
import EmptyState from '@/components/common/EmptyState.vue'

interface Props {
  platformFilter?: string
  groupIdFilter?: number | null
  refreshToken?: number
  active?: boolean
  summaryLimit?: number
}

const props = defineProps<Props>()
const { t } = useI18n()

const loading = ref(false)
const errorMessage = ref('')
const runtime = ref<GatewaySchedulerRuntimeResponse | null>(null)

const rows = computed<GatewaySchedulerRuntimeEntry[]>(() => runtime.value?.items?.slice(0, 6) ?? [])
const cacheHitRate = computed(() => {
  const transport = runtime.value?.transport
  if (!transport) return null
  const total = transport.cache_hit_total + transport.cache_miss_total
  if (total <= 0) return null
  return transport.cache_hit_total / total
})

async function fetchData() {
  if (props.active === false) return
  loading.value = true
  errorMessage.value = ''
  try {
    runtime.value = await opsAPI.getGatewaySchedulerRuntime(props.platformFilter, props.groupIdFilter ?? null, props.summaryLimit ?? 6)
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

function formatMs(value?: number | null) {
  if (value == null || Number.isNaN(value)) return '-'
  return `${Math.round(value)}ms`
}

function formatTs(value?: string | null) {
  if (!value) return '-'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return '-'
  return parsed.toLocaleTimeString()
}

watch(() => [props.platformFilter, props.groupIdFilter, props.refreshToken, props.active, props.summaryLimit], fetchData)
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
        <h3 class="text-sm font-bold text-gray-900 dark:text-white">Gateway Scheduler</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Generic path runtime selection and transport reuse</p>
      </div>
      <div class="text-right text-xs text-gray-500 dark:text-gray-400">
        <div>{{ runtime?.active_scheduling_accounts ?? 0 }} scheduling</div>
        <div>{{ runtime?.pool_accounts_total ?? 0 }} in pool</div>
        <div>{{ runtime?.metrics?.runtime_stats_account_count ?? 0 }} tracked</div>
        <div>{{ runtime?.transport?.active_clients ?? 0 }} clients</div>
      </div>
    </div>

    <div v-if="errorMessage" class="mb-4 rounded-2xl bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-900/20 dark:text-red-400">
      {{ errorMessage }}
    </div>

    <div class="grid grid-cols-2 gap-3 text-sm">
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Sticky Hit</div>
        <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">{{ formatPercent(runtime?.metrics?.sticky_hit_ratio) }}</div>
      </div>
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Scheduler Latency</div>
        <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">{{ formatMs(runtime?.metrics?.scheduler_latency_ms_avg) }}</div>
      </div>
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Transport Hit</div>
        <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">{{ formatPercent(cacheHitRate) }}</div>
      </div>
      <div class="rounded-2xl bg-gray-50 p-3 dark:bg-dark-700">
        <div class="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Switch Rate</div>
        <div class="mt-1 text-lg font-bold text-gray-900 dark:text-white">{{ formatPercent(runtime?.metrics?.account_switch_rate) }}</div>
      </div>
    </div>

    <div class="mt-4 min-h-0 flex-1 overflow-auto">
      <div v-if="loading" class="flex h-full items-center justify-center text-sm text-gray-400">{{ t('common.loading') }}</div>
      <EmptyState
        v-else-if="rows.length === 0"
        :title="t('common.noData')"
        description="No scheduler runtime data yet."
      />
      <table v-else class="min-w-full text-left text-xs">
        <thead class="text-gray-500 dark:text-gray-400">
          <tr>
            <th class="pb-2 pr-2 font-medium">Account</th>
            <th class="pb-2 pr-2 font-medium">Load</th>
            <th class="pb-2 pr-2 font-medium">TTFT</th>
            <th class="pb-2 pr-2 font-medium">Err</th>
            <th class="pb-2 pr-2 font-medium">Hits</th>
            <th class="pb-2 font-medium">Seen</th>
          </tr>
        </thead>
        <tbody class="text-gray-700 dark:text-gray-200">
          <tr v-for="row in rows" :key="row.account_id" class="border-t border-gray-100 dark:border-dark-700">
            <td class="py-2 pr-2">
              <div class="font-semibold">#{{ row.account_id }}</div>
              <div class="text-[11px] text-gray-500 dark:text-gray-400">{{ row.platform }}</div>
            </td>
            <td class="py-2 pr-2">{{ row.current_concurrency }}/{{ Math.max(row.current_concurrency, 0) + Math.max(0, row.waiting_count) }} · {{ row.load_rate }}%</td>
            <td class="py-2 pr-2">{{ formatMs(row.ttft_ewma_ms) }}</td>
            <td class="py-2 pr-2">{{ formatPercent(row.error_rate_ewma, 0) }}</td>
            <td class="py-2 pr-2">{{ row.sticky_hits }}/{{ row.load_balance_hits }}</td>
            <td class="py-2">{{ formatTs(row.last_selected_at) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
