package contracts

import (
	"encoding/json"
	"strings"
)

var MemoryTypes = []string{"decision", "invariant", "gotcha", "workflow", "guardrail", "anti_pattern", "open_question"}
var ProposalOperations = []string{"create", "update", "deprecate", "supersede"}
var ProposalScopes = []string{"personal", "team"}
var SafetyDecisions = []string{"safe", "suspicious", "unsafe"}
var ArchivistDecisions = []string{"apply", "reject", "defer"}
var LearningDestinations = []string{"memory", "skill", "mixed", "session_only", "reject"}
var LearningTargetKinds = []string{"memory", "skill"}
var ProposalStatuses = []string{"candidate", "applied", "rejected", "deferred", "session_only"}
var CandidateSources = []string{"structured", "structured_invalid"}

func Join(values []string) string {
	return strings.Join(values, "|")
}

func EnglishList(values []string) string {
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return values[0]
	}
	return strings.Join(values[:len(values)-1], ", ") + ", or " + values[len(values)-1]
}

var WorkerJSONSchema = `{
  "type": "object",
  "properties": {
    "status": {"type": "string"},
    "proposed_memory": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "type": {"type": "string", "enum": ` + jsonArray(MemoryTypes) + `},
          "title": {"type": "string"},
          "trigger": {"type": "string"},
          "summary": {"type": "string"}
        },
        "required": ["type", "title", "trigger", "summary"],
        "additionalProperties": false
      }
    },
    "proposed_skills": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "trigger": {"type": "string"},
          "summary": {"type": "string"}
        },
        "required": ["name", "trigger", "summary"],
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`

var ArchivistJSONSchema = `{
  "type": "object",
  "properties": {
    "decision": {"type": "string", "enum": ` + jsonArray(ArchivistDecisions) + `},
    "learning": {
      "type": "object",
      "properties": {
        "destination": {"type": "string", "enum": ` + jsonArray(LearningDestinations) + `},
        "operation": {"type": "string", "enum": ` + jsonArray(ProposalOperations) + `},
        "target": {
          "type": "object",
          "properties": {
            "kind": {"type": "string", "enum": ` + jsonArray(LearningTargetKinds) + `},
            "path": {"type": "string"},
            "id": {"type": "string"}
          },
          "additionalProperties": false
        },
        "quality": {
          "type": "object",
          "properties": {
            "durable": {"type": "boolean"},
            "triggerable": {"type": "boolean"},
            "evidence_backed": {"type": "boolean"},
            "non_transient": {"type": "boolean"},
            "reusable": {"type": "boolean"}
          },
          "required": ["durable", "triggerable", "evidence_backed", "non_transient", "reusable"],
          "additionalProperties": false
        },
        "notes": {"type": "array", "items": {"type": "string"}}
      },
      "required": ["destination", "operation", "quality"],
      "additionalProperties": false
    },
    "notes": {"type": "array", "items": {"type": "string"}}
  },
  "required": ["decision", "learning", "notes"],
  "additionalProperties": false
}`

func jsonArray(values []string) string {
	data, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	return string(data)
}
