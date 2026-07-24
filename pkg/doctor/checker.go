// Package doctor validates the environment and project setup for effective-next.
package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Status describes the severity of a check result.
type Status string

const (
	StatusOK       Status = "ok"
	StatusWarning  Status = "warning"
	StatusError    Status = "error"
	StatusInfo     Status = "info"
)

// Check represents a single doctor check result.
type Check struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Status  Status `json:"status"`
	Message string `json:"message"`
}

// Checker runs a series of environment and project checks.
type Checker struct {
	projectDir string
}

// NewChecker returns a Checker for the given project directory.
func NewChecker(projectDir string) *Checker {
	return &Checker{projectDir: projectDir}
}

// Run executes all checks and returns the results.
func (c *Checker) Run() []Check {
	checks := []func() Check{
		c.checkGoVersion,
		c.checkOperatingSystem,
		c.checkNode,
		c.checkPackageManager,
		c.checkNextJS,
		c.checkProjectConfig,
		c.checkProjectDirectory,
	}

	results := make([]Check, 0, len(checks))
	for _, fn := range checks {
		results = append(results, fn())
	}
	return results
}

func (c *Checker) checkGoVersion() Check {
	if runtime.Version() == "" {
		return Check{Name: "go-version", Passed: false, Status: StatusError, Message: "unable to determine Go version"}
	}
	major, minor, err := parseGoVersion(runtime.Version())
	if err != nil {
		return Check{Name: "go-version", Passed: false, Status: StatusWarning, Message: fmt.Sprintf("could not parse Go version %q: %v", runtime.Version(), err)}
	}
	text := fmt.Sprintf("Go %d.%d detected", major, minor)
	if major < 1 || (major == 1 && minor < 24) {
		return Check{Name: "go-version", Passed: false, Status: StatusError, Message: text + " (Go 1.24+ required)"}
	}
	return Check{Name: "go-version", Passed: true, Status: StatusOK, Message: text}
}

func (c *Checker) checkOperatingSystem() Check {
	return Check{Name: "operating-system", Passed: true, Status: StatusInfo, Message: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)}
}

func (c *Checker) checkNode() Check {
	out, err := execCommand("node", "--version")
	if err != nil {
		return Check{Name: "node", Passed: false, Status: StatusWarning, Message: "node not found in PATH; required for Next.js builds"}
	}
	version := strings.TrimSpace(out)
	return Check{Name: "node", Passed: true, Status: StatusOK, Message: fmt.Sprintf("node %s", version)}
}

func (c *Checker) checkPackageManager() Check {
	for _, pm := range []string{"pnpm", "yarn", "npm"} {
		out, err := execCommand(pm, "--version")
		if err == nil {
			version := strings.TrimSpace(out)
			return Check{Name: "package-manager", Passed: true, Status: StatusOK, Message: fmt.Sprintf("%s %s", pm, version)}
		}
	}
	return Check{Name: "package-manager", Passed: false, Status: StatusWarning, Message: "no package manager found (pnpm, yarn, npm)"}
}

func (c *Checker) checkNextJS() Check {
	packageJSON := filepath.Join(c.projectDir, "package.json")
	data, err := os.ReadFile(packageJSON)
	if err != nil {
		return Check{Name: "nextjs", Passed: false, Status: StatusWarning, Message: "package.json not found; cannot verify Next.js"}
	}
	content := string(data)
	if !strings.Contains(content, "next") {
		return Check{Name: "nextjs", Passed: false, Status: StatusWarning, Message: "Next.js not detected in package.json"}
	}
	out, err := execCommand("node", "-e", "console.log(require('next/package.json').version)")
	if err != nil {
		return Check{Name: "nextjs", Passed: false, Status: StatusWarning, Message: "Next.js present in package.json but node_modules/next may be missing; run install"}
	}
	version := strings.TrimSpace(out)
	major, err := parseSemverMajor(version)
	if err != nil || major < 14 {
		return Check{Name: "nextjs", Passed: false, Status: StatusWarning, Message: fmt.Sprintf("Next.js %s detected (14+ recommended)", version)}
	}
	return Check{Name: "nextjs", Passed: true, Status: StatusOK, Message: fmt.Sprintf("Next.js %s", version)}
}

func (c *Checker) checkProjectConfig() Check {
	for _, name := range []string{"next.config.js", "next.config.mjs", "next.config.ts"} {
		path := filepath.Join(c.projectDir, name)
		if _, err := os.Stat(path); err == nil {
			return Check{Name: "project-config", Passed: true, Status: StatusOK, Message: fmt.Sprintf("found %s", name)}
		}
	}
	return Check{Name: "project-config", Passed: false, Status: StatusWarning, Message: "no next.config file found"}
}

func (c *Checker) checkProjectDirectory() Check {
	info, err := os.Stat(c.projectDir)
	if err != nil {
		return Check{Name: "project-directory", Passed: false, Status: StatusError, Message: fmt.Sprintf("cannot stat project directory: %v", err)}
	}
	if !info.IsDir() {
		return Check{Name: "project-directory", Passed: false, Status: StatusError, Message: "project path is not a directory"}
	}
	return Check{Name: "project-directory", Passed: true, Status: StatusOK, Message: fmt.Sprintf("project directory is readable: %s", c.projectDir)}
}

// execCommand is a testable wrapper around exec.Command.
var execCommand = func(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = "/"
	out, err := cmd.CombinedOutput()
	return string(out), err
}

var goVersionRe = regexp.MustCompile(`^go(\d+)\.(\d+)`)

func parseGoVersion(v string) (major, minor int, err error) {
	v = strings.TrimPrefix(v, "go")
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("invalid version format")
	}
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return major, minor, nil
}

var semverRe = regexp.MustCompile(`^(\d+)(?:\.|$)`)

func parseSemverMajor(v string) (int, error) {
	v = strings.TrimPrefix(v, "v")
	m := semverRe.FindStringSubmatch(v)
	if m == nil {
		return 0, fmt.Errorf("invalid semver")
	}
	return strconv.Atoi(m[1])
}
