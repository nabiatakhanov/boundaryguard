// Command bench measures standard package loading separately from checker execution.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"boundaryguard.local/boundaryguard/internal/analyzers"
	"boundaryguard.local/boundaryguard/internal/configuration"
	"boundaryguard.local/boundaryguard/internal/synthetic"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

type sample struct {
	Packages    int     `json:"packages"`
	Fanout      int     `json:"fanout"`
	Repeat      int     `json:"repeat"`
	LoadMS      float64 `json:"load_ms"`
	AnalysisMS  float64 `json:"analysis_ms"`
	WallMS      float64 `json:"process_wall_ms"`
	CPUMS       float64 `json:"process_cpu_ms"`
	PeakBytes   uint64  `json:"peak_process_bytes"`
	Diagnostics int     `json:"diagnostics"`
}

func main() {
	worker := flag.String("worker", "", "internal: module to measure")
	sizes := flag.String("sizes", "8,32,96", "comma-separated package counts")
	fanout := flag.Int("fanout", 2, "edges per package")
	repeats := flag.Int("repeats", 2, "measured repetitions after one warm-up")
	out := flag.String("out", "artifacts/bench", "output directory for a new timestamped run")
	flag.Parse()
	if *worker != "" {
		if err := measure(*worker); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *repeats < 1 {
		log.Fatal("repeats must be positive")
	}
	if err := os.MkdirAll(*out, 0755); err != nil {
		log.Fatal(err)
	}
	runDir := filepath.Join(*out, "run-"+time.Now().UTC().Format("20060102T150405.000000000Z"))
	if err := os.Mkdir(runDir, 0755); err != nil {
		log.Fatal(err)
	}
	abs, err := filepath.Abs(runDir)
	if err != nil {
		log.Fatal(err)
	}
	var results []sample
	for _, text := range strings.Split(*sizes, ",") {
		n, err := strconv.Atoi(text)
		if err != nil {
			log.Fatal(err)
		}
		dir := filepath.Join(abs, fmt.Sprintf("graph-%d", n))
		if err := synthetic.Generate(dir, n, *fanout); err != nil {
			log.Fatal(err)
		}
		for repeat := 0; repeat <= *repeats; repeat++ {
			s, err := runWorker(dir)
			if err != nil {
				log.Fatal(err)
			}
			s.Packages = n
			s.Fanout = *fanout
			s.Repeat = repeat
			if repeat > 0 {
				results = append(results, s)
			}
			data, _ := json.MarshalIndent(s, "", "  ")
			if err := os.WriteFile(filepath.Join(abs, fmt.Sprintf("raw-%d-%d.json", n, repeat)), append(data, '\n'), 0644); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("packages=%d repeat=%d load=%.2fms analysis=%.2fms wall=%.2fms cpu=%.2fms peak=%d bytes\n", n, repeat, s.LoadMS, s.AnalysisMS, s.WallMS, s.CPUMS, s.PeakBytes)
		}
	}
	goVersion, err := exec.Command("go", "version").Output()
	if err != nil {
		log.Fatal(err)
	}
	cache, err := exec.Command("go", "env", "GOCACHE", "GOMODCACHE").Output()
	if err != nil {
		log.Fatal(err)
	}
	cachePaths := strings.Split(strings.TrimSpace(string(cache)), "\n")
	for i := range cachePaths {
		cachePaths[i] = strings.TrimSuffix(cachePaths[i], "\r")
	}
	build, ok := debug.ReadBuildInfo()
	if !ok {
		log.Fatal("build information unavailable")
	}
	metadata := map[string]any{"measured_at": time.Now().UTC().Format(time.RFC3339), "go": strings.TrimSpace(string(goVersion)), "build_info": build.String(), "cache_paths": cachePaths, "os": runtime.GOOS, "arch": runtime.GOARCH, "logical_cpus": runtime.NumCPU(), "cache": "warm: discarded repeat 0 for each generated graph; no cache eviction", "memory_scope": "worker process only; Windows sampled peak working set, Unix wait4 maxrss; excludes go command children", "cpu_scope": "worker user+system CPU only, excludes go command children", "analysis_scope": "checker.Analyze including scheduling and fact transport; no printing or package loading", "samples": results}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(abs, "results.json"), append(data, '\n'), 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Results:", filepath.Join(abs, "results.json"))
}

func measure(dir string) error {
	cfg, err := configuration.Load(filepath.Join(dir, "boundaryguard.json"))
	if err != nil {
		return err
	}
	start := time.Now()
	pkgs, err := packages.Load(&packages.Config{Dir: dir, Mode: packages.LoadAllSyntax | packages.NeedModule, Tests: true}, "./...")
	if err != nil {
		return err
	}
	if len(pkgs) == 0 || packages.PrintErrors(pkgs) > 0 {
		return fmt.Errorf("load failed")
	}
	loaded := time.Now()
	a := analyzers.New(cfg, "bench.test/graph")
	graph, err := checker.Analyze([]*analysis.Analyzer{a}, pkgs, nil)
	if err != nil {
		return err
	}
	done := time.Now()
	count := 0
	for act := range graph.All() {
		if act.Err != nil {
			return act.Err
		}
		if act.IsRoot {
			count += len(act.Diagnostics)
		}
	}
	return json.NewEncoder(os.Stdout).Encode(sample{LoadMS: float64(loaded.Sub(start)) / float64(time.Millisecond), AnalysisMS: float64(done.Sub(loaded)) / float64(time.Millisecond), Diagnostics: count})
}

func runWorker(dir string) (sample, error) {
	var result sample
	exe, err := os.Executable()
	if err != nil {
		return result, err
	}
	cmd := exec.Command(exe, "-worker", dir)
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	if err := cmd.Start(); err != nil {
		return result, err
	}
	peak, err := waitMeasured(cmd)
	if err != nil {
		return result, fmt.Errorf("worker: %w: %s", err, stderr.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return result, err
	}
	result.WallMS = float64(time.Since(start)) / float64(time.Millisecond)
	result.CPUMS = float64(cmd.ProcessState.UserTime()+cmd.ProcessState.SystemTime()) / float64(time.Millisecond)
	result.PeakBytes = peak
	return result, nil
}
