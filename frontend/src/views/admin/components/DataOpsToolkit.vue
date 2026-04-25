<template>
  <div class="space-y-6">
    <div class="card overflow-hidden border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <div class="border-b border-gray-200 bg-gradient-to-r from-amber-50 via-white to-cyan-50 px-6 py-5 dark:border-dark-700 dark:from-dark-900 dark:via-dark-900 dark:to-dark-800">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ batchOpsT('title') }}</h3>
            <p class="mt-1 max-w-3xl text-sm text-gray-600 dark:text-gray-400">
              {{ batchOpsT('description') }}
            </p>
          </div>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="referenceLoading"
            @click="loadReferenceData"
          >
            {{ referenceLoading ? t('common.loading') : batchOpsT('refreshReferenceData') }}
          </button>
        </div>
      </div>

      <div class="grid gap-5 p-6">
        <section class="rounded-2xl border border-gray-200 bg-gray-50/80 p-5 dark:border-dark-700 dark:bg-dark-800/60">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div>
              <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ batchOpsT('ungrouped.title') }}</h4>
              <p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
                {{ batchOpsT('ungrouped.description') }}
              </p>
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {{ batchOpsT('ungrouped.recentPreview', { count: ungroupedPreview?.length ?? 0 }) }}
            </div>
          </div>

          <div class="grid gap-3 md:grid-cols-4">
            <select v-model="ungroupedForm.groupId" class="input w-full">
              <option value="">{{ batchOpsT('ungrouped.selectTargetGroup') }}</option>
              <option v-for="group in groups" :key="group.id" :value="String(group.id)">
                #{{ group.id }} {{ group.name }} ({{ group.platform }})
              </option>
            </select>
            <select v-model="ungroupedForm.platform" class="input w-full">
              <option value="">{{ batchOpsT('ungrouped.allPlatforms') }}</option>
              <option v-for="platform in platformOptions" :key="platform.value" :value="platform.value">
                {{ platform.label }}
              </option>
            </select>
            <select v-model="ungroupedForm.status" class="input w-full">
              <option value="">{{ batchOpsT('status.all') }}</option>
              <option value="active">{{ batchOpsT('status.active') }}</option>
              <option value="inactive">{{ batchOpsT('status.inactive') }}</option>
              <option value="error">{{ batchOpsT('status.error') }}</option>
            </select>
            <input v-model.trim="ungroupedForm.search" class="input w-full" :placeholder="batchOpsT('ungrouped.searchPlaceholder')" />
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading.ungroupedPreview" @click="handleUngroupedPreview">
              {{ loading.ungroupedPreview ? batchOpsT('ungrouped.previewing') : batchOpsT('ungrouped.preview') }}
            </button>
            <button type="button" class="btn btn-primary btn-sm" :disabled="loading.ungroupedApply || !ungroupedForm.groupId" @click="handleUngroupedApply">
              {{ loading.ungroupedApply ? batchOpsT('ungrouped.applying') : batchOpsT('ungrouped.apply') }}
            </button>
          </div>

          <div v-if="ungroupedPreview" class="mt-4 rounded-xl border border-dashed border-gray-300 bg-white p-4 dark:border-dark-600 dark:bg-dark-900/40">
            <div class="mb-3 flex flex-wrap items-center justify-between gap-2 text-sm">
              <span class="font-medium text-gray-900 dark:text-white">{{ batchOpsT('ungrouped.total', { count: ungroupedPreview.length }) }}</span>
              <span class="text-gray-500 dark:text-gray-400">{{ batchOpsT('showFirstN', { count: 12 }) }}</span>
            </div>
            <div class="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
              <div v-for="account in ungroupedPreview.slice(0, 12)" :key="account.id" class="rounded-lg border border-gray-200 px-3 py-2 text-xs dark:border-dark-700">
                <div class="font-medium text-gray-900 dark:text-gray-100">{{ formatLooseAccountLabel(account) }}</div>
                <div class="mt-1 text-gray-500 dark:text-gray-400">{{ account.platform }} / {{ account.type }}</div>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-gray-200 bg-gray-50/80 p-5 dark:border-dark-700 dark:bg-dark-800/60">
          <div class="mb-4">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ batchOpsT('proxyMigration.title') }}</h4>
            <p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
              {{ batchOpsT('proxyMigration.description') }}
            </p>
          </div>

          <div class="grid gap-3 md:grid-cols-[2fr,1fr]">
            <textarea
              v-model="proxyMigrationForm.sourceProxyIDs"
              class="input min-h-[92px] w-full"
              :placeholder="batchOpsT('proxyMigration.sourceProxyIdsPlaceholder')"
            ></textarea>
            <select v-model="proxyMigrationForm.targetProxyId" class="input w-full">
              <option value="">{{ batchOpsT('proxyMigration.selectTargetProxy') }}</option>
              <option v-for="proxy in proxies" :key="proxy.id" :value="String(proxy.id)">
                #{{ proxy.id }} {{ proxy.host }}:{{ proxy.port }} | {{ batchOpsT('proxyOccupied', { count: proxy.account_count ?? 0 }) }}
              </option>
            </select>
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading.proxyMigrationPreview" @click="handleProxyMigrationPreview">
              {{ loading.proxyMigrationPreview ? batchOpsT('proxyMigration.previewing') : batchOpsT('proxyMigration.preview') }}
            </button>
            <button type="button" class="btn btn-danger btn-sm" :disabled="loading.proxyMigrationApply" @click="handleProxyMigrationApply">
              {{ loading.proxyMigrationApply ? batchOpsT('proxyMigration.applying') : batchOpsT('proxyMigration.apply') }}
            </button>
          </div>

          <div v-if="proxyMigrationPreview" class="mt-4 rounded-xl border border-dashed border-gray-300 bg-white p-4 dark:border-dark-600 dark:bg-dark-900/40">
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ batchOpsT('proxyMigration.hitAccounts', { count: proxyMigrationPreview.affectedAccounts.length }) }}
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ batchOpsT('proxyMigration.sourceProxies', { proxies: formatProxyList(proxyMigrationPreview.sourceProxies) }) }}
            </div>
            <div class="mt-3 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
              <div
                v-for="account in proxyMigrationPreview.affectedAccounts.slice(0, 12)"
                :key="`proxy-migration-${account.id}`"
                class="rounded-lg border border-gray-200 px-3 py-2 text-xs dark:border-dark-700"
              >
                <div class="font-medium text-gray-900 dark:text-gray-100">{{ formatProxyAccountLabel(account) }}</div>
                <div class="mt-1 text-gray-500 dark:text-gray-400">{{ account.platform }} / {{ account.type }}</div>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-gray-200 bg-gray-50/80 p-5 dark:border-dark-700 dark:bg-dark-800/60">
          <div class="mb-4">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ batchOpsT('proxyHealth.title') }}</h4>
            <p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
              {{ batchOpsT('proxyHealth.description') }}
            </p>
          </div>

          <textarea
            v-model="proxyHealthForm.sourceProxyIDs"
            class="input min-h-[92px] w-full"
            :placeholder="batchOpsT('proxyHealth.sourceProxyIdsPlaceholder')"
          ></textarea>

          <div class="mt-4 flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading.proxyHealthPreview" @click="handleProxyHealthPreview">
              {{ loading.proxyHealthPreview ? batchOpsT('proxyHealth.previewing') : batchOpsT('proxyHealth.preview') }}
            </button>
            <button type="button" class="btn btn-primary btn-sm" :disabled="loading.proxyHealthApply" @click="handleProxyHealthApply">
              {{ loading.proxyHealthApply ? batchOpsT('proxyHealth.applying') : batchOpsT('proxyHealth.apply') }}
            </button>
          </div>

          <div v-if="proxyHealthPreview" class="mt-4 rounded-xl border border-dashed border-gray-300 bg-white p-4 dark:border-dark-600 dark:bg-dark-900/40">
            <div class="grid gap-3 md:grid-cols-4">
              <div class="rounded-lg border border-gray-200 px-3 py-3 dark:border-dark-700">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ batchOpsT('proxyHealth.checkedProxies') }}</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ proxyHealthPreview.checkedProxies.length }}</div>
              </div>
              <div class="rounded-lg border border-green-200 bg-green-50 px-3 py-3 dark:border-green-900/50 dark:bg-green-900/10">
                <div class="text-xs text-green-700 dark:text-green-300">{{ batchOpsT('proxyHealth.healthyProxies') }}</div>
                <div class="mt-1 text-lg font-semibold text-green-800 dark:text-green-200">{{ proxyHealthPreview.healthyProxies.length }}</div>
              </div>
              <div class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-3 dark:border-amber-900/50 dark:bg-amber-900/10">
                <div class="text-xs text-amber-700 dark:text-amber-300">{{ batchOpsT('proxyHealth.assignments') }}</div>
                <div class="mt-1 text-lg font-semibold text-amber-800 dark:text-amber-200">{{ proxyHealthPreview.assignments.length }}</div>
              </div>
              <div class="rounded-lg border border-rose-200 bg-rose-50 px-3 py-3 dark:border-rose-900/50 dark:bg-rose-900/10">
                <div class="text-xs text-rose-700 dark:text-rose-300">{{ batchOpsT('proxyHealth.unassigned') }}</div>
                <div class="mt-1 text-lg font-semibold text-rose-800 dark:text-rose-200">{{ proxyHealthPreview.unassignedChecks.length }}</div>
              </div>
            </div>

            <div v-if="proxyHealthPreview.assignments.length > 0" class="mt-4 space-y-2">
              <div
                v-for="assignment in proxyHealthPreview.assignments.slice(0, 8)"
                :key="`assignment-${assignment.sourceProxy.id}-${assignment.targetProxy.id}`"
                class="rounded-lg border border-gray-200 px-3 py-2 text-xs dark:border-dark-700"
              >
                <div class="font-medium text-gray-900 dark:text-gray-100">
                  #{{ assignment.sourceProxy.id }} → #{{ assignment.targetProxy.id }}
                </div>
                <div class="mt-1 text-gray-500 dark:text-gray-400">
                  {{ batchOpsT('proxyHealth.moveAccounts', { count: assignment.affectedAccounts.length }) }}
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-gray-200 bg-gray-50/80 p-5 dark:border-dark-700 dark:bg-dark-800/60">
          <div class="mb-4">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ batchOpsT('opsMigration.title') }}</h4>
            <p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
              {{ batchOpsT('opsMigration.description') }}
            </p>
          </div>

          <div class="grid gap-3 md:grid-cols-4">
            <select v-model="opsMigrationForm.targetProxyId" class="input w-full">
              <option value="">{{ batchOpsT('opsMigration.selectTargetProxy') }}</option>
              <option v-for="proxy in proxies" :key="proxy.id" :value="String(proxy.id)">
                #{{ proxy.id }} {{ proxy.host }}:{{ proxy.port }}
              </option>
            </select>
            <select v-model="opsMigrationForm.endpoint" class="input w-full">
              <option value="auto">{{ batchOpsT('endpoints.auto') }}</option>
              <option value="request-errors">{{ batchOpsT('endpoints.requestErrors') }}</option>
              <option value="upstream-errors">{{ batchOpsT('endpoints.upstreamErrors') }}</option>
              <option value="requests">{{ batchOpsT('endpoints.requests') }}</option>
              <option value="system-logs">{{ batchOpsT('endpoints.systemLogs') }}</option>
            </select>
            <input v-model.trim="opsMigrationForm.keyword" class="input w-full" :placeholder="batchOpsT('opsMigration.keywordPlaceholder')" />
            <input v-model.number="opsMigrationForm.pages" type="number" min="1" max="10" class="input w-full" :placeholder="batchOpsT('opsMigration.pagesPlaceholder')" />
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading.opsPreview" @click="handleOpsPreview">
              {{ loading.opsPreview ? batchOpsT('opsMigration.previewing') : batchOpsT('opsMigration.preview') }}
            </button>
            <button type="button" class="btn btn-primary btn-sm" :disabled="loading.opsApply" @click="handleOpsApply">
              {{ loading.opsApply ? batchOpsT('opsMigration.applying') : batchOpsT('opsMigration.apply') }}
            </button>
          </div>

          <div v-if="opsPreview" class="mt-4 rounded-xl border border-dashed border-gray-300 bg-white p-4 dark:border-dark-600 dark:bg-dark-900/40">
            <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
              <span class="font-medium text-gray-900 dark:text-white">{{ batchOpsT('opsMigration.hitAccounts', { count: opsPreview.affectedAccounts.length }) }}</span>
              <span class="text-gray-500 dark:text-gray-400">{{ batchOpsT('sourceLabel', { sources: formatSourceList(opsPreview.scannedSources) }) }}</span>
            </div>
            <div class="mt-3 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
              <div
                v-for="account in opsPreview.affectedAccounts.slice(0, 12)"
                :key="`ops-migration-${account.id}`"
                class="rounded-lg border border-gray-200 px-3 py-2 text-xs dark:border-dark-700"
              >
                <div class="font-medium text-gray-900 dark:text-gray-100">{{ formatLooseAccountLabel(account) }}</div>
                <div class="mt-1 text-gray-500 dark:text-gray-400">{{ account.platform }} / {{ account.type }}</div>
              </div>
            </div>
            <details v-if="opsPreview.rawMatches?.length" class="mt-3 rounded-lg border border-gray-200 px-3 py-2 text-xs dark:border-dark-700">
              <summary class="cursor-pointer font-medium text-gray-700 dark:text-gray-300">{{ batchOpsT('showRawMatches') }}</summary>
              <pre class="mt-2 max-h-56 overflow-auto whitespace-pre-wrap break-all text-[11px] text-gray-600 dark:text-gray-400">{{ JSON.stringify(opsPreview.rawMatches.slice(0, 5), null, 2) }}</pre>
            </details>
          </div>
        </section>

        <section class="rounded-2xl border border-gray-200 bg-gray-50/80 p-5 dark:border-dark-700 dark:bg-dark-800/60">
          <div class="mb-4">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ batchOpsT('duplicates.title') }}</h4>
            <p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
              {{ batchOpsT('duplicates.description') }}
            </p>
          </div>

          <div class="grid gap-3 md:grid-cols-4">
            <select v-model="duplicateForm.fieldMode" class="input w-full">
              <option value="display">{{ batchOpsT('duplicateFields.display') }}</option>
              <option value="email">{{ batchOpsT('duplicateFields.email') }}</option>
              <option value="name">{{ batchOpsT('duplicateFields.name') }}</option>
              <option value="username">{{ batchOpsT('duplicateFields.username') }}</option>
            </select>
            <select v-model="duplicateForm.platform" class="input w-full">
              <option value="">{{ batchOpsT('ungrouped.allPlatforms') }}</option>
              <option v-for="platform in platformOptions" :key="`dup-${platform.value}`" :value="platform.value">
                {{ platform.label }}
              </option>
            </select>
            <select v-model="duplicateForm.status" class="input w-full">
              <option value="">{{ batchOpsT('status.all') }}</option>
              <option value="active">{{ batchOpsT('status.active') }}</option>
              <option value="inactive">{{ batchOpsT('status.inactive') }}</option>
              <option value="error">{{ batchOpsT('status.error') }}</option>
            </select>
            <input v-model.trim="duplicateForm.search" class="input w-full" :placeholder="batchOpsT('duplicates.searchPlaceholder')" />
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading.duplicatePreview" @click="handleDuplicatePreview">
              {{ loading.duplicatePreview ? batchOpsT('duplicates.previewing') : batchOpsT('duplicates.preview') }}
            </button>
            <button type="button" class="btn btn-danger btn-sm" :disabled="loading.duplicateApply || selectedDuplicateIDs.length === 0" @click="handleDuplicateApply">
              {{ loading.duplicateApply ? batchOpsT('duplicates.applying') : batchOpsT('duplicates.apply', { count: selectedDuplicateIDs.length }) }}
            </button>
          </div>

          <div v-if="duplicatePreview" class="mt-4 rounded-xl border border-dashed border-gray-300 bg-white p-4 dark:border-dark-600 dark:bg-dark-900/40">
            <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
              <span class="font-medium text-gray-900 dark:text-white">{{ batchOpsT('duplicates.total', { duplicates: duplicatePreview.totalDuplicates, groups: duplicatePreview.groups.length }) }}</span>
              <span class="text-gray-500 dark:text-gray-400">{{ batchOpsT('duplicates.keepSmallestId') }}</span>
            </div>

            <div class="mt-4 space-y-3">
              <div v-for="group in duplicatePreview.groups.slice(0, 12)" :key="`${group.platform}-${group.identity}`" class="rounded-lg border border-gray-200 px-4 py-3 dark:border-dark-700">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <div>
                    <div class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ group.identity || batchOpsT('fieldEmptyIdentity') }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ group.platform }} · {{ batchOpsT('duplicates.keepId', { id: group.keepId }) }}</div>
                  </div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ batchOpsT('duplicates.deleteCount', { count: group.deleteIds.length }) }}</div>
                </div>
                <div class="mt-3 space-y-2">
                  <label
                    v-for="account in group.accounts"
                    :key="`duplicate-${account.id}`"
                    class="flex items-start gap-3 rounded-md border px-3 py-2 text-xs"
                    :class="account.id === group.keepId ? 'border-emerald-200 bg-emerald-50 dark:border-emerald-900/50 dark:bg-emerald-900/10' : 'border-gray-200 dark:border-dark-700'"
                  >
                    <input
                      :checked="selectedDuplicateIDs.includes(account.id)"
                      :disabled="account.id === group.keepId"
                      type="checkbox"
                      class="mt-0.5"
                      @change="toggleDuplicateSelection(account.id, ($event.target as HTMLInputElement).checked)"
                    />
                    <div>
                      <div class="font-medium text-gray-900 dark:text-gray-100">{{ formatAccountLabel(account) }}</div>
                      <div class="mt-1 text-gray-500 dark:text-gray-400">{{ account.platform }} / {{ account.type }}</div>
                    </div>
                  </label>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { groupsAPI, proxiesAPI } from '@/api/admin'
