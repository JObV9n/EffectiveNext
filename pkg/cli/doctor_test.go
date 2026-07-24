package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/JobV9n/effectiveNext/internal/testutil"
)

func TestDoctorCommand_TextOutput(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "package.json", `{"name":"demo","dependencies":{"next":"^14.0.0"}}`)
	testutil.WriteFile(t, dir, "next.config.js", "module.exports = {}\n")

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"doctor"})

	// Save and restore cwd.
	cwd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(cwd) }()

	err := RootCmd.Execute()
	out := buf.String()

	if !strings.Contains(out, "go-version") {
		t.Fatalf("expected go-version in output, got:\n%s", out)
	}
	if !strings.Contains(out, "project-directory") {
		t.Fatalf("expected project-directory in output, got:\n%s", out)
	}
	// doctor may fail due to missing node/next binaries in CI, but command runs.
	_ = err
}

func TestDoctorCommand_JSONOutput(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "package.json", `{"name":"demo"}`)

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"doctor", "--json"})

	cwd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(cwd) }()

	err := RootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(strings.TrimSpace(out), "[") {
		t.Fatalf("expected JSON array output, got:\n%s", out)
	}
}
