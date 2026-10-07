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

export interface ReintersectionSkip {
  case_id: number
  reason:
    | 'case_finalized'
    | 'no_prior_estimate'
    | 'frequency_mismatch'
    | 'insufficient_observations'
    | 'geometry_degenerate'
}

// 站点信息变更后未结案案例的重新交汇汇总，由 PUT /stations/:id 的 meta 返回。
export interface ReintersectionSummary {
  station_id: number
  localization_changed: boolean
  affected_case_ids: number[]
  recalibrated_observation_count: number
  reintersected_case_ids: number[]
  skipped_cases: ReintersectionSkip[]
}

export const reintersectionSkipText: Record<ReintersectionSkip['reason'], string> = {
  case_finalized: '案例已结案，保持原样',
  no_prior_estimate: '尚无历史定位，待分析阶段手动运行',
  frequency_mismatch: '观测频率与案例不匹配，需人工复核',
  insufficient_observations: '启用站点有效观测不足两条',
  geometry_degenerate: '当前方位几何退化，需人工处理'
}

