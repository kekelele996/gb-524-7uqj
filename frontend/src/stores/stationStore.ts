import { create } from 'zustand'
import { stationApi, type StationUpdateResponse } from '../api/stations'
import type { ReceiverStation, StationInput } from '../types/station'

interface StationState {
  stations: ReceiverStation[]
  busy: boolean
  load: () => Promise<void>
  createStation: (input: StationInput) => Promise<ReceiverStation>
  updateStation: (id: number, input: Omit<StationInput, 'station_code'>) => Promise<StationUpdateResponse>
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
  },
  updateStation: async (id, input) => {
    const result = await stationApi.update(id, input)
    set({
      stations: get().stations
        .map((station) => (station.id === id ? result.station : station))
        .sort((a, b) => a.station_code.localeCompare(b.station_code))
    })
    return result
  }
}))
