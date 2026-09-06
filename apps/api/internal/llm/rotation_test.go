package llm

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestWorkerRotationPatternAlternates(t *testing.T) {
	a := primitive.NewObjectID()
	b := primitive.NewObjectID()
	if a == b {
		t.Fatal("expected distinct object ids")
	}
	pa := WorkerRotationPattern(a)
	pb := WorkerRotationPattern(b)
	if pa != WorkerRotationRunA && pa != WorkerRotationRunB {
		t.Fatalf("unexpected pattern %q", pa)
	}
	if pb != WorkerRotationRunA && pb != WorkerRotationRunB {
		t.Fatalf("unexpected pattern %q", pb)
	}
	if WorkerPersonaForRotation(WorkerRotationRunA) != ModelKeyCareful {
		t.Fatal("RUN_A worker persona mismatch")
	}
	if ReviewerPersonaForRotation(WorkerRotationRunA) != ModelKeyResult {
		t.Fatal("RUN_A reviewer persona mismatch")
	}
	if WorkerPersonaForRotation(WorkerRotationRunB) != ModelKeyResult {
		t.Fatal("RUN_B worker persona mismatch")
	}
	if ReviewerPersonaForRotation(WorkerRotationRunB) != ModelKeyCareful {
		t.Fatal("RUN_B reviewer persona mismatch")
	}
	if WorkerPersonaForRotation(WorkerRotationRunA) == ReviewerPersonaForRotation(WorkerRotationRunA) {
		t.Fatal("worker and reviewer must differ")
	}
}
