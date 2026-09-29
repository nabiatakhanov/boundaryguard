// Command demo builds BoundaryGuard and checks both self-contained shop modules.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	temporary, err := os.MkdirTemp("", "boundaryguard-demo-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	name := "boundaryguard"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(temporary, name)
	build := exec.Command("go", "build", "-o", binary, "./cmd/boundaryguard")
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("build BoundaryGuard: %w", err)
	}
	for _, variant := range []string{"clean", "violating"} {
		dir := filepath.Join(root, "examples", "shop", variant)
		compile := exec.Command("go", "build", "./...")
		compile.Dir = dir
		compile.Stdout, compile.Stderr = os.Stdout, os.Stderr
		if err := compile.Run(); err != nil {
			return fmt.Errorf("compile %s: %w", variant, err)
		}
		command := exec.Command(binary, "-config", "boundaryguard.json", "./...")
		command.Dir = dir
		output, runErr := command.CombinedOutput()
		fmt.Printf("--- shop/%s ---\n%s", variant, output)
		code := 0
		if runErr != nil {
			if exit, ok := runErr.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				return fmt.Errorf("run %s: %w", variant, runErr)
			}
		}
		wantCode := 0
		if variant == "violating" {
			wantCode = 3
		}
		if code != wantCode {
			return fmt.Errorf("shop/%s exited %d, want %d", variant, code, wantCode)
		}
		if variant == "violating" {
			for _, diagnostic := range []string{"BG001 [domain-no-storage]", "BG003 [domain-no-storage]", "BG002 [transport-no-global-get]"} {
				if !strings.Contains(string(output), diagnostic) {
					return fmt.Errorf("shop/violating missing %q", diagnostic)
				}
			}
		} else if len(output) != 0 {
			return fmt.Errorf("shop/clean produced unexpected output: %s", output)
		}
		fmt.Printf("expected exit %d verified\n", code)
	}
	return nil
}
