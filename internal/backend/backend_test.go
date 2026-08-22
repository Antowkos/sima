package backend

import (
	"testing"

	"github.com/antowkos/sima/internal/config"
)

func TestDoctorFindsExecutable(t *testing.T) {
	result := Doctor("test", config.BackendProfile{Kind: "codex", Executable: "/bin/echo"})
	if !result.OK {
		t.Fatalf("Doctor() failed: %#v", result)
	}
}

func TestDoctorRejectsMissingExecutable(t *testing.T) {
	result := Doctor("test", config.BackendProfile{Kind: "codex", Executable: "/definitely/missing/sima-test"})
	if result.OK {
		t.Fatalf("Doctor() unexpectedly passed: %#v", result)
	}
}
