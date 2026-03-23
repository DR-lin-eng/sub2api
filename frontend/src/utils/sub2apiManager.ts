import { accountsAPI, groupsAPI, opsAPI, proxiesAPI } from '@/api/admin'
import type {
  Account,
  AccountListFilters,
  AdminGroup,
  Proxy,
  ProxyAccountSummary,
} from '@/types'

const ACCOUNT_PAGE_SIZE = 100
const LOG_PAGE_SIZE = 200
const LOW_WRITE_CONCURRENCY = 2
const LOW_REQUEST_CONCURRENCY = 3
const DIRECT_ACCOUNT_LOOKUP_THRESHOLD = 20
const PROXY_TEST_CONCURRENCY = 8

export const LOG_ENDPOINTS = [
  'request-errors',
  'upstream-errors',
  'requests',
  'system-logs',
] as const

type LogEndpoint = (typeof LOG_ENDPOINTS)[number]

const STATUS_KEYS = new Set([
  'status',
  'status_code',
  'response_status',
  'http_status',
  'upstream_status',
  'code',
  'response_code',
])

const ACCOUNT_ID_KEYS = new Set([
  'account_id',
  'upstream_account_id',
  'scheduler_account_id',
  'accountid',
])

export interface BatchResult {
  affectedAccounts: Array<Account | ProxyAccountSummary>
  targetGroup?: AdminGroup | null
  sourceProxies?: Proxy[] | null
  targetProxy?: Proxy | null
  scannedSources?: string[] | null
  rawMatches?: any[] | null
  updateResponse?: any
  deleteResponse?: any
}

export interface DuplicatePreviewGroup {
  identity: string
  platform: string
  keepId: number
  deleteIds: number[]
  accounts: Account[]
}

export interface DuplicatePreview {
  groups: DuplicatePreviewGroup[]
  totalDuplicates: number
  fieldMode: 'display' | 'email' | 'name' | 'username'
}

export interface ProxyHealthCheck {
  proxy: Proxy
  success: boolean
  message: string
  affectedAccounts: ProxyAccountSummary[]
}

export interface ProxyMigrationAssignment {
  sourceProxy: Proxy
  targetProxy: Proxy
  affectedAccounts: ProxyAccountSummary[]
}

export interface ProxyHealthPreview {
  checkedProxies: ProxyHealthCheck[]
  healthyProxies: Proxy[]
  assignments: ProxyMigrationAssignment[]
  unassignedChecks: ProxyHealthCheck[]
  updateResponses?: any[] | null
}

export function chunked<T>(values: T[], size = 100): T[][] {
  const out: T[][] = []
  for (let index = 0; index < values.length; index += size) {
    out.push(values.slice(index, index + size))
  }
  return out
}

export async function mapWithConcurrency<T, R>(
  values: T[],
  handler: (value: T) => Promise<R>,
  maxWorkers = LOW_REQUEST_CONCURRENCY,
): Promise<R[]> {
  if (values.length === 0) return []
  const results = new Array<R>(values.length)
  let cursor = 0

  async function worker() {
    while (true) {
      const current = cursor
      cursor += 1
      if (current >= values.length) return
      results[current] = await handler(values[current])
    }
  }

  const workerCount = Math.max(1, Math.min(maxWorkers, values.length))
  await Promise.all(Array.from({ length: workerCount }, () => worker()))
  return results
}

export async function listAllAccounts(filters: AccountListFilters = {}): Promise<Account[]> {
  const accounts: Account[] = []
  const seen = new Set<number>()
  let page = 1

  while (true) {
    const response = await accountsAPI.list(page, ACCOUNT_PAGE_SIZE, filters)
    const items = Array.isArray(response.items) ? response.items : []
    if (items.length === 0) break

    let pageNewItems = 0
    for (const item of items) {
      if (!item?.id || seen.has(item.id)) continue
      seen.add(item.id)
      accounts.push(item)
      pageNewItems += 1
    }

    if (pageNewItems === 0 || items.length < ACCOUNT_PAGE_SIZE) break
    page += 1
    if (page > 1000) break
  }

  return accounts
}

