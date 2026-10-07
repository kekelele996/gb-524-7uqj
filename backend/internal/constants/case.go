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
