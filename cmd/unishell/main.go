package main

import (
	"fmt"
	"os"
)

var (
	version = "development"
	commit  = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("uniShell %s (%s)\n", version, commit)
		return
	}

	fmt.Printf("uniShell\nVersion: %s\n", version)
}
