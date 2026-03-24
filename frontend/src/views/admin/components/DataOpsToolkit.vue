<template>
  <div class="space-y-6">
    <div class="card overflow-hidden border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <div class="border-b border-gray-200 bg-gradient-to-r from-amber-50 via-white to-cyan-50 px-6 py-5 dark:border-dark-700 dark:from-dark-900 dark:via-dark-900 dark:to-dark-800">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">批量运营工具</h3>
            <p class="mt-1 max-w-3xl text-sm text-gray-600 dark:text-gray-400">
              这组工具把本地 `sub2apitool` 的高频批量功能原生接进了管理台，全部复用现有官方管理接口，不再需要额外跑 Python 工具。
            </p>
          </div>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="referenceLoading"
            @click="loadReferenceData"
          >
            {{ referenceLoading ? '加载中...' : '刷新分组/代理' }}
          </button>
        </div>
      </div>

      <div class="grid gap-5 p-6">
        <section class="rounded-2xl border border-gray-200 bg-gray-50/80 p-5 dark:border-dark-700 dark:bg-dark-800/60">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div>
              <h4 class="text-sm font-semibold text-gray-900 dark:text-white">1. 未分组账号批量入组</h4>
              <p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
                预览当前未纳入任何分组的账号，再一键写入目标分组。
              </p>
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              最近预览 {{ ungroupedPreview?.length ?? 0 }} 个账号
            </div>
          </div>

          <div class="grid gap-3 md:grid-cols-4">
            <select v-model="ungroupedForm.groupId" class="input w-full">
              <option value="">选择目标分组</option>
              <option v-for="group in groups" :key="group.id" :value="String(group.id)">
                #{{ group.id }} {{ group.name }} ({{ group.platform }})
              </option>
            </select>
            <select v-model="ungroupedForm.platform" class="input w-full">
              <option value="">全部平台</option>
              <option v-for="platform in platformOptions" :key="platform.value" :value="platform.value">
                {{ platform.label }}
              </option>
            </select>
            <select v-model="ungroupedForm.status" class="input w-full">
              <option value="">全部状态</option>
              <option value="active">正常</option>
              <option value="inactive">禁用</option>
              <option value="error">错误</option>
            </select>
            <input v-model.trim="ungroupedForm.search" class="input w-full" placeholder="搜索账号名称 / 邮箱" />
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading.ungroupedPreview" @click="handleUngroupedPreview">
              {{ loading.ungroupedPreview ? '预览中...' : '预览未分组账号' }}
            </button>
            <button type="button" class="btn btn-primary btn-sm" :disabled="loading.ungroupedApply || !ungroupedForm.groupId" @click="handleUngroupedApply">
              {{ loading.ungroupedApply ? '执行中...' : '批量写入分组' }}
            </button>
          </div>

          <div v-if="ungroupedPreview" class="mt-4 rounded-xl border border-dashed border-gray-300 bg-white p-4 dark:border-dark-600 dark:bg-dark-900/40">
            <div class="mb-3 flex flex-wrap items-center justify-between gap-2 text-sm">
              <span class="font-medium text-gray-900 dark:text-white">共 {{ ungroupedPreview.length }} 个未分组账号</span>
              <span class="text-gray-500 dark:text-gray-400">只展示前 12 个</span>
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
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">2. 代理迁移后批量删除</h4>
            <p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
              读取旧代理占用账号，批量迁移到新代理后删除旧代理。
            </p>
          </div>

          <div class="grid gap-3 md:grid-cols-[2fr,1fr]">
            <textarea
              v-model="proxyMigrationForm.sourceProxyIDs"
              class="input min-h-[92px] w-full"
              placeholder="待删除代理 ID，支持逗号/换行，例如：12,13,18"
            ></textarea>
            <select v-model="proxyMigrationForm.targetProxyId" class="input w-full">
              <option value="">选择目标代理</option>
              <option v-for="proxy in proxies" :key="proxy.id" :value="String(proxy.id)">
                #{{ proxy.id }} {{ proxy.host }}:{{ proxy.port }} | 占用 {{ proxy.account_count ?? 0 }}
              </option>
            </select>
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading.proxyMigrationPreview" @click="handleProxyMigrationPreview">
              {{ loading.proxyMigrationPreview ? '预览中...' : '预览占用账号' }}
            </button>
            <button type="button" class="btn btn-danger btn-sm" :disabled="loading.proxyMigrationApply" @click="handleProxyMigrationApply">
              {{ loading.proxyMigrationApply ? '执行中...' : '迁移账号并删除代理' }}
            </button>
          </div>

          <div v-if="proxyMigrationPreview" class="mt-4 rounded-xl border border-dashed border-gray-300 bg-white p-4 dark:border-dark-600 dark:bg-dark-900/40">
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              命中 {{ proxyMigrationPreview.affectedAccounts.length }} 个账号
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              来源代理：{{ formatProxyList(proxyMigrationPreview.sourceProxies) }}
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
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">3. 代理测活与自动迁移</h4>
            <p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
              对指定代理批量测活，自动把不可用代理上的账号迁移到当前更健康、占用更低的代理。
            </p>
          </div>

          <textarea
            v-model="proxyHealthForm.sourceProxyIDs"
            class="input min-h-[92px] w-full"
            placeholder="可选：只测这些代理，留空则测全部代理"
          ></textarea>

          <div class="mt-4 flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading.proxyHealthPreview" @click="handleProxyHealthPreview">
              {{ loading.proxyHealthPreview ? '测活中...' : '预览测活与迁移计划' }}
            </button>
            <button type="button" class="btn btn-primary btn-sm" :disabled="loading.proxyHealthApply" @click="handleProxyHealthApply">
              {{ loading.proxyHealthApply ? '执行中...' : '自动迁移不可用代理账号' }}
            </button>
          </div>

          <div v-if="proxyHealthPreview" class="mt-4 rounded-xl border border-dashed border-gray-300 bg-white p-4 dark:border-dark-600 dark:bg-dark-900/40">
            <div class="grid gap-3 md:grid-cols-4">
              <div class="rounded-lg border border-gray-200 px-3 py-3 dark:border-dark-700">
                <div class="text-xs text-gray-500 dark:text-gray-400">已测代理</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ proxyHealthPreview.checkedProxies.length }}</div>
              </div>
              <div class="rounded-lg border border-green-200 bg-green-50 px-3 py-3 dark:border-green-900/50 dark:bg-green-900/10">
                <div class="text-xs text-green-700 dark:text-green-300">健康代理</div>
                <div class="mt-1 text-lg font-semibold text-green-800 dark:text-green-200">{{ proxyHealthPreview.healthyProxies.length }}</div>
              </div>
              <div class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-3 dark:border-amber-900/50 dark:bg-amber-900/10">
                <div class="text-xs text-amber-700 dark:text-amber-300">迁移计划</div>
                <div class="mt-1 text-lg font-semibold text-amber-800 dark:text-amber-200">{{ proxyHealthPreview.assignments.length }}</div>
              </div>
              <div class="rounded-lg border border-rose-200 bg-rose-50 px-3 py-3 dark:border-rose-900/50 dark:bg-rose-900/10">
                <div class="text-xs text-rose-700 dark:text-rose-300">无法迁移</div>
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
                  迁移 {{ assignment.affectedAccounts.length }} 个账号
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-gray-200 bg-gray-50/80 p-5 dark:border-dark-700 dark:bg-dark-800/60">
          <div class="mb-4">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">4. 502 / 错误日志批量换代理</h4>
            <p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
              从请求错误、上游错误、请求明细或系统日志里找出报错账号，再批量更新代理。
            </p>
          </div>

          <div class="grid gap-3 md:grid-cols-4">
            <select v-model="opsMigrationForm.targetProxyId" class="input w-full">
              <option value="">选择目标代理</option>
              <option v-for="proxy in proxies" :key="proxy.id" :value="String(proxy.id)">
                #{{ proxy.id }} {{ proxy.host }}:{{ proxy.port }}
              </option>
            </select>
            <select v-model="opsMigrationForm.endpoint" class="input w-full">
              <option value="auto">自动选择</option>
              <option value="request-errors">请求错误</option>
              <option value="upstream-errors">上游错误</option>
              <option value="requests">请求明细</option>
              <option value="system-logs">系统日志</option>
            </select>
            <input v-model.trim="opsMigrationForm.keyword" class="input w-full" placeholder="关键字，默认 502" />
            <input v-model.number="opsMigrationForm.pages" type="number" min="1" max="10" class="input w-full" placeholder="扫描页数" />
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading.opsPreview" @click="handleOpsPreview">
              {{ loading.opsPreview ? '扫描中...' : '预览异常账号' }}
            </button>
            <button type="button" class="btn btn-primary btn-sm" :disabled="loading.opsApply" @click="handleOpsApply">
              {{ loading.opsApply ? '执行中...' : '批量更新这些账号的代理' }}
            </button>
          </div>

          <div v-if="opsPreview" class="mt-4 rounded-xl border border-dashed border-gray-300 bg-white p-4 dark:border-dark-600 dark:bg-dark-900/40">
            <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
              <span class="font-medium text-gray-900 dark:text-white">命中 {{ opsPreview.affectedAccounts.length }} 个账号</span>
              <span class="text-gray-500 dark:text-gray-400">来源：{{ opsPreview.scannedSources?.join(', ') || '无' }}</span>
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
              <summary class="cursor-pointer font-medium text-gray-700 dark:text-gray-300">查看部分原始命中日志</summary>
              <pre class="mt-2 max-h-56 overflow-auto whitespace-pre-wrap break-all text-[11px] text-gray-600 dark:text-gray-400">{{ JSON.stringify(opsPreview.rawMatches.slice(0, 5), null, 2) }}</pre>
            </details>
          </div>
        </section>

        <section class="rounded-2xl border border-gray-200 bg-gray-50/80 p-5 dark:border-dark-700 dark:bg-dark-800/60">
          <div class="mb-4">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">5. 重复账号检查与删除</h4>
            <p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
              先按字段预览重复组，默认勾选每组里除保留项之外的账号，再按你的选择删除。
            </p>
          </div>

          <div class="grid gap-3 md:grid-cols-4">
            <select v-model="duplicateForm.fieldMode" class="input w-full">
              <option value="display">显示名优先</option>
              <option value="email">邮箱</option>
              <option value="name">名称</option>
              <option value="username">用户名</option>
            </select>
            <select v-model="duplicateForm.platform" class="input w-full">
              <option value="">全部平台</option>
              <option v-for="platform in platformOptions" :key="`dup-${platform.value}`" :value="platform.value">
                {{ platform.label }}
              </option>
            </select>
            <select v-model="duplicateForm.status" class="input w-full">
              <option value="">全部状态</option>
              <option value="active">正常</option>
              <option value="inactive">禁用</option>
              <option value="error">错误</option>
            </select>
            <input v-model.trim="duplicateForm.search" class="input w-full" placeholder="可选：缩小范围搜索" />
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading.duplicatePreview" @click="handleDuplicatePreview">
              {{ loading.duplicatePreview ? '扫描中...' : '预览重复账号' }}
            </button>
            <button type="button" class="btn btn-danger btn-sm" :disabled="loading.duplicateApply || selectedDuplicateIDs.length === 0" @click="handleDuplicateApply">
              {{ loading.duplicateApply ? '删除中...' : `删除选中的重复账号 (${selectedDuplicateIDs.length})` }}
            </button>
          </div>

          <div v-if="duplicatePreview" class="mt-4 rounded-xl border border-dashed border-gray-300 bg-white p-4 dark:border-dark-600 dark:bg-dark-900/40">
            <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
              <span class="font-medium text-gray-900 dark:text-white">共 {{ duplicatePreview.totalDuplicates }} 个重复账号，分成 {{ duplicatePreview.groups.length }} 组</span>
              <span class="text-gray-500 dark:text-gray-400">默认保留每组 ID 最小的账号</span>
            </div>

            <div class="mt-4 space-y-3">
              <div v-for="group in duplicatePreview.groups.slice(0, 12)" :key="`${group.platform}-${group.identity}`" class="rounded-lg border border-gray-200 px-4 py-3 dark:border-dark-700">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <div>
                    <div class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ group.identity || '(空标识)' }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ group.platform }} · 保留 #{{ group.keepId }}</div>
                  </div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">待删 {{ group.deleteIds.length }} 个</div>
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

