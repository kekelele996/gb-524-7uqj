package constants

type CaseStatus string

const (
	CaseDraft         CaseStatus = "draft"
	CaseCollecting    CaseStatus = "collecting"
	CaseAnalyzing     CaseStatus = "analyzing"
	CasePendingReview CaseStatus = "pending_review"
	CaseConfirmed     CaseStatus = "confirmed"
	CaseClosed        CaseStatus = "closed"
)

var CaseStatuses = []CaseStatus{
	CaseDraft, CaseCollecting, CaseAnalyzing,
	CasePendingReview, CaseConfirmed, CaseClosed,
}

// OpenCaseStatuses 是尚未结案、仍允许以站点当前值重新交汇的案例状态。
// confirmed 与 closed 已形成人工结论，历史观测校正值和定位结果必须冻结。
var OpenCaseStatuses = []CaseStatus{
	CaseDraft, CaseCollecting, CaseAnalyzing, CasePendingReview,
}

// IsOpenCaseStatus 判断案例是否尚未结案（可以参与站点变更后的重新交汇）。
func IsOpenCaseStatus(status CaseStatus) bool {
	for _, candidate := range OpenCaseStatuses {
		if status == candidate {
			return true
		}
	}
	return false
}

// OpenCaseStatusValues 返回 SQL IN 条件可用的未结案状态字符串。
func OpenCaseStatusValues() []string {
	values := make([]string, 0, len(OpenCaseStatuses))
	for _, status := range OpenCaseStatuses {
		values = append(values, string(status))
	}
	return values
}

func ValidCaseStatus(value CaseStatus) bool {
	for _, status := range CaseStatuses {
		if status == value {
			return true
		}
	}
	return false
}

func CanTransitionCase(from, to CaseStatus) bool {
	switch from {
	case CaseDraft:
		return to == CaseCollecting
	case CaseCollecting:
		return to == CaseAnalyzing
	case CaseAnalyzing:
		return to == CasePendingReview
	case CasePendingReview:
		return to == CaseConfirmed || to == CaseAnalyzing
	case CaseConfirmed:
		return to == CaseClosed
	default:
		return false
	}
}

func CaseStatusValues() []string {
	values := make([]string, 0, len(CaseStatuses))
	for _, status := range CaseStatuses {
		values = append(values, string(status))
	}
	return values
}
