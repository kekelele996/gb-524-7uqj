import { create } from 'zustand'
import { observationApi } from '../api/observations'
import type { BearingObservation, BatchValidation, ObservationInput } from '../types/observation'

interface ObservationState {
  observations: BearingObservation[]
  validation: BatchValidation | null
  busy: boolean
  load: (caseId?: number, stationId?: number) => Promise<void>
  createObservation: (input: ObservationInput) => Promise<BearingObservation>
  excludeObservation: (id: number, reason: string) => Promise<void>
  validateCase: (caseId: number) => Promise<BatchValidation>
}

export const useObservationStore = create<ObservationState>((set, get) => ({
  observations: [],
  validation: null,
  busy: false,
  load: async (caseId, stationId) => {
    set({ busy: true })
    try {
      const response = await observationApi.list(caseId, stationId)
      set({ observations: response.data })
    } finally {
      set({ busy: false })
    }
  },
  createObservation: async (input) => {
    const response = await observationApi.create(input)
    set({ observations: [response.data, ...get().observations] })
    return response.data
  },
  excludeObservation: async (id, reason) => {
    const response = await observationApi.exclude(id, reason)
    set({ observations: get().observations.map((item) => item.id === id ? { ...item, ...response.data } : item) })
  },
  validateCase: async (caseId) => {
    const response = await observationApi.validateCase(caseId)
    set({ validation: response.data })
    return response.data
  }
}))

