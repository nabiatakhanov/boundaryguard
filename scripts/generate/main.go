// Command generate creates a new synthetic module without overwriting files.
package main

import (
	"boundaryguard.local/boundaryguard/internal/synthetic"
	"flag"
	"log"
)

func main() {
	dir := flag.String("out", "", "new output directory (parent must exist)")
	n := flag.Int("packages", 32, "number of packages")
	fanout := flag.Int("fanout", 2, "forward edges per package")
	flag.Parse()
	if *dir == "" {
		log.Fatal("-out is required")
	}
	if err := synthetic.Generate(*dir, *n, *fanout); err != nil {
		log.Fatal(err)
	}
}