import { useAppStore } from '@/stores'
import type { Account, AdminGroup, Proxy, ProxyAccountSummary } from '@/types'
import {
  assignUngroupedAccounts,
  autoMigrateAccountsFromUnhealthyProxies,
  deleteAccounts,
  migrateAccountsAndDeleteProxies,
  migrateAccountsFromOpsLogs,
  previewAccountsFromOpsLogs,
  previewDuplicateAccounts,
  previewProxyHealthAutoMigration,
  previewProxyMigration,
  previewUngroupedAccounts,
  type BatchResult,
  type DuplicatePreview,
  type ProxyHealthPreview,
} from '@/utils/sub2apiManager'

const { t } = useI18n()
const appStore = useAppStore()
const batchOpsPrefix = 'admin.dataManagement.batchOps'

function batchOpsT(key: string, params?: Record<string, unknown>) {
  return t(`${batchOpsPrefix}.${key}`, params ?? {})
}

function formatSourceName(source: string): string {
  switch (source) {
    case 'request-errors':
      return batchOpsT('endpoints.requestErrors')
    case 'upstream-errors':
      return batchOpsT('endpoints.upstreamErrors')
    case 'requests':
      return batchOpsT('endpoints.requests')
    case 'system-logs':
      return batchOpsT('endpoints.systemLogs')
    case 'auto':
      return batchOpsT('endpoints.auto')
    default:
      return source
  }
}