const appStore = useAppStore()

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
    appStore.showError(error?.message || '读取分组和代理列表失败')
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
    throw new Error('请先选择目标代理')
  }
  return parsed
}

function getRequiredGroupID(value: string): number {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error('请先选择目标分组')
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
  if (!items || items.length === 0) return '无'
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
    appStore.showSuccess(`已找到 ${ungroupedPreview.value.length} 个未分组账号`)
  } catch (error: any) {
    appStore.showError(error?.message || '预览未分组账号失败')
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
  const toastID = beginToast('正在批量写入分组', '将未分组账号写入目标分组')
  try {
    const result = await assignUngroupedAccounts(targetGroupID, {
      platform: ungroupedForm.platform,
      search: ungroupedForm.search,
      status: ungroupedForm.status,
    })
    finishToast(
      toastID,
      'success',
      `已写入 ${result.affectedAccounts.length} 个账号`,
      result.targetGroup ? `目标分组：#${result.targetGroup.id} ${result.targetGroup.name}` : undefined,
    )
    ungroupedPreview.value = result.affectedAccounts as Account[]
  } catch (error: any) {
    finishToast(toastID, 'error', error?.message || '批量写入分组失败')
  } finally {
    loading.ungroupedApply = false
  }
}

async function handleProxyMigrationPreview() {
  const sourceProxyIDs = parseIDList(proxyMigrationForm.sourceProxyIDs)
  if (sourceProxyIDs.length === 0) {
    appStore.showError('请至少填写一个待删除代理 ID')
    return
  }

  loading.proxyMigrationPreview = true
  try {
    proxyMigrationPreview.value = await previewProxyMigration(sourceProxyIDs)
    appStore.showSuccess(`命中 ${proxyMigrationPreview.value.affectedAccounts.length} 个账号`)
  } catch (error: any) {
    appStore.showError(error?.message || '预览代理迁移失败')
  } finally {
    loading.proxyMigrationPreview = false
  }
}

