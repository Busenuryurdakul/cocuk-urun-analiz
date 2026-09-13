package llm

import "testing"

func TestShouldEscalateQualityLongPrompt(t *testing.T) {
	prompt := stringsRepeat("Bu urun guvenligi icin detayli analiz gerektiriyor. ", 20)
	if !ShouldEscalateQuality("analysis", prompt, "OK") {
		t.Fatal("expected escalation for long prompt")
	}
}

func TestShouldEscalateQualityShortPrompt(t *testing.T) {
	if ShouldEscalateQuality("analysis", "Kisa ozet ver.", "Bu kisa bir ozet.") {
		t.Fatal("did not expect escalation for short prompt")
	}
}

func TestShouldEscalateQualitySkipsReviewTask(t *testing.T) {
	if ShouldEscalateQuality("review", stringsRepeat("x", 500), "y") {
		t.Fatal("review task should not quality-escalate")
	}
}

func TestShouldEscalateQualitySkipsRatingPrediction(t *testing.T) {
	prompt := stringsRepeat("x", 400)
	if ShouldEscalateQuality(TaskTypeRatingPrediction, prompt, "5") {
		t.Fatal("rating_prediction should not quality-escalate short numeric outputs")
	}
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
