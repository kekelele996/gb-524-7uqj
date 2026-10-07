package localization

import (
	"errors"
	"fmt"
	"math"
)

const (
	EarthRadiusM     = 6371008.8
	AlgorithmVersion = "wls-bearing-v1"
)

var ErrInsufficientObservations = errors.New("at least two active observations are required")

type DegenerateError struct {
	ConditionNumber float64
	Reason          string
}

func (e *DegenerateError) Error() string {
	return fmt.Sprintf("degenerate bearing geometry: %s (condition %.3f)", e.Reason, e.ConditionNumber)
}

type Input struct {
	ObservationID uint    `json:"observation_id"`
	StationCode   string  `json:"station_code"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	BearingDeg    float64 `json:"bearing_deg"`
	AccuracyDeg   float64 `json:"accuracy_deg"`
	QualityWeight float64 `json:"quality_weight"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Residual struct {
	ObservationID uint    `json:"observation_id"`
	StationCode   string  `json:"station_code"`
	ObservedDeg   float64 `json:"observed_deg"`
	PredictedDeg  float64 `json:"predicted_deg"`
	ResidualDeg   float64 `json:"residual_deg"`
	Standardized  float64 `json:"standardized"`
}

type Result struct {
	Point              GeoPoint   `json:"point"`
	LocalPoint         Point      `json:"local_point"`
	ResidualDeg        float64    `json:"residual_deg"`
	ConditionNumber    float64    `json:"condition_number"`
	UncertaintyRadiusM float64    `json:"uncertainty_radius_m"`
	UsedObservationIDs []uint     `json:"used_observation_ids"`
	Residuals          []Residual `json:"residuals"`
	OutlierIDs         []uint     `json:"outlier_ids"`
}

type Run struct {
	Primary   Result  `json:"primary"`
	Candidate *Result `json:"candidate,omitempty"`
}

type matrix2 struct {
	a00 float64
	a01 float64
	a11 float64
	b0  float64
	b1  float64
}

func Solve(inputs []Input, conditionLimit float64) (Result, error) {
	if len(inputs) < 2 {
		return Result{}, ErrInsufficientObservations
	}
	frame := NewFrame(inputs)
	matrix := buildNormalMatrix(frame, inputs)
	point, condition, minEigenvalue, err := matrix.solve(conditionLimit)
	if err != nil {
		return Result{}, err
	}
	residuals, weightedRMS := ComputeResiduals(frame, point, inputs)
	result := Result{
		Point:              frame.ToGeo(point),
		LocalPoint:         point,
		ResidualDeg:        weightedRMS,
		ConditionNumber:    condition,
		UncertaintyRadiusM: estimateUncertainty(frame, point, inputs, weightedRMS, minEigenvalue),
		UsedObservationIDs: make([]uint, 0, len(inputs)),
		Residuals:          residuals,
		OutlierIDs:         []uint{},
	}
	for _, input := range inputs {
		result.UsedObservationIDs = append(result.UsedObservationIDs, input.ObservationID)
	}
	return result, nil
}

func buildNormalMatrix(frame Frame, inputs []Input) matrix2 {
	var matrix matrix2
	for _, input := range inputs {
		station := frame.ToLocal(input.Latitude, input.Longitude)
		direction := BearingDirection(input.BearingDeg)
		normalX := -direction.Y
		normalY := direction.X
		weight := input.QualityWeight / math.Max(input.AccuracyDeg*input.AccuracyDeg, 0.01)
		matrix.a00 += weight * normalX * normalX
		matrix.a01 += weight * normalX * normalY
		matrix.a11 += weight * normalY * normalY
		projection := normalX*station.X + normalY*station.Y
		matrix.b0 += weight * normalX * projection
		matrix.b1 += weight * normalY * projection
	}
	return matrix
}

func (m matrix2) solve(conditionLimit float64) (Point, float64, float64, error) {
	trace := m.a00 + m.a11
	discriminant := math.Sqrt(math.Max((m.a00-m.a11)*(m.a00-m.a11)+4*m.a01*m.a01, 0))
	maxEigenvalue := (trace + discriminant) / 2
	minEigenvalue := (trace - discriminant) / 2
	if minEigenvalue <= 1e-10 || maxEigenvalue <= 0 {
		return Point{}, math.Inf(1), minEigenvalue, &DegenerateError{ConditionNumber: math.Inf(1), Reason: "方位线平行或有效权重不足"}
	}
	condition := maxEigenvalue / minEigenvalue
	if condition > conditionLimit {
		return Point{}, condition, minEigenvalue, &DegenerateError{ConditionNumber: condition, Reason: "方位几何近共线"}
	}
	determinant := m.a00*m.a11 - m.a01*m.a01
	if math.Abs(determinant) < 1e-12 {
		return Point{}, condition, minEigenvalue, &DegenerateError{ConditionNumber: condition, Reason: "法方程不可逆"}
	}
	return Point{
		X: (m.b0*m.a11 - m.a01*m.b1) / determinant,
		Y: (m.a00*m.b1 - m.a01*m.b0) / determinant,
	}, condition, minEigenvalue, nil
}

func estimateUncertainty(frame Frame, point Point, inputs []Input, residualDeg, minEigenvalue float64) float64 {
	var weightedDistance, totalWeight, accuracySquared float64
	for _, input := range inputs {
		station := frame.ToLocal(input.Latitude, input.Longitude)
		distance := math.Hypot(point.X-station.X, point.Y-station.Y)
		weight := input.QualityWeight / math.Max(input.AccuracyDeg*input.AccuracyDeg, 0.01)
		weightedDistance += distance * weight
		accuracySquared += input.AccuracyDeg * input.AccuracyDeg * weight
		totalWeight += weight
	}
	if totalWeight <= 0 {
		return 0
	}
	meanDistance := weightedDistance / totalWeight
	angularError := math.Sqrt(accuracySquared/totalWeight + residualDeg*residualDeg)
	geometryFactor := 1 / math.Sqrt(math.Max(minEigenvalue, 1e-6))
	radius := meanDistance * math.Tan(degreesToRadians(angularError)) * math.Max(1, geometryFactor)
	return math.Max(10, math.Min(radius, 50000))
}
