package llm

import "go.mongodb.org/mongo-driver/bson/primitive"

const (
	WorkerRotationRunA = "RUN_A"
	WorkerRotationRunB = "RUN_B"
)

// WorkerRotationPattern alternates worker/reviewer assignment per analysis run.
// RUN A: worker=careful_analyst, reviewer=result_analyst
// RUN B: worker=result_analyst, reviewer=careful_analyst
func WorkerRotationPattern(runID primitive.ObjectID) string {
	if runID[11]%2 == 0 {
		return WorkerRotationRunA
	}
	return WorkerRotationRunB
}

func WorkerPersonaForRotation(pattern string) string {
	if pattern == WorkerRotationRunB {
		return ModelKeyResult
	}
	return ModelKeyCareful
}

func ReviewerPersonaForRotation(pattern string) string {
	if pattern == WorkerRotationRunB {
		return ModelKeyCareful
	}
	return ModelKeyResult
}
