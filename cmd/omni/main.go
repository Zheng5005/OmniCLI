package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	resume := flag.Bool("resume", false, "Resume the last active session")
	flag.Parse()

	if *resume {
		fmt.Println("Resuming last session...")
	}

	fmt.Println("OmniCLI starting...")

	// TODO: Load config, init history, create agent, start TUI
	_ = resume
	os.Exit(0)
}
