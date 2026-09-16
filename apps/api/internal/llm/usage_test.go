package llm

import "testing"

func TestIsUpgradeOrFallback(t *testing.T) {
	cases := []struct {
		name     string
		fallback bool
		reason   string
		want     bool
	}{
		{name: "plain analysis", reason: "task=analysis;primary=careful_analyst;fallback=result_analyst;policy=1.0.0"},
		{name: "deep analysis task is not an upgrade", reason: "task=deep_analysis;primary=result_analyst;fallback=careful_analyst;policy=1.0.0"},
		{name: "de-escalated is not an upgrade", reason: "policy=de-escalated;task=analysis"},
		{name: "quality_escalation false is not an upgrade", reason: "task=analysis;quality_escalation=false"},
		{name: "quality escalation", reason: "task=analysis;primary=careful_analyst;fallback=result_analyst;quality_escalation=true", want: true},
		{name: "fallback flag", fallback: true, reason: "task=analysis;primary=careful_analyst;fallback=result_analyst", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsUpgradeOrFallback(tc.fallback, tc.reason)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
