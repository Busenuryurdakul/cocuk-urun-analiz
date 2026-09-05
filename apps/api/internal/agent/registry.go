package agent

type ToolAvailability string

const (
	ToolAvailable   ToolAvailability = "AVAILABLE"
	ToolPartial     ToolAvailability = "PARTIAL"
	ToolUnavailable ToolAvailability = "UNAVAILABLE"
)

type SandboxClass string

const (
	SandboxInternal        SandboxClass = "INTERNAL"
	SandboxNetworkOutbound SandboxClass = "NETWORK_OUTBOUND"
)

type ToolDefinition struct {
	Name         string
	Version      string
	Purpose      string
	Availability ToolAvailability
	SandboxClass SandboxClass
	MinRole      string
}

// Frozen 18-tool registry — names must match FINAL_MASTER_PROMPT §13.
var frozenRegistry = []ToolDefinition{
	{Name: "dataset_validator", Version: "frozen-v1", Purpose: "Validate dataset eligibility and records", Availability: ToolAvailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "ecommerce_fetcher", Version: "frozen-v1", Purpose: "Fetch permitted product URLs", Availability: ToolUnavailable, SandboxClass: SandboxNetworkOutbound, MinRole: "ANALYST"},
	{Name: "product_normalizer", Version: "frozen-v1", Purpose: "Normalize product fields", Availability: ToolUnavailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "import_planner", Version: "frozen-v1", Purpose: "Plan import strategy", Availability: ToolUnavailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "fetch_policy_checker", Version: "frozen-v1", Purpose: "Enforce fetch security policies", Availability: ToolAvailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "import_diff_generator", Version: "frozen-v1", Purpose: "Diff import versions", Availability: ToolUnavailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "review_sampler", Version: "frozen-v1", Purpose: "Sample reviews max 100", Availability: ToolAvailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "review_analyzer", Version: "frozen-v1", Purpose: "Analyze sampled reviews", Availability: ToolUnavailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "price_history_analyzer", Version: "frozen-v1", Purpose: "Deterministic price analytics", Availability: ToolUnavailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "safety_analyzer", Version: "frozen-v1", Purpose: "Safety signals", Availability: ToolUnavailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "age_analyzer", Version: "frozen-v1", Purpose: "Target age assessment", Availability: ToolUnavailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "material_analyzer", Version: "frozen-v1", Purpose: "Material analysis", Availability: ToolUnavailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "market_analyzer", Version: "frozen-v1", Purpose: "Market/risk signals", Availability: ToolUnavailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "compliance_checker", Version: "frozen-v1", Purpose: "Compliance validation", Availability: ToolAvailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "pii_redactor", Version: "frozen-v1", Purpose: "PII redaction", Availability: ToolAvailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "policy_evaluator", Version: "frozen-v1", Purpose: "Policy profile evaluation", Availability: ToolAvailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "evidence_validator", Version: "frozen-v1", Purpose: "Validate claim-evidence linkage", Availability: ToolUnavailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
	{Name: "report_generator", Version: "frozen-v1", Purpose: "Generate analysis report", Availability: ToolUnavailable, SandboxClass: SandboxInternal, MinRole: "ANALYST"},
}

type Registry struct {
	tools map[string]ToolDefinition
	order []string
}

func NewRegistry() *Registry {
	r := &Registry{tools: make(map[string]ToolDefinition, len(frozenRegistry))}
	for _, t := range frozenRegistry {
		r.tools[t.Name] = t
		r.order = append(r.order, t.Name)
	}
	return r
}

func (r *Registry) Get(name string) (ToolDefinition, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) IsPlannable(name string) bool {
	t, ok := r.tools[name]
	if !ok {
		return false
	}
	return t.Availability == ToolAvailable
}

func (r *Registry) CanonicalContent() []byte {
	var b []byte
	for _, name := range r.order {
		t := r.tools[name]
		line := name + "|" + t.Version + "|" + string(t.Availability) + "|" + t.Purpose + "\n"
		b = append(b, []byte(line)...)
	}
	return b
}

func (r *Registry) All() []ToolDefinition {
	out := make([]ToolDefinition, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.tools[name])
	}
	return out
}

const MaxReviewSample = 100
