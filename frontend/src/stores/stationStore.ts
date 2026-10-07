import { create } from 'zustand'
import { stationApi } from '../api/stations'
import type { ReceiverStation, StationInput } from '../types/station'

interface StationState {
  stations: ReceiverStation[]
  busy: boolean
  load: () => Promise<void>
  createStation: (input: StationInput) => Promise<ReceiverStation>
}

export const useStationStore = create<StationState>((set, get) => ({
  stations: [],
  busy: false,
  load: async () => {
    set({ busy: true })
    try {
      const response = await stationApi.list()
      set({ stations: response.data })
    } finally {
      set({ busy: false })
    }
  },
  createStation: async (input) => {
    const response = await stationApi.create(input)
    set({ stations: [...get().stations, response.data].sort((a, b) => a.station_code.localeCompare(b.station_code)) })
    return response.data
  }
}))

