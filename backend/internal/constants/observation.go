package constants

type ObservationQuality string

const (
	QualityGood     ObservationQuality = "good"
	QualityFair     ObservationQuality = "fair"
	QualityPoor     ObservationQuality = "poor"
	QualityExcluded ObservationQuality = "excluded"
)

var ObservationQualities = []ObservationQuality{
	QualityGood, QualityFair, QualityPoor, QualityExcluded,
}

func ValidObservationQuality(value ObservationQuality) bool {
	for _, quality := range ObservationQualities {
		if quality == value {
			return true
		}
	}
	return false
}

func QualityWeight(value ObservationQuality) float64 {
	switch value {
	case QualityGood:
		return 1
	case QualityFair:
		return 0.55
	case QualityPoor:
		return 0.2
	default:
		return 0
	}
}

const (
	EstimateComplete   = "complete"
	EstimateDegenerate = "degenerate"
	EstimateCandidate  = "outlier_candidate"
)
