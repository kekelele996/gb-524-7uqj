package localization

import (
	"errors"
	"math"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	inputs := []Input{{Latitude: 31.23, Longitude: 121.47}, {Latitude: 31.24, Longitude: 121.48}}
	frame := NewFrame(inputs)
	point := frame.ToLocal(31.23123, 121.47567)
	geo := frame.ToGeo(point)
	if math.Abs(geo.Latitude-31.23123) > 1e-8 || math.Abs(geo.Longitude-121.47567) > 1e-8 {
		t.Fatalf("round trip mismatch: %+v", geo)
	}
}

func TestSolvePerpendicularBearings(t *testing.T) {
	inputs := []Input{
		{ObservationID: 1, StationCode: "W", Latitude: 31.23, Longitude: 121.45, BearingDeg: 90, AccuracyDeg: 1, QualityWeight: 1},
		{ObservationID: 2, StationCode: "S", Latitude: 31.21, Longitude: 121.47, BearingDeg: 0, AccuracyDeg: 1, QualityWeight: 1},
		{ObservationID: 3, StationCode: "E", Latitude: 31.23, Longitude: 121.49, BearingDeg: 270, AccuracyDeg: 1, QualityWeight: 1},
	}
	result, err := Solve(inputs, 1000)
	if err != nil {
		t.Fatalf("solve returned error: %v", err)
	}
	if math.Abs(result.Point.Latitude-31.23) > 0.0001 || math.Abs(result.Point.Longitude-121.47) > 0.0001 {
		t.Fatalf("unexpected estimate: %+v", result.Point)
	}
	if result.ResidualDeg > 0.01 {
		t.Fatalf("expected near-zero residual, got %.6f", result.ResidualDeg)
	}
}

func TestSolveRejectsParallelGeometry(t *testing.T) {
	inputs := []Input{
		{ObservationID: 1, Latitude: 31.20, Longitude: 121.45, BearingDeg: 0, AccuracyDeg: 1, QualityWeight: 1},
		{ObservationID: 2, Latitude: 31.21, Longitude: 121.46, BearingDeg: 0, AccuracyDeg: 1, QualityWeight: 1},
		{ObservationID: 3, Latitude: 31.22, Longitude: 121.47, BearingDeg: 180, AccuracyDeg: 1, QualityWeight: 1},
	}
	_, err := Solve(inputs, 1000)
	var degenerate *DegenerateError
	if !errors.As(err, &degenerate) {
		t.Fatalf("expected DegenerateError, got %v", err)
	}
}

func TestOutlierCandidateKeepsPrimaryEvidence(t *testing.T) {
	inputs := []Input{
		{ObservationID: 1, StationCode: "W", Latitude: 31.23, Longitude: 121.44, BearingDeg: 90, AccuracyDeg: 1, QualityWeight: 1},
		{ObservationID: 2, StationCode: "S", Latitude: 31.20, Longitude: 121.47, BearingDeg: 0, AccuracyDeg: 1, QualityWeight: 1},
		{ObservationID: 3, StationCode: "E", Latitude: 31.23, Longitude: 121.50, BearingDeg: 270, AccuracyDeg: 1, QualityWeight: 1},
		{ObservationID: 4, StationCode: "N", Latitude: 31.26, Longitude: 121.47, BearingDeg: 225, AccuracyDeg: 1, QualityWeight: 1},
	}
	run, err := SolveWithOutlierCandidate(inputs, 1000, true)
	if err != nil {
		t.Fatalf("solve with outlier returned error: %v", err)
	}
	if run.Candidate == nil {
		t.Fatal("expected an explainable outlier candidate")
	}
	if len(run.Primary.UsedObservationIDs) != 4 || len(run.Primary.OutlierIDs) != 1 {
		t.Fatalf("primary evidence was not retained: %+v", run.Primary)
	}
	if len(run.Candidate.UsedObservationIDs) != 3 {
		t.Fatalf("candidate should use three observations: %+v", run.Candidate)
	}
}
