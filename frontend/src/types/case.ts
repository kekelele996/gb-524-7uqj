export type CaseStatus = 'draft' | 'collecting' | 'analyzing' | 'pending_review' | 'confirmed' | 'closed'
export type CasePriority = 'low' | 'normal' | 'high'

export interface InterferenceCase {
  id: number
  case_code: string
  title: string
  frequency_center_hz: number
  case_status: CaseStatus
  priority: CasePriority
  opened_by: number
  reviewer_id: number | null
  conclusion: string
  review_reason: string
  version: number
  closed_at: string | null
  created_at: string
  updated_at: string
}

export interface CaseSummary {
  id: number
  case_code: string
  title: string
  frequency_center_hz: number
  case_status: CaseStatus
  priority: CasePriority
  version: number
  observation_count: number
  active_observation_count: number
  estimate_count: number
  conclusion: string
}

export interface CaseInput {
  case_code: string
  title: string
  frequency_center_hz: number
  priority: CasePriority
}

export interface CaseTransition {
  target_status: CaseStatus
  version: number
  conclusion?: string
  reason?: string
}

