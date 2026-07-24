package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/JobV9n/effectiveNext/internal/testutil"
)

func TestBuildCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"build", "--help"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "preprocesses") {
		t.Fatalf("expected help text mentioning preprocess, got:\n%s", out)
	}
}

func TestBuildCommand_Flags(t *testing.T) {
	cmd := buildCmd
	if cmd == nil {
		t.Fatal("buildCmd is nil")
	}

	flags := []string{"workers", "no-cache", "skip-next", "dry-run"}
	for _, flag := range flags {
		f := cmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("expected flag %q to be defined", flag)
		}
	}
}

func TestBuildCommand_DryRun(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "package.json", `{"name":"test"}`)

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"build", "--dry-run"})

	cwd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(cwd) }()

	err := RootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "dry run") && !strings.Contains(out, "preprocess") {
		t.Fatalf("expected dry run or preprocessing output, got:\n%s", out)
	}
}

func TestBuildCommand_SkipNext(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "package.json", `{"name":"test"}`)

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"build", "--skip-next"})

	cwd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(cwd) }()

	err := RootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "skip-next") && !strings.Contains(out, "skipping") {
		t.Fatalf("expected skip-next message, got:\n%s", out)
	}
}

func TestBuildCommand_NoCache(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "package.json", `{"name":"test"}`)

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"build", "--no-cache", "--skip-next"})

	cwd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(cwd) }()

	err := RootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDevCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"dev", "--help"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "file watcher") && !strings.Contains(out, "incremental") {
		t.Fatalf("expected help text mentioning file watcher or incremental, got:\n%s", out)
	}
}

func TestDevCommand_Flags(t *testing.T) {
	cmd := devCmd
	if cmd == nil {
		t.Fatal("devCmd is nil")
	}

	flags := []string{"port", "hostname"}
	for _, flag := range flags {
		f := cmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("expected flag %q to be defined", flag)
		}
	}
}

func TestCleanCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"clean", "--help"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Remove") && !strings.Contains(out, ".effective-next") {
		t.Fatalf("expected help text, got:\n%s", out)
	}
}

func TestAnalyzeCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"analyze", "--help"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "analyze") && !strings.Contains(out, "Module") {
		t.Fatalf("expected help text, got:\n%s", out)
	}
}

func TestGraphCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"graph", "--help"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "dependency") && !strings.Contains(out, "Dependency") {
		t.Fatalf("expected help text, got:\n%s", out)
	}
}

func TestBenchmarkCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"benchmark", "--help"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "benchmark") {
		t.Fatalf("expected help text, got:\n%s", out)
	}
}

func TestWatchCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"watch", "--help"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Watch") && !strings.Contains(out, "watch") {
		t.Fatalf("expected help text, got:\n%s", out)
	}
}

func TestCacheCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"cache", "--help"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "cache") {
		t.Fatalf("expected help text, got:\n%s", out)
	}
}

func TestRootCommand_GlobalFlags(t *testing.T) {
	cmd := RootCmd
	if cmd == nil {
		t.Fatal("RootCmd is nil")
	}

	flags := []string{"config", "verbose", "quiet", "json"}
	for _, flag := range flags {
		f := cmd.PersistentFlags().Lookup(flag)
		if f == nil {
			t.Errorf("expected persistent flag %q to be defined", flag)
		}
	}
}

func TestRootCommand_Version(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"--version"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "effective-next") {
		t.Fatalf("expected version output, got:\n%s", out)
	}
}
