export type StationStatus = 'active' | 'calibration_due' | 'inactive'

export interface ReceiverStation {
  id: number
  station_code: string
  name: string
  latitude: number
  longitude: number
  antenna_bias_deg: number
  accuracy_deg: number
  station_status: StationStatus
  calibrated_at: string | null
  created_at: string
  updated_at: string
}

export interface StationInput {
  station_code: string
  name: string
  latitude: number
  longitude: number
  antenna_bias_deg: number
  accuracy_deg: number
  station_status: StationStatus
  calibrated_at: string | null
}

export interface StationCoverage {
  station_id: number
  observation_count: number
  last_observed_at: string | null
}

export interface ReintersectedCase {
  case_id: number
  case_code: string
  estimate_id?: number
  status: 'reintersected' | 'skipped'
  reason?: string
}

export interface StationReintersection {
  triggered: boolean
  open_cases: ReintersectedCase[]
  estimates_created: number
  skipped_cases: ReintersectedCase[]
}