function formatSourceList(sources?: string[] | null): string {
  if (!sources || sources.length === 0) return t('common.none')
  return sources.map((item) => formatSourceName(item)).join(', ')
}

const platformOptions = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
  { value: 'sora', label: 'Sora' },
]

const groups = ref<AdminGroup[]>([])
const proxies = ref<Proxy[]>([])
const referenceLoading = ref(false)

const ungroupedForm = reactive({
  groupId: '',
  platform: '',
  status: '',
  search: '',
})

const proxyMigrationForm = reactive({
  sourceProxyIDs: '',
  targetProxyId: '',
})

const proxyHealthForm = reactive({
  sourceProxyIDs: '',
})

const opsMigrationForm = reactive({
  endpoint: 'auto',
  keyword: '502',
  pages: 3,
  targetProxyId: '',
})

const duplicateForm = reactive({
  fieldMode: 'display' as 'display' | 'email' | 'name' | 'username',
  platform: '',
  status: '',
  search: '',
})

const loading = reactive({
  ungroupedPreview: false,
  ungroupedApply: false,
  proxyMigrationPreview: false,
  proxyMigrationApply: false,
  proxyHealthPreview: false,
  proxyHealthApply: false,
  opsPreview: false,
  opsApply: false,
  duplicatePreview: false,
  duplicateApply: false,
})

