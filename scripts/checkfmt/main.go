// Command checkfmt makes gofmt's file listing a failing CI check.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if err := check("."); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func check(root string) error {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "artifacts" || entry.Name() == "bin" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("list Go sources: %w", err)
	}
	// An empty argument list would make gofmt read from stdin. Bound each
	// invocation so fixture growth does not exceed Windows' command-line limit.
	for start := 0; start < len(files); start += 64 {
		end := min(start+64, len(files))
		args := append([]string{"-l"}, files[start:end]...)
		output, err := exec.Command("gofmt", args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("gofmt: %w\n%s", err, output)
		}
		if len(output) != 0 {
			return fmt.Errorf("run gofmt on:\n%s", output)
		}
	}
	return nil
}