export async function getAccountsByIds(accountIds: number[]): Promise<Account[]> {
  const normalizedIds = Array.from(new Set(accountIds.map((id) => Number(id)).filter((id) => Number.isFinite(id) && id > 0)))
  if (normalizedIds.length === 0) return []

  if (normalizedIds.length <= DIRECT_ACCOUNT_LOOKUP_THRESHOLD) {
    const results = await mapWithConcurrency(normalizedIds, async (accountId) => {
      try {
        return await accountsAPI.getById(accountId)
      } catch {
        return null
      }
    })
    return results.filter((item): item is Account => !!item)
  }

  const accounts = await listAllAccounts()
  const byId = new Map(accounts.map((account) => [account.id, account]))
  return normalizedIds.map((id) => byId.get(id)).filter((item): item is Account => !!item)
}

export async function batchUpdateAccounts(accountIds: number[], updates: Record<string, unknown>) {
  const batches = chunked(
    Array.from(new Set(accountIds.map((id) => Number(id)).filter((id) => Number.isFinite(id) && id > 0))).sort((a, b) => a - b),
    100,
  )
  return mapWithConcurrency(batches, (batch) => accountsAPI.bulkUpdate(batch, updates), LOW_WRITE_CONCURRENCY)
}

export async function previewUngroupedAccounts(filters: {
  platform?: string
  search?: string
  status?: string
} = {}): Promise<Account[]> {
  return listAllAccounts({
    platform: filters.platform || '',
    search: filters.search || '',
    status: filters.status || '',
    group: 'ungrouped',
  })
}

export async function assignUngroupedAccounts(
  groupId: number,
  filters: {
    platform?: string
    search?: string
    status?: string
  } = {},
): Promise<BatchResult> {
  const accounts = await previewUngroupedAccounts(filters)
  const accountIds = accounts.map((account) => account.id)
  const updateResponse = accountIds.length > 0 ? await batchUpdateAccounts(accountIds, { group_ids: [groupId] }) : []
  const groups = await groupsAPI.getAll()
  const targetGroup = groups.find((group) => group.id === groupId) ?? null
  return {
    affectedAccounts: accounts,
    targetGroup,
    updateResponse,
  }
}

export async function previewProxyMigration(sourceProxyIds: number[]): Promise<BatchResult> {
  const proxies = await proxiesAPI.getAllWithCount()
  const selectedIds = new Set(sourceProxyIds.map((id) => Number(id)).filter((id) => Number.isFinite(id) && id > 0))
  const sourceProxies = proxies.filter((proxy) => selectedIds.has(proxy.id))
  const proxyAccounts = await mapWithConcurrency(
    Array.from(selectedIds),
    (proxyId) => proxiesAPI.getProxyAccounts(proxyId),
    LOW_REQUEST_CONCURRENCY,
  )
  const accountsById = new Map<number, ProxyAccountSummary>()
  for (const accounts of proxyAccounts) {
    for (const account of accounts) {
      accountsById.set(account.id, account)
    }
  }
  return {
    affectedAccounts: Array.from(accountsById.values()).sort((a, b) => a.id - b.id),
    sourceProxies,
  }
}

export async function migrateAccountsAndDeleteProxies(
  sourceProxyIds: number[],
  targetProxyId: number | null,
): Promise<BatchResult> {
  const preview = await previewProxyMigration(sourceProxyIds)
  if (targetProxyId != null && sourceProxyIds.includes(targetProxyId)) {
    throw new Error('目标代理不能包含在待删除代理里。')
  }
  if (targetProxyId == null && preview.affectedAccounts.length > 0) {
    throw new Error('当前只支持迁移到指定代理，请先选择目标代理。')
  }

  let updateResponse: any[] = []
  if (targetProxyId != null) {
    const accountIds = preview.affectedAccounts
      .map((account) => account.id)
      .filter((id) => id > 0)
    if (accountIds.length > 0) {
      updateResponse = await batchUpdateAccounts(accountIds, { proxy_id: targetProxyId })
    }
  }
  const deleteResponse = await proxiesAPI.batchDelete(sourceProxyIds)
  const proxies = await proxiesAPI.getAllWithCount()
  return {
    ...preview,
    targetProxy: targetProxyId != null ? proxies.find((proxy) => proxy.id === targetProxyId) ?? null : null,
    updateResponse,
    deleteResponse,
  }
}

export function proxyUsageCount(proxy: Proxy): number {
  return Number(proxy.account_count || 0)
}

function shouldFetchProxyAccounts(proxy: Proxy): boolean {
  if (proxyUsageCount(proxy) > 0) return true
  return typeof proxy.account_count !== 'number'
}

