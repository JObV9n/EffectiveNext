package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/JobV9n/effectiveNext/pkg/doctor"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate the environment and project setup",
	Long: `doctor checks the local environment for everything effectiveNext needs:
Go version, Node.js, package managers, Next.js installation, and project
configuration. It reports problems and suggestions without modifying any files.`,
	RunE: runDoctor,
}

func init() {
	RootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	checker := doctor.NewChecker(dir)
	results := checker.Run()

	if jsonOut {
		return outputDoctorJSON(cmd, results)
	}
	return outputDoctorText(cmd, results)
}

func outputDoctorJSON(cmd *cobra.Command, results []doctor.Check) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

func outputDoctorText(cmd *cobra.Command, results []doctor.Check) error {
	failed := 0
	var lines []string

	for _, r := range results {
		icon := "✓"
		if !r.Passed {
			icon = "✗"
			failed++
		}
		lines = append(lines, fmt.Sprintf("%s %s: %s", icon, r.Name, r.Message))
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.Join(lines, "\n"))

	if failed > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%d check(s) failed.\n", failed)
		return fmt.Errorf("doctor found %d problem(s)", failed)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nAll checks passed.")
	return nil
}
