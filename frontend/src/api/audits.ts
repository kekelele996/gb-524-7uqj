import { apiClient } from './client'
import type { AuditEvent } from '../types/audit'

export interface AuditFilters {
  actorEmail?: string
  action?: string
  entityType?: string
}

export const auditApi = {
  list: (filters: AuditFilters = {}) => {
    const params = new URLSearchParams({ page_size: '100' })
    if (filters.actorEmail) params.set('actor_email', filters.actorEmail)
    if (filters.action) params.set('action', filters.action)
    if (filters.entityType) params.set('entity_type', filters.entityType)
    return apiClient.getPage<AuditEvent[]>(`/audits?${params}`)
  }
}

