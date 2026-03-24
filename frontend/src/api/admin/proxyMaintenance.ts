import { apiClient } from '../client'
import type {
  CreateProxyMaintenancePlanRequest,
  ProxyMaintenancePlan,
  ProxyMaintenanceResult,
  UpdateProxyMaintenancePlanRequest,
} from '@/types'

export async function listPlans(): Promise<ProxyMaintenancePlan[]> {
  const { data } = await apiClient.get<ProxyMaintenancePlan[]>('/admin/proxy-maintenance-plans')
  return data ?? []
}

export async function createPlan(req: CreateProxyMaintenancePlanRequest): Promise<ProxyMaintenancePlan> {
  const { data } = await apiClient.post<ProxyMaintenancePlan>('/admin/proxy-maintenance-plans', req)
  return data
}

export async function updatePlan(id: number, req: UpdateProxyMaintenancePlanRequest): Promise<ProxyMaintenancePlan> {
  const { data } = await apiClient.put<ProxyMaintenancePlan>(`/admin/proxy-maintenance-plans/${id}`, req)
  return data
}

export async function deletePlan(id: number): Promise<void> {
  await apiClient.delete(`/admin/proxy-maintenance-plans/${id}`)
}

export async function listResults(planId: number, limit?: number): Promise<ProxyMaintenanceResult[]> {
  const { data } = await apiClient.get<ProxyMaintenanceResult[]>(`/admin/proxy-maintenance-plans/${planId}/results`, {
    params: limit ? { limit } : undefined
  })
  return data ?? []
}

export async function runNow(sourceProxyIDs?: number[]): Promise<ProxyMaintenanceResult> {
  const { data } = await apiClient.post<ProxyMaintenanceResult>('/admin/proxy-maintenance/run-now', {
    source_proxy_ids: sourceProxyIDs ?? []
  }, {
    timeout: 0
  })
  return data
}

export const proxyMaintenanceAPI = {
  listPlans,
  createPlan,
  updatePlan,
  delete: deletePlan,
  listResults,
  runNow
}

export default proxyMaintenanceAPI
