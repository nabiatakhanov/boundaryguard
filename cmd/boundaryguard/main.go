// Command boundaryguard checks architecture policies in the current Go module.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"boundaryguard.local/boundaryguard/internal/analyzers"
	"boundaryguard.local/boundaryguard/internal/configuration"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// singlechecker owns flag parsing, package loading and the analysis protocol.
// Its native JSON mode returns 0 for diagnostics, so a small parent process
// inspects that same JSON and restores useful CI exit status (without rewriting it).
func main() {
	if jsonEnabled(os.Args[1:]) && os.Getenv("BOUNDARYGUARD_JSON_CHILD") != "1" {
		os.Exit(jsonDriver())
	}
	path, help, err := configArgument(os.Args[1:])
	if err != nil {
		fail(err, 2)
	}
	var cfg *configuration.Config
	module := ""
	if help {
		cfg, err = configuration.Parse([]byte(`{"version":1,"layers":[],"dependency_rules":[],"forbidden_symbols":[]}`))
	} else {
		cfg, err = configuration.Load(path)
		if err == nil {
			module, err = currentModule()
		}
	}
	if err != nil {
		fail(err, 2)
	}
	a := analyzers.New(cfg, module)
	a.Flags.String("config", path, "required JSON policy path, relative to invocation directory")
	singlechecker.Main(a)
}

func fail(err error, code int) {
	fmt.Fprintln(os.Stderr, "boundaryguard:", err)
	os.Exit(code)
}

func configArgument(args []string) (path string, help bool, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		for _, unsupported := range []string{"fix", "diff"} {
			name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
			if name == unsupported {
				return "", false, fmt.Errorf("-%s is unsupported: architecture diagnostics require a design decision", unsupported)
			}
			if value, ok := strings.CutPrefix(name, unsupported+"="); ok {
				if enabled, e := strconv.ParseBool(value); e == nil && enabled {
					return "", false, fmt.Errorf("-%s is unsupported: architecture diagnostics require a design decision", unsupported)
				}
			}
		}
		if arg == "-h" || arg == "--help" || arg == "-help" {
			help = true
		}
		if arg == "-config" || arg == "--config" {
			if i+1 == len(args) || strings.HasPrefix(args[i+1], "-") {
				return "", false, fmt.Errorf("-config requires a path")
			}
			i++
			path = args[i]
		} else if strings.HasPrefix(arg, "-config=") || strings.HasPrefix(arg, "--config=") {
			path = strings.SplitN(arg, "=", 2)[1]
		}
	}
	if path == "" && !help {
		return "", false, fmt.Errorf("-config is required (use -h for help)")
	}
	return path, help, nil
}

func currentModule() (string, error) {
	cmd := exec.Command("go", "list", "-m", "-json")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("discover main module: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var module struct{ Path, GoMod string }
	if err := json.Unmarshal(out, &module); err != nil {
		return "", fmt.Errorf("decode main module: %w", err)
	}
	if module.Path == "" || module.GoMod == "" || module.GoMod == os.DevNull {
		return "", fmt.Errorf("run inside a Go module; workspace-wide and GOPATH CLI analysis are unsupported")
	}
	return module.Path, nil
}

func jsonEnabled(args []string) bool {
	enabled := false
	for _, arg := range args {
		arg = strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
		switch {
		case arg == "json":
			enabled = true
		case strings.HasPrefix(arg, "json="):
			if value, err := strconv.ParseBool(strings.TrimPrefix(arg, "json=")); err == nil {
				enabled = value
			}
		}
	}
	return enabled
}

func jsonDriver() int {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), "BOUNDARYGUARD_JSON_CHILD=1")
	cmd.Stderr = os.Stderr
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	if _, writeErr := os.Stdout.Write(stdout.Bytes()); writeErr != nil {
		return 1
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if stdout.Len() == 0 { // e.g. driver help
		return 0
	}
	return jsonStatus(stdout.Bytes())
}

func jsonStatus(data []byte) int {
	var packages map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &packages); err != nil {
		fmt.Fprintln(os.Stderr, "boundaryguard: invalid driver JSON:", err)
		return 1
	}
	code := 0
	for _, checks := range packages {
		for _, result := range checks {
			var diagnostics []json.RawMessage
			if json.Unmarshal(result, &diagnostics) == nil {
				if len(diagnostics) > 0 {
					code = 3
				}
			} else {
				// Driver analyzer failures are objects containing an error.
				return 1
			}
		}
	}
	return code
}