export async function inspectProxyHealth(proxy: Proxy): Promise<ProxyHealthCheck> {
  let success = false
  let message = ''
  try {
    const payload = await proxiesAPI.testProxy(proxy.id)
    success = !!payload.success
    message = String(payload.message || (success ? '测活成功' : '测活失败'))
  } catch (error: any) {
    message = error?.message || '代理测活失败'
  }

  let affectedAccounts: ProxyAccountSummary[] = []
  if (!success && shouldFetchProxyAccounts(proxy)) {
    try {
      affectedAccounts = await proxiesAPI.getProxyAccounts(proxy.id)
    } catch (error: any) {
      message = `${message} | 读取占用账号失败: ${error?.message || error}`
    }
  }

  return { proxy, success, message, affectedAccounts }
}

function pickTargetProxy(healthyProxies: Proxy[], projectedUsage: Record<number, number>): Proxy | null {
  if (healthyProxies.length === 0) return null
  return [...healthyProxies].sort((a, b) => {
    const aUsage = projectedUsage[a.id] ?? proxyUsageCount(a)
    const bUsage = projectedUsage[b.id] ?? proxyUsageCount(b)
    if (aUsage !== bUsage) return aUsage - bUsage
    return a.id - b.id
  })[0] ?? null
}

export async function previewProxyHealthAutoMigration(sourceProxyIds?: number[]): Promise<ProxyHealthPreview> {
  const proxies = await proxiesAPI.getAllWithCount()
  const selected = sourceProxyIds && sourceProxyIds.length > 0
    ? proxies.filter((proxy) => sourceProxyIds.includes(proxy.id))
    : proxies
  if (selected.length === 0) {
    throw new Error('没有匹配到要测活的代理 ID。')
  }

  const checkedProxies = await mapWithConcurrency(selected, inspectProxyHealth, PROXY_TEST_CONCURRENCY)
  const healthyProxies = checkedProxies.filter((item) => item.success).map((item) => item.proxy)
  const projectedUsage: Record<number, number> = Object.fromEntries(
    healthyProxies.map((proxy) => [proxy.id, proxyUsageCount(proxy)]),
  )

  const assignments: ProxyMigrationAssignment[] = []
  const unassignedChecks: ProxyHealthCheck[] = []
  for (const check of checkedProxies) {
    if (check.success || check.affectedAccounts.length === 0) continue
    const targetProxy = pickTargetProxy(healthyProxies, projectedUsage)
    if (!targetProxy) {
      unassignedChecks.push(check)
      continue
    }
    projectedUsage[targetProxy.id] = (projectedUsage[targetProxy.id] ?? proxyUsageCount(targetProxy)) + check.affectedAccounts.length
    assignments.push({
      sourceProxy: check.proxy,
      targetProxy,
      affectedAccounts: check.affectedAccounts,
    })
  }

  return {
    checkedProxies,
    healthyProxies,
    assignments,
    unassignedChecks,
  }
}

export async function autoMigrateAccountsFromUnhealthyProxies(
  sourceProxyIds?: number[],
  preview?: ProxyHealthPreview,
): Promise<ProxyHealthPreview> {
  const plan = preview ?? await previewProxyHealthAutoMigration(sourceProxyIds)
  if (plan.unassignedChecks.length > 0) {
    throw new Error(`没有足够的可用代理可接收这些不可用代理上的账号: ${plan.unassignedChecks.map((item) => item.proxy.id).join(', ')}`)
  }
  const updateResponses: any[] = []
  for (const assignment of plan.assignments) {
    const accountIds = assignment.affectedAccounts.map((account) => account.id).filter((id) => id > 0)
    if (accountIds.length === 0) continue
    updateResponses.push(...(await batchUpdateAccounts(accountIds, { proxy_id: assignment.targetProxy.id })))
  }
  return {
    ...plan,
    updateResponses,
  }
}

function normalizeOpsEndpoint(endpoint: string): LogEndpoint {
  const trimmed = endpoint.trim().toLowerCase()
  if (!trimmed || trimmed === 'auto') return 'request-errors'
  if (LOG_ENDPOINTS.includes(trimmed as LogEndpoint)) return trimmed as LogEndpoint
  if (trimmed.endsWith('request-errors')) return 'request-errors'
  if (trimmed.endsWith('upstream-errors')) return 'upstream-errors'
  if (trimmed.endsWith('requests')) return 'requests'
  if (trimmed.endsWith('system-logs')) return 'system-logs'
  return 'request-errors'
}

