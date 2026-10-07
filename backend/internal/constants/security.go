package constants

const (
	RoleObserver = "observer"
	RoleAnalyst  = "analyst"
	RoleReviewer = "reviewer"
	RoleAdmin    = "admin"
)

var Roles = []string{RoleObserver, RoleAnalyst, RoleReviewer, RoleAdmin}

func ValidRole(role string) bool {
	for _, candidate := range Roles {
		if role == candidate {
			return true
		}
	}
	return false
}

func CanReview(role string) bool {
	return role == RoleReviewer || role == RoleAdmin
}

func CanAnalyze(role string) bool {
	return role == RoleAnalyst || role == RoleAdmin
}

func CanObserve(role string) bool {
	return role == RoleObserver || role == RoleAnalyst || role == RoleAdmin
}
