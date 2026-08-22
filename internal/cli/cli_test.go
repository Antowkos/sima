package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"sima", "version"}, &out, &err)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, err.String())
	}
	if !strings.Contains(out.String(), "sima ") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"sima", "wat"}, &out, &err)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %s", code, err.String())
	}
	if !strings.Contains(err.String(), "unknown command") {
		t.Fatalf("unexpected stderr: %q", err.String())
	}
}