async function fetchOpsPage(
  endpoint: LogEndpoint,
  page: number,
  pageSize: number,
  extraQuery: Record<string, any> = {},
): Promise<any[]> {
  switch (endpoint) {
    case 'request-errors': {
      const data = await opsAPI.listRequestErrors({ page, page_size: pageSize, ...extraQuery })
      return data.items || []
    }
    case 'upstream-errors': {
      const data = await opsAPI.listUpstreamErrors({ page, page_size: pageSize, ...extraQuery })
      return data.items || []
    }
    case 'requests': {
      const data = await opsAPI.listRequestDetails({ page, page_size: pageSize, ...extraQuery })
      return data.items || []
    }
    case 'system-logs': {
      const data = await opsAPI.listSystemLogs({ page, page_size: pageSize, ...extraQuery })
      return data.items || []
    }
  }
}

function toInt(value: any): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) return Math.trunc(value)
  if (typeof value === 'string' && /^\d+$/.test(value.trim())) return Number.parseInt(value.trim(), 10)
  return null
}

function extractStatusValues(item: any): Set<number> {
  const values = new Set<number>()
  if (Array.isArray(item)) {
    for (const nested of item) {
      for (const value of extractStatusValues(nested)) values.add(value)
    }
    return values
  }
  if (!item || typeof item !== 'object') return values

  for (const [key, value] of Object.entries(item)) {
    const lowered = key.toLowerCase()
    if (STATUS_KEYS.has(lowered) || lowered.includes('status') || lowered.endsWith('_code')) {
      const parsed = toInt(value)
      if (parsed != null) values.add(parsed)
    }
    for (const nested of extractStatusValues(value)) values.add(nested)
  }
  return values
}

function extractAccountIds(item: any): Set<number> {
  const accountIds = new Set<number>()
  if (Array.isArray(item)) {
    for (const nested of item) {
      for (const accountId of extractAccountIds(nested)) accountIds.add(accountId)
    }
    return accountIds
  }
  if (!item || typeof item !== 'object') return accountIds

  for (const [key, value] of Object.entries(item)) {
    const lowered = key.toLowerCase()
    if (ACCOUNT_ID_KEYS.has(lowered)) {
      const parsed = toInt(value)
      if (parsed != null) accountIds.add(parsed)
    } else if (lowered === 'account' && value && typeof value === 'object') {
      const parsed = toInt((value as Record<string, any>).id)
      if (parsed != null) accountIds.add(parsed)
    } else if (lowered === 'accounts' && Array.isArray(value)) {
      for (const nested of value) {
        const parsed = toInt(nested?.id)
        if (parsed != null) accountIds.add(parsed)
      }
    }
    for (const nestedId of extractAccountIds(value)) accountIds.add(nestedId)
  }
  return accountIds
}

function itemMatchesKeyword(item: any, keyword: string): boolean {
  const target = keyword.trim().toLowerCase()
  if (!target) return true

  const statusValues = extractStatusValues(item)
  if (/^\d+$/.test(target) && statusValues.has(Number.parseInt(target, 10))) {
    return true
  }

  const serialized = JSON.stringify(item).toLowerCase()
  if (new RegExp(`\\b${target.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\b`).test(serialized)) {
    return true
  }
  if (target === '502' && serialized.includes('bad gateway')) {
    return true
  }
  return false
}

export async function previewAccountsFromOpsLogs(params: {
  endpoint?: string
  pages?: number
  pageSize?: number
  keyword?: string
  extraQuery?: Record<string, any>
} = {}): Promise<BatchResult> {
  const endpoint = params.endpoint === 'auto' || !params.endpoint
    ? null
    : normalizeOpsEndpoint(params.endpoint)
  const endpoints = endpoint ? [endpoint] : [...LOG_ENDPOINTS]
  const pages = Math.max(1, params.pages ?? 3)
  const pageSize = Math.max(1, params.pageSize ?? LOG_PAGE_SIZE)
  const keyword = params.keyword ?? '502'
  const extraQuery = params.extraQuery ?? {}

  const tasks = endpoints.flatMap((candidate) =>
    Array.from({ length: pages }, (_, index) => ({ endpoint: candidate, page: index + 1 })),
  )

  const responses = await mapWithConcurrency(tasks, async (task) => ({
    endpoint: task.endpoint,
    page: task.page,
    items: await fetchOpsPage(task.endpoint, task.page, pageSize, extraQuery),
  }))

  const matches: any[] = []
  const accountIds = new Set<number>()
  const sourcesHit = new Set<string>()
  const firstEmptyPage = new Map<string, number>()

  for (const response of responses) {
    const emptyPage = firstEmptyPage.get(response.endpoint)
    if (emptyPage != null && response.page >= emptyPage) continue
    if (!response.items || response.items.length === 0) {
      firstEmptyPage.set(response.endpoint, response.page)
      continue
    }
    for (const item of response.items) {
      if (!itemMatchesKeyword(item, keyword)) continue
      const ids = extractAccountIds(item)
      if (ids.size === 0) continue
      sourcesHit.add(response.endpoint)
      matches.push(item)
      for (const accountId of ids) accountIds.add(accountId)
    }
  }

  const affectedAccounts = await getAccountsByIds(Array.from(accountIds))
  return {
    affectedAccounts,
    scannedSources: Array.from(sourcesHit).sort(),
    rawMatches: matches.slice(0, 50),
  }
}