const ungroupedPreview = ref<Account[] | null>(null)
const proxyMigrationPreview = ref<BatchResult | null>(null)
const proxyHealthPreview = ref<ProxyHealthPreview | null>(null)
const opsPreview = ref<BatchResult | null>(null)
const duplicatePreview = ref<DuplicatePreview | null>(null)
const selectedDuplicateIDs = ref<number[]>([])

async function loadReferenceData() {
  referenceLoading.value = true
  try {
    const [groupList, proxyList] = await Promise.all([
      groupsAPI.getAll(),
      proxiesAPI.getAllWithCount(),
    ])
    groups.value = groupList
    proxies.value = proxyList
  } catch (error: any) {
    appStore.showError(error?.message || batchOpsT('loadReferenceDataFailed'))
  } finally {
    referenceLoading.value = false
  }
}

function parseIDList(raw: string): number[] {
  return Array.from(
    new Set(
      raw
        .split(/[\s,]+/)
        .map((item) => Number(item.trim()))
        .filter((item) => Number.isFinite(item) && item > 0),
    ),
  )
}

function getRequiredTargetProxyID(value: string): number {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error(batchOpsT('targetProxyRequired'))
  }
  return parsed
}

function getRequiredGroupID(value: string): number {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error(batchOpsT('targetGroupRequired'))
  }
  return parsed
}

