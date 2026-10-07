package localization

import "math"

func ComputeResiduals(frame Frame, point Point, inputs []Input) ([]Residual, float64) {
	residuals := make([]Residual, 0, len(inputs))
	var weightedSquares, totalWeight float64
	for _, input := range inputs {
		station := frame.ToLocal(input.Latitude, input.Longitude)
		predicted := BearingFromVector(point.X-station.X, point.Y-station.Y)
		residualValue := normalizeAngleDelta(input.BearingDeg - predicted)
		standardized := math.Abs(residualValue) / math.Max(input.AccuracyDeg, 0.1)
		weight := input.QualityWeight / math.Max(input.AccuracyDeg*input.AccuracyDeg, 0.01)
		weightedSquares += residualValue * residualValue * weight
		totalWeight += weight
		residuals = append(residuals, Residual{
			ObservationID: input.ObservationID,
			StationCode:   input.StationCode,
			ObservedDeg:   input.BearingDeg,
			PredictedDeg:  predicted,
			ResidualDeg:   residualValue,
			Standardized:  standardized,
		})
	}
	if totalWeight == 0 {
		return residuals, 0
	}
	return residuals, math.Sqrt(weightedSquares / totalWeight)
}

func WorstResidual(residuals []Residual) (Residual, bool) {
	var worst Residual
	found := false
	for _, residual := range residuals {
		if !found || residual.Standardized > worst.Standardized {
			worst = residual
			found = true
		}
	}
	return worst, found
}

func ResidualByObservation(residuals []Residual) map[uint]Residual {
	result := make(map[uint]Residual, len(residuals))
	for _, residual := range residuals {
		result[residual.ObservationID] = residual
	}
	return result
}
