package agent

type ToolCapability struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Availability string `json:"availability"`
}

type CapabilitySnapshot struct {
	ToolRegistryVersion      string           `json:"toolRegistryVersion"`
	ToolPolicyVersion        string           `json:"toolPolicyVersion"`
	PlannerVersion           string           `json:"plannerVersion"`
	ObservationSchemaVersion string           `json:"observationSchemaVersion"`
	MaxToolLoopIterations    int              `json:"maxToolLoopIterations"`
	TotalRunTimeoutSeconds   int              `json:"totalRunTimeoutSeconds"`
	AvailableTools           []ToolCapability `json:"availableTools"`
}

func (s *Service) BuildCapabilitySnapshot() CapabilitySnapshot {
	tools := make([]ToolCapability, 0)
	for _, def := range s.Registry.All() {
		if def.Availability != ToolAvailable {
			continue
		}
		tools = append(tools, ToolCapability{
			Name:         def.Name,
			Version:      def.Version,
			Availability: string(def.Availability),
		})
	}
	return CapabilitySnapshot{
		ToolRegistryVersion:      RegistryVersion(s.Registry),
		ToolPolicyVersion:        ToolPolicyVersion(s.Registry),
		PlannerVersion:           PythonPlannerVersion(),
		ObservationSchemaVersion: ObservationSchemaVersion(),
		MaxToolLoopIterations:    DefaultMaxToolLoopIterations,
		TotalRunTimeoutSeconds:   int(DefaultTotalRunTimeout.Seconds()),
		AvailableTools:           tools,
	}
}

// PythonPlannerVersion is the canonical planner version label exposed to Python orchestrator.
func PythonPlannerVersion() string {
	return "python-deterministic-planner-v1"
}