function formatAccountLabel(account: Account): string {
  const email = String(account.extra?.email_address || account.name || account.extra?.username || '-')
  return `#${account.id} ${email}`
}

function formatLooseAccountLabel(account: Account | ProxyAccountSummary): string {
  if ('proxy_id' in account) {
    return formatAccountLabel(account)
  }
  return `#${account.id} ${account.name || '-'}`
}

function formatProxyAccountLabel(account: ProxyAccountSummary): string {
  return `#${account.id} ${account.name || '-'}`
}

function formatProxyList(items?: Proxy[] | null): string {
  if (!items || items.length === 0) return t('common.none')
  return items.map((item) => `#${item.id}`).join(', ')
}

function beginToast(title: string, subtitle: string) {
  return appStore.showToast('info', title, undefined, {
    title,
    subtitle,
  })
}

function finishToast(toastID: string, type: 'success' | 'warning' | 'error', message: string, subtitle?: string) {
  appStore.updateToast(toastID, {
    type,
    message,
    title: message,
    subtitle,
    duration: 4000,
  })
}

async function handleUngroupedPreview() {
  loading.ungroupedPreview = true
  try {
    ungroupedPreview.value = await previewUngroupedAccounts({
      platform: ungroupedForm.platform,
      search: ungroupedForm.search,
      status: ungroupedForm.status,
    })
    appStore.showSuccess(batchOpsT('ungrouped.previewSuccess', { count: ungroupedPreview.value.length }))
  } catch (error: any) {
    appStore.showError(error?.message || batchOpsT('ungrouped.previewFailed'))
  } finally {
    loading.ungroupedPreview = false
  }
}

