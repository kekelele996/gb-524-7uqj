package service

import (
	"encoding/json"
	"math"
	"testing"

	"spectrum-interference-triangulation/backend/internal/util"
)

func TestJSONSafeNumberSerializesDegenerateCondition(t *testing.T) {
	details := map[string]any{
		"condition_number": util.JSONSafeNumber(math.Inf(1)),
		"reason":           "parallel bearings",
	}

	payload, err := json.Marshal(details)
	if err != nil {
		t.Fatalf("marshal degenerate error details: %v", err)
	}
	if string(payload) != `{"condition_number":"+Infinity","reason":"parallel bearings"}` {
		t.Fatalf("unexpected error details: %s", payload)
	}
}
