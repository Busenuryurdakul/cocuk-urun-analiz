package reviewinsight

import "testing"

func TestClassifySafetyAndPositive(t *testing.T) {
	safe := classifyText("choking hazard small parts", nil)
	if len(safe.kinds) == 0 {
		t.Fatal("expected safety kind")
	}
	pos := classifyText("harika ve sağlam, tavsiye ederim", ptr(5))
	if !pos.positive {
		t.Fatal("expected positive")
	}
}

func ptr(v float64) *float64 { return &v }