async function handleUngroupedApply() {
  let targetGroupID = 0
  try {
    targetGroupID = getRequiredGroupID(ungroupedForm.groupId)
  } catch (error: any) {
    appStore.showError(error.message)
    return
  }

  loading.ungroupedApply = true
  const toastID = beginToast(batchOpsT('ungrouped.applyingTitle'), batchOpsT('ungrouped.applyingSubtitle'))
  try {
    const result = await assignUngroupedAccounts(targetGroupID, {
      platform: ungroupedForm.platform,
      search: ungroupedForm.search,
      status: ungroupedForm.status,
    })
    finishToast(
      toastID,
      'success',
      batchOpsT('ungrouped.applied', { count: result.affectedAccounts.length }),
      result.targetGroup
        ? batchOpsT('ungrouped.targetGroup', { id: result.targetGroup.id, name: result.targetGroup.name })
        : undefined,
    )
    ungroupedPreview.value = result.affectedAccounts as Account[]
  } catch (error: any) {
    finishToast(toastID, 'error', error?.message || batchOpsT('ungrouped.applyFailed'))
  } finally {
    loading.ungroupedApply = false
  }
}

async function handleProxyMigrationPreview() {
  const sourceProxyIDs = parseIDList(proxyMigrationForm.sourceProxyIDs)
  if (sourceProxyIDs.length === 0) {
    appStore.showError(batchOpsT('proxyMigration.sourceProxyIdsRequired'))
    return
  }

  loading.proxyMigrationPreview = true
  try {
    proxyMigrationPreview.value = await previewProxyMigration(sourceProxyIDs)
    appStore.showSuccess(batchOpsT('proxyMigration.previewSuccess', { count: proxyMigrationPreview.value.affectedAccounts.length }))
  } catch (error: any) {
    appStore.showError(error?.message || batchOpsT('proxyMigration.previewFailed'))
  } finally {
    loading.proxyMigrationPreview = false
  }
}