async function handleProxyMigrationApply() {
  const sourceProxyIDs = parseIDList(proxyMigrationForm.sourceProxyIDs)
  if (sourceProxyIDs.length === 0) {
    appStore.showError('请至少填写一个待删除代理 ID')
    return
  }

  let targetProxyID = 0
  try {
    targetProxyID = getRequiredTargetProxyID(proxyMigrationForm.targetProxyId)
  } catch (error: any) {
    appStore.showError(error.message)
    return
  }

  if (!window.confirm(`确认把 ${sourceProxyIDs.join(', ')} 上的账号迁移后删除这些代理吗？`)) {
    return
  }

  loading.proxyMigrationApply = true
  const toastID = beginToast('正在迁移账号并删除代理', `源代理：${sourceProxyIDs.join(', ')}`)
  try {
    proxyMigrationPreview.value = await migrateAccountsAndDeleteProxies(sourceProxyIDs, targetProxyID)
    finishToast(
      toastID,
      'success',
      `已处理 ${proxyMigrationPreview.value.affectedAccounts.length} 个账号`,
      proxyMigrationPreview.value.targetProxy ? `目标代理：#${proxyMigrationPreview.value.targetProxy.id}` : undefined,
    )
    await loadReferenceData()
  } catch (error: any) {
    finishToast(toastID, 'error', error?.message || '迁移账号并删除代理失败')
  } finally {
    loading.proxyMigrationApply = false
  }
}