export async function migrateAccountsFromOpsLogs(
  targetProxyId: number | null,
  params: {
    endpoint?: string
    pages?: number
    pageSize?: number
    keyword?: string
    extraQuery?: Record<string, any>
  } = {},
): Promise<BatchResult> {
  const preview = await previewAccountsFromOpsLogs(params)
  if (targetProxyId == null && preview.affectedAccounts.length > 0) {
    throw new Error('命中账号后，请先选择目标代理。')
  }
  let updateResponse: any[] = []
  if (targetProxyId != null) {
    const accountIds = preview.affectedAccounts.map((account) => account.id)
    if (accountIds.length > 0) {
      updateResponse = await batchUpdateAccounts(accountIds, { proxy_id: targetProxyId })
    }
  }
  const proxies = await proxiesAPI.getAllWithCount()
  return {
    ...preview,
    targetProxy: targetProxyId != null ? proxies.find((proxy) => proxy.id === targetProxyId) ?? null : null,
    updateResponse,
  }
}

function duplicateIdentity(account: Account, fieldMode: DuplicatePreview['fieldMode']): string {
  switch (fieldMode) {
    case 'email':
      return String(account.extra?.email_address || '').trim()
    case 'name':
      return String(account.name || '').trim()
    case 'username':
      return String((account.extra?.username as string) || '').trim()
    default:
      return String(account.extra?.email_address || account.name || (account.extra?.username as string) || '').trim()
  }
}

export async function previewDuplicateAccounts(params: {
  fieldMode?: DuplicatePreview['fieldMode']
  platform?: string
  search?: string
  status?: string
} = {}): Promise<DuplicatePreview> {
  const fieldMode = params.fieldMode ?? 'display'
  const accounts = await listAllAccounts({
    platform: params.platform || '',
    search: params.search || '',
    status: params.status || '',
  })

  const grouped = new Map<string, Account[]>()
  for (const account of accounts) {
    const identity = duplicateIdentity(account, fieldMode)
    if (!identity) continue
    const key = `${String(account.platform).toLowerCase()}::${identity.toLowerCase()}`
    grouped.set(key, [...(grouped.get(key) ?? []), account])
  }

  const groups: DuplicatePreviewGroup[] = []
  let totalDuplicates = 0
  for (const items of grouped.values()) {
    if (items.length < 2) continue
    const ordered = [...items].sort((a, b) => a.id - b.id)
    const keepId = ordered[0].id
    const deleteIds = ordered.slice(1).map((item) => item.id)
    totalDuplicates += deleteIds.length
    groups.push({
      identity: duplicateIdentity(ordered[0], fieldMode),
      platform: ordered[0].platform,
      keepId,
      deleteIds,
      accounts: ordered,
    })
  }

  groups.sort((a, b) => {
    if (a.platform !== b.platform) return a.platform.localeCompare(b.platform)
    return a.identity.localeCompare(b.identity)
  })

  return { groups, totalDuplicates, fieldMode }
}

export async function deleteAccounts(accountIds: number[]) {
  const normalizedIds = Array.from(new Set(accountIds.map((id) => Number(id)).filter((id) => Number.isFinite(id) && id > 0)))
  return mapWithConcurrency(
    normalizedIds,
    async (accountId) => ({
      accountId,
      response: await accountsAPI.delete(accountId),
    }),
    LOW_WRITE_CONCURRENCY,
  )
}
