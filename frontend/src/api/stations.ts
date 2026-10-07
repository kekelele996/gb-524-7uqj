import { apiClient } from './client'
import type { ApiResponse } from '../types/api'
import type { ReceiverStation, StationCoverage, StationInput, StationReintersection } from '../types/station'

export interface StationUpdateResponse {
  station: ReceiverStation
  reintersection: StationReintersection | null
}

type StationUpdateApiResponse = ApiResponse<ReceiverStation> & {
  meta?: { reintersection?: StationReintersection }
}

export const stationApi = {
  list: (status = '') => apiClient.getPage<ReceiverStation[]>(`/stations?page_size=100${status ? `&status=${status}` : ''}`),
  get: (id: number) => apiClient.get<ReceiverStation>(`/stations/${id}`),
  coverage: (id: number) => apiClient.get<StationCoverage>(`/stations/${id}/coverage`),
  create: (input: StationInput) => apiClient.post<ReceiverStation>('/stations', input),
  update: async (id: number, input: Omit<StationInput, 'station_code'>): Promise<StationUpdateResponse> => {
    const response = await apiClient.put<ReceiverStation, StationUpdateApiResponse>(`/stations/${id}`, input)
    return {
      station: response.data,
      reintersection: response.meta?.reintersection ?? null
    }
  }
}

