import { apiClient } from './client'
import type { LocalizationEstimate, LocalizationRunResult } from '../types/localization'

export const localizationApi = {
  list: (caseId?: number) => apiClient.get<LocalizationEstimate[]>(`/localizations${caseId ? `?case_id=${caseId}` : ''}`),
  get: (id: number) => apiClient.get<LocalizationEstimate>(`/localizations/${id}`),
  run: (caseId: number, allowOutlier: boolean) => apiClient.post<LocalizationRunResult>('/localizations/run', {
    case_id: caseId,
    allow_outlier: allowOutlier
  })
}

