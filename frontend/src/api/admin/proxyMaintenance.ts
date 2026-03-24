import { apiClient } from '../client'
import type {
  CreateProxyMaintenancePlanRequest,
  ProxyMaintenancePlan,
  ProxyMaintenanceResult,
  ProxyMaintenanceTask,
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

export async function runNow(sourceProxyIDs?: number[]): Promise<ProxyMaintenanceTask> {
  const { data } = await apiClient.post<ProxyMaintenanceTask>('/admin/proxy-maintenance/run-now', {
    source_proxy_ids: sourceProxyIDs ?? []
  }, {
    timeout: 0
  })
  return data
}

export async function getTask(taskID: string): Promise<ProxyMaintenanceTask> {
  const { data } = await apiClient.get<ProxyMaintenanceTask>(`/admin/proxy-maintenance/tasks/${taskID}`, {
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
  runNow,
  getTask
}

export default proxyMaintenanceAPI