async function handleProxyHealthPreview() {
  loading.proxyHealthPreview = true
  try {
    const sourceProxyIDs = parseIDList(proxyHealthForm.sourceProxyIDs)
    proxyHealthPreview.value = await previewProxyHealthAutoMigration(sourceProxyIDs.length > 0 ? sourceProxyIDs : undefined)
    appStore.showSuccess(`已完成 ${proxyHealthPreview.value.checkedProxies.length} 个代理的测活预览`)
  } catch (error: any) {
    appStore.showError(error?.message || '代理测活预览失败')
  } finally {
    loading.proxyHealthPreview = false
  }
}

async function handleProxyHealthApply() {
  const sourceProxyIDs = parseIDList(proxyHealthForm.sourceProxyIDs)
  if (!window.confirm('确认自动迁移不可用代理上的账号吗？')) {
    return
  }

  loading.proxyHealthApply = true
  const toastID = beginToast('正在迁移不可用代理账号', '系统会优先选择更健康、占用更低的代理')
  try {
    proxyHealthPreview.value = await autoMigrateAccountsFromUnhealthyProxies(
      sourceProxyIDs.length > 0 ? sourceProxyIDs : undefined,
      proxyHealthPreview.value ?? undefined,
    )
    finishToast(
      toastID,
      'success',
      `已执行 ${proxyHealthPreview.value.assignments.length} 组代理迁移`,
      `健康代理 ${proxyHealthPreview.value.healthyProxies.length} 个`,
    )
    await loadReferenceData()
  } catch (error: any) {
    finishToast(toastID, 'error', error?.message || '自动迁移失败')
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
    appStore.showSuccess(`已命中 ${opsPreview.value.affectedAccounts.length} 个异常账号`)
  } catch (error: any) {
    appStore.showError(error?.message || '扫描异常日志失败')
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
  const toastID = beginToast('正在批量切换异常账号代理', `关键字：${opsMigrationForm.keyword || '502'}`)
  try {
    opsPreview.value = await migrateAccountsFromOpsLogs(targetProxyID, {
      endpoint: opsMigrationForm.endpoint,
      pages: opsMigrationForm.pages,
      keyword: opsMigrationForm.keyword || '502',
    })
    finishToast(
      toastID,
      'success',
      `已批量更新 ${opsPreview.value.affectedAccounts.length} 个账号`,
      opsPreview.value.scannedSources?.length ? `来源：${opsPreview.value.scannedSources.join(', ')}` : undefined,
    )
    await loadReferenceData()
  } catch (error: any) {
    finishToast(toastID, 'error', error?.message || '批量换代理失败')
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
    appStore.showSuccess(`找到 ${duplicatePreview.value.totalDuplicates} 个重复账号`)
  } catch (error: any) {
    appStore.showError(error?.message || '扫描重复账号失败')
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
    appStore.showError('当前没有勾选任何待删除账号')
    return
  }
  if (!window.confirm(`确认删除这 ${selectedDuplicateIDs.value.length} 个重复账号吗？`)) {
    return
  }

  loading.duplicateApply = true
  const toastID = beginToast('正在删除重复账号', `已选中 ${selectedDuplicateIDs.value.length} 个账号`)
  try {
    await deleteAccounts(selectedDuplicateIDs.value)
    finishToast(toastID, 'success', `已删除 ${selectedDuplicateIDs.value.length} 个重复账号`)
    await handleDuplicatePreview()
  } catch (error: any) {
    finishToast(toastID, 'error', error?.message || '删除重复账号失败')
  } finally {
    loading.duplicateApply = false
  }
}

onMounted(async () => {
  await loadReferenceData()
})
</script>
