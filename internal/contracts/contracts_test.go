package contracts

import (
	"encoding/json"
	"testing"
)

func TestSchemasAreValidJSON(t *testing.T) {
	for name, schema := range map[string]string{
		"worker":    WorkerJSONSchema,
		"archivist": ArchivistJSONSchema,
	} {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
			t.Fatalf("%s schema is invalid JSON: %v", name, err)
		}
		if parsed["additionalProperties"] != false {
			t.Fatalf("%s schema must reject top-level additional properties", name)
		}
	}
}

func TestMemoryTypeContractExcludesRejectedApproach(t *testing.T) {
	for _, value := range MemoryTypes {
		if value == "rejected_approach" {
			t.Fatal("memory type contract should use anti_pattern/guardrail instead of rejected_approach")
		}
	}
	if Join(MemoryTypes) != "decision|invariant|gotcha|workflow|guardrail|anti_pattern|open_question" {
		t.Fatalf("unexpected memory type contract: %s", Join(MemoryTypes))
	}
}
