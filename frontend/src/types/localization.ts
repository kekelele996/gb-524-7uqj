export interface ResidualEvidence {
  observation_id: number
  station_code: string
  observed_deg: number
  predicted_deg: number
  residual_deg: number
  standardized: number
}

export interface LocalizationInputSnapshot {
  observation_id: number
  station_code: string
  latitude: number
  longitude: number
  bearing_deg: number
  accuracy_deg: number
  quality_weight: number
}

export interface LocalizationEstimate {
  id: number
  case_id: number
  parent_estimate_id: number | null
  algorithm_version: string
  latitude: number
  longitude: number
  uncertainty_radius_m: number
  residual_deg: number
  condition_number: number
  geometry_degenerate: boolean
  used_observation_ids_json: number[]
  outlier_ids_json: number[]
  residuals_json: ResidualEvidence[]
  input_snapshot_json: LocalizationInputSnapshot[]
  estimate_status: 'complete' | 'degenerate' | 'outlier_candidate'
  created_by: number
  created_at: string
}

export interface LocalizationRunResult {
  primary: LocalizationEstimate
  candidate?: LocalizationEstimate
}

