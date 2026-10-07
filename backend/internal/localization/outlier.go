package localization

const (
	outlierStandardizedThreshold = 2.5
	minimumImprovementRatio      = 0.2
)

func SolveWithOutlierCandidate(inputs []Input, conditionLimit float64, allowOutlier bool) (Run, error) {
	primary, err := Solve(inputs, conditionLimit)
	if err != nil {
		return Run{}, err
	}
	run := Run{Primary: primary}
	if !allowOutlier || len(inputs) < 4 {
		return run, nil
	}
	worst, found := WorstResidual(primary.Residuals)
	if !found || worst.Standardized < outlierStandardizedThreshold {
		return run, nil
	}
	remaining := make([]Input, 0, len(inputs)-1)
	for _, input := range inputs {
		if input.ObservationID != worst.ObservationID {
			remaining = append(remaining, input)
		}
	}
	if len(remaining) < 3 {
		return run, nil
	}
	candidate, err := Solve(remaining, conditionLimit)
	if err != nil {
		return run, nil
	}
	if primary.ResidualDeg <= 0 || candidate.ResidualDeg > primary.ResidualDeg*(1-minimumImprovementRatio) {
		return run, nil
	}
	primary.OutlierIDs = []uint{worst.ObservationID}
	candidate.OutlierIDs = []uint{worst.ObservationID}
	run.Primary = primary
	run.Candidate = &candidate
	return run, nil
}
