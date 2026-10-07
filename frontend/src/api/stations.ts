import { apiClient } from './client'
import type { MetaResponse } from '../types/api'
import type { ReceiverStation, ReintersectionSummary, StationCoverage, StationInput } from '../types/station'

export interface StationUpdateMeta {
  reintersection: ReintersectionSummary
}

export type StationUpdateResponse = MetaResponse<ReceiverStation, StationUpdateMeta>

export const stationApi = {
  list: (status = '') => apiClient.getPage<ReceiverStation[]>(`/stations?page_size=100${status ? `&status=${status}` : ''}`),
  get: (id: number) => apiClient.get<ReceiverStation>(`/stations/${id}`),
  coverage: (id: number) => apiClient.get<StationCoverage>(`/stations/${id}/coverage`),
  create: (input: StationInput) => apiClient.post<ReceiverStation>('/stations', input),
  update: (id: number, input: Omit<StationInput, 'station_code'>) =>
    apiClient.putMeta<ReceiverStation, StationUpdateMeta>(`/stations/${id}`, input)
}