async function handleProxyMigrationApply() {
  const sourceProxyIDs = parseIDList(proxyMigrationForm.sourceProxyIDs)
  if (sourceProxyIDs.length === 0) {
    appStore.showError(batchOpsT('proxyMigration.sourceProxyIdsRequired'))
    return
  }

  let targetProxyID = 0
  try {
    targetProxyID = getRequiredTargetProxyID(proxyMigrationForm.targetProxyId)
  } catch (error: any) {
    appStore.showError(error.message)
    return
  }

  if (!window.confirm(batchOpsT('proxyMigration.confirm', { proxyIds: sourceProxyIDs.join(', ') }))) {
    return
  }

  loading.proxyMigrationApply = true
  const toastID = beginToast(
    batchOpsT('proxyMigration.applyingTitle'),
    batchOpsT('proxyMigration.applyingSubtitle', { proxyIds: sourceProxyIDs.join(', ') }),
  )
  try {
    proxyMigrationPreview.value = await migrateAccountsAndDeleteProxies(sourceProxyIDs, targetProxyID)
    finishToast(
      toastID,
      'success',
      batchOpsT('proxyMigration.applied', { count: proxyMigrationPreview.value.affectedAccounts.length }),
      proxyMigrationPreview.value.targetProxy
        ? batchOpsT('proxyMigration.targetProxy', { id: proxyMigrationPreview.value.targetProxy.id })
        : undefined,
    )
    await loadReferenceData()
  } catch (error: any) {
    finishToast(toastID, 'error', error?.message || batchOpsT('proxyMigration.applyFailed'))
  } finally {
    loading.proxyMigrationApply = false
  }
}

async function handleProxyHealthPreview() {
  loading.proxyHealthPreview = true
  try {
    const sourceProxyIDs = parseIDList(proxyHealthForm.sourceProxyIDs)
    proxyHealthPreview.value = await previewProxyHealthAutoMigration(sourceProxyIDs.length > 0 ? sourceProxyIDs : undefined)
    appStore.showSuccess(batchOpsT('proxyHealth.previewSuccess', { count: proxyHealthPreview.value.checkedProxies.length }))
  } catch (error: any) {
    appStore.showError(error?.message || batchOpsT('proxyHealth.previewFailed'))
  } finally {
    loading.proxyHealthPreview = false
  }
}

async function handleProxyHealthApply() {
  const sourceProxyIDs = parseIDList(proxyHealthForm.sourceProxyIDs)
  if (!window.confirm(batchOpsT('proxyHealth.confirm'))) {
    return
  }

  loading.proxyHealthApply = true
  const toastID = beginToast(batchOpsT('proxyHealth.applyingTitle'), batchOpsT('proxyHealth.applyingSubtitle'))
  try {
    proxyHealthPreview.value = await autoMigrateAccountsFromUnhealthyProxies(
      sourceProxyIDs.length > 0 ? sourceProxyIDs : undefined,
      proxyHealthPreview.value ?? undefined,
    )
    finishToast(
      toastID,
      'success',
      batchOpsT('proxyHealth.applied', { count: proxyHealthPreview.value.assignments.length }),
      batchOpsT('proxyHealth.healthySummary', { count: proxyHealthPreview.value.healthyProxies.length }),
    )
    await loadReferenceData()
  } catch (error: any) {
    finishToast(toastID, 'error', error?.message || batchOpsT('proxyHealth.applyFailed'))
  } finally {
    loading.proxyHealthApply = false
  }
}

