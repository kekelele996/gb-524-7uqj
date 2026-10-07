import { apiClient } from './client'
import type { CaseInput, CaseSummary, CaseTransition, InterferenceCase } from '../types/case'

export const caseApi = {
  list: (status = '') => apiClient.getPage<CaseSummary[]>(`/cases?page_size=100${status ? `&status=${status}` : ''}`),
  get: (id: number) => apiClient.get<InterferenceCase>(`/cases/${id}`),
  create: (input: CaseInput) => apiClient.post<InterferenceCase>('/cases', input),
  transition: (id: number, input: CaseTransition) => apiClient.post<InterferenceCase>(`/cases/${id}/transition`, input)
}

