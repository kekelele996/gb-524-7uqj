import { apiClient } from './client'
import type { ReceiverStation, StationCoverage, StationInput } from '../types/station'

export const stationApi = {
  list: (status = '') => apiClient.getPage<ReceiverStation[]>(`/stations?page_size=100${status ? `&status=${status}` : ''}`),
  get: (id: number) => apiClient.get<ReceiverStation>(`/stations/${id}`),
  coverage: (id: number) => apiClient.get<StationCoverage>(`/stations/${id}/coverage`),
  create: (input: StationInput) => apiClient.post<ReceiverStation>('/stations', input),
  update: (id: number, input: Omit<StationInput, 'station_code'>) => apiClient.put<ReceiverStation>(`/stations/${id}`, input)
}