async function handleOpsPreview() {
  loading.opsPreview = true
  try {
    opsPreview.value = await previewAccountsFromOpsLogs({
      endpoint: opsMigrationForm.endpoint,
      pages: opsMigrationForm.pages,
      keyword: opsMigrationForm.keyword || '502',
    })
    appStore.showSuccess(batchOpsT('opsMigration.previewSuccess', { count: opsPreview.value.affectedAccounts.length }))
  } catch (error: any) {
    appStore.showError(error?.message || batchOpsT('opsMigration.previewFailed'))
  } finally {
    loading.opsPreview = false
  }
}

async function handleOpsApply() {
  let targetProxyID = 0
  try {
    targetProxyID = getRequiredTargetProxyID(opsMigrationForm.targetProxyId)
  } catch (error: any) {
    appStore.showError(error.message)
    return
  }

  loading.opsApply = true
  const toastID = beginToast(
    batchOpsT('opsMigration.applyingTitle'),
    batchOpsT('opsMigration.applyingSubtitle', { keyword: opsMigrationForm.keyword || '502' }),
  )
  try {
    opsPreview.value = await migrateAccountsFromOpsLogs(targetProxyID, {
      endpoint: opsMigrationForm.endpoint,
      pages: opsMigrationForm.pages,
      keyword: opsMigrationForm.keyword || '502',
    })
    finishToast(
      toastID,
      'success',
      batchOpsT('opsMigration.applied', { count: opsPreview.value.affectedAccounts.length }),
      opsPreview.value.scannedSources?.length
        ? batchOpsT('sourceLabel', { sources: formatSourceList(opsPreview.value.scannedSources) })
        : undefined,
    )
    await loadReferenceData()
  } catch (error: any) {
    finishToast(toastID, 'error', error?.message || batchOpsT('opsMigration.applyFailed'))
  } finally {
    loading.opsApply = false
  }
}

async function handleDuplicatePreview() {
  loading.duplicatePreview = true
  try {
    duplicatePreview.value = await previewDuplicateAccounts({
      fieldMode: duplicateForm.fieldMode,
      platform: duplicateForm.platform,
      search: duplicateForm.search,
      status: duplicateForm.status,
    })
    selectedDuplicateIDs.value = duplicatePreview.value.groups.flatMap((group) => group.deleteIds)
    appStore.showSuccess(batchOpsT('duplicates.previewSuccess', { count: duplicatePreview.value.totalDuplicates }))
  } catch (error: any) {
    appStore.showError(error?.message || batchOpsT('duplicates.previewFailed'))
  } finally {
    loading.duplicatePreview = false
  }
}

function toggleDuplicateSelection(accountID: number, checked: boolean) {
  if (checked) {
    if (!selectedDuplicateIDs.value.includes(accountID)) {
      selectedDuplicateIDs.value = [...selectedDuplicateIDs.value, accountID]
    }
    return
  }
  selectedDuplicateIDs.value = selectedDuplicateIDs.value.filter((item) => item !== accountID)
}

async function handleDuplicateApply() {
  if (selectedDuplicateIDs.value.length === 0) {
    appStore.showError(batchOpsT('duplicates.noSelection'))
    return
  }
  if (!window.confirm(batchOpsT('duplicates.confirm', { count: selectedDuplicateIDs.value.length }))) {
    return
  }

  loading.duplicateApply = true
  const toastID = beginToast(
    batchOpsT('duplicates.applyingTitle'),
    batchOpsT('duplicates.applyingSubtitle', { count: selectedDuplicateIDs.value.length }),
  )
  try {
    await deleteAccounts(selectedDuplicateIDs.value)
    finishToast(toastID, 'success', batchOpsT('duplicates.applied', { count: selectedDuplicateIDs.value.length }))
    await handleDuplicatePreview()
  } catch (error: any) {
    finishToast(toastID, 'error', error?.message || batchOpsT('duplicates.applyFailed'))
  } finally {
    loading.duplicateApply = false
  }
}

onMounted(async () => {
  await loadReferenceData()
})
</script>
