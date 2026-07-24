package cli

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/JobV9n/effectiveNext/pkg/config"
	"github.com/JobV9n/effectiveNext/pkg/graph"
	"github.com/JobV9n/effectiveNext/pkg/hash"
	"github.com/JobV9n/effectiveNext/pkg/scanner"
	"github.com/spf13/cobra"
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run performance benchmarks",
	Long: `benchmark runs performance tests and generates reports:
- Cold build time
- File scanning throughput
- Parser speed
- Memory and disk usage`,
	RunE: runBenchmark,
}

var (
	benchSuite  string
	benchOutput string
	benchFormat string
)

func init() {
	benchmarkCmd.Flags().StringP("suite", "s", "all", "Benchmark suites to run: all, scan, hash, graph")
	benchmarkCmd.Flags().StringP("output", "o", "", "Output report file")
	benchmarkCmd.Flags().StringP("format", "f", "text", "Report format: json, text")
	RootCmd.AddCommand(benchmarkCmd)
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	results := make(map[string]float64)

	if benchSuite == "all" || benchSuite == "scan" {
		cmd.PrintErrln("effective-next: benchmarking file scanner...")
		start := time.Now()
		s := scanner.New(scanner.Options{
			Root:    dir,
			Config:  config.Default().Scanner,
			Workers: runtime.NumCPU(),
		})
		files, err := s.Scan()
		if err != nil {
			return fmt.Errorf("scan benchmark: %w", err)
		}
		scanDuration := time.Since(start)
		results["scan_duration_ms"] = float64(scanDuration.Milliseconds())
		results["scan_files"] = float64(len(files))
		if scanDuration.Seconds() > 0 {
			results["scan_throughput"] = float64(len(files)) / scanDuration.Seconds()
		}
		cmd.PrintErrln(fmt.Sprintf("  scanned %d files in %v (%.0f files/s)", len(files), scanDuration.Round(time.Millisecond), results["scan_throughput"]))
	}

	if benchSuite == "all" || benchSuite == "hash" {
		cmd.PrintErrln("effective-next: benchmarking content hashing...")
		data := make([]byte, 1024*1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
		iterations := 10000
		start := time.Now()
		for i := 0; i < iterations; i++ {
			hash.Hash64(data)
		}
		hashDuration := time.Since(start)
		results["hash_duration_ms"] = float64(hashDuration.Milliseconds())
		results["hash_throughput"] = float64(iterations) / hashDuration.Seconds()
		cmd.PrintErrln(fmt.Sprintf("  %d hashes in %v (%.0f hashes/s)", iterations, hashDuration.Round(time.Millisecond), results["hash_throughput"]))
	}

	if benchSuite == "all" || benchSuite == "graph" {
		cmd.PrintErrln("effective-next: benchmarking dependency graph...")
		g := graph.New()
		nodeCount := 10000
		start := time.Now()
		for i := 0; i < nodeCount; i++ {
			g.AddNode(graph.Node{
				ID:   fmt.Sprintf("node-%d", i),
				Path: fmt.Sprintf("src/file%d.ts", i),
			})
			if i > 0 {
				g.AddEdge(fmt.Sprintf("node-%d", i-1), fmt.Sprintf("node-%d", i), graph.EdgeStaticImport, false)
			}
		}
		addDuration := time.Since(start)
		results["graph_add_ms"] = float64(addDuration.Milliseconds())

		start = time.Now()
		_, err := g.TopologicalSort()
		sortDuration := time.Since(start)
		results["graph_sort_ms"] = float64(sortDuration.Milliseconds())
		if err != nil {
			return fmt.Errorf("graph benchmark: %w", err)
		}
		cmd.PrintErrln(fmt.Sprintf("  added %d nodes in %v, topological sort in %v", nodeCount, addDuration.Round(time.Millisecond), sortDuration.Round(time.Millisecond)))
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	results["heap_alloc_mb"] = float64(memStats.HeapAlloc) / (1024 * 1024)
	results["sys_mb"] = float64(memStats.Sys) / (1024 * 1024)
	cmd.PrintErrln(fmt.Sprintf("\nMemory: heap=%.1f MB, sys=%.1f MB", results["heap_alloc_mb"], results["sys_mb"]))

	if benchOutput != "" {
		if benchFormat == "json" {
			data := fmt.Sprintf(`{"benchmarks":%v,"memory":{"heap_mb":%.1f,"sys_mb":%.1f}}`,
				results, results["heap_alloc_mb"], results["sys_mb"])
			if err := os.WriteFile(benchOutput, []byte(data), 0o644); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
		} else {
			content := fmt.Sprintf("effectiveNext Benchmark Report\n=======================\n")
			for k, v := range results {
				content += fmt.Sprintf("%-25s %.2f\n", k, v)
			}
			if err := os.WriteFile(benchOutput, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
		}
		cmd.Println(fmt.Sprintf("effective-next: report written to %s", benchOutput))
	}

	return nil
}
